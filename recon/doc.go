// Package execution is HertaSDK v0.2-recon: the v0.2.0 execution contract
// rebuilt from first principles (see CONTRACT.md).
//
// The exported API and behavior are those of v0.2.0, except the ten
// defects CONTRACT §8 corrects. Internals are laid out for v1.0:
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
//
// An uncontended call takes one atomic add on the gate, one short critical
// section per claim, and no goroutines, allocations or contexts beyond the
// optional per-attempt deadline. Cost appears only where a call blocks.
package execution
