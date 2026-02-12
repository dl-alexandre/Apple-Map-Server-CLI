package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dl-alexandre/Apple-Map-Server-CLI/internal/commands"
)

func TestHelpGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run(nil, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	assertGolden(t, "help.golden", stdout.String())
}

func TestVersionGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"version"}, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	assertGolden(t, "version.golden", stdout.String())
}

func TestGeocodeHelpGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := Run([]string{"geocode", "--help"}, stdout, stderr)

	if code != commands.ExitSuccess {
		t.Fatalf("expected exit %d, got %d", commands.ExitSuccess, code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	assertGolden(t, "geocode-help.golden", stdout.String())
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}

	expected := string(data)
	if got != expected {
		t.Fatalf("golden mismatch for %s\nexpected:\n%q\n\ngot:\n%q", name, expected, got)
	}
}
