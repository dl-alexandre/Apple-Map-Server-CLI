package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/commands"
)

func Run(args []string, stdout, stderr io.Writer) int {
	authTokenCmd := commands.NewAuthTokenCommand()
	geocodeCmd := commands.NewGeocodeCommand()
	versionCmd := commands.NewVersionCommand()
	pingCmd := commands.NewPingCommand()
	reverseCmd := commands.NewReverseCommand()

	var ordered []commands.Command
	lookup := map[string]commands.Command{}

	usage := func(w io.Writer) {
		writeUsage(w, ordered)
	}

	lookupFn := func(name string) (commands.Command, bool) {
		cmd, ok := lookup[name]
		return cmd, ok
	}

	helpCmd := commands.NewHelpCommand(usage, lookupFn)

	ordered = []commands.Command{helpCmd, authTokenCmd, geocodeCmd, reverseCmd, pingCmd, versionCmd}
	for _, cmd := range ordered {
		lookup[cmd.Name] = cmd
	}

	if len(args) == 0 {
		return helpCmd.Run(nil, stdout, stderr)
	}

	cmd, consumed := matchCommand(args, ordered)
	if cmd.Name == "" {
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		usage(stderr)
		return commands.ExitUsageError
	}

	return cmd.Run(args[consumed:], stdout, stderr)
}

func writeUsage(w io.Writer, cmds []commands.Command) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  ams <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")

	width := 0
	for _, cmd := range cmds {
		name := displayName(cmd)
		if len(name) > width {
			width = len(name)
		}
	}

	for _, cmd := range cmds {
		name := displayName(cmd)
		padding := strings.Repeat(" ", width-len(name))
		fmt.Fprintf(w, "  %s%s  %s\n", name, padding, cmd.Summary)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit codes:")
	fmt.Fprintln(w, "  0 success")
	fmt.Fprintln(w, "  1 runtime/API error")
	fmt.Fprintln(w, "  2 usage error")
}

func displayName(cmd commands.Command) string {
	if cmd.UsageLine != "" {
		return cmd.UsageLine
	}
	return cmd.Name
}

func matchCommand(args []string, cmds []commands.Command) (commands.Command, int) {
	var matched commands.Command
	consumed := 0

	for _, cmd := range cmds {
		parts := strings.Fields(cmd.Name)
		if len(parts) == 0 || len(args) < len(parts) {
			continue
		}

		if matchesParts(args, parts) && len(parts) > consumed {
			matched = cmd
			consumed = len(parts)
		}
	}

	return matched, consumed
}

func matchesParts(args, parts []string) bool {
	for i, part := range parts {
		if args[i] != part {
			return false
		}
	}
	return true
}
