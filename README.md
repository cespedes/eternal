# eternal

eternal is a tool to store your shell history for all your sessions,
all your accounts and all your machines.

It is inspired by [github.com/atuinsh/atuin](https://github.com/atuinsh/atuin).

In order to use it, you have to configure your shell to execute `eternal init`
on startup, and then `eternal start` and `eternal end` at the start and end
of each line in your shell.

You can use `eternal history` to retreive your shell history.

All of those invocations of `eternal` (`init`, `start`, `end` and `history`)
establish a connection to a local daemon (created by `eternal daemon`) who is
listening in a UNIX socket in your local machine.

That daemon can store your history in a local SQLite database, or forward it
to another daemon, in a remote machine.

# SQLite

When using SQLite, the database is created with:

    CREATE TABLE eternal_session(id INTEGER primary key, created timestamp not null default (datetime()), uuid text unique not null, hostname text not null, username text not null, tty text not null, pid int not null);

    CREATE TABLE eternal_command (id INTEGER primary key, session_id integer not null references eternal_session(id), cwd text not null, start timestamp not null default (datetime()), exit int, duration int, command text not null);

# PostgreSQL

    create table eternal_session(id serial primary key, created timestamp not null default now(), uuid text not null default gen_random_uuid(), hostname text not null, username text not null, tty text not null, pid int not null);
    create table eternal_command (id serial primary key, session_id integer not null references eternal_session(id), cwd text not null, start timestamp not null default date_trunc('second',now()), exit int, duration int, command text not null);

