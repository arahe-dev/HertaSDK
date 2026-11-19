# Frozen faithful migration rules

Written before checks or oracle execution in this evaluation. Target SDK and
compiler: ff3f185ba7c4df825791420b73508d659ca0a432.

1. Snapshot all 20 final programs and their own tests, excluding vendored SDKs.
   H1/H2/H3 use bundle-herta; R uses oracle2-workspace; R2 uses workspace.
   Never write to the pilot. Run against the target SDK with a local replace.
2. Preserve every original .he byte except insertions into operation contract
   blocks. For every retrying operation, insert classify copying its reachable
   Go classifier's ordered error checks. Split OR arms in source order. Preserve
   context checks, mapping untagged returns to permanent; append _ => permanent.
   Nil success is handled by the runtime, not a classify arm. Nonretrying ops
   need no classifier and keep legacy propagation (including unused scaffolding).
3. Charge t1: H1 timeout, deadline => uncertain; declined, canceled => permanent;
   unavailable => transient. H2 unavailable => transient, timeout => uncertain,
   deadline/canceled => permanent. H3 timeout => uncertain, canceled/deadline =>
   permanent, unavailable => transient. R unavailable then timeout. R2 timeout
   then unavailable. All use permanent fallback.
4. All t3 charge classifiers: unavailable => transient, timeout => uncertain.
   t3 reserve: H1/R/R2 the same; H2 deadline/canceled => permanent before that
   order; H3 canceled/deadline => permanent, timeout => uncertain, unavailable
   => transient. Preserve retry sets, timeouts, effects and resource claims.
5. Insert only actual nested governed edges: every checkout -> reserve, charge;
   H1 transfer -> move_funds and move_funds -> debit, credit; H2 transfer -> debit
   only (credit is external after transfer.Do); H3/R/R2 transfer -> debit, credit.
   No external Service-entry .Do is a nested edge. No backend call is an op edge.
6. Context selectors cannot be spelled in this prototype's classify grammar.
   Add behavior-neutral ContextCanceled/ContextDeadlineExceeded Go aliases in a
   separate harness file only where required; leave all original Go untouched.
   These aliases are explicitly a harness accommodation, not an .he-only edit.
7. Freeze all 20 migrated .he files and alias files, original evidence, and
   the rules (SHA-256 plus UTC timestamp) before ANY oracle run. Do not revise
   migrations after results. Authority variants are separate post-freeze copies.
8. 'Pass check' means eliminate closed-only migration diagnostics, not repair
   preexisting semantic hazards: faithful classification order can and should
   retain HC010. Do not silently repair it to obtain a green build.
9. Run the real check -closed and build -closed for every model. A diagnostic
   blocks CLI generation. Record BLOCKED distinctly, then supplement with the
   same GenerateClosed API without Check so blocked programs can be exercised.
   This diagnostic-bypass output is NOT a successful herta build or acceptance.
10. Fresh baseline: retained original generated Go + new SDK. Own tests, original
    oracle and combined original+oracle2 run separately, once each (-count=1).
    Use exact oracle_test.go/harder_test.go bytes from R's retained workspace,
    verifying original oracle bytes against pilot/oracle. Compile own tests
    separately from oracle files to avoid accidental test selection overlap.
    Every Go test has a 12s package timeout and 35s subprocess watchdog; logs
    preserve exit codes, named failures, diagnostics and observed guard errors.
11. C: reorder only uncertain/transient arms in flagged classifier blocks;
    leave Go classifiers byte-identical. Run combined oracle2. For H3/t3 and
    H3/t4 separately drop one calls edge; additionally change only H3/t4's
    transfer handler to pass from_key as the nested debit key, preserving all
    original body statements as well. Focused tests assert exact sentinel
    identity with errors.Is and use a 250ms caller timeout.
12. Confusion tables use fresh unmodified baseline oracle2 as the behavioral
    label, with a second mechanism-sensitive interpretation. Oracle passing
    does not prove absence of other hazards. Do not equate R/t3's order warning
    to its missing-uncertain-retry bug or R/t4's key-value collision to a static
    serialize-key proof.
