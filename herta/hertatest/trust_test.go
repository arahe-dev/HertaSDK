package hertatest

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTrustGoldens(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "herta")
	if out, err := exec.Command("go", "build", "-o", bin, "../cmd/herta").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	cases, err := Discover("../testdata/trust", DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	summary, ok := Summary(Run(context.Background(), cases, bin+" trust {args} {file}", DefaultLimits))
	if !ok {
		t.Fatal(summary)
	}
	t.Log(summary)
	for _, args := range [][]string{{"trust"}, {"trust", "missing.he"}} {
		cmd := exec.Command(bin, args...)
		if err := cmd.Run(); err == nil || cmd.ProcessState.ExitCode() != 2 {
			t.Fatalf("%v: expected input error exit 2, got %v", args, err)
		}
	}
}
