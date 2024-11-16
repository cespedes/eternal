#
# This file must be sourced at the beginning of each session.
# It should work in either "bash" or "zsh".
#

if [ -z "$BASH" -a -z "$ZSH_NAME" ]
then
	echo This shell does not appear to be BASH or ZSH: eternal is not supported here.
	return
fi

__eternal_start_command() {
	eternal start "$1"
	__eternal_start_time=${EPOCHREALTIME-}
}

__eternal_end_command() {
	local EXIT=$? __eternal_end_time=${EPOCHREALTIME-}
	eternal end "${EXIT}" "${__eternal_start_time}" "${__eternal_end_time}"
	unset __eternal_start_time
}

if [ "$BASH" ]
then
	if [ -f "`dirname ${BASH_ARGV[0]}`/bash-preexec.sh" ]
	then
		source "`dirname ${BASH_ARGV[0]}`/bash-preexec.sh"
	fi
fi

export ETERNAL_SESSION=$( eternal init )

precmd_functions+=(__eternal_end_command)
preexec_functions+=(__eternal_start_command)
