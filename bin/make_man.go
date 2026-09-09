//go:build ignore

// make_man converts the local manual into a roff man page without pandoc.
package main

import (
	"fmt"
	"os"

	"github.com/cpuguy83/go-md2man/v2/md2man"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s INPUT.md OUTPUT.1\n", os.Args[0])
		os.Exit(2)
	}
	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(os.Args[2], md2man.Render(input), 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
