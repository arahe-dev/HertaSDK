// Package system tests Herta as a whole (tier 3 of the correctness lab).
//
//   - differential: seeded random scenarios replayed step by step on every
//     implementation against the model of the rules it claims; divergences
//     not explained by a CONTRACT §8 delta fail with a minimized repro.
//   - load and chaos: concurrent random workloads with invariant monitors in
//     every handler (capacity, key exclusion, unsafe retry), a Stats ledger
//     reconciled against the per-call trace, and leak checks after drain;
//     chaos adds panics, ctx-ignoring handlers, cancellations and
//     concurrent Shutdown at random points, all under a watchdog.
//   - twin: a digital twin of the Mirai backend (renderer 8 / Reject,
//     db-write 4 / Wait shared by events, catalogue and handoff) under
//     open-loop traffic, judged by SLO-style properties. LAB_LEVEL=full
//     runs it as a 5-minute soak with steady-state memory checks.
package system
