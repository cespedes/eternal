package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mitchellh/go-ps"
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
	var session Session
	session.Hostname, err = os.Hostname()
	if err != nil {
		return err
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	session.Username = user.Username

	// TTY: we first try the Linux way:
	session.TTY, err = os.Readlink("/proc/self/fd/0")
	if err != nil {
		// If unavailable, we execute "tty":
		cmd := exec.Command("tty")
		cmd.Stdin = os.Stdin
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf(`cannot exec "tty": %w`, err)
		}
		session.TTY = strings.TrimSpace(string(out))
	}

	session.PID = os.Getppid()

	proc, err := ps.FindProcess(session.PID)
	if err != nil {
		return err
	}
	session.Shell = proc.Executable()
	proc, err = ps.FindProcess(proc.PPid())
	session.Parent = proc.Executable()

	session.OS = runtime.GOOS + "/" + runtime.GOARCH
	session.Origin, _, _ = strings.Cut(os.Getenv("SSH_CLIENT"), " ")

	log.Printf("eternal init: session=%+v", session)

	// log.Printf("Sending to daemon: init %s %s %s %d", hostname, username, tty, ppid)
	_, err = c.Write([]byte(fmt.Sprintf("init %s %s %s %d",
		session.Hostname, session.Username, session.TTY, session.PID)))
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
