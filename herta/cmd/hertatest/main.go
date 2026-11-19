// Command hertatest runs the Herta golden-file suite: one fast command with
// hard caps on case count and runtime.
//
//	hertatest -cases herta/testdata/cases -exec "herta check {file}"
//
// -exec is a command template: {file} expands to the case's input.he and
// {args} to its args.txt. The literal value "selftest" selects the built-in
// echo engine for exercising the runner itself.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/arahe-dev/hertasdk/herta/hertatest"
)

func main() {
	cases := flag.String("cases", "herta/testdata/cases", "directory of case folders")
	execT := flag.String("exec", hertatest.Selftest, `check command template ({file}, {args}) or "selftest"`)
	maxCases := flag.Int("max-cases", hertatest.DefaultLimits.MaxCases, "hard cap on case count")
	caseTimeout := flag.Duration("case-timeout", hertatest.DefaultLimits.CaseTimeout, "per-case timeout")
	totalTimeout := flag.Duration("timeout", hertatest.DefaultLimits.TotalTimeout, "whole-suite timeout")
	flag.Parse()

	lim := hertatest.DefaultLimits
	lim.MaxCases = *maxCases
	lim.CaseTimeout = *caseTimeout
	lim.TotalTimeout = *totalTimeout

	found, err := hertatest.Discover(*cases, lim)
	if err != nil {
		fmt.Fprintln(os.Stderr, "hertatest:", err)
		os.Exit(2)
	}
	fmt.Printf("hertatest: %d cases, cap %d, %s total / %s per case\n",
		len(found), lim.MaxCases, lim.TotalTimeout, lim.CaseTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	results := hertatest.Run(ctx, found, *execT, lim)
	summary, ok := hertatest.Summary(results)
	fmt.Print(summary)
	if !ok {
		os.Exit(1)
	}
}
