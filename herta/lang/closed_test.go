package lang

import (
	"reflect"
	"strings"
	"testing"
)

func TestClosedChecks(t *testing.T) {
	for _, tc := range []struct{ name, policy, body, want string }{
		{"no retry", "", "return 0, nil", ""},
		{"retry", "retry max 1 on { transient }", "return 0, nil", "HC016"},
		{"classified retry", "retry max 2 on { transient } classify { _ => permanent }", "return 0, nil", ""},
		{"empty classifier", "classify {}", "return 0, nil", "HC009"},
		{"missing fallback", "classify { ErrA => permanent }", "return 0, nil", "HC009"},
		{"fail", "", "fail permanent, ErrA", "HC018"},
		{"inline fail", "", "if true { fail permanent, ErrA }", "HC018"},
		{"sdk fail", "", "return 0, hertasdk . Fail (hertasdk.Permanent, ErrA)", "HC018"},
		{"multiline sdk fail", "", "return 0, hertasdk .\nFail (hertasdk.Permanent, ErrA)", "HC018"},
		{"sdk failure", "", "return 0, & hertasdk . Failure {Outcome: hertasdk.Permanent, Err: ErrA}", "HC018"},
		{"multiline sdk failure", "", "return 0, &hertasdk .\nFailure {Outcome: hertasdk.Permanent, Err: ErrA}", "HC018"},
		{"string", "", `return "hertasdk.Fail() &hertasdk.Failure{} fail permanent, ErrA", nil`, ""},
		{"comment", "", "return 0, nil // hertasdk.Fail()", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Parse("runtime main { resource db capacity 1 }\nop main::a() -> u64 { effect pure " + tc.policy + " }\nhandler main::a() -> (u64, error) { " + tc.body + "\n}")
			if err != nil {
				t.Fatal(err)
			}
			ds := CheckClosed(p)
			if tc.want == "" {
				if len(ds) > 0 {
					t.Fatalf("unexpected diagnostics: %v", ds)
				}
			} else if len(ds) != 1 || !strings.Contains(ds[0].Msg, tc.want) {
				t.Fatalf("want only %s, got %v", tc.want, ds)
			}
			for _, d := range Check(p) {
				if strings.Contains(d.Msg, "HC016") || strings.Contains(d.Msg, "HC018") {
					t.Fatalf("closed diagnostic leaked: %v", d)
				}
			}
		})
	}
}

func TestCallees(t *testing.T) {
	op := &Op{Runtime: "main", Calls: []string{"b", "b"}}
	h := &Handler{Params: []Param{{Name: "client", Type: "cOp"}}, Body: []Stmt{
		&RawStmt{Text: `call b(x); call d(); client . Do (ctx, Input{}); "call ignored()"`},
	}}
	if got, want := Callees(op, h), []string{"main::b", "main::d", "main::c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("callees=%v want %v", got, want)
	}
	if got := Callees(&Op{}, nil); len(got) != 0 {
		t.Fatalf("empty op: %v", got)
	}
}
