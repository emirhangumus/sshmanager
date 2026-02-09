package main

import (
	"log"
	"os"

	"github.com/emirhangumus/sshmanager/internal/app"
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
		log.Fatal(err)
	}
}
