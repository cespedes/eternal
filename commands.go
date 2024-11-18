package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"
)

var commands = map[string]func([]string) error{
	"daemon":  cmdDaemon,
	"init":    cmdInit,
	"start":   cmdStart,
	"end":     cmdEnd,
	"history": cmdHistory,
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

	m := map[string]any{"action": "start"}

	m["session"], err = getSession()
	if err != nil {
		return err
	}
	m["working_dir"], err = os.Getwd()
	if err != nil {
		m["working_dir"] = "(error)"
	}
	m["command"] = args[2]
	m["start"] = time.Now().UnixMicro()
	// log.Printf("eternal start: Sending to daemon: %v", m)
	enc := json.NewEncoder(c)
	err = enc.Encode(m)
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

	m := map[string]any{"action": "end"}

	m["session"], err = getSession()
	if err != nil {
		return err
	}
	m["status"] = args[2]
	if f, err := strconv.ParseFloat(args[3], 64); err == nil {
		m["start"] = int(f * 1_000_000)
	}
	if f, err := strconv.ParseFloat(args[4], 64); err == nil {
		m["end"] = int(f * 1_000_000)
	}
	if m["end"] == nil {
		m["end"] = time.Now().UnixMicro()
	}
	// log.Printf("eternal end: Sending to daemon: %v", m)
	enc := json.NewEncoder(c)
	err = enc.Encode(m)
	if err != nil {
		return err
	}

	return nil
}
