#!/usr/bin/env python3
"""Frozen closed-world pilot evaluation. No writes to the original pilot."""
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
import csv
import datetime
import difflib
import gzip
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parent
SDK = Path(os.environ.get('HERTA_SDK', '/home/ubuntu/repos/HertaSDK')).resolve()
PILOT = Path(os.environ.get('HERTA_PILOT', '/home/ubuntu/work/pilot')).resolve()
RUNS = ['H1', 'H2', 'H3', 'R', 'R2']
TARGET = 'ff3f185ba7c4df825791420b73508d659ca0a432'
ROWS = []

def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()

def write(p, s):
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(s)

def js(p, obj):
    write(p, json.dumps(obj, indent=2) + '\n')

def command(args, cwd=SDK, timeout=35):
    env = dict(os.environ, GOWORK='off')
    try:
        p = subprocess.run([str(x) for x in args], cwd=cwd, env=env,
                           stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                           text=True, timeout=timeout)
        return p.returncode, p.stdout
    except subprocess.TimeoutExpired as e:
        out = e.stdout or b''
        if isinstance(out, bytes): out = out.decode(errors='replace')
        return 124, out + '\nPROCESS WATCHDOG EXPIRED\n'

def sources():
    return {**{f'H{i}': PILOT/f'results/herta-{i}/bundle-herta' for i in (1,2,3)},
            'R': PILOT/'results/ref-herta/oracle2-workspace',
            'R2': PILOT/'results/ref2-herta/workspace'}

STD = [('ErrUnavailable','transient'), ('ErrTimeout','uncertain')]
PERM_CTX = [('ContextDeadlineExceeded','permanent'), ('ContextCanceled','permanent')]

def classifier(run, task, op):
    if task == 't1':
        if run == 'H1':
            return [('ErrTimeout','uncertain'), ('ContextDeadlineExceeded','uncertain'),
                    ('ErrDeclined','permanent'), ('ContextCanceled','permanent'), ('ErrUnavailable','transient')]
        if run == 'H2': return STD + PERM_CTX
        if run == 'H3': return [('ErrTimeout','uncertain')] + PERM_CTX[::-1] + [('ErrUnavailable','transient')]
        if run == 'R2': return STD[::-1]
        return STD
    if op == 'reserve':
        if run == 'H2': return PERM_CTX + STD
        if run == 'H3': return PERM_CTX[::-1] + STD[::-1]
    return STD

def edges(run, task, op):
    if task == 't3' and op == 'checkout': return ['reserve','charge']
    if task == 't4':
        if op == 'transfer':
            return ['move_funds'] if run == 'H1' else ['debit'] if run == 'H2' else ['debit','credit']
        if run == 'H1' and op == 'move_funds': return ['debit','credit']
    return []

def migrate(text, run, task):
    aliases = set()
    def opedit(m):
        name, body = m.group(1), m.group(2)
        additions = []
        if re.search(r'\bretry\b', body):
            arms = classifier(run, task, name)
            aliases.update(n for n, _ in arms if n.startswith('Context'))
            additions.append('classify { ' + ', '.join(f'{n} => {o}' for n,o in arms) + ', _ => permanent }')
        calls = edges(run, task, name)
        if calls: additions.append('calls ' + ', '.join(calls))
        return m.group(0) + ('\n    ' + '\n    '.join(additions) + '\n' if additions else '')
    # Capture an op contract up to its matching closing brace, including nested
    # resource/retry braces. Only insert immediately before that closing brace.
    inserts = []
    for m in re.finditer(r'\bop main::(\w+)\(', text):
        start = text.index('{', m.end())
        depth, end = 1, start+1
        while depth:
            depth += (text[end] == '{') - (text[end] == '}')
            end += 1
        name, body = m.group(1), text[start+1:end-1]
        additions = []
        if re.search(r'\bretry\b', body):
            arms = classifier(run, task, name)
            aliases.update(n for n,_ in arms if n.startswith('Context'))
            additions.append('classify { ' + ', '.join(f'{n} => {o}' for n,o in arms) + ', _ => permanent }')
        calls = edges(run, task, name)
        if calls: additions.append('calls ' + ', '.join(calls))
        if additions: inserts.append((end-1, '\n    ' + '\n    '.join(additions) + '\n'))
    for pos, insertion in reversed(inserts): text = text[:pos] + insertion + text[pos:]
    alias = ''
    if aliases:
        alias = f'package {task}\n\nimport "context"\n\n' + '\n'.join(
            f'var {n} = context.{n.removeprefix("Context")}' for n in sorted(aliases)) + '\n'
    return text, alias

