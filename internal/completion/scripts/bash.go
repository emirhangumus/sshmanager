package scripts

const BashScript = `_sshmanager() {
  local cur candidates
  cur="${COMP_WORDS[COMP_CWORD]}"
  candidates="$(sshmanager -complete "$cur" 2>/dev/null)" || return 0
  COMPREPLY=( $(compgen -W "$candidates" -- "$cur") )
}
complete -o default -F _sshmanager sshmanager
`
