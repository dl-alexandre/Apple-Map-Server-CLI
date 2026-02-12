package commands

import (
	"fmt"
	"io"
	"strings"
)

const helpUsage = `Usage:
  ams help [command]

Show help for a command.
`

func NewHelpCommand(usage func(io.Writer), lookup func(string) (Command, bool)) Command {
	return Command{
		Name:      "help",
		UsageLine: "help [command]",
		Summary:   "Show help for a command",
		Usage:     helpUsage,
		Run: func(args []string, stdout, stderr io.Writer) int {
			if len(args) == 0 {
				usage(stdout)
				return ExitSuccess
			}

			name := strings.Join(args, " ")
			cmd, ok := lookup(name)
			if !ok {
				fmt.Fprintf(stderr, "unknown command: %s\n", name)
				fmt.Fprint(stderr, helpUsage)
				return ExitUsageError
			}

			fmt.Fprint(stdout, cmd.Usage)
			return ExitSuccess
		},
	}
}
