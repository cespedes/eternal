package main

import (
	"errors"
	"log"
	"net"
	"os"

	"github.com/google/uuid"
)

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
			data := buf[0:nr]
			log.Printf("Got: %s\n", data)
			c.Write([]byte(uuid.NewString()))
			return
		}(c)
	}
	return nil
}
