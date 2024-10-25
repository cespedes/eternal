package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"path"
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
			c.Write([]byte("received: " + string(data)))
			return
		}(c)
	}
	return nil
}

func cmd_init(args []string) error {
	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()
	log.Println("I am the init.")
	// We need a few things: hostname, logname, tty, parent PID:
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	logname := user.Username
	tty, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		return err
	}
	ppid := os.Getppid()

	_, err = c.Write([]byte(fmt.Sprintf("init %s %s %s %d", hostname, logname, tty, ppid)))
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
	return nil
}

func cmd_start(args []string) error {
	log.Println("I am the start.")
	return nil
}

func cmd_end(args []string) error {
	log.Println("I am the end.")
	return nil
}

func connect() (net.Conn, error) {
	return net.Dial("unix", socketName())
}

func socketName() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	filename := "eternal"
	if dir == "" {
		filename = "/tmp/" + os.Getenv("LOGNAME") + "-eternal"
	} else {
		filename = path.Join(dir, filename)
	}
	return filename
}
