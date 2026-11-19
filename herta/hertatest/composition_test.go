package hertatest

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckerGoldens(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "herta")
	cmd := exec.Command("go", "build", "-o", bin, "../cmd/herta")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build checker: %v\n%s", err, out)
	}
	cases, err := Discover("../testdata/cases", DefaultLimits)
	if err != nil {
		t.Fatal(err)
	}
	results := Run(context.Background(), cases, bin+" check {file}", DefaultLimits)
	summary, ok := Summary(results)
	if !ok {
		t.Fatal(summary)
	}
	t.Log(summary)
}
