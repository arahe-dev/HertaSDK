package doe

import (
	"fmt"
	"math"
	"strings"
)

// writePoints tabulates each design point's median throughput and p99 per arm.
func writePoints(b *strings.Builder, x *Experiment, trials []Trial) {
	if x.Kind == "sweep" {
		return
	}
	tput, p99 := mustResponse("throughput"), mustResponse("lat_p99")
	b.WriteString("## Per-point medians\n\n")
	head := []string{"pt", "levels " + strings.Join(Symbols(x.Factors), "")}
	for _, a := range x.Arms {
		head = append(head, a.Label+" ops/s", a.Label+" p99")
	}
	var rows [][]string
	for p, pt := range x.Points {
		row := []string{fmt.Sprint(p), "`" + levelString(pt.Levels, "") + "`"}
		for a := range x.Arms {
			row = append(row, human(pointMedian(trials, p, a, tput), tput.Unit),
				human(pointMedian(trials, p, a, p99), p99.Unit))
		}
		rows = append(rows, row)
	}
	table(b, head, rows)
}

func mustResponse(n string) Response {
	rs, ok := ResponseByName(n)
	if !ok {
		panic("doe: no response " + n)
	}
	return rs
}

func pointMedian(trials []Trial, p, a int, rs Response) float64 {
	var ys []float64
	for i := range trials {
		if t := &trials[i]; t.Point == p && t.Arm == a && t.Res.Err == "" {
			ys = append(ys, rs.Value(&t.Res))
		}
	}
	return median(ys)
}

// writeCurves renders each sweep axis as a table and two log-scaled charts.
func writeCurves(b *strings.Builder, an Analysis) {
	if len(an.Curves) == 0 {
		return
	}
	b.WriteString("## Sweeps\n\nMedians over replicates. Bars are log-scaled within each chart.\n\n")
	for _, axis := range SweepAxes {
		var cs []CurvePoint
		for _, c := range an.Curves {
			if c.Axis == axis {
				cs = append(cs, c)
			}
		}
		if len(cs) == 0 {
			continue
		}
		fmt.Fprintf(b, "### %s\n\n", axis)
		var rows [][]string
		for _, c := range cs {
			rows = append(rows, []string{human(c.X, ""), c.Arm, human(c.Throughput, "ops/s"), human(c.P50, "ns"),
				human(c.P99, "ns"), human(c.P999, "ns"), human(c.Allocs, ""), human(c.PeakGoroutines, ""),
				human(c.ErrRate, "fraction")})
		}
		table(b, []string{axis, "arm", "ops/s", "p50", "p99", "p99.9", "allocs/op", "extra goroutines", "errors"}, rows)
		chart(b, "throughput (ops/s)", cs, func(c CurvePoint) float64 { return c.Throughput }, "ops/s")
		chart(b, "p99 latency", cs, func(c CurvePoint) float64 { return c.P99 }, "ns")
	}
}

// chart draws one log-scaled horizontal bar per (x, arm).
func chart(b *strings.Builder, title string, cs []CurvePoint, get func(CurvePoint) float64, unit string) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, c := range cs {
		if v := get(c); v > 0 {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if math.IsInf(lo, 0) {
		return
	}
	span := math.Max(math.Log(hi/lo), 1e-9)
	fmt.Fprintf(b, "```\n%s\n", title)
	for _, c := range cs {
		v := get(c)
		frac := 0.02 + 0.98*math.Log(v/lo)/span
		fmt.Fprintf(b, "%6s %-8s %-40s %s\n", human(c.X, ""), c.Arm, bar(frac, 40), human(v, unit))
	}
	b.WriteString("```\n\n")
}
