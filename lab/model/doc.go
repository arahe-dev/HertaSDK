// Package model holds the lab's reference semantics: small, obviously
// correct, single-threaded models of every Herta subsystem, written straight
// from docs/contract.md.
//
//   - Sem      the weighted FIFO budget semaphore (§4.1)
//   - Keys     the per-key FIFO locker (§4.2)
//   - Exec     the retry loop and its one classifier (§5)
//   - Validate construction checks (§3.2)
//   - System   all of the above composed with admission and drain (§4, §6, §7)
//
// Every model runs in one of two modes selected by Rules.Fixed: false
// reproduces v0.2.0 (the oracle, defects included); true applies the §8
// deltas. A suite compares each implementation against the mode it claims,
// so a divergence between implementations is explained exactly when both
// agree with their own model.
//
// The models never block and never read a clock: time is an explicit
// virtual duration, which is what lets the lab replay them step by step
// against real implementations running inside a testing/synctest bubble.
package model
