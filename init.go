package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
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

	m := map[string]string{"action": "init"}

	// We need a few things: hostname, username, tty, parent PID:
	m["hostname"], err = os.Hostname()
	if err != nil {
		return err
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	m["username"] = user.Username

	// TTY: we first try the Linux way:
	m["tty"], err = os.Readlink("/proc/self/fd/0")
	if err != nil {
		// If unavailable, we execute "tty":
		cmd := exec.Command("tty")
		cmd.Stdin = os.Stdin
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf(`cannot exec "tty": %w`, err)
		}
		m["tty"] = strings.TrimSpace(string(out))
	}

	pid := os.Getppid()
	m["pid"] = strconv.Itoa(pid)

	proc, err := ps.FindProcess(pid)
	if err != nil {
		return err
	}
	m["shell"] = proc.Executable()
	proc, err = ps.FindProcess(proc.PPid())
	m["parent"] = proc.Executable()

	m["os"] = runtime.GOOS + "/" + runtime.GOARCH
	m["origin"], _, _ = strings.Cut(os.Getenv("SSH_CLIENT"), " ")

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = c.Write(b)
	if err != nil {
		return err
	}

	buf := make([]byte, 1024)
	nr, err := c.Read(buf)
	if err != nil {
		return err
	}
	data := buf[0:nr]

	var session struct {
		Session string `json:"session"`
	}
	err = json.Unmarshal(data, &session)
	if err != nil {
		return err
	}
	fmt.Println(session.Session)
	return nil
}
