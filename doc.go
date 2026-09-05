// Package execution implements Herta V0: a minimal in-process execution
// fabric shared by heterogeneous application operations.
//
// One execution model, many operations: each Operation declares a Policy
// (Effect, resource Requirements, Admission mode, Timeout, SerializeKey,
// Retry) and the Runtime arbitrates admission against shared, finite
// resource budgets.
//
// Contract highlights (all enforced or documented here):
//
//   - Construction-time validation: unsafe or impossible Policies
//     (unknown resources, Units <= 0, Units > capacity, NonIdempotent +
//     retry-on-Uncertain) are rejected in NewRuntime/NewOperation, never
//     at request time.
//   - Shutdown semantics: rejects NEW work, WITHDRAWS work still waiting
//     for key/resource admission (wakes those waiters via the runtime
//     stop-admission context), and DRAINS work that already began
//     executing. Executing handlers are never force-cancelled.
//   - Timeout is COOPERATIVE: a per-attempt deadline propagated through
//     context.Context. Herta does not forcibly terminate handlers; the
//     handler (or its provider adapter) must honor the context. Worst
//     case wall time ≈ MaxAttempts × Timeout.
//   - Error classification is the handler/adapter's job (see Failure and
//     Fail). Herta never invents Uncertain on its own — except the one
//     conservative promotion described in outcome.go.
//
// Internal lifecycle (not exported as enums; see runtime.go):
//
//	SUBMITTED → ADMITTED → WAITING_FOR_KEY → WAITING_FOR_RESOURCES
//	          → EXECUTING → Success | Permanent | Transient | Throttled
//	                       | Uncertain
//	(Shutdown between ADMITTED and EXECUTING → CANCELLED_BEFORE_EXECUTION)
//
// This package is deliberately generic: no application-domain vocabulary,
// and only golang.org/x/sync as an external dependency.
//
// See README.md for the frozen V0 contract.
package execution
