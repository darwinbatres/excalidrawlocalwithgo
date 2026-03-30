package testutil

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

// TestLogger returns a zerolog.Logger that writes to test output.
func TestLogger(t *testing.T) zerolog.Logger {
	t.Helper()
	return zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
}

// NopLogger returns a no-op logger for tests that don't need log output.
func NopLogger() zerolog.Logger {
	return zerolog.Nop()
}

// SkipIfNoDocker skips the test if Docker is not available.
func SkipIfNoDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") == "" {
		// Check docker availability via DOCKER_HOST or default socket
		if _, err := os.Stat("/var/run/docker.sock"); os.IsNotExist(err) {
			if os.Getenv("DOCKER_HOST") == "" {
				t.Skip("Docker not available, skipping integration test")
			}
		}
	}
}
