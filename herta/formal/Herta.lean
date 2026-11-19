import Std

/-!
Execution contract at base ffe32e6. Citations are repository-root Go file:line.
Time is Nat milliseconds; deadlines are absolute, attempts are zero-indexed.
We model admitted, sequential calls, not scheduling, panic, shutdown or budgets.
-/
namespace Herta

inductive Effect where
  | pure | idempotent | nonidempotent
  deriving DecidableEq, Repr

inductive Outcome where
  | success | transient | permanent | throttled | uncertain
  deriving DecidableEq, Repr

inductive RawError where
  | plain | canceled | deadline
  deriving DecidableEq, Repr

inductive Path where
  | legacy | closed
  deriving DecidableEq, Repr

structure Reply where
  duration : Nat
  applied : Bool
  error : Option RawError
  /-- Cooperative backend waits precisely until the effective context deadline. -/
  hangs : Bool := false
  /-- evidence.go: the adapter called Dispatch before this reply. Conservative
  default; DispatchBeforeApply below is the adapter's whole obligation. -/
  dispatched : Bool := true
  deriving Repr

abbrev Backend := Nat → Reply
abbrev Classifier := Nat → RawError → Option Outcome

structure Policy where
  effect : Effect
  maxAttempts : Nat := 1
  timeout : Nat := 0
  callerDeadline : Nat
  retryOn : List Outcome := []
  path : Path := .legacy
  /-- Zero is SDK behavior (attempt.go:5–7); positive values are an extension. -/
  backoff : Nat := 0
  /-- Policy.TrackDispatch (evidence.go). -/
  trackDispatch : Bool := false

def normalize : Outcome → Outcome
  | .success => .permanent
  | o => o

/-- policy.go:38–45; validate.go:69–88. Only validated retry behavior is modeled.
An invalid nonidempotent/Uncertain request cannot grant permission here. -/
def mayRetry (p : Policy) (o : Outcome) : Bool :=
  p.retryOn.contains o && (o == .transient ||
    (o == .uncertain && p.effect != .nonidempotent))

/-- outcome.go:106–114: legacy tag wins, even for cancellation/deadline.
Closed attempt.go:23–37: caller ending wins; own deadline bypasses classifier.
Closed classification normalizes Success (other invalid enum values are omitted).
The caller check here is only after a nonnil error; late success still wins. -/
def resolve (path : Path) (raw : RawError) (tag : Option Outcome)
    (ownExpired callerEnded : Bool) : Outcome :=
  match path with
  | .legacy => match tag with
    | some o => o
    | none => if ownExpired && raw != .canceled then .uncertain else .permanent
  | .closed =>
    if callerEnded then .permanent
    else if ownExpired then .uncertain
    else normalize (tag.getD .permanent)

/-- evidence.go:55–64, single operation: after Dispatch, Transient is no longer
evidence of "not applied". Nested ancestor marking is not modeled here. -/
def evidence (p : Policy) (r : Reply) (o : Outcome) : Outcome :=
  if p.trackDispatch && r.dispatched && o == .transient then .uncertain else o

structure Observation where
  elapsed : Nat
  applied : Nat
  outcome : Outcome
  ownExpired : Bool
  callerEnded : Bool
  deriving Repr

/-- attempt.go:52–61: expiry sampled at return, not at first timer event.
Noncooperative handlers may return late, including nil-error success.
For cooperative hangs the context ends at min(own deadline, caller deadline).
At equal deadlines caller wins, since ctx.Err()!=nil prevents own promotion. -/
def observe (p : Policy) (b : Backend) (c : Classifier) (i now : Nat) : Observation :=
  let r := b i
  let limit := if p.timeout == 0 then p.callerDeadline
    else min (now + p.timeout) p.callerDeadline
  let finish := if r.hangs then limit else now + r.duration
  let raw := if r.hangs then some RawError.deadline else r.error
  let callerEnded := decide (p.callerDeadline ≤ finish)
  let ownExpired := raw.isSome && p.timeout != 0 &&
    decide (now + p.timeout ≤ finish) && !callerEnded
  { elapsed := finish - now
    applied := if r.applied then 1 else 0
    outcome := match raw with
      | none => .success
      | some e => evidence p r (resolve p.path e (c i e) ownExpired callerEnded)
    ownExpired := ownExpired
    callerEnded := callerEnded }

