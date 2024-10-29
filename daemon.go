package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

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

const chanSize = 50

type Session struct {
	OS       string // GOOS/GOARCH (eg, "linux/amd64", "darwin/amd64"...)
	Shell    string
	Parent   string
	Origin   string // remote IP if this is a SSH connection
	Hostname string
	Username string
	TTY      string
	PID      int
}

type Entry struct {
	Session
	WorkingDir string
	Timestamp  string
	Command    string
	ExitStatus int
	Duration   int // milliseconds
}

type Command struct {
	Input  map[string]string
	Output chan map[string]string
}

func (c Command) String() string {
	return fmt.Sprint(c.Input)
}

func cmdDaemon(args []string) error {
	if c, err := connect(); err == nil {
		c.Close()
		return errors.New("daemon already running")
	}
	log.Println("Starting daemon")
	os.Remove(socketName())
	ln, err := net.Listen("unixpacket", socketName())
	if err != nil {
		return err
	}
	defer ln.Close()

	var dbdir, dbfile string
	switch runtime.GOOS {
	case "darwin":
		dbdir = filepath.Join(os.Getenv("HOME"), "Application Support", "eternal")
	default:
		dbdir = filepath.Join(os.Getenv("HOME"), ".local", "share", "eternal")
	}
	err = os.MkdirAll(dbdir, 0700)
	if err != nil {
		return err
	}
	dbfile = filepath.Join(dbdir, "history.db")
	db, err := sql.Open("sqlite", dbfile)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS eternal_session(id INTEGER primary key, created timestamp not null default (datetime('now','localtime')), session text unique not null, os text not null default '', shell text not null default '', parent string not null default '', origin text not null default '', hostname text not null, username text not null, tty text not null, pid int not null);
		CREATE TABLE IF NOT EXISTS eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), working_dir text not null, start timestamp not null default (datetime('now','localtime')), exit int, duration int, command text not null);
	`)
	if err != nil {
		return fmt.Errorf("trying to create SQL tables: %w", err)
	}

	cc := make(chan Command, chanSize)
	go daemonBackend(db, cc)

	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			buf := make([]byte, 1024)
			nr, err := c.Read(buf)
			if err != nil {
				return
			}
			data := buf[0:nr]
			// log.Printf("Got: %q", data)

			var cmd Command
			err = json.Unmarshal(data, &cmd.Input)
			if err != nil {
				return
			}
			cmd.Output = make(chan map[string]string, 1)
			cc <- cmd
			for response := range cmd.Output {
				b, err := json.Marshal(response)
				if err != nil {
					return
				}
				c.Write(b)
			}
		}(c)
	}
	return nil
}

func daemonBackend(db *sql.DB, cc chan Command) {
	var err error

	for cmd := range cc {
		switch cmd.Input["action"] {
		case "init":
			// Expected: init os hostname username tty pid
			m := cmd.Input
			m["session"] = uuid.NewString()
			err = sqliteInit(db, m)
			if err != nil {
				return
			}
			cmd.Output <- map[string]string{"session": m["session"]}
			close(cmd.Output)
		case "start":
			// Expected: start session working_dir command
			m := cmd.Input
			_, err := sqliteStartCommand(db, m)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			close(cmd.Output)
		case "end":
			// Expected: end session exit tstamp_start tstamp_end
			m := cmd.Input
			err = sqliteEndCommand(db, m)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			close(cmd.Output)
		case "history":
			m := cmd.Input
			history, err := sqliteHistory(db, m["session"])
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}

			for _, e := range history {
				cmd.Output <- map[string]string{
					"os":          e.OS,
					"shell":       e.Shell,
					"parent":      e.Parent,
					"origin":      e.Origin,
					"hostname":    e.Hostname,
					"username":    e.Username,
					"tty":         e.TTY,
					"pid":         strconv.Itoa(e.PID),
					"working_dir": e.WorkingDir,
					"timestamp":   e.Timestamp,
					"command":     e.Command,
					"exit_status": strconv.Itoa(e.ExitStatus),
					"duration":    strconv.Itoa(e.Duration),
				}
			}
			close(cmd.Output)
		default:
			log.Printf("Error: got %q\n", cmd)
		}
	}
}

// CREATE TABLE IF NOT EXISTS eternal_session(id INTEGER primary key, created timestamp not null default (datetime('now','localtime')), session text unique not null, os text not null default '', shell text not null default '', parent string not null default '', origin text not null default '', hostname text not null, username text not null, tty text not null, pid int not null);

func sqliteInit(db *sql.DB, m map[string]string) error {
	log.Printf("sqliteInit(%v)", m)
	_, err := db.Exec(`
		INSERT INTO eternal_session(session, os, shell, parent, origin,
			hostname, username, tty, pid)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m["session"], m["os"], m["shell"], m["parent"], m["origin"],
		m["hostname"], m["username"], m["tty"], m["pid"])
	if err != nil {
		return err
	}
	return nil
}

