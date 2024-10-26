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
