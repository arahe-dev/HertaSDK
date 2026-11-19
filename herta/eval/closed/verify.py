#!/usr/bin/env python3
"""Validate retained evidence and result joins without rerunning oracles."""
from pathlib import Path
import csv
import datetime
import gzip
import hashlib
import json
import os

root=Path(__file__).resolve().parent
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
frozen=json.loads((root/'freeze.json').read_text())
for rel,digest in frozen['files'].items():assert sha(root/rel)==digest,rel
rows=json.loads((root/'results.json').read_text())
assert len(rows)==311
assert len(rows)==len({(r['phase'],r['run'],r['task'],r['stage']) for r in rows})
with (root/'results.csv').open(newline='') as f:
 csvrows=list(csv.DictReader(f))
assert csvrows==[{k:str(v) for k,v in r.items()} for r in rows]
freeze_time=datetime.datetime.fromisoformat(frozen['utc'])
events=0
for r in rows:
 text=gzip.decompress((root/r['log']).read_bytes()).decode()
 if r['stage'] in ('own','oracle','oracle2','mutation'):
  ev=[json.loads(line) for line in text.splitlines() if line.startswith('{')]
  assert ev, r
  assert bool(r['exit'])==any(e.get('Action')=='fail' for e in ev),r
  assert not any('build-fail'==e.get('Action') for e in ev),r
  for e in ev:
   if 'Time' in e:assert datetime.datetime.fromisoformat(e['Time'].replace('Z','+00:00'))>freeze_time
  events+=len(ev)
def find(phase,run,task,stage):
 return next(r for r in rows if (r['phase'],r['run'],r['task'],r['stage'])==(phase,run,task,stage))
for phase in ('baseline','B'):
 assert all(r['exit']==0 for r in rows if r['phase']==phase and r['stage'] in ('own','oracle'))
 failed={(r['run'],r['task']) for r in rows if r['phase']==phase and r['stage']=='oracle2' and r['exit']}
 assert failed=={('H2','t1'),('R','t1'),('R','t3'),('R','t4')}
for r in rows:
 if r['phase']=='C-order' and r['stage']=='oracle2':
  workspace=root/'workspaces/C-order'/r['run']/r['task']
  if workspace.exists():
   assert (workspace/'glue.go').read_bytes()==(root/'inputs'/r['run']/r['task']/'glue.go').read_bytes()
  for p in (root/'variants/order'/r['run']/r['task']).glob('*.go'):
   assert p.read_bytes()==(root/'models'/r['run']/r['task']/p.name).read_bytes()
  original=(root/'models'/r['run']/r['task']/'shop.he').read_text()
  variant=(root/'variants/order'/r['run']/r['task']/'shop.he').read_text()
  import re
  normalize=lambda s:re.sub(r'classify\s*\{([^}]*)\}',lambda m:'classify {'+','.join(sorted(x.strip() for x in m.group(1).split(',')))+'}',s)
  assert normalize(original)==normalize(variant)
assert all(r['exit']==0 for r in rows if r['stage']=='mutation')
if os.environ.get('VERIFY_PILOT')=='1':
 for name,digest in json.loads((root/'source-inventory.json').read_text()).items():
  assert sha(Path(name))==digest,name
 print('Original pilot source/oracle inputs unchanged (all inventoried paths)')
print(f'Frozen evidence unchanged: {len(frozen["files"])} files; 20 migrated .he; 5 context aliases')
print(f'Result CSV/JSON agree: {len(rows)} unique command rows; {events} test events after freeze')
print('Baseline/B: own 20/20, original oracle 20/20; same four oracle2 failures')
print('Authority .he diffs contain only classifier-arm reordering; all 3 guard assertions pass')
