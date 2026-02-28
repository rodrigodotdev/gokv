package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/infrastructure/config"
)

func TestInitializeContainerWithoutAOF(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:              "0",
		AOFEnabled:        false,
		AOFFilePath:       "",
		CleanupIntervalMs: 1000,
		ConnTimeoutSec:    300,
	}

	ctn, cleanup, err := InitializeContainer(cfg)
	if err != nil {
		t.Fatalf("InitializeContainer() error = %v", err)
	}
	defer cleanup()

	if ctn.store == nil {
		t.Error("store is nil")
	}
	if ctn.persistence != nil {
		t.Error("persistence should be nil when AOF is disabled")
	}
	if ctn.commandHandler == nil {
		t.Error("commandHandler is nil")
	}
	if ctn.tcpHandler == nil {
		t.Error("tcpHandler is nil")
	}
	if ctn.parser == nil {
		t.Error("parser is nil")
	}
	if ctn.formatter == nil {
		t.Error("formatter is nil")
	}
}

func TestInitializeContainerWithAOF(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	aofPath := filepath.Join(tmpDir, "test.aof")

	cfg := config.Config{
		Port:              "0",
		AOFEnabled:        true,
		AOFFilePath:       aofPath,
		CleanupIntervalMs: 1000,
		ConnTimeoutSec:    300,
	}

	ctn, cleanup, err := InitializeContainer(cfg)
	if err != nil {
		t.Fatalf("InitializeContainer() error = %v", err)
	}
	defer cleanup()
	defer func() { _ = ctn.Close() }()

	if ctn.persistence == nil {
		t.Error("persistence should not be nil when AOF is enabled")
	}

	// Verify AOF file was created
	if _, err := os.Stat(aofPath); os.IsNotExist(err) {
		t.Error("AOF file was not created")
	}
}

func TestContainerClose(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:              "0",
		AOFEnabled:        false,
		CleanupIntervalMs: 1000,
		ConnTimeoutSec:    300,
	}

	ctn, cleanup, err := InitializeContainer(cfg)
	if err != nil {
		t.Fatalf("InitializeContainer() error = %v", err)
	}
	defer cleanup()

	// Close should not error when persistence is nil
	if err := ctn.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestContainerReplayCommand(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:              "0",
		AOFEnabled:        false,
		CleanupIntervalMs: 1000,
		ConnTimeoutSec:    300,
	}

	ctn, cleanup, err := InitializeContainer(cfg)
	if err != nil {
		t.Fatalf("InitializeContainer() error = %v", err)
	}
	defer cleanup()

	ctx := context.Background()

	// Replay a SET command
	ctn.replayCommand(ctx, &command.Command{
		Type: command.SET,
		Args: []string{"key1", "value1"},
	})

	// Verify the key was set in the store
	val, exists := ctn.store.Get(ctx, "key1")
	if !exists {
		t.Fatal("key1 should exist after replay")
	}
	if val != "value1" {
		t.Errorf("key1 = %q, want %q", val, "value1")
	}
}
