# eternal

eternal is a tool to store your shell history for all your sessions,
all your accounts and all your machines.

It is inspired by [github.com/atuinsh/atuin](https://github.com/atuinsh/atuin).

In order to use it, you have to configure your shell to execute `eternal init`
on startup, and then `eternal start` and `eternal end` at the start and end
of each line in your shell.  This is automatically handled by sourcing
the file `eternal.sh` from your Bash or ZSH.

You can use `eternal history` to retreive your shell history.

All of those invocations of `eternal` (`init`, `start`, `end` and `history`)
establish a connection to a local daemon (created by `eternal daemon`) who is
listening in a UNIX socket in your local machine.

The daemon can also listen in a TCP socket for remote connections using HTTP
and Websockets (TODO).

That daemon can store your history in a local SQLite database, or in a PostgreSQL
database (TODO), or forward it to another daemon in a remote machine (TODO).

# Invocation

The different ways to invocate `eternal` are:

## `eternal daemon [-config filename]`

**WARNING**: specifying configuration file is not yet implemented.

This reads the configuration file, if supplied.  If not, it reads the
configuration file in `$HOME/.config/eternal.json`.

Those files tell the daemon how to listen to incoming connections,
and where to redirect the outgoing ones.

If there is no configuration file, the default configuration, in Linux, is:

    listen unix:$XDG_RUNTIME_DIR/eternal
    output sqlite://$HOME/.local/share/eternal/history.db

In MacOSX the default configuration is:

    listen unix:$TMPDIR/eternal-$USER
    output sqlite://$HOME/Application%20Support/eternal/history.db

## `eternal init`

This connects to a local daemon using UNIX sockets.

If it cannot connect, it tries to launch a local daemon with `eternal daemon` and tries again.

It sends to the daemon the following JSON object:

    {
        "action": "init",
        "hostname": (host name),
        "username": (user name)
        "tty": (TTY of controlling terminal),
        "pid": (PID of shell),
        "shell": (name of the shell),
        "parent": (name of the shell's parent),
        "os": (OS + "/" + ARCH),
        "origin": (IP address of client, if connected via SSH)
    }

The daemon sends the following response:

    {
        "session": (UUID of new session)
    }

IF all goes well, `eternal init` writes the session using standard output.

## `eternal start <command-and-args>`

This connects to the local daemon and sends the JSON object:

    {
        "action": "start",
        "session": (content of environmen variable $ETERNAL_SESSION),
        "working_dir": (working directory),
        "command": (command),
        "start": (start of command, as an integer number of microseconds since epoch)
    }

## `eternal end <exit-status> <start-timestamp> <end-timestamp>`

`<start-timestamp>` and `<end-timestamp>` can be empty, or a float with the number of
seconds since the epoch.

This connects to the local daemon and sends the JSON object:

    {
        "action": "end",
        "session": (content of environmen variable $ETERNAL_SESSION),
        "start": (start of command, as an integer number of microseconds since epoch)
        "end": (start of command, as an integer number of microseconds since epoch)
    }

If the arguments `<start-timestamp>` and `<end-timestamp>` are empty, the `"start"` element
is not sent, and the `"end"` is set to the current time.

## `eternal history`

# SQLite

When using SQLite, the database is created with:

    CREATE TABLE eternal_session (
        id       INTEGER PRIMARY KEY,
        created  TIMESTAMP NOT NULL DEFAULT (datetime('now','localtime')),
        uuid     TEXT UNIQUE NOT NULL,
        os       TEXT NOT NULL DEFAULT '',
        shell    TEXT NOT NULL DEFAULT '',
        parent   TEXT NOT NULL DEFAULT '',
        origin   TEXT NOT NULL DEFAULT '',
        hostname TEXT NOT NULL,
        username TEXT NOT NULL,
        tty      TEXT NOT NULL,
        pid      INTEGER NOT NULL
    );

    CREATE TABLE eternal_command (
        id          INTEGER PRIMARY KEY,
        session_id  INTEGER NOT NULL REFERENCES eternal_session(id),
        working_dir TEXT NOT NULL,
        start       TIMESTAMP NOT NULL DEFAULT (datetime('now','localtime')),
        exit        INTEGER,
        duration    INTEGER,
        command     TEXT NOT NULL
    );

# PostgreSQL

In PostgreSQL the schema is almost the same:

    CREATE TABLE eternal_session (
        id       SERIAL PRIMARY KEY,
        created  TIMESTAMP NOT NULL DEFAULT now(),
        uuid     TEXT UNIQUE NOT NULL DEFAULT gen_random_uuid(),
        os       TEXT NOT NULL DEFAULT '',
        shell    TEXT NOT NULL DEFAULT '',
        parent   TEXT NOT NULL DEFAULT '',
        origin   TEXT NOT NULL DEFAULT '',
        hostname TEXT NOT NULL,
        username TEXT NOT NULL,
        tty      TEXT NOT NULL,
        pid      INTEGER NOT NULL
    );

    CREATE TABLE eternal_command (
        id          SERIAL PRIMARY KEY,
        session_id  INTEGER NOT NULL REFERENCES eternal_session(id),
        working_dir TEXT NOT NULL,
        start       TIMESTAMP NOT NULL DEFAULT date_trunc('second', now()),
        exit        INTEGER,
        duration    INTEGER,
        command     TEXT NOT NULL
    );
