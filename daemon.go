package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// The daemon is executed using "eternal daemon".
// It listens to connections from one or more listeners,
// and stores the data into one backend.

// Usage:
//
// eternal daemon [-listen listener]... [-backend scheme]
//
// The default backend is:
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
	var dbdir, dbfile string
	switch runtime.GOOS {
	case "darwin":
		dbdir = filepath.Join(os.Getenv("HOME"), "Application Support", "eternal")
	default:
		dbdir = filepath.Join(os.Getenv("HOME"), ".local", "share", "eternal")
	}
	dbfile = filepath.Join(dbdir, "history.db")
	db, err := sql.Open("sqlite", dbfile)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS eternal_session(id INTEGER primary key, created timestamp not null default (datetime()), uuid text unique not null, hostname text not null, username text not null, tty text not null, pid int not null);
		CREATE TABLE IF NOT EXISTS eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), cwd text not null, start timestamp not null default (datetime()), exit int, duration int, command text not null);
	`)

	err = os.MkdirAll(dbdir, 0700)
	if err != nil {
		return err
	}
	defer l.Close()
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		// log.Println("Accepted new connection")
		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 1024)
			nr, err := c.Read(buf)
			if err != nil {
				return
			}
			data := string(buf[0:nr])
			log.Printf("Got: %q", data)
			cmd, rest, found := strings.Cut(data, " ")
			if !found {
				log.Printf("Error: got %q\n", data)
			}
			switch cmd {
			case "init":
				// Expected: init hostname username tty pid
				f := strings.Fields(rest)
				if len(f) != 4 {
					log.Printf("Error: got %q\n", data)
				}
				uuid := uuid.NewString()
				hostname := f[0]
				username := f[1]
				tty := f[2]
				pid := f[3]
				err = daemonNewSession(db, uuid, hostname, username, tty, pid)
				if err != nil {
					return
				}
				c.Write([]byte(uuid))
			case "start":
				// Expected: start session cwd\000command
				sess, rest, ok := strings.Cut(rest, " ")
				if !ok {
					log.Printf("Error 2: got %q\n", data)
				}
				cwd, command, ok := strings.Cut(rest, "\000")
				if !ok {
					log.Printf("Error 3: got %q\n", data)
				}
				id, err := daemonStartCommand(db, sess, cwd, command)
				if err != nil {
					log.Printf("Error: %v\n", err)
					return
				}
				log.Printf("START: sess=%s cwd=%q command=%q", sess, cwd, command)
				c.Write([]byte(strconv.Itoa(id)))
			case "end":
				// Expected: end session id exit tstamp_start tstamp_end
				c.Write([]byte("ok"))
				f := strings.Fields(rest)
				if len(f) != 5 {
					log.Printf("Error: got %q\n", data)
				}
				sess := f[0]
				id := f[1]
				exit := f[2]
				timeStart := f[3]
				timeEnd := f[4]
				err = daemonEndCommand(db, sess, id, exit, timeStart, timeEnd)
				if err != nil {
					log.Printf("Error: %v\n", err)
					return
				}
			default:
				log.Printf("Error: got %q\n", data)
			}
			return
		}(c)
	}
	return nil
}

func daemonNewSession(db *sql.DB, uuid string, hostname string, username string, tty string, pid string) error {
	_, err := db.Exec(`
		INSERT INTO eternal_session(uuid, hostname, username, tty, pid)
		VALUES (?, ?, ?, ?, ?)
	`, uuid, hostname, username, tty, pid)
	if err != nil {
		return err
	}
	return nil
}

func daemonStartCommand(db *sql.DB, sess string, cwd string, command string) (int, error) {
	var id int
	row := db.QueryRow(`
		INSERT INTO eternal_command(session_id, cwd, command)
		SELECT id, ?, ? FROM eternal_session WHERE uuid=?
		RETURNING id
	`, cwd, command, sess)
	err := row.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("INSERTING command: %w", err)
	}
	return id, nil
}

func daemonEndCommand(db *sql.DB, sess string, id string, exit string, timeStart string, timeEnd string) error {
	t1, err := strconv.ParseFloat(timeStart, 64)
	if err != nil {
		return err
	}
	t2, err := strconv.ParseFloat(timeEnd, 64)
	if err != nil {
		return err
	}
	duration := uint((t2 - t1) * 1_000_000_000)
	_, err = db.Exec(`
		UPDATE eternal_command
		SET exit=?, duration=?
		WHERE id=? AND session_id=(SELECT id FROM eternal_session WHERE uuid=?)
	`, exit, duration, id, sess)
	if err != nil {
		return fmt.Errorf("UPDATING command: %w", err)
	}
	return nil
}
