package service

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetProcessStats_Basic(t *testing.T) {
	stats := getProcessStats()

	assert.Equal(t, os.Getpid(), stats.PID, "PID should match current process")
	assert.Greater(t, stats.UptimeSeconds, int64(-1), "uptime should be non-negative")
	assert.NotEmpty(t, stats.StartTime, "start time should be set")

	// On Linux /proc/self/statm should be readable
	if _, err := os.ReadFile("/proc/self/statm"); err == nil {
		assert.Greater(t, stats.RSS, uint64(0), "RSS should be > 0 on Linux")
	}

	// On Linux /proc/self/fd should be readable
	if _, err := os.ReadDir("/proc/self/fd"); err == nil {
		assert.Greater(t, stats.OpenFDs, 0, "open FDs should be > 0 on Linux")
	}
}
