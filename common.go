package main

import (
	"errors"
	"net"
	"os"
	"os/user"
	"path/filepath"
)

//
// Default socket location:
//
// - $XDG_RUNTIME_DIR/eternal
// - $TMPDIR/$USER-eternal
// - /tmp/$USER-eternal
//

func socketName() string {
	filename := "eternal"
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir != "" {
		return filepath.Join(dir, filename)
	}

	username := os.Getenv("USER")
	user, err := user.Current()
	if err == nil {
		username = user.Username
	}
	if username != "" {
		filename = username + "-" + filename
	}

	dir = os.Getenv("TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	return filepath.Join(dir, filename)
}

func connect() (net.Conn, error) {
	return net.Dial("unix", socketName())
}

func getSession() (string, error) {
	session := os.Getenv("ETERNAL_SESSION")
	if session == "" {
		return "", errors.New("no ETERNAL_SESSION in environment")
	}
	return session, nil
}
