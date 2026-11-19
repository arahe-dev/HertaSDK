// Diagnostic-bypass generator for evaluation only, not a compiler acceptance path.
package main

import (
	"fmt"
	"os"

	"github.com/arahe-dev/hertasdk/herta/gogen"
	"github.com/arahe-dev/hertasdk/herta/lang"
)

func main() {
	src, err := os.ReadFile(os.Args[1])
	if err != nil { panic(err) }
	p, err := lang.Parse(string(src))
	if err != nil { panic(err) }
	out, err := gogen.GenerateClosed(p, os.Args[2])
	if err != nil { panic(err) }
	fmt.Print(out)
}