def prepare():
    if (ROOT/'freeze.json').exists(): raise SystemExit('Refusing to overwrite frozen migrations')
    inventory = {}
    for run, source in sources().items():
        for i in range(1,5):
            task = f't{i}'
            dest = ROOT/'inputs'/run/task
            for p in sorted((source/task).iterdir()):
                if p.suffix == '.go' or p.name == 'shop.he':
                    if p.name in ('oracle_test.go','harder_test.go'): continue
                    dest.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(p, dest/p.name)
                    inventory[str(p)] = sha(p)
            text, alias = migrate((dest/'shop.he').read_text(), run, task)
            model = ROOT/'models'/run/task/'shop.he'
            write(model, text)
            if alias: write(model.with_name('closed_context.go'), alias)
            write(model.with_name('migration.diff'), ''.join(difflib.unified_diff(
                (dest/'shop.he').read_text().splitlines(True), text.splitlines(True),
                fromfile=f'{run}/{task}/original.he', tofile=f'{run}/{task}/migrated.he')))
    for i in range(1,5):
        for name in ('oracle_test.go','harder_test.go'):
            p = sources()['R']/f't{i}'/name
            if name == 'oracle_test.go': assert p.read_bytes() == (PILOT/f'oracle/t{i}'/name).read_bytes()
            assert p.read_bytes() == (sources()['R2']/f't{i}'/name).read_bytes()
            d = ROOT/'oracles'/f't{i}'/name
            d.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(p, d)
            inventory[str(p)] = sha(p)
    js(ROOT/'source-inventory.json', inventory)
    frozen = [ROOT/'migration-rules.md', ROOT/'source-inventory.json']
    frozen += sorted((ROOT/'inputs').rglob('*')) + sorted((ROOT/'models').rglob('*')) + sorted((ROOT/'oracles').rglob('*'))
    manifest = {str(p.relative_to(ROOT)): sha(p) for p in frozen if p.is_file()}
    js(ROOT/'freeze.json', {'utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
                            'sdk_commit':TARGET, 'files':manifest})
    write(ROOT/'models.sha256', ''.join(f'{v}  {k}\n' for k,v in manifest.items() if k.startswith('models/')))
    print(f'Frozen {len(manifest)} files; all 20 migrations frozen before oracle execution', flush=True)

def verify():
    f = json.loads((ROOT/'freeze.json').read_text())
    for rel, digest in f['files'].items(): assert sha(ROOT/rel) == digest, rel
    return f

def log(phase, run, task, stage, rc, output):
    p = ROOT/'logs'/phase/run/task/(stage+'.log.gz')
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_bytes(gzip.compress(output.encode(), mtime=0))
    events = []
    for line in output.splitlines():
        try:
            event = json.loads(line)
            if isinstance(event, dict): events.append(event)
        except (ValueError, TypeError): pass
    text = ''.join(e.get('Output','') for e in events) if events else output
    guards = [n for n,s in [('ErrUndeclaredCall','undeclared call'),('ErrKeyReacquired','serialize key reacquired')] if s in text]
    failures = [e['Test'] for e in events if e.get('Action')=='fail' and e.get('Test')]
    row = {'phase':phase,'run':run,'task':task,'stage':stage,'exit':rc,
           'codes':';'.join(sorted(set(re.findall(r'HC\d{3}', text)))),
           'guards':';'.join(guards),'failures':';'.join(failures),
           'log':str(p.relative_to(ROOT)), 'excerpt':'\n'.join(line for line in text.splitlines()
               if any(x in line for x in ['--- FAIL:', 'hertasdk:', 'undefined:', 'declared and not used', 'idempotent inventory', 'arbitrary account', 'uncertain charge', 'WATCHDOG']))[:12000]}
    ROWS.append(row)
    return row

def recorded(phase,run,task,stage,args,cwd=SDK):
    p = ROOT/'logs'/phase/run/task/(stage+'.log.gz')
    if p.exists():
        out = gzip.decompress(p.read_bytes()).decode()
        if stage in ('own','oracle','oracle2'):
            events = [json.loads(x) for x in out.splitlines() if x.startswith('{')]
            rc = int(any(e.get('Action') == 'fail' for e in events))
        else:
            rc = int(bool(re.search(r'HC\d{3}',out)))
        log(phase,run,task,stage,rc,out)
        return rc,out
    rc,out=command(args,cwd)
    log(phase,run,task,stage,rc,out)
    return rc,out

