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

const chanSize = 50

type Entry struct {
	Hostname   string
	Username   string
	TTY        string
	PID        int
	WorkingDir string
	Timestamp  string
	Cmd        string
	ExitStatus int
	Elapsed    int // milliseconds
}

type Command struct {
	Name     string
	Args     string
	Response chan string
	History  chan []Entry
}

func (c Command) String() string {
	return c.Name + " " + c.Args
}

func cmdDaemon(args []string) error {
	if c, err := connect(); err == nil {
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
		CREATE TABLE IF NOT EXISTS eternal_session(id INTEGER primary key, created timestamp not null default (datetime()), uuid text unique not null, hostname text not null, username text not null, tty text not null, pid int not null);
		CREATE TABLE IF NOT EXISTS eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), cwd text not null, start timestamp not null default (datetime()), exit int, duration int, command text not null);
	`)
	if err != nil {
		return fmt.Errorf("trying to create SQL tables: %w", err)
	}

	cc := make(chan Command, chanSize)
	go daemonBackendSqlite(db, cc)

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

			var cmd Command
			cmd.Name, cmd.Args, _ = strings.Cut(data, " ")
			if cmd.Name == "init" {
				cmd.Response = make(chan string, 1)
			}
			if cmd.Name == "history" {
				cmd.History = make(chan []Entry, 1)
			}
			cc <- cmd
			if cmd.Name == "init" {
				response := <-cmd.Response
				c.Write([]byte(response))
			}
			if cmd.Name == "history" {
				history := <-cmd.History
				for _, e := range history {
					c.Write([]byte(fmt.Sprintf("%s %s\n", e.Timestamp, e.Cmd)))
				}
				_ = history
			}
		}(c)
	}
	return nil
}

func daemonBackendSqlite(db *sql.DB, cc chan Command) {
	var err error

	for cmd := range cc {
		switch cmd.Name {
		case "init":
			// Expected: init hostname username tty pid
			f := strings.Fields(cmd.Args)
			if len(f) != 4 {
				log.Printf("Error: got %q\n", cmd)
			}
			uuid := uuid.NewString()
			hostname := f[0]
			username := f[1]
			tty := f[2]
			pid := f[3]
			err = sqliteNewSession(db, uuid, hostname, username, tty, pid)
			log.Printf("New session: host=%q user=%q tty=%q pid=%s", hostname, username, tty, pid)
			if err != nil {
				return
			}
			cmd.Response <- uuid
		case "start":
			// Expected: start session cwd\000command
			sess, rest, ok := strings.Cut(cmd.Args, " ")
			if !ok {
				log.Printf("Error 2: got %q\n", cmd)
			}
			cwd, command, ok := strings.Cut(rest, "\000")
			if !ok {
				log.Printf("Error 3: got %q\n", cmd)
			}
			_, err := sqliteStartCommand(db, sess, cwd, command)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			log.Printf("START: sess=%s cwd=%q command=%q", sess, cwd, command)
		case "end":
			// Expected: end session exit tstamp_start tstamp_end
			f := strings.Fields(cmd.Args)
			if len(f) != 4 {
				log.Printf("Error: got %q\n", cmd)
			}
			sess := f[0]
			exit := f[1]
			timeStart := f[2]
			timeEnd := f[3]
			err = sqliteEndCommand(db, sess, exit, timeStart, timeEnd)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			log.Printf("END: sess=%s exit=%s start=%s end=%s", sess, exit, timeStart, timeEnd)
		case "history":
			f := strings.Fields(cmd.Args)
			if len(f) != 1 {
				log.Printf("Error: got %q\n", cmd)
			}
			sess := f[0]
			history, err := sqliteHistory(db, sess)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			log.Println("HISTORY")
			cmd.History <- history
		default:
			log.Printf("Error: got %q\n", cmd)
		}
	}
}

func sqliteNewSession(db *sql.DB, uuid string, hostname string, username string, tty string, pid string) error {
	_, err := db.Exec(`
		INSERT INTO eternal_session(uuid, hostname, username, tty, pid)
		VALUES (?, ?, ?, ?, ?)
	`, uuid, hostname, username, tty, pid)
	if err != nil {
		return err
	}
	return nil
}

func sqliteStartCommand(db *sql.DB, sess string, cwd string, command string) (int, error) {
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

func sqliteEndCommand(db *sql.DB, sess string, exit string, timeStart string, timeEnd string) error {
	t1, err := strconv.ParseFloat(timeStart, 64)
	if err != nil {
		return err
	}
	t2, err := strconv.ParseFloat(timeEnd, 64)
	if err != nil {
		return err
	}
	duration := uint((t2 - t1) * 1_000_000)
	_, err = db.Exec(`
		UPDATE eternal_command
		SET exit=?, duration=?
		WHERE exit IS NULL AND id=(SELECT MAX(id) FROM eternal_command WHERE session_id=(SELECT id FROM eternal_session WHERE uuid=?))
	`, exit, duration, sess)
	if err != nil {
		return fmt.Errorf("UPDATING command: %w", err)
	}
	return nil
}

// CREATE TABLE eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), cwd text not null, start timestamp not null default (datetime()), exit int, duration int, command text not null);

func sqliteHistory(db *sql.DB, sess string) ([]Entry, error) {
	var e Entry
	rows, err := db.Query(`
		SELECT
			s.hostname, s.username, s.tty, s.pid,
			c.cwd, datetime(c.start), COALESCE(c.exit,0), COALESCE(c.duration,0), c.command
		FROM eternal_command c
		LEFT JOIN eternal_session s ON c.session_id=s.id
		ORDER BY c.id
	`, sess)
	if err != nil {
		return nil, fmt.Errorf("SELECT command: %w", err)
	}
	var h []Entry
	for rows.Next() {
		if err = rows.Scan(&e.Hostname, &e.Username, &e.TTY, &e.PID,
			&e.WorkingDir, &e.Timestamp, &e.ExitStatus, &e.Elapsed, &e.Cmd); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		h = append(h, e)
	}
	return h, nil
}

// select * FROM eternal_command where session_id=4 and exit is null and id=(select max(id) FROM eternal_command where session_id=4);
