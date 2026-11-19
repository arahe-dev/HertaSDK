package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/arahe-dev/hertasdk/herta/experiments/retrycheck"
	"golang.org/x/tools/go/analysis"
)

type finding struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type report struct {
	Files    int       `json:"files"`
	Findings []finding `json:"findings"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: retrycheck <git checkout>")
		os.Exit(2)
	}
	if err := scan(os.Args[1], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func include(path string) bool {
	if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		switch part {
		case "vendor", "testdata", "fixtures":
			return false
		}
	}
	return true
}

// scan parses tracked files across all platforms and modules. It never builds
// the checkout, resolves its dependencies, or runs its code.
func scan(root string, out io.Writer) error {
	cmd := exec.Command("git", "ls-files", "--full-name", "-z", "--", "*.go")
	cmd.Dir = root
	paths, err := cmd.Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return fmt.Errorf("list tracked files: %w: %s", err, exit.Stderr)
		}
		return fmt.Errorf("list tracked files: %w", err)
	}

	fileset := token.NewFileSet()
	r := report{Findings: []finding{}}
	type group struct {
		dir string
		pkg string
	}
	groups := make(map[group][]*ast.File)
	var order []group
	for _, raw := range bytes.Split(paths, []byte{0}) {
		path := string(raw)
		if !include(path) {
			continue
		}
		full := filepath.Join(root, path)
		info, err := os.Lstat(full)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		src, err := os.ReadFile(full)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		file, err := parser.ParseFile(fileset, path, src, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		if ast.IsGenerated(file) {
			continue
		}
		r.Files++
		key := group{filepath.Dir(path), file.Name.Name}
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], file)
	}
	for _, key := range order {
		a := retrycheck.Analyzer
		_, err := a.Run(&analysis.Pass{
			Analyzer: a,
			Fset:     fileset,
			Files:    groups[key],
			Report: func(d analysis.Diagnostic) {
				loc := fileset.PositionFor(d.Pos, false)
				r.Findings = append(r.Findings, finding{loc.Filename, loc.Line, loc.Column, d.Category, d.Message})
			},
		})
		if err != nil {
			return err
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