structure Result where
  calls : Nat := 0
  applied : Nat := 0
  elapsed : Nat := 0
  outcome : Outcome := .permanent
  deriving Repr

def prepend (a : Observation) (delay : Nat) (r : Result) : Result :=
  { calls := 1 + r.calls, applied := a.applied + r.applied
    elapsed := a.elapsed + delay + r.elapsed, outcome := r.outcome }

/-- attempt.go:8–45. The caller precheck runs even between retries; the closed
post-handler caller check returns immediately. No retry after a nil error:
legacy hand-built Success tags are errors but never retryable, so counts agree.
Result.outcome abstracts error identity and promotion wrappers. -/
def loop (p : Policy) (b : Backend) (c : Classifier) : Nat → Nat → Nat → Result
  | 0, _, _ => {}
  | fuel + 1, i, now =>
    if now ≥ p.callerDeadline then {}
    else
      let a := observe p b c i now
      if fuel > 0 && mayRetry p a.outcome &&
          !(p.path == .closed && a.callerEnded) then
        prepend a p.backoff (loop p b c fuel (i + 1) (now + a.elapsed + p.backoff))
      else { calls := 1, applied := a.applied, elapsed := a.elapsed, outcome := a.outcome }

def run (p : Policy) (b : Backend) (c : Classifier) : Result :=
  loop p b c (max 1 p.maxAttempts) 0 0

def untagged : Classifier := fun _ _ => none
def transientTag : Classifier := fun _ _ => some .transient

theorem observation_applied_le_one (p : Policy) (b : Backend) (c : Classifier)
    (i now : Nat) : (observe p b c i now).applied ≤ 1 := by
  simp only [observe]
  split <;> simp_all
#print axioms observation_applied_le_one

/-- This explicit hypothesis is the trust assumption (informal "axiom") that
a raw Go classifier introduces: Transient promises the attempt did not execute
(docs/failure-semantics.md:62–65). It is NOT a Lean axiom. Own-deadline hygiene
alone is insufficient: a lying Transient tag before timeout also duplicates.
The condition covers both paths and all backend attempts/times. -/
def TransientHonest (p : Policy) (b : Backend) (c : Classifier) : Prop :=
  ∀ i now, (observe p b c i now).outcome = .transient →
    (observe p b c i now).applied = 0

def DeadlineHygiene (c : Classifier) : Prop :=
  ∀ i, c i .deadline = none ∨ c i .deadline = some .uncertain

theorem retry_nonidempotent (p : Policy) (he : p.effect = .nonidempotent)
    (o : Outcome) (h : mayRetry p o = true) : o = .transient := by
  cases o <;> simp_all [mayRetry]
#print axioms retry_nonidempotent

theorem loop_safety (p : Policy) (b : Backend) (c : Classifier)
    (he : p.effect = .nonidempotent) (honest : TransientHonest p b c)
    (fuel i now : Nat) : (loop p b c fuel i now).applied ≤ 1 := by
  induction fuel generalizing i now with
  | zero => simp [loop]
  | succ fuel ih =>
    simp only [loop]
    split
    · simp
    · split
      · rename_i h
        have hr : mayRetry p (observe p b c i now).outcome = true := by
          simp only [Bool.and_eq_true] at h
          exact h.1.2
        have hz := honest i now (retry_nonidempotent p he _ hr)
        simpa [prepend, hz] using ih (i + 1) (now + (observe p b c i now).elapsed + p.backoff)
      · exact observation_applied_le_one p b c i now
#print axioms loop_safety

/-- T1: deadline hygiene plus the Transient trust assumption imply safety for
ANY backend. The proof is stronger: honesty suffices without deadline hygiene,
since nonidempotent operations can only retry Transient (policy.go:40–44). -/
theorem T1 (p : Policy) (b : Backend) (c : Classifier)
    (he : p.effect = .nonidempotent) (_hu : .uncertain ∉ p.retryOn)
    (_hd : DeadlineHygiene c) (honest : TransientHonest p b c) :
    (run p b c).applied ≤ 1 := by
  exact loop_safety p b c he honest _ _ _
#print axioms T1

def twoAttempts : Policy :=
  { effect := .nonidempotent, maxAttempts := 2, timeout := 300
    callerDeadline := 1000, retryOn := [.transient] }

