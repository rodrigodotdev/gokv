// Package persistence provides append-only file (AOF) based durability for
// gokv, allowing write commands to be logged to disk and replayed on startup.
package persistence

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
)

// AOF implements an append-only file persistence strategy. Write commands are
// serialised in a RESP-like format that safely handles values containing
// spaces, and fsynced to disk after every append.
//
// Format (per command):
//
//	*<argc>\r\n
//	$<len>\r\n<arg>\r\n   (repeated argc times)
//
// The first argument is always the command name (e.g. "SET"), followed by
// the command's arguments.
type AOF struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

// NewAOF opens or creates the AOF file at filePath and returns a
// PersistenceRepository backed by it.
func NewAOF(filePath string) (repository.PersistenceRepository, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("opening AOF file %s: %w", filePath, err)
	}

	return &AOF{
		filePath: filePath,
		file:     file,
	}, nil
}

// Append writes a command in RESP-like format to the AOF
// file and fsyncs to ensure durability.
func (a *AOF) Append(ctx context.Context, cmd *command.Command) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Build RESP-like entry: *<count>\r\n followed by $<len>\r\n<data>\r\n for each element.
	// Elements = [cmd.Type] + cmd.Args.
	elements := append([]string{cmd.Type.String()}, cmd.Args...)

	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(elements))
	for _, el := range elements {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(el), el)
	}
	entry := b.String()

	_, err := a.file.WriteString(entry)
	if err != nil {
		return fmt.Errorf("writing to AOF file: %w", err)
	}

	if err := a.file.Sync(); err != nil {
		return fmt.Errorf("syncing AOF file: %w", err)
	}

	return nil
}

// Replay reads the AOF file and calls the handler for each valid write command.
// The handler is responsible for executing the command against the store.
// This keeps AOF focused on I/O and parsing, avoiding duplicated dispatch logic.
func (a *AOF) Replay(ctx context.Context, handler repository.CommandHandler) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	file, err := os.Open(a.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("opening AOF file for replay %s: %w", a.filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			slog.Error("failed to close AOF file during replay", "error", err)
		}
	}()

	reader := bufio.NewReader(file)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		elements, err := readRESPArray(reader)
		if err != nil {
			return err
		}
		if elements == nil {
			break // EOF
		}
		if len(elements) == 0 {
			continue
		}

		cmdType := command.Type(elements[0])
		if !cmdType.IsWriteCommand() {
			continue
		}

		cmd := &command.Command{
			Type: cmdType,
			Args: []string{},
		}
		if len(elements) > 1 {
			cmd.Args = elements[1:]
		}

		handler(ctx, cmd)
	}

	return nil
}

// readRESPArray reads one RESP array from the reader.
// Returns nil, nil on clean EOF (no more data).
func readRESPArray(r *bufio.Reader) ([]string, error) {
	// Read the *<count>\r\n header
	header, err := r.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading RESP header: %w", err)
	}

	header = trimCRLF(header)
	if header == "" {
		return nil, nil
	}

	if header[0] != '*' {
		return nil, fmt.Errorf("expected RESP array header '*', got %q", header)
	}

	count, err := strconv.Atoi(header[1:])
	if err != nil {
		return nil, fmt.Errorf("parsing RESP array count: %w", err)
	}

	elements := make([]string, 0, count)
	for range count {
		// Read $<len>\r\n
		lenLine, err := r.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading RESP bulk length: %w", err)
		}

		lenLine = trimCRLF(lenLine)
		if lenLine == "" || lenLine[0] != '$' {
			return nil, fmt.Errorf("expected RESP bulk string '$', got %q", lenLine)
		}

		strLen, err := strconv.Atoi(lenLine[1:])
		if err != nil {
			return nil, fmt.Errorf("parsing RESP bulk length: %w", err)
		}

		// Read exactly strLen bytes + \r\n
		buf := make([]byte, strLen+2) // +2 for \r\n
		_, err = io.ReadFull(r, buf)
		if err != nil {
			return nil, fmt.Errorf("reading RESP bulk data: %w", err)
		}

		elements = append(elements, string(buf[:strLen]))
	}

	return elements, nil
}

// trimCRLF removes trailing \r\n or \n from a string.
func trimCRLF(s string) string {
	s = s[:len(s)-1] // remove \n (always present from ReadString('\n'))
	if s != "" && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}

// Close closes the underlying AOF file. It is safe to call Close on a nil file.
func (a *AOF) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		if err := a.file.Close(); err != nil {
			return fmt.Errorf("closing AOF file: %w", err)
		}
	}

	return nil
}

// AOFProviderConfig holds the configuration for the AOF persistence provider.
type AOFProviderConfig struct {
	Enabled  bool
	FilePath string
}

// NewAOFProvider returns a PersistenceRepository based on the given config.
// If persistence is disabled (cfg.Enabled == false) it returns nil, nil.
func NewAOFProvider(cfg AOFProviderConfig) (repository.PersistenceRepository, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	return NewAOF(cfg.FilePath)
}
