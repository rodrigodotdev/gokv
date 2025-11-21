package protocol

import (
	"fmt"
	"strings"

	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

type Command struct {
	Type command.Type
	Args []string
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ParseCommand(input string) (*Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty command")
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmdType := command.Type(strings.ToUpper(parts[0]))
	if !cmdType.IsValid() {
		return nil, fmt.Errorf("unknown command: %s", parts[0])
	}

	cmd := &Command{
		Type: cmdType,
		Args: []string{},
	}

	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}

	return cmd, nil
}

func (p *Parser) FormatResponse(resp interface{}) string {
	switch v := resp.(type) {
	case string:
		return v
	case int, int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return p.FormatOK()
		}
		return p.FormatError("operation failed")
	case nil:
		return p.FormatNil()
	case error:
		return p.FormatError(v.Error())
	default:
		return fmt.Sprintf("%v", resp)
	}
}

func (s *Parser) FormatOK() string { return "OK" }

func (s *Parser) FormatNil() string { return "nil" }

func (s *Parser) FormatError(error string) string {
	return fmt.Sprintf("ERR: %s", error)
}
