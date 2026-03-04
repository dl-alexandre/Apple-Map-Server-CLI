package commands

import (
	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/cli"
)

// AutoUpdateCheck performs an automatic background update check
// This is called once at startup and is non-blocking
func AutoUpdateCheck() {
	cli.AutoUpdateCheck()
}
