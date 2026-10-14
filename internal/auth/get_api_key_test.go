package auth

import (
	"errors"
	"net/http"
	"testing"
)

func TestGetAPIKey(t *testing.T) {
	headers := make(http.Header)
	headers.Set("Authorization", "ApiKey test-api-key")

	key, err := GetAPIKey(headers)
	if err != nil {
		t.Fatalf("GetAPIKey() returned an unexpected error: %v", err)
	}
	if key != "test-api-key" {
		t.Errorf("GetAPIKey() = %q, want %q", key, "test-api-key")
	}
}

func TestGetAPIKeyMissingHeader(t *testing.T) {
	key, err := GetAPIKey(make(http.Header))
	if !errors.Is(err, ErrNoAuthHeaderIncluded) {
		t.Errorf("GetAPIKey() error = %v, want %v", err, ErrNoAuthHeaderIncluded)
	}
	if key != "" {
		t.Errorf("GetAPIKey() = %q, want an empty key", key)
	}
}

func TestGetAPIKeyMalformedHeader(t *testing.T) {
	for _, header := range []string{"ApiKey", "Bearer test-api-key"} {
		t.Run(header, func(t *testing.T) {
			headers := make(http.Header)
			headers.Set("Authorization", header)

			key, err := GetAPIKey(headers)
			if err == nil {
				t.Fatal("GetAPIKey() accepted a malformed authorization header")
			}
			if key != "" {
				t.Errorf("GetAPIKey() = %q, want an empty key", key)
			}
		})
	}
}
