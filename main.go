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
	if len(args) < 4 {
		return nil
	}
	c, err := connect()
	if err != nil {
		return err
	}

	session := args[2]
	command := args[3]
	// log.Printf("Sending to daemon: start %s", cmdline)
	_, err = c.Write([]byte(fmt.Sprintf("start %s %s", session, command)))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	nr, err := c.Read(buf)
	if err != nil {
		return err
	}
	data := buf[0:nr]
	log.Printf("Got: %s\n", data)
	fmt.Println(string(data))

	log.Println("I am the start.")
	return nil
}

func cmd_end(args []string) error {
	if len(args) < 6 {
		return nil
	}
	c, err := connect()
	if err != nil {
		return err
	}

	session := args[2]
	id := args[3]
	status := args[4]
	duration := args[5]

	// log.Printf("Sending to daemon: end %s", cmdline)
	_, err = c.Write([]byte(fmt.Sprintf("end %s %s %s %s", session, id, status, duration)))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	nr, err := c.Read(buf)
	if err != nil {
		return err
	}
	data := buf[0:nr]
	log.Printf("Got: %s\n", data)
	fmt.Println(string(data))

	log.Println("I am the end.")
	return nil
}
