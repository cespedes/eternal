#
# This file must be sourced at the beginning of each session.
# It depends on the ZSH-like 'preexec' and 'precmd' functions,
# which can be enabled for bash using https://github.com/rcaloras/bash-preexec
#

ETERNAL_SESSION=$( eternal init )
export ETERNAL_SESSION

__eternal_preexec() {
	__eternal_history_id=$( eternal start "$ETERNAL_SESSION" "$1" )
	__eternal_start_time=${EPOCHREALTIME-}
}

__eternal_precmd() {
	local EXIT=$? __eternal_end_time=${EPOCHREALTIME-}

	[[ ! $__eternal_history_id ]] && return

	eternal end "$ETERNAL_SESSION" "$__eternal_history_id" "${EXIT}" "${__eternal_start_time}" "${__eternal_end_time}"
	unset __eternal_history_id __eternal_start_time
}

precmd_functions+=(__eternal_precmd)
preexec_functions+=(__eternal_preexec)

