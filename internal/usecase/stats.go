package usecase

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// KeySizer provides the number of keys in the store.
// Stats only needs this subset of the store interface.
type KeySizer interface {
	Size(ctx context.Context) int
}

// Stats tracks server-level runtime statistics such as uptime, total commands
// processed, and connection metrics.
type Stats struct {
	startTime        time.Time
	totalCommands    int64
	totalConnections int64
	activeConns      int64
	keyspace         KeySizer
}

// NewStats returns a Stats instance that queries keyspace for key counts.
func NewStats(keyspace KeySizer) *Stats {
	return &Stats{
		startTime: time.Now(),
		keyspace:  keyspace,
	}
}

// IncrementCommands atomically increments the total commands processed counter.
func (s *Stats) IncrementCommands() {
	atomic.AddInt64(&s.totalCommands, 1)
}

// IncrementConnections atomically increments both the total and active
// connection counters. Call this when a new client connects.
func (s *Stats) IncrementConnections() {
	atomic.AddInt64(&s.totalConnections, 1)
	atomic.AddInt64(&s.activeConns, 1)
}

// DecrementConnections atomically decrements the active connection counter.
// Call this when a client disconnects.
func (s *Stats) DecrementConnections() {
	atomic.AddInt64(&s.activeConns, -1)
}

// GetInfo returns a multi-line string containing server statistics formatted
// in a Redis-compatible INFO layout.
func (s *Stats) GetInfo(ctx context.Context) string {
	uptimeDuration := time.Since(s.startTime)
	uptime := uptimeDuration.String()
	uptimeInSeconds := int64(uptimeDuration / time.Second)
	uptimeInDays := uptimeInSeconds / 86400
	activeClients := atomic.LoadInt64(&s.activeConns)
	totalConns := atomic.LoadInt64(&s.totalConnections)
	commands := atomic.LoadInt64(&s.totalCommands)
	databaseSize := s.keyspace.Size(ctx)

	info := fmt.Sprintf(`# Server
uptime_in_seconds:%d
uptime_in_days:%d
uptime_human:%s

# Clients
connected_clients:%d
total_connections_received:%d

# Stats
total_commands_processed:%d

# Keyspace
db0:keys=%d`,
		uptimeInSeconds,
		uptimeInDays,
		uptime,
		activeClients,
		totalConns,
		commands,
		databaseSize,
	)

	return info
}
