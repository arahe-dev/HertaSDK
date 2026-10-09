// Package mutate is the lab's mutation tester: it measures how much of an
// implementation's behavior its tests actually pin down.
//
// Generate parses a package's non-test files and enumerates mutants with
// the operators that matter for a concurrency runtime: flipped and shifted
// conditionals, deleted and swapped effect statements (release, unlock,
// cancel, Add, Done), flipped booleans, swallowed errors, off-by-one
// integers, and removed or inlined defers. Each edit is a byte splice at
// the AST node's position, so comments and line numbers survive.
//
// Run copies the module once per worker into a scratch directory, applies
// one mutant at a time, and runs a cascade of stages (typically the
// package's own tests, then the lab suites). A mutant is killed by the
// first stage that fails or hangs past its deadline, survives if every
// stage passes, and is a compile mutant if the package no longer builds.
// A plain test failure kills only if its stage fails again when re-run,
// so a timing assertion tripped by the run's own load kills
// nothing; such failures are reported as flaky. Surviving mutants that a
// human has proven equivalent are listed in a triage file and excluded
// from the adjusted score.
//
// A saved run (run.json) can be re-triaged without running anything, and
// resumed: kills stand, while survivors, new mutants and every mutant of
// an edited file are judged again.
//
// The command lives in lab/mutate/cmd/mutate; lab/mutate/run.sh holds the
// orig and recon presets.
package mutate
