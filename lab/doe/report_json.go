package doe

import (
	"encoding/json"
	"math"
	"strconv"
)

// num is a float64 that encodes NaN and ±Inf as JSON null.
type num float64

func (n num) MarshalJSON() ([]byte, error) {
	f := float64(n)
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return []byte("null"), nil
	}
	return strconv.AppendFloat(nil, f, 'g', 6, 64), nil
}

// Summary is summary.json: the machine-readable digest of an experiment.
type Summary struct {
	Meta        Meta                    `json:"meta"`
	Design      *designJSON             `json:"design,omitempty"`
	Comparisons []comparisonJSON        `json:"comparisons,omitempty"`
	Noise       []noiseJSON             `json:"noise,omitempty"`
	TopEffects  map[string][]effectJSON `json:"top_effects,omitempty"`
	Curves      []curveJSON             `json:"curves,omitempty"`
}

// Meta describes how and where an experiment ran.
type Meta struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Command      string   `json:"command"`
	Started      string   `json:"started"`
	Go           string   `json:"go"`
	OS           string   `json:"os"`
	Arch         string   `json:"arch"`
	CPUs         int      `json:"cpus"`
	GOMAXPROCS   int      `json:"gomaxprocs"`
	Arms         []string `json:"arms"` // "label=impl"
	Reps         int      `json:"reps"`
	Seed         uint64   `json:"seed"`
	WarmupMs     float64  `json:"warmup_ms"`
	DurationMs   float64  `json:"duration_ms"`
	Trials       int      `json:"trials"`
	FailedTrials int      `json:"failed_trials"`
	ElapsedS     float64  `json:"elapsed_s"`
}

type designJSON struct {
	Runs       int          `json:"runs"`
	Resolution int          `json:"resolution"`
	Factors    []factorJSON `json:"factors"`
	Generators []string     `json:"generators,omitempty"`
	Defining   []string     `json:"defining,omitempty"`
}

type factorJSON struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Low    string `json:"low"`
	High   string `json:"high"`
}

type comparisonJSON struct {
	Response     string   `json:"response"`
	Unit         string   `json:"unit"`
	Base         string   `json:"base"`
	Cand         string   `json:"candidate"`
	Verdict      string   `json:"verdict"`
	BaseMean     num      `json:"base_mean"`
	CandMean     num      `json:"candidate_mean"`
	Effect       num      `json:"effect"` // log2 units for log responses
	SE           num      `json:"se"`
	P            num      `json:"p"`
	Ratio        num      `json:"ratio"`
	Lo           num      `json:"ci_lo"` // ratio (log) or difference (raw)
	Hi           num      `json:"ci_hi"`
	Interactions []string `json:"interactions,omitempty"`
}

type noiseJSON struct {
	Response string  `json:"response"`
	Scope    string  `json:"scope"`
	SD       num     `json:"sd"`
	CV       num     `json:"cv"`
	DF       float64 `json:"df"`
}

type effectJSON struct {
	Scope       string `json:"scope"`
	Effect      string `json:"effect"`
	Estimate    num    `json:"estimate"`
	SE          num    `json:"se"`
	P           num    `json:"p"`
	Ratio       num    `json:"ratio"`
	Lenth       string `json:"lenth,omitempty"`
	Significant bool   `json:"significant"`
}

type curveJSON struct {
	Axis           string `json:"axis"`
	Arm            string `json:"arm"`
	X              num    `json:"x"`
	Throughput     num    `json:"throughput"`
	P50            num    `json:"p50_ns"`
	P99            num    `json:"p99_ns"`
	P999           num    `json:"p999_ns"`
	Allocs         num    `json:"allocs_op"`
	PeakGoroutines num    `json:"peak_goroutines"`
	ErrRate        num    `json:"err_rate"`
}

func marshalSummary(s Summary) ([]byte, error) { return json.MarshalIndent(s, "", "  ") }