def workspace(phase, run, task, model=None):
    dest = ROOT/'workspaces'/phase/run/task
    if dest.exists(): shutil.rmtree(dest)
    shutil.copytree(ROOT/'inputs'/run/task, dest)
    write(dest/'go.mod', f'module closedeval/{run}/{task}\n\ngo 1.26.0\nrequire github.com/arahe-dev/hertasdk v0.0.0\nreplace github.com/arahe-dev/hertasdk => {SDK}\n')
    if model:
        shutil.copy2(model/'shop.he', dest/'shop.he')
        for p in model.glob('*.go'): shutil.copy2(p,dest/p.name)
    return dest

def tests(phase, run, task, dest, stages=('own','oracle','oracle2')):
    base = sorted(p.name for p in dest.glob('*.go') if not p.name.endswith('_test.go'))
    own = sorted(p.name for p in dest.glob('*_test.go'))
    for stage in stages:
        extras = own if stage == 'own' else ['oracle_test.go'] + (['harder_test.go'] if stage == 'oracle2' else [])
        if stage != 'own':
            for name in extras: shutil.copy2(ROOT/'oracles'/task/name, dest/name)
        recorded(phase,run,task,stage,['go','test','-json','-count=1','-timeout=12s'] + base + extras,dest)

def evaluate_one(job):
    phase,run,task = job
    dest = workspace(phase,run,task, ROOT/'models'/run/task if phase == 'B' else None)
    if phase != 'baseline':
        rc, out = recorded(phase,run,task,'check',[ROOT/'bin/herta','check','-closed',dest/'shop.he'])
        rc, out = recorded(phase,run,task,'build',[ROOT/'bin/herta','build','-closed','-pkg',task,dest/'shop.he'])
        if rc:
            rc, out = command([ROOT/'bin/force',dest/'shop.he',task])
            log(phase,run,task,'diagnostic-bypass-generation',rc,out if rc else 'GenerateClosed without Check (explicit evaluation bypass)\n')
        if rc:
            print(f'{phase} {run}/{task} generation failed',flush=True)
            return
        write(dest/'ops_gen.go',out)
    tests(phase,run,task,dest)
    print(f'{phase} {run}/{task} finished', flush=True)

def reorder(text):
    def replace(m):
        arms = [x.strip() for x in m.group(1).split(',')]
        transient = next((i for i,a in enumerate(arms) if '=> transient' in a), None)
        uncertain = next((i for i,a in enumerate(arms) if '=> uncertain' in a), None)
        if transient is not None and uncertain is not None and transient < uncertain:
            arm = arms.pop(uncertain)
            arms.insert(transient,arm)
        return 'classify { ' + ', '.join(arms) + ' }'
    return re.sub(r'classify\s*\{([^}]*)\}',replace,text)

def authority():
    flagged = [(r,t) for r in RUNS for t in ('t1','t2','t3','t4')
               if any(x['phase']=='B' and x['run']==r and x['task']==t and 'HC010' in x['codes'] for x in ROWS)]
    for run,task in flagged:
        model = ROOT/'variants/order'/run/task
        shutil.copytree(ROOT/'models'/run/task,model,dirs_exist_ok=True)
        before = (model/'shop.he').read_text()
        after = reorder(before)
        assert before != after
        write(model/'shop.he',after)
        dest = workspace('C-order',run,task,model)
        assert (dest/'glue.go').read_bytes() == (ROOT/'inputs'/run/task/'glue.go').read_bytes()
        for stage,args in [('check',['check','-closed']),('build',['build','-closed','-pkg',task])]:
            rc,out=command([ROOT/'bin/herta']+args+[dest/'shop.he'])
            log('C-order',run,task,stage,rc,out)
        if rc == 0:
            write(dest/'ops_gen.go',out)
            tests('C-order',run,task,dest,('oracle2',))
    for task in ('t3','t4'):
        model = ROOT/'variants/drop-calls/H3'/task
        shutil.copytree(ROOT/'models/H3'/task,model,dirs_exist_ok=True)
        text = (model/'shop.he').read_text()
        old,new = ('calls reserve, charge','calls charge') if task=='t3' else ('calls debit, credit','calls credit')
        assert old in text
        write(model/'shop.he', text.replace(old,new))
        guard_mutation('C-drop','H3',task,model,'ErrUndeclaredCall')
    model = ROOT/'variants/same-key/H3/t4'
    shutil.copytree(ROOT/'models/H3/t4',model,dirs_exist_ok=True)
    text=(model/'shop.he').read_text()
    needle='return Unit{}, svc.transferHost(ctx, from, to, cents)'
    assert needle in text
    injection='if _, err := svc.debit.Do(ctx, MainDebitInput{Svc: svc, User: from, UserKey: from_key, Cents: cents}); err != nil { return Unit{}, err }\n'
    write(model/'shop.he',text.replace(needle,injection+needle))
    guard_mutation('C-key','H3','t4',model,'ErrKeyReacquired')

