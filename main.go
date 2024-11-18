package main

import (
	"errors"
	"fmt"
	"os"
)

func usage() error {
	return errors.New("usage: eternal { init | start | end | history } [options]")
}

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return usage()
	}
	if f := commands[os.Args[1]]; f != nil {
		return f(args)
	}
	return usage()
}
