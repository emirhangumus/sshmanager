#compdef sshmanager
_sshmanager() {
  local prefix=$words[2]
  local -a hosts
  hosts=(${(f)"$(sshmanager --complete "$prefix")"})
  compadd -S '' -- $hosts
}
