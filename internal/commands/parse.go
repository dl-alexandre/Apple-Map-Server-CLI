package commands

import (
	"errors"
	"io"

	"github.com/alecthomas/kong"
)

var errHelpRequested = errors.New("help requested")

// parseOptions uses Kong for the standalone command adapters as well as the
// top-level CLI parser. This keeps direct command tests and internal callers
// aligned with the public flag syntax.
func parseOptions(args []string, options any) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return errHelpRequested
		}
	}
	parser, err := kong.New(options, kong.Writers(io.Discard, io.Discard), kong.Exit(func(int) {}))
	if err != nil {
		return err
	}
	_, err = parser.Parse(args)
	return err
}
