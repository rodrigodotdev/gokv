package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rodrigodotdev/gokv/internal/container"
	"github.com/rodrigodotdev/gokv/internal/infrastructure/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	cfg.RegisterFlags(flag.CommandLine)
	flag.Parse()

	ctn, cleanup, err := container.InitializeContainer(cfg)
	if err != nil {
		return err
	}

	defer func() {
		if cleanup != nil {
			cleanup()
		}
		if err := ctn.Close(); err != nil {
			slog.Error("failed to close container", "error", err)
		}
	}()

	if cfg.AOFEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := ctn.ReplayAOF(ctx); err != nil {
			slog.Warn("failed to replay AOF", "error", err)
		} else {
			slog.Info("AOF replay completed")
		}
	}

	ctn.Start(time.Duration(cfg.CleanupIntervalMs) * time.Millisecond)
	defer ctn.Stop()

	listener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return err
	}

	slog.Info("server listening", "port", cfg.Port, "aof", cfg.AOFEnabled)

	// Graceful shutdown: stop accepting new connections on signal,
	// then wait for active connections to finish.
	var wg sync.WaitGroup

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutting down: stopping new connections...")
		if err := listener.Close(); err != nil {
			slog.Error("failed to close listener", "error", err)
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			// net.ErrClosed is expected when the listener is closed during
			// graceful shutdown. Any other error (e.g. too many open files)
			// is transient — log it and keep accepting.
			if errors.Is(err, net.ErrClosed) {
				break
			}
			slog.Error("failed to accept connection", "error", err)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			ctn.HandleConnection(conn)
		}()
	}

	slog.Info("waiting for active connections to finish...")
	wg.Wait()
	slog.Info("all connections closed, goodbye")

	return nil
}
