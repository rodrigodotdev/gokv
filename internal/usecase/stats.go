package usecase

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rodrigodotdev/gokv/internal/domain/repository"
)

type Stats struct {
	startTime        time.Time
	totalCommands    int64
	totalConnections int64
	keyspace         repository.KeyValueRepository
}

func NewStats(keyspace repository.KeyValueRepository) *Stats {
	return &Stats{
		startTime: time.Now(),
		keyspace:  keyspace,
	}
}

func (s *Stats) IncrementCommands() {
	atomic.AddInt64(&s.totalCommands, 1)
}

func (s *Stats) IncrementConnections() {
	atomic.AddInt64(&s.totalConnections, 1)
}

func (s *Stats) GetInfo(ctx context.Context) string {
	uptimeDuration := time.Since(s.startTime)
	uptime := uptimeDuration.String()
	uptimeInSeconds := int64(uptimeDuration / time.Second)
	uptimeInDays := uptimeInSeconds / 86400
	connections := atomic.LoadInt64(&s.totalConnections)
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
keyspace_hits:0
keyspace_misses:0

# Keyspace
db0:keys=%d,expires=0,avg_ttl=0`,
		uptimeInSeconds,
		uptimeInDays,
		uptime,
		connections,
		connections,
		commands,
		databaseSize,
	)

	return info
}
