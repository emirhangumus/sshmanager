_sshmanager() {
  local cur=${COMP_WORDS[COMP_CWORD]}
  COMPREPLY=( $(compgen -W "$(sshmanager --complete "$cur")" -- "$cur") )
}
complete -F _sshmanager -o nospace sshmanager
