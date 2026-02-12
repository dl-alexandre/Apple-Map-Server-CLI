package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/version"
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
			fs := flag.NewFlagSet("version", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			if err := fs.Parse(args); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					fmt.Fprint(stdout, versionUsage)
					return ExitSuccess
				}
				fmt.Fprintln(stderr, err)
				fmt.Fprint(stderr, versionUsage)
				return ExitUsageError
			}

			if fs.NArg() != 0 {
				fmt.Fprintln(stderr, "version accepts no arguments")
				fmt.Fprint(stderr, versionUsage)
				return ExitUsageError
			}

			fmt.Fprintf(stdout, "ams version %s\n", version.Version)
			fmt.Fprintf(stdout, "commit %s\n", version.Commit)
			fmt.Fprintf(stdout, "date %s\n", version.Date)
			return ExitSuccess
		},
	}
}
