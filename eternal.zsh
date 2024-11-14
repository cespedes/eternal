#
# This file must be sourced at the beginning of each session.
# It depends on the ZSH-like 'preexec' and 'precmd' functions,
# which can be enabled for bash using https://github.com/rcaloras/bash-preexec
#

autoload -U add-zsh-hook

zmodload zsh/datetime 2>/dev/null

export ETERNAL_SESSION=$( eternal init )

__eternal_preexec() {
	eternal start "$1"
	__eternal_start_time=${EPOCHREALTIME-}
}

__eternal_precmd() {
	local EXIT=$? __eternal_end_time=${EPOCHREALTIME-}

	eternal end "${EXIT}" "${__eternal_start_time}" "${__eternal_end_time}"
	unset __eternal_start_time
}

add-zsh-hook preexec __eternal_preexec
add-zsh-hook precmd __eternal_precmd
