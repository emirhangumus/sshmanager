package main

import (
	"errors"
	"log"
	"os"

	"github.com/emirhangumus/sshmanager/v2/internal/app"
	"github.com/emirhangumus/sshmanager/v2/internal/cli/commands"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := app.Run(os.Args, app.BuildInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
	}); err != nil {
		var exitErr *commands.ExecExitError
		if errors.As(err, &exitErr) {
			if exitErr.Detail != "" {
				log.Print(exitErr.Detail)
			}
			os.Exit(exitErr.Code)
		}
		log.Fatal(err)
	}
}
