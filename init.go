package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"
)

func cmdInit(args []string) error {
	log.Println("eternal starting")
	c, err := connect()
	if err != nil {
		// No daemon found; let's launch it and retry
		path, err := os.Executable()
		if err != nil {
			return err
		}
		args := []string{os.Args[0], "daemon"}
		env := os.Environ()

		// fds := []uintptr{
		// 	os.Stdin.Fd(),
		// }

		// Use ForkExec to fork the process and re-execute the command
		_, err = syscall.ForkExec(path, args, &syscall.ProcAttr{
			Env: env,
			// Files: fds,
		})
		if err != nil {
			return err
		}

		for range 100 { // wait for daemon, for up to 5 seconds
			c, err = connect()
			if err == nil {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		if err != nil {
			return fmt.Errorf("cannot create daemon: %w", err)
		}
	}
	defer c.Close()
	// We need a few things: hostname, username, tty, parent PID:
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	username := user.Username

	// TTY: we first try the Linux way:
	tty, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		// If unavailable, we execute "tty":
		cmd := exec.Command("tty")
		cmd.Stdin = os.Stdin
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf(`cannot exec "tty": %w`, err)
		}
		tty = strings.TrimSpace(string(out))
	}

	ppid := os.Getppid()

	// log.Printf("Sending to daemon: init %s %s %s %d", hostname, username, tty, ppid)
	_, err = c.Write([]byte(fmt.Sprintf("init %s %s %s %d", hostname, username, tty, ppid)))
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	nr, err := c.Read(buf)
	if err != nil {
		return err
	}
	data := buf[0:nr]
	// log.Printf("Got: %s\n", data)
	fmt.Println(string(data))
	return nil
}
