package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

	for _, want := range []string{"Usage: ams <command>", "auth token", "search", "check-update", "version"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("help output missing %q:\n%s", want, stdout.String())
		}
	}
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

	for _, want := range []string{"Usage: ams geocode", "--json", "--limit", "--file", "--concurrency", "Address to geocode"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("geocode help missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestKongParsesCommandFlagsAndArguments(t *testing.T) {
	ctx, cli, err := ParseForTest([]string{"geocode", "-limit", "7", "--json", "San", "Francisco"})
	if err != nil {
		t.Fatalf("ParseForTest() error = %v", err)
	}
	if got := commandName(ctx.Command()); got != "geocode" {
		t.Fatalf("command = %q, want geocode", got)
	}
	if cli.Geocode.Limit != 7 || !cli.Geocode.JSON {
		t.Fatalf("parsed geocode flags = %#v", cli.Geocode)
	}
	if got := strings.Join(cli.Geocode.Query, " "); got != "San Francisco" {
		t.Fatalf("query = %q, want San Francisco", got)
	}
}

func TestKongParsesSearchAutocompleteRoute(t *testing.T) {
	ctx, cli, err := ParseForTest([]string{"search", "autocomplete", "coffee", "shops", "--near", "37.7,-122.4", "--limit", "3"})
	if err != nil {
		t.Fatalf("ParseForTest() error = %v", err)
	}
	if got := commandName(ctx.Command()); got != "search" {
		t.Fatalf("command = %q, want search", got)
	}
	if got := strings.Join(cli.Search.Query, " "); got != "autocomplete coffee shops" {
		t.Fatalf("search query = %q", got)
	}
	if cli.Search.Near != "37.7,-122.4" || cli.Search.Limit != 3 {
		t.Fatalf("parsed autocomplete flags = %#v", cli.Search)
	}
}

func TestKongRejectsUnknownFlag(t *testing.T) {
	if _, _, err := ParseForTest([]string{"ping", "--not-a-flag"}); err == nil {
		t.Fatal("ParseForTest() accepted an unknown flag")
	}
}

func TestKongParsesExistingCommandRoutes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "auth token", args: []string{"auth", "token", "--raw"}, want: "auth token"},
		{name: "auth check", args: []string{"auth", "check", "--token", "jwt"}, want: "auth check"},
		{name: "auth jwt", args: []string{"auth", "jwt", "--team-id", "TEAM123456", "--private-key", "key.p8"}, want: "auth jwt"},
		{name: "cache clear", args: []string{"cache", "clear"}, want: "cache clear"},
		{name: "config set", args: []string{"config", "set", "maps_token", "value"}, want: "config set"},
		{name: "snapshot", args: []string{"snapshot", "Cupertino", "--zoom", "14", "--format", "jpg"}, want: "snapshot"},
		{name: "unified", args: []string{"unified", "Golden", "Gate", "Bridge", "--near", "37.7,-122.4"}, want: "unified"},
		{name: "check-update", args: []string{"check-update", "--force"}, want: "check-update"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _, err := ParseForTest(tt.args)
			if err != nil {
				t.Fatalf("ParseForTest() error = %v", err)
			}
			if got := commandName(ctx.Command()); got != tt.want {
				t.Fatalf("command = %q, want %q", got, tt.want)
			}
		})
	}
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