func sqliteStartCommand(db *sql.DB, m map[string]string) (int, error) {
	log.Printf("sqliteStartCommand(%v)", m)
	var id int
	row := db.QueryRow(`
		INSERT INTO eternal_command(session_id, working_dir, command)
		SELECT id, ?, ? FROM eternal_session WHERE session=?
		RETURNING id
	`, m["working_dir"], m["command"], m["session"])
	err := row.Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("INSERTING command: %w", err)
	}
	return id, nil
}

func sqliteEndCommand(db *sql.DB, m map[string]string) error {
	log.Printf("sqliteEndCommand(%v)", m)
	t1, err := strconv.ParseFloat(m["start"], 64)
	if err != nil {
		return err
	}
	t2, err := strconv.ParseFloat(m["end"], 64)
	if err != nil {
		return err
	}
	duration := uint((t2 - t1) * 1_000_000)
	_, err = db.Exec(`
		UPDATE eternal_command
		SET exit=?, duration=?
		WHERE exit IS NULL AND id=(SELECT MAX(id) FROM eternal_command WHERE session_id=(SELECT id FROM eternal_session WHERE session=?))
	`, m["status"], duration, m["session"])
	if err != nil {
		return fmt.Errorf("UPDATING command: %w", err)
	}
	return nil
}

// CREATE TABLE IF NOT EXISTS eternal_session(id INTEGER primary key, created timestamp not null default (datetime('now','localtime')), session text unique not null, os text not null default '', shell text not null default '', parent string not null default '', origin text not null default '', hostname text not null, username text not null, tty text not null, pid int not null);
// CREATE TABLE IF NOT EXISTS eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), working_dir text not null, start timestamp not null default (datetime('now','localtime')), exit int, duration int, command text not null);

func sqliteHistory(db *sql.DB, session string) ([]Entry, error) {
	var e Entry
	rows, err := db.Query(`
		SELECT
			s.os, s.shell, s.parent, s.origin,
			s.hostname, s.username, s.tty, s.pid,
			c.working_dir, datetime(c.start) AS start, c.command,
			COALESCE(c.exit,'0') AS status, COALESCE(c.duration,'0') AS duration
		FROM eternal_command c
		LEFT JOIN eternal_session s ON c.session_id=s.id
		ORDER BY c.id
	`, session)
	if err != nil {
		return nil, fmt.Errorf("SELECT command: %w", err)
	}
	var h []Entry
	for rows.Next() {
		if err = rows.Scan(&e.OS, &e.Shell, &e.Parent, &e.Origin,
			&e.Hostname, &e.Username, &e.TTY, &e.PID,
			&e.WorkingDir, &e.Timestamp, &e.Command,
			&e.ExitStatus, &e.Duration); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		h = append(h, e)
	}
	return h, nil
}

// select * FROM eternal_command where session_id=4 and exit is null and id=(select max(id) FROM eternal_command where session_id=4);
