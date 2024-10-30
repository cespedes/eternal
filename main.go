package main

import (
	"encoding/json"
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

	m := map[string]string{"action": "start"}

	m["session"], err = getSession()
	if err != nil {
		return err
	}
	m["working_dir"], err = os.Getwd()
	if err != nil {
		m["working_dir"] = "(error)"
	}
	m["command"] = args[2]
	// log.Printf("eternal start: Sending to daemon: %v", m)
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = c.Write(b)
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

	m := map[string]string{"action": "end"}

	m["session"], err = getSession()
	if err != nil {
		return err
	}
	m["status"] = args[2]
	m["start"] = args[3]
	m["end"] = args[4]
	// log.Printf("eternal end: Sending to daemon: %v", m)
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = c.Write(b)
	if err != nil {
		return err
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
