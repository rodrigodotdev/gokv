// Package usecase implements the application's core business logic, including
// command dispatch, statistics tracking, and coordination between the storage
// and persistence layers.
package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
)

// KeyValueStore defines the data operations that CommandHandler requires from
// the underlying store. Defined at the consumer per Go interface conventions.
type KeyValueStore interface {
	Set(ctx context.Context, key, value string)
	Get(ctx context.Context, key string) (string, bool)
	Del(ctx context.Context, key string) int
	Expire(ctx context.Context, key string, seconds int) bool
	TTL(ctx context.Context, key string) int64
	Persist(ctx context.Context, key string) bool
	Keys(ctx context.Context, pattern string) []string
	Exists(ctx context.Context, key string) bool
	Size(ctx context.Context) int
}

// StatsCollector tracks command execution metrics.
// Defined at the consumer (CommandHandler) per Go interface conventions.
type StatsCollector interface {
	IncrementCommands()
	GetInfo(ctx context.Context) string
}

// CommandHandler is the central command dispatcher. It validates and executes
// incoming commands against the key-value store, persists write operations, and
// records execution metrics.
type CommandHandler struct {
	store   KeyValueStore
	persist repository.PersistenceRepository
	stats   StatsCollector
}

// NewCommandHandler returns a CommandHandler wired to the given store,
// persistence layer, and stats collector. If persist is nil, write commands
// will not be logged.
func NewCommandHandler(
	store KeyValueStore,
	persist repository.PersistenceRepository,
	stats StatsCollector,
) *CommandHandler {
	return &CommandHandler{
		store:   store,
		persist: persist,
		stats:   stats,
	}
}

// HandleCommand dispatches a parsed command to the appropriate handler method
// and returns a domain.Result.
func (h *CommandHandler) HandleCommand(ctx context.Context, cmd *command.Command) domain.Result {
	h.stats.IncrementCommands()

	switch cmd.Type {
	case command.SET:
		return h.handleSet(ctx, cmd.Args)
	case command.GET:
		return h.handleGet(ctx, cmd.Args)
	case command.DEL:
		return h.handleDel(ctx, cmd.Args)
	case command.EXPIRE:
		return h.handleExpire(ctx, cmd.Args)
	case command.TTL:
		return h.handleTTL(ctx, cmd.Args)
	case command.PERSIST:
		return h.handlePersist(ctx, cmd.Args)
	case command.KEYS:
		return h.handleKeys(ctx, cmd.Args)
	case command.EXISTS:
		return h.handleExists(ctx, cmd.Args)
	case command.PING:
		return h.handlePing(ctx, cmd.Args)
	case command.INFO:
		return h.handleInfo(ctx, cmd.Args)
	default:
		return domain.Error(fmt.Errorf("%w: %s", domain.ErrUnknownCommand, cmd.Type))
	}
}

func (h *CommandHandler) handleSet(ctx context.Context, args []string) domain.Result {
	if len(args) < 2 {
		return domain.Error(fmt.Errorf("%w: SET requires at least 2 arguments", domain.ErrWrongArgs))
	}

	if ctx.Err() != nil {
		return domain.Error(ctx.Err())
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	h.store.Set(ctx, key, value)
	h.persistCommand(ctx, command.SET, args)

	return domain.OK()
}

func (h *CommandHandler) handleGet(ctx context.Context, args []string) domain.Result {
	if len(args) != 1 {
		return domain.Error(fmt.Errorf("%w: GET requires exactly 1 argument", domain.ErrWrongArgs))
	}

	value, exists := h.store.Get(ctx, args[0])
	if !exists {
		return domain.Nil()
	}

	return domain.Value(value)
}

func (h *CommandHandler) handleDel(ctx context.Context, args []string) domain.Result {
	if len(args) < 1 {
		return domain.Error(fmt.Errorf("%w: DEL requires at least 1 argument", domain.ErrWrongArgs))
	}

	count := 0
	for _, key := range args {
		if h.store.Del(ctx, key) == 1 {
			count++
			h.persistCommand(ctx, command.DEL, []string{key})
		}
	}

	return domain.Value(count)
}

func (h *CommandHandler) handleExpire(ctx context.Context, args []string) domain.Result {
	if len(args) != 2 {
		return domain.Error(fmt.Errorf("%w: EXPIRE requires exactly 2 arguments", domain.ErrWrongArgs))
	}

	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return domain.Error(fmt.Errorf("invalid expiration time: %w", err))
	}

	success := h.store.Expire(ctx, args[0], seconds)
	if success {
		h.persistCommand(ctx, command.EXPIRE, args)
	}

	return domain.Value(success)
}

func (h *CommandHandler) handleTTL(ctx context.Context, args []string) domain.Result {
	if len(args) != 1 {
		return domain.Error(fmt.Errorf("%w: TTL requires exactly 1 argument", domain.ErrWrongArgs))
	}

	ttl := h.store.TTL(ctx, args[0])
	return domain.Value(ttl)
}

func (h *CommandHandler) handlePersist(ctx context.Context, args []string) domain.Result {
	if len(args) != 1 {
		return domain.Error(fmt.Errorf("%w: PERSIST requires exactly 1 argument", domain.ErrWrongArgs))
	}

	success := h.store.Persist(ctx, args[0])
	if success {
		h.persistCommand(ctx, command.PERSIST, args)
	}

	return domain.Value(success)
}

func (h *CommandHandler) handleKeys(ctx context.Context, args []string) domain.Result {
	pattern := "*"
	if len(args) > 0 {
		pattern = args[0]
	}

	keys := h.store.Keys(ctx, pattern)
	if len(keys) == 0 {
		return domain.Nil()
	}

	return domain.Value(keys)
}

func (h *CommandHandler) handleExists(ctx context.Context, args []string) domain.Result {
	if len(args) < 1 {
		return domain.Error(fmt.Errorf("%w: EXISTS requires at least 1 argument", domain.ErrWrongArgs))
	}

	count := 0
	for _, key := range args {
		if h.store.Exists(ctx, key) {
			count++
		}
	}

	return domain.Value(count)
}

func (h *CommandHandler) handlePing(ctx context.Context, args []string) domain.Result {
	message := "PONG"
	if len(args) > 0 {
		message = strings.Join(args, " ")
	}

	return domain.Value(message)
}

func (h *CommandHandler) handleInfo(ctx context.Context, _ []string) domain.Result {
	// NOTE: Redis INFO accepts an optional section argument (e.g. "INFO server").
	// Section filtering is not currently supported; all sections are always returned.
	return domain.Value(h.stats.GetInfo(ctx))
}

// persistCommand persists a write command to the AOF log.
func (h *CommandHandler) persistCommand(ctx context.Context, cmdType command.Type, args []string) {
	if h.persist == nil {
		return
	}

	cmd := &command.Command{Type: cmdType, Args: args}
	if err := h.persist.Append(ctx, cmd); err != nil {
		slog.Error("failed to persist command", "command", cmdType.String(), "error", err)
	}
}