def appliesThenTimesOut : Backend := fun _ =>
  { duration := 301, applied := true, error := some .deadline }

/-- T2: legacy Go-side handler tag overrides Δ3 promotion. The backend applies,
ignores the context for 1 ms past its own deadline, then returns tagged error.
This is the pilot r1 failure mechanism, not closed Policy.Classify behavior. -/
theorem T2 : ∃ b : Backend, (run twoAttempts b transientTag).applied = 2 := by
  exact ⟨appliesThenTimesOut, by decide⟩
#print axioms T2

def appliesThenPlain : Backend := fun _ =>
  { duration := 1, applied := true, error := some .plain }

def liesBeforeDeadline : Classifier := fun _ e =>
  if e == .deadline then none else some .transient

/-- Necessity of the T1 trust assumption even with perfect deadline hygiene. -/
theorem T1_assumption_necessary : DeadlineHygiene liesBeforeDeadline ∧
    (run twoAttempts appliesThenPlain liesBeforeDeadline).applied = 2 := by
  constructor
  · intro i; exact Or.inl rfl
  · decide
#print axioms T1_assumption_necessary

def r1 : Policy :=
  { effect := .nonidempotent, maxAttempts := 4, timeout := 300
    callerDeadline := 1000, retryOn := [.transient] }

def hangsUntilContext : Backend := fun _ =>
  { duration := 0, applied := true, error := none, hangs := true }

/-- T3a: per-attempt 300ms reading, legacy Transient handler tags. Four calls:
300+300+300+100=1000ms; the last is ended by the 1000ms caller deadline.
This is not four full 300ms waits and not an invented end-to-end 300ms timer. -/
theorem T3_per_attempt : (run r1 hangsUntilContext transientTag).calls = 4 ∧
    (run r1 hangsUntilContext transientTag).elapsed = 1000 := by decide
#print axioms T3_per_attempt

/-- T3b: SDK default Δ3/Δ4 promotion, untagged own deadline -> Uncertain.
Nonidempotent retry-on-Transient stops after one 300ms call (<=550ms oracle
bound). This matches the oracle timing without changing timeout into a total
call budget: it is classification, not an alternative SDK deadline reading. -/
theorem T3_default : (run r1 hangsUntilContext untagged).calls = 1 ∧
    (run r1 hangsUntilContext untagged).elapsed = 300 ∧
    (run r1 hangsUntilContext untagged).elapsed ≤ 550 := by decide
#print axioms T3_default

/-- keys.go:29–60, operation.go:122–140: a nonempty key already held by the
outer call cannot be handed to its synchronous nested Wait call. The outer
retains it until the nested call returns. Without a deadline this blocks
forever; finite deadline returns failure, including deadline zero. -/
abbrev HeldKeys := List Nat

def nestedAcquire (held : HeldKeys) (key deadline : Nat) : Bool × Nat :=
  if held.contains key then (false, deadline) else (true, 0)

theorem T4_key (held : HeldKeys) (key deadline : Nat) (h : key ∈ held) :
    nestedAcquire held key deadline = (false, deadline) := by
  simp [nestedAcquire, h]
#print axioms T4_key

/-- Each outer invocation calls a successful nonidempotent inner once, then
loses/returns an untagged response after its own timeout. Reply.applied records
that inner effect (one, not an arbitrary external-effect multiplicity). -/
def nestedInner : Backend := fun _ =>
  { duration := 301, applied := true, error := some .deadline }

def outer (e : Effect) : Policy :=
  { twoAttempts with effect := e, retryOn := [.uncertain] }

/-- T4 composition witness: declaring the outer Pure/Idempotent does not make
its nested nonidempotent side effect safe to replay. Keys here are distinct.
This models the hazard, not acceptance by Herta's composition checker. -/
theorem T4_composition :
    (run (outer .pure) nestedInner untagged).applied > 1 ∧
    (run (outer .idempotent) nestedInner untagged).applied > 1 := by decide
#print axioms T4_composition

