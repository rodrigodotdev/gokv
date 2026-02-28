// Package config provides a unified configuration struct for the gokv server.
// Values are loaded from environment variables and can be overridden by
// command-line flags.
package config

import (
	"flag"
	"os"
	"strconv"
)

// Default values for server configuration.
const (
	DefaultPort            = "6379"
	DefaultAOFEnabled      = false
	DefaultAOFFilePath     = "data.aof"
	DefaultCleanupInterval = 1000 // milliseconds
	DefaultConnTimeoutSec  = 300  // seconds (5 minutes)
)

// Config holds all server configuration values.
type Config struct {
	// Port is the TCP port the server listens on.
	Port string

	// AOFEnabled controls whether append-only file persistence is active.
	AOFEnabled bool

	// AOFFilePath is the path to the AOF persistence file.
	AOFFilePath string

	// CleanupIntervalMs is the interval in milliseconds between expired-key
	// cleanup sweeps.
	CleanupIntervalMs int

	// ConnTimeoutSec is the idle connection timeout in seconds. Connections
	// that remain open longer than this duration are closed.
	ConnTimeoutSec int
}

// Load returns a Config populated from environment variables, with defaults
// applied for any unset values. Environment variables:
//
//   - GOKV_PORT           — TCP listen port (default "6379")
//   - GOKV_AOF_ENABLED    — enable AOF persistence: "true" or "1" (default false)
//   - GOKV_AOF_FILE_PATH  — AOF file path (default "data.aof")
//   - GOKV_CLEANUP_INTERVAL_MS — cleanup sweep interval in ms (default 1000)
func Load() Config {
	cfg := Config{
		Port:              DefaultPort,
		AOFEnabled:        DefaultAOFEnabled,
		AOFFilePath:       DefaultAOFFilePath,
		CleanupIntervalMs: DefaultCleanupInterval,
		ConnTimeoutSec:    DefaultConnTimeoutSec,
	}

	if v := os.Getenv("GOKV_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("GOKV_AOF_ENABLED"); v == "true" || v == "1" {
		cfg.AOFEnabled = true
	}
	if v := os.Getenv("GOKV_AOF_FILE_PATH"); v != "" {
		cfg.AOFFilePath = v
	}
	if v := os.Getenv("GOKV_CLEANUP_INTERVAL_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			cfg.CleanupIntervalMs = ms
		}
	}
	if v := os.Getenv("GOKV_CONN_TIMEOUT_SEC"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			cfg.ConnTimeoutSec = sec
		}
	}

	return cfg
}

// RegisterFlags registers command-line flags that override Config values.
// Call flag.Parse() after RegisterFlags to apply overrides.
func (c *Config) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Port, "port", c.Port, "TCP port to listen on")
	fs.BoolVar(&c.AOFEnabled, "aof", c.AOFEnabled, "Enable append-only file persistence")
	fs.StringVar(&c.AOFFilePath, "aof-file", c.AOFFilePath, "Path to the AOF file")
	fs.IntVar(&c.CleanupIntervalMs, "cleanup-interval", c.CleanupIntervalMs, "Expired-key cleanup interval in milliseconds")
	fs.IntVar(&c.ConnTimeoutSec, "conn-timeout", c.ConnTimeoutSec, "Idle connection timeout in seconds")
}
