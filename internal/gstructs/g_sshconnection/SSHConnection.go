package g_sshconnection

type SSHConnection struct {
	Index       string
	Username    string
	Host        string
	Password    string
	Description string
	Alias       string
}

func (c SSHConnection) String() string {
	s := c.Index + ". " + c.Username + "@" + c.Host + " - " + c.Description
	if c.Alias != "" {
		s += " (" + c.Alias + ")"
	}
	return s
}