/-- T5 local retry rule for ANY backend and classifier: the closed path
structurally overrides classification on own-expiry (attempt.go:23–39).
An already-applied effect cannot be replayed *because of this attempt's own
deadline*. This is not whole-run safety against dishonest earlier Transients. -/
theorem T5 (p : Policy) (b : Backend) (c : Classifier) (i now : Nat)
    (hp : p.path = .closed) (he : p.effect = .nonidempotent)
    (_hu : .uncertain ∉ p.retryOn)
    (err : (if (b i).hangs then some RawError.deadline else (b i).error).isSome = true)
    (expired : (observe p b c i now).ownExpired = true) :
    mayRetry p (observe p b c i now).outcome = false := by
  cases hr : (if (b i).hangs then some RawError.deadline else (b i).error) with
  | none => simp [hr] at err
  | some e =>
    simp only [observe, hr] at expired ⊢
    rw [expired, hp]
    have safe : ∀ ended, mayRetry p (evidence p (b i) (resolve .closed e (c i e) true ended)) = false := by
      intro ended
      cases ended <;> simp [resolve, evidence, mayRetry, he] <;> split <;> simp [mayRetry, he]
    exact safe _
#print axioms T5

/-- Edge checks: a pre-ended caller makes no backend call; a late nil-error
return still succeeds; empty retry sets stop at the first plain failure.
These use attempt.go:10–21, not a hard preemption interpretation of Timeout. -/
theorem execution_edges :
    (run { r1 with callerDeadline := 0 } hangsUntilContext untagged).calls = 0 ∧
    (run r1 (fun _ => ⟨1100, true, none, false, true⟩) untagged).outcome = .success ∧
    (run { r1 with retryOn := [] } appliesThenPlain transientTag).calls = 1 ∧
    (run { r1 with maxAttempts := 0 } appliesThenPlain transientTag).calls = 1 := by decide
#print axioms execution_edges

/-- Closed-world counterpart of the exact T2 witness, in addition to T5's
universal local rule: any classifier response is bypassed on own expiry. -/
theorem T5_witness_blocked :
    (run { twoAttempts with path := .closed } appliesThenTimesOut transientTag).applied = 1 := by
  decide
#print axioms T5_witness_blocked

/-- The adapter obligation dispatch tracking replaces TransientHonest with:
every application happens after Dispatch. It says nothing about error labels. -/
def DispatchBeforeApply (b : Backend) : Prop :=
  ∀ i, (b i).applied = true → (b i).dispatched = true

theorem evidence_transient (p : Policy) (r : Reply) (o : Outcome)
    (ht : p.trackDispatch = true) (h : evidence p r o = .transient) :
    r.dispatched = false := by
  unfold evidence at h
  cases hd : r.dispatched
  · rfl
  · simp only [ht, hd, Bool.true_and] at h
    split at h <;> simp_all

theorem dispatch_honest (p : Policy) (b : Backend) (c : Classifier)
    (ht : p.trackDispatch = true) (hb : DispatchBeforeApply b) :
    TransientHonest p b c := by
  intro i now h
  have hnot : (b i).applied = false := by
    cases ha : (b i).applied
    · rfl
    · have hd := hb i ha
      simp only [observe] at h
      split at h
      · simp at h
      · simp [evidence_transient p (b i) _ ht h] at hd
  simp [observe, hnot]

/-- T6: with dispatch tracking, ANY classifier (including one that labels
every error Transient) cannot duplicate a nonidempotent effect, provided only
that the adapter dispatches before applying. -/
theorem T6 (p : Policy) (b : Backend) (c : Classifier)
    (he : p.effect = .nonidempotent) (ht : p.trackDispatch = true)
    (hb : DispatchBeforeApply b) : (run p b c).applied ≤ 1 :=
  loop_safety p b c he (dispatch_honest p b c ht hb) _ _ _
#print axioms T6

def appliesBeforeDispatch : Backend := fun _ =>
  { duration := 1, applied := true, error := some .plain, dispatched := false }

/-- The boundary study's mislabeled lost ack: two applications untracked,
one tracked; an adapter that applies before Dispatch still gets two. -/
theorem T6_witness :
    (run { twoAttempts with path := .closed } appliesThenPlain transientTag).applied = 2 ∧
    (run { twoAttempts with path := .closed, trackDispatch := true }
      appliesThenPlain transientTag).applied = 1 ∧
    (run { twoAttempts with path := .closed, trackDispatch := true }
      appliesBeforeDispatch transientTag).applied = 2 := by
  decide
#print axioms T6_witness

end Herta
