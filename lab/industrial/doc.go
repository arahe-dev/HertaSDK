// Package industrial tests Herta at deployment scale (tier 4 of the lab):
// what a production runtime meets on a bad day, judged against CONTRACT
// §10's budgets and the drain promise of §6.
//
//   - spike: open-loop traffic at 10× the base rate for a burst, then back;
//     the queue must shed by deadline, never violate capacity, clear its
//     backlog within a second and leave the whole budget free.
//   - drain: Shutdown with ~1000 callers parked on a budget or a key; every
//     waiter must be withdrawn and the drain must take one service time.
//   - keys: 1e6 distinct keys pass through; idle keys must cost nothing.
//   - waiters: 10k callers parked at once; each must cost ≤ 2 allocations
//     and ≤ 256 B of heap beyond its own goroutine, and withdraw promptly.
//   - goroutines: goroutines created per Do, read from the Go scheduler.
//   - soak: the Mirai twin held for LAB_SOAK (default 3s); a soak of a
//     minute or more must hold heap, goroutines and db-write p99 steady.
//
// Sizes scale with LAB_LEVEL (quick by default, full for the industrial
// figures). Every test logs its numbers and, when LAB_OUT names a file,
// appends them there as one JSON line, so a study can tabulate them.
package industrial
