package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/jhillyerd/enmime"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [inputfile]\n", os.Args[0])
	os.Exit(2)
}

func main() {
	if len(os.Args) < 1 {
		usage()
	}
}