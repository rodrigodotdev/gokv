package usecase

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

// mockKeySizer implements KeySizer for testing.
type mockKeySizer struct {
	size int
}

func (m *mockKeySizer) Size(_ context.Context) int {
	return m.size
}

func TestStatsIncrementCommands(t *testing.T) {
	t.Parallel()

	stats := NewStats(&mockKeySizer{size: 0})
	stats.IncrementCommands()
	stats.IncrementCommands()
	stats.IncrementCommands()

	got := atomic.LoadInt64(&stats.totalCommands)
	if got != 3 {
		t.Errorf("totalCommands = %d, want 3", got)
	}
}

func TestStatsIncrementDecrementConnections(t *testing.T) {
	t.Parallel()

	stats := NewStats(&mockKeySizer{size: 0})

	stats.IncrementConnections()
	stats.IncrementConnections()

	totalConns := atomic.LoadInt64(&stats.totalConnections)
	activeConns := atomic.LoadInt64(&stats.activeConns)

	if totalConns != 2 {
		t.Errorf("totalConnections = %d, want 2", totalConns)
	}
	if activeConns != 2 {
		t.Errorf("activeConns = %d, want 2", activeConns)
	}

	stats.DecrementConnections()
	activeConns = atomic.LoadInt64(&stats.activeConns)

	if activeConns != 1 {
		t.Errorf("activeConns after decrement = %d, want 1", activeConns)
	}
}

func TestStatsGetInfoContainsSections(t *testing.T) {
	t.Parallel()

	stats := NewStats(&mockKeySizer{size: 5})
	stats.IncrementCommands()
	stats.IncrementConnections()

	info := stats.GetInfo(context.Background())

	requiredSections := []string{
		"# Server",
		"# Clients",
		"# Stats",
		"# Keyspace",
	}

	for _, section := range requiredSections {
		if !strings.Contains(info, section) {
			t.Errorf("GetInfo missing section %q", section)
		}
	}

	requiredFields := []string{
		"uptime_in_seconds:",
		"uptime_in_days:",
		"uptime_human:",
		"connected_clients:1",
		"total_connections_received:1",
		"total_commands_processed:1",
		"db0:keys=5",
	}

	for _, field := range requiredFields {
		if !strings.Contains(info, field) {
			t.Errorf("GetInfo missing field %q\ngot:\n%s", field, info)
		}
	}
}
