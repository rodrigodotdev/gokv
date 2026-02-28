// Package protocol implements the text-based command protocol used to
// communicate with gokv over a TCP connection.
package protocol

import (
	"fmt"
	"strings"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

// Parser handles parsing of raw text input into commands.
type Parser struct{}

// NewParser returns a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// ParseCommand parses a raw text line into a Command. It returns
// domain.ErrEmptyCommand for blank input and domain.ErrUnknownCommand for
// unrecognised command types.
//
// NOTE: The text protocol uses whitespace to delimit arguments, so consecutive
// spaces within a value are collapsed into a single space during parsing.
// For example, "SET key hello  world" is parsed identically to
// "SET key hello world". The command handler reassembles multi-word values
// with strings.Join(args, " ").
func (p *Parser) ParseCommand(input string) (*command.Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, domain.ErrEmptyCommand
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, domain.ErrEmptyCommand
	}

	cmdType := command.Type(strings.ToUpper(parts[0]))
	if !cmdType.IsValid() {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnknownCommand, parts[0])
	}

	cmd := &command.Command{
		Type: cmdType,
		Args: []string{},
	}

	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}

	return cmd, nil
}
