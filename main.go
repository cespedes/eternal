package main

import (
	"errors"
	"fmt"
	"log"
	"os"
)

func usage() error {
	return errors.New("usage: eternal { init | start | end } [options]")
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
	switch os.Args[1] {
	case "daemon":
		return cmd_daemon(args)
	case "init":
		return cmd_init(args)
	case "start":
		return cmd_start(args)
	case "end":
		return cmd_end(args)
	default:
		return usage()
	}
}

func cmd_start(args []string) error {
	log.Println("I am the start.")
	return nil
}

func cmd_end(args []string) error {
	log.Println("I am the end.")
	return nil
}
