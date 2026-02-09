package prompt

type DefaultPromptTextError struct {
	NoSSHConnectionsFound            string
	AliasNotFoundX                   string
	SSHPassNotFound                  string
	ConnectionToXFailedX             string
	InvalidSelectionX                string
	FailedToAddConnectionX           string
	FailedToStoreUpdatedConnectionsX string
	FailedToLoadConnectionsX         string
	FailedToLoadConfigX              string
	FailedToInstallCompletionX       string
}

type DefaultPromptTextSuccess struct {
	SSHConnectionSaved   string
	SSHConnectionRemoved string
	SSHConnectionUpdated string
	SSHConnectionRenamed string
	CompletionInstalledX string
	CompletionGeneratedX string
	AllFilesRemoved      string
	OperationCancelled   string
}

type DefaultPromptText struct {
	EnterHost                 string
	EnterUsername             string
	EnterPort                 string
	EnterAuthMode             string
	EnterPassword             string
	EnterIdentityFile         string
	EnterProxyJump            string
	EnterLocalForwards        string
	EnterRemoteForwards       string
	EnterExtraSSHArgs         string
	EnterGroup                string
	EnterTags                 string
	EnterDescription          string
	EnterAlias                string
	EditHost                  string
	EditUsername              string
	EditPort                  string
	EditAuthMode              string
	EditPassword              string
	EditIdentityFile          string
	EditProxyJump             string
	EditLocalForwards         string
	EditRemoteForwards        string
	EditExtraSSHArgs          string
	EditGroup                 string
	EditTags                  string
	EditDescription           string
	EditAlias                 string
	SelectAnSSHConnection     string
	SelectAConnectionToEdit   string
	SelectAConnectionToRemove string
	Exit                      string
	ConnectToSSH              string
	AddSSHConnection          string
	EditSSHConnection         string
	RemoveSSHConnection       string
	RenameSSHConnection       string
	ErrorMessages             DefaultPromptTextError
	SuccessMessages           DefaultPromptTextSuccess
}

var DefaultPromptTexts = DefaultPromptText{
	EnterHost:                 "Enter Host",
	EnterUsername:             "Enter Username",
	EnterPort:                 "Enter Port (optional, default 22)",
	EnterAuthMode:             "Enter Auth Mode (password|key|agent)",
	EnterPassword:             "Enter Password",
	EnterIdentityFile:         "Enter Identity File (required for key mode)",
	EnterProxyJump:            "Enter ProxyJump (optional)",
	EnterLocalForwards:        "Enter Local Forwards (optional, comma-separated [bind_address:]port:host:hostport)",
	EnterRemoteForwards:       "Enter Remote Forwards (optional, comma-separated [bind_address:]port:host:hostport)",
	EnterExtraSSHArgs:         "Enter Extra SSH Args (optional, comma-separated tokens)",
	EnterGroup:                "Enter Group (optional)",
	EnterTags:                 "Enter Tags (optional, comma-separated)",
	EnterDescription:          "Enter Description",
	EnterAlias:                "Enter Alias",
	EditHost:                  "Edit Host",
	EditUsername:              "Edit Username",
	EditPort:                  "Edit Port (optional, default 22)",
	EditAuthMode:              "Edit Auth Mode (password|key|agent)",
	EditPassword:              "Edit Password",
	EditIdentityFile:          "Edit Identity File (required for key mode)",
	EditProxyJump:             "Edit ProxyJump (optional)",
	EditLocalForwards:         "Edit Local Forwards (optional, comma-separated [bind_address:]port:host:hostport)",
	EditRemoteForwards:        "Edit Remote Forwards (optional, comma-separated [bind_address:]port:host:hostport)",
	EditExtraSSHArgs:          "Edit Extra SSH Args (optional, comma-separated tokens)",
	EditGroup:                 "Edit Group (optional)",
	EditTags:                  "Edit Tags (optional, comma-separated)",
	EditDescription:           "Edit Description",
	EditAlias:                 "Edit Alias",
	SelectAnSSHConnection:     "Select an SSH connection",
	SelectAConnectionToEdit:   "Select a connection to edit",
	SelectAConnectionToRemove: "Select a connection to remove",
	Exit:                      "Exit",
	ConnectToSSH:              "Connect to SSH",
	AddSSHConnection:          "Add SSH Connection",
	EditSSHConnection:         "Edit SSH Connection",
	RemoveSSHConnection:       "Remove SSH Connection",
	RenameSSHConnection:       "Rename SSH Alias",
	ErrorMessages: DefaultPromptTextError{
		NoSSHConnectionsFound:            "No SSH connections found.",
		AliasNotFoundX:                   "No SSH connection found for alias: %s",
		SSHPassNotFound:                  "Error: sshpass not found. Please install it using your package manager.",
		ConnectionToXFailedX:             "Connection to %s failed: %v",
		InvalidSelectionX:                "Invalid selection: %v",
		FailedToAddConnectionX:           "Failed to add connection: %v",
		FailedToStoreUpdatedConnectionsX: "Failed to store updated connections: %v",
		FailedToLoadConnectionsX:         "Failed to load connections: %v",
		FailedToLoadConfigX:              "Failed to load config: %v",
		FailedToInstallCompletionX:       "Failed to install completion: %v",
	},
	SuccessMessages: DefaultPromptTextSuccess{
		SSHConnectionSaved:   "SSH connection saved.",
		SSHConnectionRemoved: "SSH connection removed.",
		SSHConnectionUpdated: "SSH connection updated.",
		SSHConnectionRenamed: "SSH alias renamed.",
		CompletionInstalledX: "Completion installed for shell: %s",
		CompletionGeneratedX: "Completion script generated for shell: %s",
		AllFilesRemoved:      "All SSH connections and key files removed.",
		OperationCancelled:   "Operation cancelled.",
	},
}
