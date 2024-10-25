package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"syscall"
	"time"
)

func cmd_init(args []string) error {
	c, err := connect()
	if err != nil {
		// No daemon found; let's launch it and retry
		path, err := os.Executable()
		if err != nil {
			return err
		}
		args := []string{os.Args[0], "daemon"}
		env := os.Environ()

		// Use ForkExec to fork the process and re-execute the command
		_, err = syscall.ForkExec(path, args, &syscall.ProcAttr{
			Env: env,
		})
		if err != nil {
			return err
		}

		time.Sleep(100 * time.Millisecond)
		c, err = connect()
		if err != nil {
			return err
		}
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
