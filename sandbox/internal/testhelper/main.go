// Command testhelper is compiled and invoked by the sandbox integration tests.
// It calls sandbox.Init() first, then for each path on the command line
// tries os.Open and prints one line:
//
//	OK:<path>
//	ERR:<path>:<err>
package main

import (
	"fmt"
	"os"

	"github.com/leftmike/research/sandbox"
)

func main() {
	sandbox.Init()
	for _, p := range os.Args[1:] {
		f, err := os.Open(p)
		if err != nil {
			fmt.Printf("ERR:%s:%v\n", p, err)
			continue
		}
		f.Close()
		fmt.Printf("OK:%s\n", p)
	}
}
