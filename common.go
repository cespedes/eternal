package main

import (
	"net"
	"os"
	"path"
)

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
