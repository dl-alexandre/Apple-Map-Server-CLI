package commands

import (
	"errors"
	"flag"
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
			fs := flag.NewFlagSet("check-update", flag.ContinueOnError)
			fs.SetOutput(io.Discard)

			var force bool
			var clear bool
			fs.BoolVar(&force, "force", false, "Force check, bypassing the cache")
			fs.BoolVar(&clear, "clear", false, "Clear the update cache")

			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, checkUpdateUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, checkUpdateUsage)
				return ExitUsageError
			}

			if fs.NArg() != 0 {
				fmt.Fprintln(stderr, "check-update accepts no arguments")
				fmt.Fprint(stderr, checkUpdateUsage)
				return ExitUsageError
			}

			// Handle clear cache flag
			if clear {
				if err := cli.ClearUpdateCache(); err != nil {
					fmt.Fprintf(stderr, "Failed to clear cache: %v\n", err)
					return ExitRuntimeError
				}
				fmt.Fprintln(stdout, "✓ Update cache cleared")
				return ExitSuccess
			}

			// Force refresh by clearing cache if requested
			if force {
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
