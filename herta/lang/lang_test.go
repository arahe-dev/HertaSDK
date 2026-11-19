package lang

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files")

// Golden tests: each testdata/<name>.he is parsed; a clean parse dumps the
// AST, a parse error prints the error, and a passing parse runs the checker
// whose diagnostics are printed. Run with -update to regenerate.
func TestGolden(t *testing.T) {
	files, err := filepath.Glob("testdata/*.he")
	if err != nil || len(files) == 0 {
		t.Fatalf("no testdata: %v", err)
	}
	for _, f := range files {
		f := f
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			var out strings.Builder
			prog, err := Parse(string(src))
			if err != nil {
				out.WriteString("parse error: " + err.Error() + "\n")
			} else {
				diags := Check(prog)
				if len(diags) == 0 {
					out.WriteString(Dump(prog))
					out.WriteString("ok\n")
				} else {
					for _, d := range diags {
						out.WriteString(d.String() + "\n")
					}
				}
			}
			golden := strings.TrimSuffix(f, ".he") + ".golden"
			if *update {
				if err := os.WriteFile(golden, []byte(out.String()), 0644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("missing golden %s (run with -update)", golden)
			}
			if out.String() != string(want) {
				t.Fatalf("golden mismatch for %s\ngot:\n%s\nwant:\n%s", f, out.String(), want)
			}
		})
	}
}
