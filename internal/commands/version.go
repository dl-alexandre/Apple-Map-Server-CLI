package commands

import (
	"errors"
	"fmt"
	"io"

	cliver "github.com/dl-alexandre/cli-tools/version"
)

const versionUsage = `Usage:
  ams version

Show version info.
`

func NewVersionCommand() Command {
	return Command{
		Name:      "version",
		UsageLine: "version",
		Summary:   "Show version info",
		Usage:     versionUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			if err := parseOptions(args, &struct{}{}); err != nil {
				if errors.Is(err, errHelpRequested) {
					fmt.Fprint(stdout, versionUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, versionUsage)
				return ExitUsageError
			}

			fmt.Fprintf(stdout, "ams version %s\n", cliver.Version)
			fmt.Fprintf(stdout, "commit %s\n", cliver.GitCommit)
			fmt.Fprintf(stdout, "date %s\n", cliver.BuildTime)
			return ExitSuccess
		},
	}
}