def guard_mutation(phase,run,task,model,want):
    dest=workspace(phase,run,task,model)
    rc,out=command([ROOT/'bin/herta','check','-closed',dest/'shop.he'])
    log(phase,run,task,'check',rc,out)
    rc,out=command([ROOT/'bin/herta','build','-closed','-pkg',task,dest/'shop.he'])
    log(phase,run,task,'build',rc,out)
    if rc:
        rc,out=command([ROOT/'bin/force',dest/'shop.he',task])
        log(phase,run,task,'diagnostic-bypass-generation',rc,'GenerateClosed without Check\n' if not rc else out)
    assert rc == 0,out
    write(dest/'ops_gen.go',out)
    request = '_, err := s.Checkout(ctx,"A","sku",1)' if task=='t3' else 'err := s.Transfer(ctx,"A","B",1)'
    test=f'''package {task}
import("context";"errors";"testing";"time";"github.com/arahe-dev/hertasdk")
func TestClosedMutation(t *testing.T) {{
 b:=&Backend{{}}; b.SetBalance("A",10); s,e:=New(b);if e!=nil{{t.Fatal(e)}}
 ctx,cancel:=context.WithTimeout(context.Background(),250*time.Millisecond);defer cancel()
 started:=time.Now(); {request}
 if !errors.Is(err,hertasdk.{want})||hertasdk.OutcomeOf(err)!=hertasdk.Permanent{{t.Fatalf("wanted {want}: %v",err)}}
 if elapsed:=time.Since(started);elapsed>=200*time.Millisecond{{t.Fatalf("guard too slow: %v",elapsed)}}
 t.Logf("expected {want}: %v; elapsed=%v",err,time.Since(started))
}}
'''
    write(dest/'mutation_test.go',test)
    write(model/'mutation_test.go.txt',test)
    base=sorted(p.name for p in dest.glob('*.go') if not p.name.endswith('_test.go'))
    rc,out=command(['go','test','-json','-count=1','-timeout=3s']+base+['mutation_test.go'],dest)
    log(phase,run,task,'mutation',rc,out)
    tests(phase,run,task,dest,('oracle2',))

def main():
    mode=sys.argv[1]
    if mode=='prepare': prepare();return
    verify()
    (ROOT/'bin').mkdir(exist_ok=True)
    for args in (['go','build','-o',ROOT/'bin/herta','./herta/cmd/herta'],
                 ['go','build','-o',ROOT/'bin/force',ROOT/'force.go']):
        rc,out=command(args);assert rc==0,out
    if mode in ('run','resume'):
        if mode=='resume': ROWS.extend(json.loads((ROOT/'results.json').read_text()))
        for phase in (('A','B') if mode=='resume' else ('baseline','A','B')):
            with ThreadPoolExecutor(max_workers=4) as pool:
                list(pool.map(evaluate_one,[(phase,r,f't{i}') for r in RUNS for i in range(1,5)]))
            js(ROOT/'results.json',ROWS)
        authority()
        js(ROOT/'results.json',ROWS)
    elif mode=='authority':
        ROWS.extend(json.loads((ROOT/'results.json').read_text()))
        authority();js(ROOT/'results.json',ROWS)
    verify()
    with (ROOT/'results.csv').open('w',newline='') as f:
        w=csv.DictWriter(f,fieldnames=list(ROWS[0]));w.writeheader();w.writerows(ROWS)
    print('Frozen artifacts unchanged; wrote results.json/results.csv',flush=True)

if __name__=='__main__': main()
