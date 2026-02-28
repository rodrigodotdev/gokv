// container.go wires together all application dependencies and provides a
// single entry point for initialising the gokv server.
package container

import (
	"context"
	"net"
	"time"

	"github.com/rodrigodotdev/gokv/internal/adapter/handler"
	"github.com/rodrigodotdev/gokv/internal/adapter/protocol"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
	"github.com/rodrigodotdev/gokv/internal/infrastructure/config"
	"github.com/rodrigodotdev/gokv/internal/infrastructure/persistence"
	"github.com/rodrigodotdev/gokv/internal/infrastructure/storage"
	"github.com/rodrigodotdev/gokv/internal/usecase"
)

// Container holds all initialised application components and exposes them for
// use by the server's main loop.
type Container struct {
	store          *storage.Store
	persistence    repository.PersistenceRepository
	commandHandler *usecase.CommandHandler
	tcpHandler     *handler.TCPHandler
	parser         *protocol.Parser
	formatter      *protocol.Formatter
	replayHandler  *usecase.CommandHandler
}

// InitializeContainer creates and wires all application dependencies based on
// the provided configuration. It returns the container, a cleanup function,
// and any initialisation error.
func InitializeContainer(cfg config.Config) (*Container, func(), error) {
	store := storage.NewStore()

	aofCfg := persistence.AOFProviderConfig{
		Enabled:  cfg.AOFEnabled,
		FilePath: cfg.AOFFilePath,
	}

	persist, err := persistence.NewAOFProvider(aofCfg)
	if err != nil {
		return nil, nil, err
	}

	parser := protocol.NewParser()
	formatter := protocol.NewFormatter()

	stats := usecase.NewStats(store)

	commandHandler := usecase.NewCommandHandler(store, persist, stats)

	// replayHandler processes commands without persistence (for AOF replay).
	replayHandler := usecase.NewCommandHandler(store, nil, stats)

	connTimeout := time.Duration(cfg.ConnTimeoutSec) * time.Second
	tcpHandler := handler.NewTCPHandler(commandHandler, parser, formatter, stats, connTimeout)

	ctn := &Container{
		store:          store,
		persistence:    persist,
		commandHandler: commandHandler,
		tcpHandler:     tcpHandler,
		parser:         parser,
		formatter:      formatter,
		replayHandler:  replayHandler,
	}

	return ctn, func() {}, nil
}

// Close releases resources held by the container, including closing the
// persistence layer if enabled.
func (c *Container) Close() error {
	if c.persistence != nil {
		return c.persistence.Close()
	}

	return nil
}

// Start begins background processes such as expired-key cleanup.
func (c *Container) Start(cleanupInterval time.Duration) {
	c.store.StartCleanup(cleanupInterval)
}

// Stop halts background processes started by Start.
func (c *Container) Stop() {
	c.store.StopCleanup()
}

// HandleConnection delegates a TCP connection to the handler.
func (c *Container) HandleConnection(conn net.Conn) {
	c.tcpHandler.HandleConnection(conn)
}

// ReplayAOF replays the AOF file if persistence is enabled.
// Returns nil if persistence is disabled.
func (c *Container) ReplayAOF(ctx context.Context) error {
	if c.persistence == nil {
		return nil
	}

	replayHandler := func(ctx context.Context, cmd *command.Command) {
		c.replayHandler.HandleCommand(ctx, cmd)
	}

	return c.persistence.Replay(ctx, replayHandler)
}

// replayCommand executes a command against the store without persisting it.
// Used during AOF replay to avoid re-writing commands back to the log.
func (c *Container) replayCommand(ctx context.Context, cmd *command.Command) {
	c.replayHandler.HandleCommand(ctx, cmd)
}
