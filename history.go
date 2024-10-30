package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func showDuration(d int) string {
	if d < 0 {
		return ""
	}
	if d < 1000 {
		return fmt.Sprintf("%dµs", d)
	}
	if d < 10_000 {
		return fmt.Sprintf("%d.%dms", d/1000, (d%1000)/100)
	}
	d /= 1000
	if d < 1000 {
		return fmt.Sprintf("%dms", d)
	}
	if d < 10_000 {
		return fmt.Sprintf("%d.%02ds", d/1000, (d%1000)/10)
	}
	if d < 60_000 {
		return fmt.Sprintf("%d.%ds", d/1000, (d%1000)/100)
	}
	d /= 1000
	if d < 600 {
		return fmt.Sprintf("%dm%ds", d/60, d%60)
	}
	d /= 60
	if d < 60 {
		return fmt.Sprintf("%dm", d)
	}
	if d < 600 {
		return fmt.Sprintf("%dh%dm", d/60, d%60)
	}
	d /= 60
	if d < 24 {
		return fmt.Sprintf("%dh", d)
	}
	if d < 240 {
		return fmt.Sprintf("%dd%dh", d/24, d%24)
	}
	d /= 24
	return fmt.Sprintf("%dd", d)
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

	m := map[string]string{"action": "history"}

	m["session"], err = getSession()
	if err != nil {
		return err
	}

	// log.Printf("eternal history: Sending to daemon: %v", m)
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = c.Write(b)
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
		var o map[string]string
		err = json.Unmarshal(data, &o)
		if err != nil {
			return err
		}
		d, err := strconv.Atoi(o["duration"])
		if err != nil {
			d = -1
		}
		fmt.Printf("%s (%-6s) %5s %s\n", o["timestamp"], strings.TrimPrefix(o["tty"], "/dev/"), showDuration(d), o["command"])
	}

	return nil
}
