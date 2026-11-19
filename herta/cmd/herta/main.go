// Command herta is the Herta language compiler front end.
//
//	herta check <file.he>   parse + contract-check, print diagnostics
//	herta ast <file.he>     print the parsed AST dump (golden format)
package main

import (
	"fmt"
	"os"

	"github.com/arahe-dev/hertasdk/herta/lang"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: herta check|ast <file.he>")
		os.Exit(2)
	}
	src, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "herta:", err)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "check":
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
	case "ast":
		prog, err := lang.Parse(string(src))
		if err != nil {
			fmt.Fprintln(os.Stdout, err)
			os.Exit(1)
		}
		fmt.Print(lang.Dump(prog))
	default:
		fmt.Fprintln(os.Stderr, "usage: herta check|ast <file.he>")
		os.Exit(2)
	}
}
