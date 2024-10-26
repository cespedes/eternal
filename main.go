package main

import (
	"errors"
	"fmt"
	"io"
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
		return cmdDaemon(args)
	case "init":
		return cmdInit(args)
	case "start":
		return cmdStart(args)
	case "end":
		return cmdEnd(args)
	case "history":
		return cmdHistory(args)
	default:
		return usage()
	}
}

func cmdStart(args []string) error {
	if len(args) != 3 {
		return usage()
	}
	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	session, err := getSession()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "(error)"
	}
	command := args[2]
	// log.Printf("eternal start: Sending to daemon: start %s %s %s", session, cmd, command)
	// We use '\000' between cwd and command so that any of them can contain spaces
	_, err = c.Write([]byte(fmt.Sprintf("start %s %s\000%s", session, cwd, command)))
	if err != nil {
		return err
	}

	return nil
}

func cmdEnd(args []string) error {
	if len(args) != 5 {
		return usage()
	}
	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	session, err := getSession()
	if err != nil {
		return err
	}
	status := args[2]
	start := args[3]
	end := args[4]

	// log.Printf("eternal end: Sending to daemon: end %s %s %s %s", session, status, start, end)
	_, err = c.Write([]byte(fmt.Sprintf("end %s %s %s %s", session, status, start, end)))
	if err != nil {
		return err
	}

	return nil
}

func cmdHistory(args []string) error {
	if len(args) != 2 {
		return usage()
	}
	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	session, err := getSession()
	if err != nil {
		return err
	}

	// log.Printf("eternal history: Sending to daemon: history")
	_, err = c.Write([]byte(fmt.Sprintf("history %s", session)))
	if err != nil {
		return err
	}
	for {
		buf := make([]byte, 1024)
		nr, err := c.Read(buf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		data := buf[0:nr]
		fmt.Print(string(data))
	}

	return nil
}

func getSession() (string, error) {
	session := os.Getenv("ETERNAL_SESSION")
	if session == "" {
		return "", errors.New("no ETERNAL_SESSION in environment")
	}
	return session, nil
}
