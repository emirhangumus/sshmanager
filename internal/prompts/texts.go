package prompts

type DefaultPromptTextError struct {
	NoSSHConnectionsFound            string
	SSHPassNotFound                  string
	ConnectionToXFailedX             string
	InvalidSelectionX                string
	FailedToAddConnectionX           string
	ErrorLoadingEncryptionKeyX       string
	FailedToStoreUpdatedConnectionsX string
	DecryptionDataFailedX            string
	InvalidDataFormatX               string
}

type DefaultPromptTextSuccess struct {
	SSHConnectionSaved   string
	SSHConnectionRemoved string
}

type DefaultPromptText struct {
	BackToMainMenu            string
	EnterHost                 string
	EnterUsername             string
	EnterPassword             string
	EnterDescription          string
	SelectAnSSHConnection     string
	Exit                      string
	ConnectToSSH              string
	AddSSHConnection          string
	RemoveSSHConnection       string
	SelectAConnectionToRemove string
	SSHConnectionRemoved      string
	ErrorMessages             DefaultPromptTextError
	SuccessMessages           DefaultPromptTextSuccess
}

var DefaultPromptTexts = DefaultPromptText{
	BackToMainMenu:            "Back to main menu",
	EnterHost:                 "Enter Host",
	EnterUsername:             "Enter Username",
	EnterPassword:             "Enter Password",
	EnterDescription:          "Enter Description",
	SelectAnSSHConnection:     "Select an SSH connection",
	Exit:                      "Exit",
	ConnectToSSH:              "Connect to SSH",
	AddSSHConnection:          "Add SSH Connection",
	RemoveSSHConnection:       "Remove SSH Connection",
	SelectAConnectionToRemove: "Select a connection to remove",
	SSHConnectionRemoved:      "SSH connection removed.",
	ErrorMessages: DefaultPromptTextError{
		NoSSHConnectionsFound:            "No SSH connections found.",
		SSHPassNotFound:                  "Error: sshpass not found. Please install it using your package manager.",
		ConnectionToXFailedX:             "Connection to %s failed: %v",
		InvalidSelectionX:                "Invalid selection: %s",
		FailedToAddConnectionX:           "Failed to add connection: %s",
		ErrorLoadingEncryptionKeyX:       "Error loading encryption key: %s",
		FailedToStoreUpdatedConnectionsX: "Failed to store updated connections: %s",
		DecryptionDataFailedX:            "Decryption data failed: %v",
		InvalidDataFormatX:               "Invalid data format: %s",
	},
	SuccessMessages: DefaultPromptTextSuccess{
		SSHConnectionSaved:   "SSH connection saved.",
		SSHConnectionRemoved: "SSH connection removed.",
	},
}
