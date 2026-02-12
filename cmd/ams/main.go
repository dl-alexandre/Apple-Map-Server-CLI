package main

import (
	"os"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:], os.Stdout, os.Stderr))
}
