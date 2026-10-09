// Package doe is the lab's Design-of-Experiments engine.
//
// It drives any impl.Impl with a synthetic workload (a closed-loop client
// pool or open-loop Poisson arrivals), measures a fixed set of responses per
// run, and analyses two-level factorial designs in which the implementation
// itself is a crossed factor (X). One code path measures every
// implementation, so a difference in a response is a difference in the
// implementation; labelling one implementation twice ("a=orig,b=orig")
// turns any experiment into an A/A calibration of the lab's own noise.
//
// Layers, bottom up:
//
//	clock*.go, hist.go, stats.go   TSC clock; allocation-free histogram; t, normal, Lenth
//	workload.go, handler.go        what one run does; in-handler safety monitors
//	engine.go, driver.go           one run: callers, warm-up, window, shutdown under load
//	recorder.go, sampler.go        per-call capture; peak goroutines and heap
//	result.go, response.go         what a run reports, and on which scale it is analysed
//	design.go, factor.go           2^(k-p) designs, alias structure; the factor catalog
//	experiments.go, sweep.go       screen, full factorial, sweep and A/B experiments
//	arm.go, plan.go                arms; replicates in randomised blocks
//	analysis.go, compare.go        effects, standard errors, impl comparisons
//	followup.go                    picking follow-up factors from a screen
//	report*.go, format.go          runs.csv, effects.csv, summary.json, REPORT.md
package doe
