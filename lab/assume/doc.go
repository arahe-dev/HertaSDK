// Package assume makes the hidden assumptions of Herta's design
// falsifiable (tier 1 of the correctness lab).
//
// Every test states, in its doc comment:
//
//	Assumption:     what the design takes for granted;
//	Why it matters: what breaks, and where, if it is false;
//	Verdict:        what the test asserts, and the measured value it logs.
//
// Platform tests measure the machine and the Go runtime the budgets of
// CONTRACT §10 are set on. Contract tests pin one rule of CONTRACT §4–§6
// each, on every selected implementation, mostly inside a testing/synctest
// bubble so that order and time are exact. deltas_test.go is the ledger of
// CONTRACT §8: one test per delta, asserting v0.2's defect on orig and the
// fix on everything after it.
package assume
