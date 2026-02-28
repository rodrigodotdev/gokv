// Package handler provides connection-level handlers that bridge network I/O
// with the application's command processing pipeline.
package handler

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"time"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

// CommandProcessor executes parsed commands and returns domain results.
// Defined at the consumer (TCPHandler) per Go interface conventions.
type CommandProcessor interface {
	HandleCommand(ctx context.Context, cmd *command.Command) domain.Result
}

// CommandParser parses raw input into commands.
type CommandParser interface {
	ParseCommand(input string) (*command.Command, error)
}

// ResponseFormatter formats domain values into protocol strings.
type ResponseFormatter interface {
	FormatResponse(resp any) string
	FormatOK() string
	FormatNil() string
	FormatError(msg string) string
}

// ConnectionTracker tracks connection lifecycle events.
type ConnectionTracker interface {
	IncrementConnections()
	DecrementConnections()
}

// TCPHandler reads commands from a TCP connection, dispatches them through
// a CommandProcessor, and writes formatted responses back to the client.
type TCPHandler struct {
	commands    CommandProcessor
	parser      CommandParser
	formatter   ResponseFormatter
	connTracker ConnectionTracker
	connTimeout time.Duration
}

// NewTCPHandler returns a TCPHandler wired to the given command processor,
// parser, response formatter, connection tracker, and idle connection timeout.
func NewTCPHandler(
	commands CommandProcessor,
	parser CommandParser,
	formatter ResponseFormatter,
	connTracker ConnectionTracker,
	connTimeout time.Duration,
) *TCPHandler {
	return &TCPHandler{
		commands:    commands,
		parser:      parser,
		formatter:   formatter,
		connTracker: connTracker,
		connTimeout: connTimeout,
	}
}

// HandleConnection runs the read-eval-print loop for a single TCP connection.
// It blocks until the client disconnects, sends a QUIT command, or the
// connection's context times out.
func (h *TCPHandler) HandleConnection(conn net.Conn) {
	h.connTracker.IncrementConnections()
	defer h.connTracker.DecrementConnections()
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close connection", "error", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), h.connTimeout)
	defer cancel()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		cmd, err := h.parser.ParseCommand(line)
		if err != nil {
			response := h.formatter.FormatError(err.Error())
			if writeErr := h.writeResponse(conn, response); writeErr != nil {
				return
			}
			continue
		}

		if cmd.Type == command.QUIT {
			_ = h.writeResponse(conn, h.formatter.FormatOK())
			return
		}

		result := h.commands.HandleCommand(ctx, cmd)
		response := h.formatResult(result)
		if writeErr := h.writeResponse(conn, response); writeErr != nil {
			return
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		slog.Error("failed to read from connection", "error", err)
	}
}

// formatResult converts a domain.Result into a protocol-formatted string.
func (h *TCPHandler) formatResult(r domain.Result) string {
	if err := r.Err(); err != nil {
		return h.formatter.FormatError(err.Error())
	}

	if r.IsNil() {
		return h.formatter.FormatNil()
	}

	val, _ := r.Val()
	return h.formatter.FormatResponse(val)
}

// writeResponse sends a newline-terminated response string to the connection.
func (h *TCPHandler) writeResponse(conn net.Conn, response string) error {
	_, err := conn.Write([]byte(response + "\n"))
	if err != nil {
		slog.Error("failed to write to connection", "error", err)
	}
	return err
}
