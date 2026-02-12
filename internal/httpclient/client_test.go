package httpclient

import "testing"

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	t.Setenv("AMS_BASE_URL", "notaurl")
	_, err := New()
	if err == nil {
		t.Fatalf("expected error for invalid base url")
	}
}

func TestNewNormalizesBaseURL(t *testing.T) {
	t.Setenv("AMS_BASE_URL", "https://example.com/")
	client, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.BaseURL != "https://example.com" {
		t.Fatalf("expected normalized base url, got %q", client.BaseURL)
	}
}
