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
	CompletionInstalledX string
	CompletionGeneratedX string
	AllFilesRemoved      string
	OperationCancelled   string
}

type DefaultPromptText struct {
	EnterHost                 string
	EnterUsername             string
	EnterPassword             string
	EnterDescription          string
	EnterAlias                string
	EditHost                  string
	EditUsername              string
	EditPassword              string
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
	ErrorMessages             DefaultPromptTextError
	SuccessMessages           DefaultPromptTextSuccess
}

var DefaultPromptTexts = DefaultPromptText{
	EnterHost:                 "Enter Host",
	EnterUsername:             "Enter Username",
	EnterPassword:             "Enter Password",
	EnterDescription:          "Enter Description",
	EnterAlias:                "Enter Alias",
	EditHost:                  "Edit Host",
	EditUsername:              "Edit Username",
	EditPassword:              "Edit Password",
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
		CompletionInstalledX: "Completion installed for shell: %s",
		CompletionGeneratedX: "Completion script generated for shell: %s",
		AllFilesRemoved:      "All SSH connections and key files removed.",
		OperationCancelled:   "Operation cancelled.",
	},
}
