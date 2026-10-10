// Package hertasdk is an in-process execution contract runtime: ordinary Go
// handlers declare what they consume, what repeating them means, and what a
// failure means, and one Runtime arbitrates shared capacity among them.
//
// A Runtime owns named, weighted budgets (ResourceSpec). An Operation wraps a
// plain handler with a Policy: the budgets it claims, its Effect (Pure,
// Idempotent, NonIdempotent), Wait or Reject admission, an optional per-key
// serialization, a cooperative per-attempt Timeout and a RetryPolicy. Unsafe
// contracts, such as retrying an Uncertain outcome of a NonIdempotent
// operation, fail in NewOperation, never under load. Shutdown stops
// admission, withdraws waiters and drains running calls.
//
// The normative contract is docs/contract.md; examples/quickstart shows one
// runtime shared by two operations.
//
// Cost: an uncontended call takes one atomic add on the gate and one short
// critical section per claim, with no goroutines and no allocations beyond
// the optional per-attempt deadline. Cost appears only where a call blocks.
//
// Layout:
//
//	runtime.go    the gate word (stopping | drained | in-flight), Shutdown, Stats
//	operation.go  NewOperation's snapshot, Do, the one admission path
//	attempt.go    the attempt loop and per-attempt deadline
//	outcome.go    Outcome, Failure, the one classifier, the retry bitset
//	errors.go     sentinels and every error Do constructs
//	resource.go   the FIFO weighted semaphore behind each budget
//	keys.go       per-key FIFO mutual exclusion
//	waiter.go     the intrusive waiter both of them park on
//	policy.go     the declarative contract types
//	validate.go   construction-time validation
package hertasdk
