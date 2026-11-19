package gogen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arahe-dev/hertasdk/herta/lang"
)

func TestClosedBuild(t *testing.T) {
	src, err := os.ReadFile("../examples/closed.he")
	if err != nil {
		t.Fatal(err)
	}
	p, err := lang.Parse(string(src))
	if err != nil {
		t.Fatal(err)
	}
	if ds := lang.CheckClosed(p); len(ds) != 0 {
		t.Fatal(ds)
	}
	out, err := GenerateClosed(p, "closedtest")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/closed.golden")
	if err != nil {
		t.Fatal(err)
	}
	if out != string(want) {
		t.Fatalf("closed build differs from golden:\n%s", out)
	}
	t.Run("compiled runtime", func(t *testing.T) {
		compileGenerated(t, out, closedHost, "")
	})
	t.Run("undefined sentinel", func(t *testing.T) {
		compileGenerated(t, strings.ReplaceAll(out, "ErrTimeout", "ErrMissing"), closedHost, "undefined: ErrMissing")
	})
	plain, err := Generate(p, "closedtest")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "Classify:") || strings.Contains(plain, "Closed:") || strings.Contains(plain, "Calls:") {
		t.Fatal("closed policy leaked into ordinary build")
	}
}

func TestClosedClassifierFallbacks(t *testing.T) {
	for _, classify := range []string{"", "classify { _ => permanent }", "classify { _ => permanent, ErrUnreachable => transient }"} {
		p, err := lang.Parse("runtime main { resource db capacity 1 }\nop main::a() -> u64 { effect pure " + classify + " }\nhandler main::a() -> (u64, error) { return 1, nil }")
		if err != nil {
			t.Fatal(err)
		}
		out, err := GenerateClosed(p, "closedtest")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, `"errors"`) != strings.Contains(classify, "ErrUnreachable") {
			t.Fatal("incorrect errors import")
		}
		compileGenerated(t, out, "package closedtest\nimport \"errors\"\nvar ErrUnreachable = errors.New(\"unreachable\")\n", "")
	}
}

func TestClosedTypedCallee(t *testing.T) {
	p, err := lang.Parse(`runtime main { resource db capacity 1 }
op main::a(client: bOp) -> u64 { effect pure }
handler main::a(client: bOp) -> (u64, error) { return client.Do(ctx, MainBInput{}) }
op main::b() -> u64 { effect pure }
handler main::b() -> (u64, error) { return 7, nil }
`)
	if err != nil {
		t.Fatal(err)
	}
	if ds := lang.CheckClosed(p); len(ds) != 0 {
		t.Fatal(ds)
	}
	out, err := GenerateClosed(p, "closedtest")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `Calls:  []string{"main::b"}`) {
		t.Fatalf("typed edge not lowered: %s", out)
	}
	compileGenerated(t, out, `package closedtest
import "github.com/arahe-dev/hertasdk"
type bOp = *hertasdk.Operation[MainBInput, uint64]
`, "")
}

func compileGenerated(t *testing.T, generated, host, wantError string) {
	t.Helper()
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": "module closedtest\n\ngo 1.26.0\n\nrequire github.com/arahe-dev/hertasdk v0.0.0\nreplace github.com/arahe-dev/hertasdk => " + filepath.ToSlash(root) + "\n",
		"ops.go": generated, "host_test.go": host,
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "test", "-count=1", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if wantError == "" {
		if err != nil {
			t.Fatalf("compiled closed build: %v\n%s", err, out)
		}
	} else if err == nil || !strings.Contains(string(out), wantError) {
		t.Fatalf("wanted compilation failure %q: %v\n%s", wantError, err, out)
	}
}

const closedHost = `package closedtest
import (
    "context"
    "errors"
    "testing"
    "github.com/arahe-dev/hertasdk"
)
var ErrTimeout = errors.New("timeout")
var ErrUnavailable = errors.New("unavailable")
var charge *hertasdk.Operation[MainChargeInput, uint64]
var rogue *hertasdk.Operation[int, uint64]
var attempts int
var nested bool
func chargeHost(ctx context.Context) (uint64, error) {
    attempts++
    if nested { return rogue.Do(ctx, 0) }
    return 0, hertasdk.Fail(hertasdk.Transient, errors.Join(ErrUnavailable, ErrTimeout))
}
func checkoutHost(ctx context.Context) (uint64, error) { return charge.Do(ctx, MainChargeInput{}) }
func TestGeneratedPolicies(t *testing.T) {
    rt, err := NewMainRuntime()
    if err != nil { t.Fatal(err) }
    charge, err = NewMainCharge(rt)
    if err != nil { t.Fatal(err) }
    checkout, err := NewMainCheckout(rt)
    if err != nil { t.Fatal(err) }
    _, err = checkout.Do(context.Background(), MainCheckoutInput{})
    if attempts != 1 || hertasdk.OutcomeOf(err) != hertasdk.Uncertain || !errors.Is(err, ErrTimeout) {
        t.Fatalf("classifier or declared helper call failed: attempts=%d err=%v", attempts, err)
    }
    rogue, err = hertasdk.NewOperation(rt, "main::rogue", func(context.Context, int) (uint64, error) {
        t.Fatal("undeclared handler ran"); return 0, nil
    }, hertasdk.Policy[int]{Effect: hertasdk.Pure})
    if err != nil { t.Fatal(err) }
    nested = true
    _, err = charge.Do(context.Background(), MainChargeInput{})
    if !errors.Is(err, hertasdk.ErrUndeclaredCall) || hertasdk.OutcomeOf(err) != hertasdk.Permanent {
        t.Fatalf("undeclared helper call escaped: %v", err)
    }
}
`
