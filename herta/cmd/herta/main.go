// Command herta is the Herta language compiler front end.
//
//	herta check <file.he>          parse + contract-check, print diagnostics
//	herta ast <file.he>            print the parsed AST dump (golden format)
//	herta build [-pkg name] <file.he>   compile to Go calling hertasdk, on stdout
//	herta trust [-closed] [-json] [-strict] <file.he>   print transitive trust assumptions
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/arahe-dev/hertasdk/herta/gogen"
	"github.com/arahe-dev/hertasdk/herta/lang"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: herta check|ast|build|trust [flags] <file.he>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "trust":
		fs := flag.NewFlagSet("trust", flag.ExitOnError)
		closed := fs.Bool("closed", false, "model closed-world build mode")
		asJSON := fs.Bool("json", false, "print machine-readable trust assumptions")
		strict := fs.Bool("strict", false, "fail if any required trust assumption remains")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "usage: herta trust [-closed] [-json] [-strict] <file.he>")
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
		report := lang.Trust(prog, *closed)
		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(report); err != nil {
				fmt.Fprintln(os.Stderr, "herta:", err)
				os.Exit(2)
			}
		} else {
			fmt.Print(lang.DumpTrust(report))
		}
		if *strict && lang.TrustRequired(report) {
			os.Exit(1)
		}
	case "check", "ast":
		fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
		closed := fs.Bool("closed", false, "enforce closed-world contracts")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			fmt.Fprintf(os.Stderr, "usage: herta %s [-closed] <file.he>\n", os.Args[1])
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
		if os.Args[1] == "ast" {
			fmt.Print(lang.Dump(prog))
			return
		}
		check := lang.Check
		if *closed {
			check = lang.CheckClosed
		}
		diags := check(prog)
		for _, d := range diags {
			fmt.Fprintln(os.Stdout, d)
		}
		if len(diags) > 0 {
			os.Exit(1)
		}
	case "build":
		fs := flag.NewFlagSet("build", flag.ExitOnError)
		pkg := fs.String("pkg", "main", "package name for the generated file")
		closed := fs.Bool("closed", false, "enforce closed-world contracts")
		fs.Parse(os.Args[2:])
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "usage: herta build [-closed] [-pkg name] <file.he>")
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
		check := lang.Check
		if *closed {
			check = lang.CheckClosed
		}
		diags := check(prog)
		for _, d := range diags {
			fmt.Fprintln(os.Stdout, d)
		}
		if len(diags) > 0 {
			os.Exit(1)
		}
		generate := gogen.Generate
		if *closed {
			generate = gogen.GenerateClosed
		}
		out, err := generate(prog, *pkg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "herta:", err)
			os.Exit(2)
		}
		fmt.Print(out)
	default:
		fmt.Fprintln(os.Stderr, "usage: herta check|ast|build|trust [flags] <file.he>")
		os.Exit(2)
	}
}
