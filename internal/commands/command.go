package commands

import "io"

const (
	ExitSuccess      = 0
	ExitRuntimeError = 1
	ExitUsageError   = 2
)

type Command struct {
	Name      string
	UsageLine string
	Summary   string
	Usage     string
	Run       func(args []string, stdout, stderr io.Writer) int
}
