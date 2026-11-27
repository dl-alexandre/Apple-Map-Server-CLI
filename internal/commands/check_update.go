package commands

import (
	"errors"
	"fmt"
	"io"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/cli"
)

const checkUpdateUsage = `Usage:
  ams check-update

Check for available updates.

Flags:
  --force   Force check, bypassing the cache
  --clear   Clear the update cache
`

func NewCheckUpdateCommand() Command {
	return Command{
		Name:      "check-update",
		UsageLine: "check-update",
		Summary:   "Check for available updates",
		Usage:     checkUpdateUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			var options struct {
				Force bool `help:"Force check, bypassing the cache"`
				Clear bool `help:"Clear the update cache"`
			}
			if err := parseOptions(args, &options); err != nil {
				if errors.Is(err, errHelpRequested) {
					fmt.Fprint(stdout, checkUpdateUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, checkUpdateUsage)
				return ExitUsageError
			}

			// Handle clear cache flag
			if options.Clear {
				if err := cli.ClearUpdateCache(); err != nil {
					fmt.Fprintf(stderr, "Failed to clear cache: %v\n", err)
					return ExitRuntimeError
				}
				fmt.Fprintln(stdout, "✓ Update cache cleared")
				return ExitSuccess
			}

			// Force refresh by clearing cache if requested
			if options.Force {
				_ = cli.ClearUpdateCache() // #nosec G104 - best effort, failure is non-critical
			}

			// Check for updates
			_, err := cli.CheckForUpdates()
			if err != nil {
				fmt.Fprintf(stderr, "Error checking for updates: %v\n", err)
				return ExitRuntimeError
			}

			return ExitSuccess
		},
	}
}
