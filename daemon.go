package main

import (
	"errors"
	"log"
	"net"
	"os"
	"strings"

	"github.com/google/uuid"
)

// The daemon is executed using "eternal daemon".
// It listens to connections from one or more listeners,
// and stores the data into one backend.

// Usage:
//
// eternal daemon [-listen listener]... [-backend scheme]
//
// The default listener is:
// - MaxOS: sqlite://$HOME/Library/Application Support/eternal/history.db
// - other: sqlite://$HOME/.local/share/eternal/history.db
//

func cmd_daemon(args []string) error {
	c, err := connect()
	if err == nil {
		c.Close()
		return errors.New("daemon already running")
	}
	log.Println("Starting daemon")
	os.Remove(socketName())
	l, err := net.Listen("unix", socketName())
	if err != nil {
		return err
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		log.Println("Accepted new connection")
		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 1024)
			nr, err := c.Read(buf)
			if err != nil {
				return
			}
			data := string(buf[0:nr])
			log.Printf("Got: %q", data)
			cmd, after, found := strings.Cut(data, " ")
			if !found {
				log.Printf("Error: got %q\n", data)
			}
			_ = after
			switch cmd {
			case "init":
				c.Write([]byte(uuid.NewString()))
			case "start":
				c.Write([]byte("42"))
			case "end":
				c.Write([]byte("ok"))
			default:
				log.Printf("Error: got %q\n", data)
			}
			return
		}(c)
	}
	return nil
}
