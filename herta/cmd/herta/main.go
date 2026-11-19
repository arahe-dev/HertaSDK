// Command herta is the Herta language compiler front end.
//
//	herta check <file.he>          parse + contract-check, print diagnostics
//	herta ast <file.he>            print the parsed AST dump (golden format)
//	herta build [-pkg name] <file.he>   compile to Go calling hertasdk, on stdout
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/arahe-dev/hertasdk/herta/gogen"
	"github.com/arahe-dev/hertasdk/herta/lang"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: herta check|ast|build [-pkg name] <file.he>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "check", "ast":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "usage: herta %s <file.he>\n", os.Args[1])
			os.Exit(2)
		}
		src, err := os.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "herta:", err)
			os.Exit(2)
		}
		prog, err := lang.Parse(string(src))
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			os.Exit(1)
		}
		if os.Args[1] == "ast" {
			fmt.Print(lang.Dump(prog))
			return
		}
		diags := lang.Check(prog)
		for _, d := range diags {
			fmt.Fprintln(os.Stdout, d)
		}
		if len(diags) > 0 {
			os.Exit(1)
		}
	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		pkg := fs.String("pkg", "main", "package name for the generated file")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "usage: herta build [-pkg name] <file.he>")
			os.Exit(2)
		}
		src, err := os.ReadFile(fs.Arg(0))
		if err != nil {
			fmt.Fprintln(os.Stderr, "herta:", err)
			os.Exit(2)
		}
		prog, err := lang.Parse(string(src))
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			os.Exit(1)
		}
		diags := lang.Check(prog)
		for _, d := range diags {
			fmt.Fprintln(os.Stdout, d)
		}
		if len(diags) > 0 {
			os.Exit(1)
		}
		out, err := gogen.Generate(prog, *pkg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "herta:", err)
			os.Exit(2)
		}
		fmt.Print(out)
	default:
		fmt.Fprintln(os.Stderr, "usage: herta check|ast|build [-pkg name] <file.he>")
		os.Exit(2)
	}
}
