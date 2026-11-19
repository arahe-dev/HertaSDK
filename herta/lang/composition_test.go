package lang

import (
	"strings"
	"testing"
)

func TestComposition(t *testing.T) {
	const rt = "runtime main { resource db capacity 8 }\n"
	const leaf = "op main::b(id: string) -> u64 { effect nonidempotent }\nhandler main::b(id: string) -> (u64, error) { return 0, nil }\n"
	cases := []struct{ name, source, want string }{
		{"empty", "", ""},
		{"opaque", rt + leaf, ""},
		{"strings are not calls", rt + leaf + `op main::a(b: bOp) -> u64 { effect pure }
handler main::a(b: bOp) -> (u64, error) { return "b.Do(ctx, x) call b(x)", nil }`, ""},
		{"multiline spaced raw call", rt + leaf + `op main::a(b: bOp) -> u64 { effect pure }
handler main::a(b: bOp) -> (u64, error) {
return b . Do (
ctx, Input{Id: "a"})
}`, "HC004"},
		{"unknown", rt + `op main::a() -> u64 { effect pure calls missing }
handler main::a() -> (u64, error) { return 0, nil }`, "HC013"},
		{"invalid dedupe", rt + `op main::a() -> u64 { effect nonidempotent dedupe key missing }
handler main::a() -> (u64, error) { return 0, nil }`, "HC014"},
		{"empty classify", rt + `op main::a() -> u64 { effect pure classify {} }
handler main::a() -> (u64, error) { return 0, nil }`, "HC009"},
		{"first match wins", rt + `error ErrA ambiguous
op main::a() -> u64 {
effect nonidempotent
retry max 2 on { transient }
classify { _ => permanent, ErrA => permanent }
}
handler main::a() -> (u64, error) { return 0, nil }`, ""},
		{"ambiguous retry", rt + `error ErrA ambiguous
op main::a() -> u64 {
effect nonidempotent
retry max 2 on { transient }
classify { ErrA => transient, _ => permanent }
}
handler main::a() -> (u64, error) { return 0, nil }`, "HC005"},
		{"forward renamed key", rt + `op main::a(user: string) -> u64 { effect pure serialize key user }
handler main::a(user: string) -> (u64, error) { call b(user) }
op main::b(other: string) -> u64 { effect pure serialize key other }
handler main::b(other: string) -> (u64, error) { return 0, nil }`, "HC001"},
		{"sequential sibling resources", rt + `op main::a() -> u64 { effect pure calls b, c }
handler main::a() -> (u64, error) { return 0, nil }
op main::b() -> u64 { effect pure resources { db units 8 } }
handler main::b() -> (u64, error) { return 0, nil }
op main::c() -> u64 { effect pure resources { db units 8 } }
handler main::c() -> (u64, error) { return 0, nil }`, ""},
		{"raw renamed key", rt + `op main::a(user: string, b: bOp) -> u64 { effect pure serialize key user }
handler main::a(user: string, b: bOp) -> (u64, error) { return b.Do(ctx, Input{OtherKey: user}) }
op main::b(other_key: string) -> u64 { effect pure serialize key other_key }
handler main::b(other_key: string) -> (u64, error) { return 0, nil }`, "HC001"},
		{"effective detached idempotent", rt + `op main::a() -> u64 { effect idempotent calls b }
handler main::a() -> (u64, error) { return 0, nil }
op main::b() -> u64 { effect pure detached calls c }
handler main::b() -> (u64, error) { return 0, nil }
op main::c() -> u64 { effect idempotent }
handler main::c() -> (u64, error) { return 0, nil }`, "HC008"},
		{"resource sum no overflow", `runtime main { resource db capacity 9223372036854775807 }
op main::a() -> u64 { effect pure resources { db units 9223372036854775807 } calls b }
handler main::a() -> (u64, error) { return 0, nil }
op main::b() -> u64 { effect pure resources { db units 2 } }
handler main::b() -> (u64, error) { return 0, nil }`, "HC003"},
		{"detached deadline boundary", rt + `op main::a() -> u64 { effect pure timeout 1ms calls b }
handler main::a() -> (u64, error) { return 0, nil }
op main::b() -> u64 { effect pure detached timeout 2ms calls c }
handler main::b() -> (u64, error) { return 0, nil }
op main::c() -> u64 { effect pure timeout 2ms }
handler main::c() -> (u64, error) { return 0, nil }`, ""},
		{"huge retry no overflow", rt + `op main::a() -> u64 { effect pure retry max 9223372036854775807 on { transient } calls b budget 9223372036854775807 }
handler main::a() -> (u64, error) { return 0, nil }
op main::b() -> u64 { effect pure retry max 2 on { transient } }
handler main::b() -> (u64, error) { return 0, nil }`, "HC006"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Parse(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			var messages []string
			for _, d := range Check(p) {
				messages = append(messages, d.String())
			}
			got := strings.Join(messages, "\n")
			if tc.want == "" && got != "" {
				t.Fatalf("unexpected diagnostics: %s", got)
			}
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("want %s, got %s", tc.want, got)
			}
		})
	}
}

func TestCompositionParseErrors(t *testing.T) {
	for _, clause := range []string{
		"budget 0", "budget 9223372036854775808", "detached for nobody",
		"dedupe nope id", "classify { _ => nope }", "classify { _ -> permanent }",
		"calls", "lock", "budget 2 budget 3",
	} {
		t.Run(clause, func(t *testing.T) {
			if _, err := Parse("op main::a() -> u64 { effect pure " + clause + " }"); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}
