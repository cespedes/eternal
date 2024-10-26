package main

import (
	"net"
	"os"
	"os/user"
	"path"
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
		return path.Join(dir, filename)
	}

	username := os.Getenv("USER")
	user, err := user.Current()
	if err == nil {
		username = user.Username
	}
	username = user.Username
	if username != "" {
		filename = username + "-" + filename
	}

	dir = os.Getenv("TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	return path.Join(dir, filename)
}

func connect() (net.Conn, error) {
	return net.Dial("unix", socketName())
}
