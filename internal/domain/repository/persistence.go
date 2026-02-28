package repository

import (
	"context"

	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

// CommandHandler is a function that processes a parsed command during replay.
type CommandHandler func(ctx context.Context, cmd *command.Command)

// PersistenceRepository defines the contract for append-only persistence.
type PersistenceRepository interface {
	Append(ctx context.Context, cmd *command.Command) error
	Replay(ctx context.Context, handler CommandHandler) error
	Close() error
}
