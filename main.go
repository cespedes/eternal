package main

import (
	"errors"
	"fmt"
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
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "(error)"
	}
	command := args[3]
	// We use '\000' between cwd and command so that any of them can contain spaces
	_, err = c.Write([]byte(fmt.Sprintf("start %s %s\000%s", session, cwd, command)))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	nr, err := c.Read(buf)
	if err != nil {
		return err
	}
	data := buf[0:nr]
	fmt.Println(string(data))

	// log.Printf("Sent: `start %s %q %s`. Received: %s", session, cwd, command, data)
	return nil
}

func cmd_end(args []string) error {
	if len(args) < 7 {
		return nil
	}
	c, err := connect()
	if err != nil {
		return err
	}

	session := args[2]
	id := args[3]
	status := args[4]
	start := args[5]
	end := args[6]

	// log.Printf("Sending to daemon: end %s", cmdline)
	_, err = c.Write([]byte(fmt.Sprintf("end %s %s %s %s %s", session, id, status, start, end)))
	if err != nil {
		return err
	}

	/*
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)
		if err != nil {
			return err
		}
		data := buf[0:nr]
	*/

	// log.Printf("Sent: \"end %s %s %s %s %s\"", session, id, status, start, end)
	return nil
}
