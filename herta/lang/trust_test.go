package lang

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTrust(t *testing.T) {
	const rt = "runtime main { resource db capacity 8 }\n"
	const leaf = `op main::b() -> u64 { effect nonidempotent }
handler main::b() -> (u64, error) { return 0, nil }
`
	tests := []struct {
		name, source, op, kind string
		closed, want, required bool
	}{
		{"legacy ignores classify", rt + `op main::a() -> u64 { effect nonidempotent retry max 2 on { transient } classify { _ => permanent } }`, "main::a", "transient-honest", false, true, true},
		{"closed uses classify", rt + `op main::a() -> u64 { effect nonidempotent retry max 2 on { transient } classify { _ => permanent } }`, "main::a", "transient-honest", true, false, false},
		{"nonidempotent required", rt + `op main::a() -> u64 { effect nonidempotent retry max 2 on { transient } classify { ErrA => transient, _ => permanent } }`, "main::a", "transient-honest", true, true, true},
		{"idempotent informational", rt + `op main::a() -> u64 { effect idempotent retry max 2 on { transient } classify { ErrA => transient, _ => permanent } }`, "main::a", "transient-honest", true, true, false},
		{"one attempt", rt + `op main::a() -> u64 { effect nonidempotent retry max 1 on { transient } }`, "main::a", "transient-honest", false, false, false},
		{"unreachable arm", rt + `op main::a() -> u64 { effect nonidempotent retry max 2 on { transient } classify { _ => permanent, ErrA => transient } }`, "main::a", "transient-honest", true, false, false},
		{"missing closed classify remains legacy", rt + leaf, "main::b", "legacy-tag-path", true, true, true},
		{"amplification", rt + leaf + `op main::a() -> u64 { effect pure retry max 2 on { uncertain } calls b }`, "main::a", "nested-retry-amplification", true, true, true},
		{"unknown", rt + `op main::a() -> u64 { effect pure calls missing }`, "main::a", "unresolved-callee", true, true, true},
		{"same key", rt + `op main::a(id: string) -> u64 { effect pure serialize key id calls b }
op main::b(id: string) -> u64 { effect pure serialize key id }`, "main::a", "same-serialize-key", true, true, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Parse(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, op := range Trust(p, tc.closed) {
				for _, a := range op.Assumptions {
					if op.Op == tc.op && a.Origin == tc.op && a.Kind == tc.kind {
						found = true
						if a.Required != tc.required || len(a.Theorems) == 0 {
							t.Fatalf("bad assumption: %+v", a)
						}
					}
				}
			}
			if found != tc.want {
				t.Fatalf("found %s = %v, want %v", tc.kind, found, tc.want)
			}
		})
	}
}

func TestTrustTransitiveDeterministic(t *testing.T) {
	p, err := Parse(`runtime main { resource db capacity 1 }
op main::z() -> u64 { effect pure calls b, a }
op main::a() -> u64 { effect pure calls b }
op main::b() -> u64 { effect nonidempotent calls z retry max 2 on { transient } }
`)
	if err != nil {
		t.Fatal(err)
	}
	report := Trust(p, false)
	if len(report) != 3 || report[0].Op != "main::a" || !TrustRequired(report) {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, op := range report {
		count := 0
		for _, a := range op.Assumptions {
			if a.Origin == "main::b" && a.Kind == "transient-honest" {
				count++
				if len(a.Via) > 2 || a.Via[0] != op.Op {
					t.Fatalf("not shortest provenance: %+v", a)
				}
			}
		}
		if count != 1 {
			t.Fatalf("propagation count = %d", count)
		}
	}
	for range 10 {
		if !reflect.DeepEqual(report, Trust(p, false)) {
			t.Fatal("nondeterministic ledger")
		}
	}
	encoded, err := json.Marshal(report)
	if err != nil || !strings.Contains(string(encoded), `"theorems":["T1"]`) {
		t.Fatalf("JSON: %s %v", encoded, err)
	}
}

func TestTrustEmptyAndOpaque(t *testing.T) {
	if got := Trust(&Program{}, false); len(got) != 0 || TrustRequired(got) {
		t.Fatalf("empty program: %+v", got)
	}
	p, err := Parse(`runtime main { resource db capacity 1 }
op main::a() -> u64 { effect pure classify { _ => permanent } }
handler main::a() -> (u64, error) { return 0, nil }`)
	if err != nil {
		t.Fatal(err)
	}
	report := Trust(p, true)
	if got := DumpTrust(report); got != "main::a\n  trust: opaque-handler only [required; T1,T4_composition]\n" {
		t.Fatal(got)
	}
	if !TrustRequired(report) {
		t.Fatal("raw Go must remain a required assumption")
	}
	report[0].Assumptions = append(report[0].Assumptions, report[0].Assumptions[0])
	if !strings.Contains(DumpTrust(report), "trust: opaque-handler only") {
		t.Fatal("inherited opaque assumptions should retain the opaque-only summary")
	}
}
