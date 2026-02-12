package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteGeocodeTableLimit(t *testing.T) {
	payload := []byte(`{"results":[{"formattedAddress":"One","coordinate":{"latitude":1,"longitude":2}},{"formattedAddress":"Two","coordinate":{"latitude":3,"longitude":4}}]}`)

	stdout := &bytes.Buffer{}
	if !writeGeocodeTable(stdout, payload, 1) {
		t.Fatalf("expected table render to succeed")
	}

	output := stdout.String()
	if !strings.Contains(output, "One") {
		t.Fatalf("expected first result in output, got %q", output)
	}
	if strings.Contains(output, "Two") {
		t.Fatalf("expected output to be limited to 1 row, got %q", output)
	}
}
