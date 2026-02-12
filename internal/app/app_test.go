package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/commands"
)

func TestRunNoArgsShowsHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(nil, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected usage in stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"nope"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected error in stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"version"}, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if !strings.Contains(stdout.String(), "ams version") {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunPingMissingEnv(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"ping"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Fatalf("expected env error, got %q", stderr.String())
	}
}

func TestRunHelpForCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"help", "version"}, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if !strings.Contains(stdout.String(), "ams version") {
		t.Fatalf("expected version usage in stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunAuthTokenMissingEnv(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"auth", "token"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Fatalf("expected env error, got %q", stderr.String())
	}
}

func TestRunGeocodeMissingEnv(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"geocode", "Cupertino"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Fatalf("expected env error, got %q", stderr.String())
	}
}

func TestRunReverseMissingEnv(t *testing.T) {
	t.Setenv("AMS_MAPS_TOKEN", "")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"reverse", "0,0"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing required env vars") {
		t.Fatalf("expected env error, got %q", stderr.String())
	}
}

func TestRunGeocodeInvalidLimit(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"geocode", "--limit", "0", "Cupertino"}, stdout, stderr)

	if code != commands.ExitUsageError {
		t.Fatalf("expected exit %d, got %d", commands.ExitUsageError, code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "limit must be at least 1") {
		t.Fatalf("expected limit error, got %q", stderr.String())
	}
}
