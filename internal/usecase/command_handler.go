package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rodrigodotdev/gokv/internal/adapter/protocol"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
)

type CommandHandler struct {
	store   repository.KeyValueRepository
	persist repository.PersistenceRepository
	parser  *protocol.Parser
	stats   *Stats
}

func NewCommandHandler(
	store repository.KeyValueRepository,
	persist repository.PersistenceRepository,
	parser *protocol.Parser,
	stats *Stats,
) *CommandHandler {
	return &CommandHandler{
		store:   store,
		persist: persist,
		parser:  parser,
		stats:   stats,
	}
}

func (h *CommandHandler) HandleCommand(ctx context.Context, cmd *protocol.Command) string {
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
		return h.parser.FormatError(fmt.Sprintf("unknown command type: %s", cmd.Type))
	}
}

func (h *CommandHandler) handleSet(ctx context.Context, args []string) string {
	if len(args) < 2 {
		return h.parser.FormatError("SET command requires at least 2 arguments")
	}

	key := args[0]
	value := strings.Join(args[1:], " ")

	h.store.Set(ctx, key, value)

	if h.persist != nil {
		h.persist.Append(ctx, command.SET.String(), args)
	}

	return h.parser.FormatOK()
}

func (h *CommandHandler) handleGet(ctx context.Context, args []string) string {
	if len(args) != 1 {
		return h.parser.FormatError("GET command requires exactly 1 argument")
	}

	key := args[0]
	value, exists := h.store.Get(ctx, key)
	if !exists {
		return h.parser.FormatNil()
	}

	return h.parser.FormatResponse(value)
}

func (h *CommandHandler) handleDel(ctx context.Context, args []string) string {
	if len(args) < 1 {
		return h.parser.FormatError("DEL command requires at least 1 argument")
	}

	count := 0
	for _, key := range args {
		count += h.store.Del(ctx, key)
	}

	if h.persist != nil && count > 0 {
		for _, key := range args {
			h.persist.Append(ctx, command.DEL.String(), []string{key})
		}
	}

	return h.parser.FormatResponse(count)
}

func (h *CommandHandler) handleExpire(ctx context.Context, args []string) string {
	if len(args) != 2 {
		return h.parser.FormatError("EXPIRE command requires exactly 2 arguments")
	}

	key := args[0]
	seconds, err := strconv.Atoi(args[1])
	if err != nil {
		return h.parser.FormatError("invalid expiration time")
	}

	success := h.store.Expire(ctx, key, seconds)
	if h.persist != nil && success {
		h.persist.Append(ctx, command.EXPIRE.String(), args)
	}

	return h.parser.FormatResponse(success)
}

func (h *CommandHandler) handleTTL(ctx context.Context, args []string) string {
	if len(args) != 1 {
		return h.parser.FormatError("TTL command requires exactly 1 argument")
	}

	key := args[0]
	ttl := h.store.TTL(ctx, key)

	return h.parser.FormatResponse(ttl)
}

func (h *CommandHandler) handlePersist(ctx context.Context, args []string) string {
	if len(args) != 1 {
		return h.parser.FormatError("PERSIST command requires exactly 1 argument")
	}

	key := args[0]

	success := h.store.Persist(ctx, key)
	if h.persist != nil && success {
		h.persist.Append(ctx, command.PERSIST.String(), args)
	}

	return h.parser.FormatResponse(success)
}

func (h *CommandHandler) handleKeys(ctx context.Context, args []string) string {
	pattern := "*"
	if len(args) > 0 {
		pattern = args[0]
	}

	keys := h.store.Keys(ctx, pattern)
	if len(keys) == 0 {
		return h.parser.FormatNil()
	}

	return strings.Join(keys, " ")
}

func (h *CommandHandler) handleExists(ctx context.Context, args []string) string {
	if len(args) != 1 {
		return h.parser.FormatError("EXISTS command requires exactly 1 argument")
	}

	key := args[0]
	exists := h.store.Exists(ctx, key)

	return h.parser.FormatResponse(exists)
}

func (h *CommandHandler) handlePing(ctx context.Context, args []string) string {
	message := "PONG"
	if len(args) > 0 {
		message = strings.Join(args, " ")
	}

	return h.parser.FormatResponse(message)
}

func (h *CommandHandler) handleInfo(ctx context.Context, args []string) string {
	return h.stats.GetInfo(ctx)
}
