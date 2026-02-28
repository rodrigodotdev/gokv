package protocol

import (
	"errors"
	"strings"
	"testing"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

func TestParseCommand(t *testing.T) {
	p := NewParser()

	tests := []struct {
		name     string
		input    string
		wantType command.Type
		wantArgs []string
		wantErr  error
	}{
		{
			name:    "empty string returns ErrEmptyCommand",
			input:   "",
			wantErr: domain.ErrEmptyCommand,
		},
		{
			name:    "whitespace-only returns ErrEmptyCommand",
			input:   "   \t\n  ",
			wantErr: domain.ErrEmptyCommand,
		},
		{
			name:    "unknown command returns ErrUnknownCommand",
			input:   "FOOBAR",
			wantErr: domain.ErrUnknownCommand,
		},
		{
			name:     "valid command with no args (PING)",
			input:    "PING",
			wantType: command.PING,
			wantArgs: []string{},
		},
		{
			name:     "valid command with args (SET key value)",
			input:    "SET key value",
			wantType: command.SET,
			wantArgs: []string{"key", "value"},
		},
		{
			name:     "case insensitive lowercase",
			input:    "set key value",
			wantType: command.SET,
			wantArgs: []string{"key", "value"},
		},
		{
			name:     "case insensitive mixed case",
			input:    "Set key value",
			wantType: command.SET,
			wantArgs: []string{"key", "value"},
		},
		{
			name:     "case insensitive uppercase",
			input:    "SET key value",
			wantType: command.SET,
			wantArgs: []string{"key", "value"},
		},
		{
			name:     "multi-word value (SET key hello world)",
			input:    "SET key hello world",
			wantType: command.SET,
			wantArgs: []string{"key", "hello", "world"},
		},
		{
			name:     "QUIT command parses correctly",
			input:    "QUIT",
			wantType: command.QUIT,
			wantArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := p.ParseCommand(tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if cmd != nil {
					t.Fatalf("expected nil command on error, got %+v", cmd)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd == nil {
				t.Fatal("expected non-nil command, got nil")
			}
			if cmd.Type != tt.wantType {
				t.Errorf("type: got %q, want %q", cmd.Type, tt.wantType)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Fatalf("args length: got %d, want %d", len(cmd.Args), len(tt.wantArgs))
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("arg[%d]: got %q, want %q", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseCommand_MultiWordValueReconstruction(t *testing.T) {
	// Verify that multi-word values parsed by the parser can be correctly
	// reconstructed by joining args with spaces, which is what the command
	// handler does with strings.Join(args[1:], " ").
	p := NewParser()

	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
	}{
		{
			name:      "two-word value",
			input:     "SET greeting hello world",
			wantKey:   "greeting",
			wantValue: "hello world",
		},
		{
			name:      "three-word value",
			input:     "SET msg one two three",
			wantKey:   "msg",
			wantValue: "one two three",
		},
		{
			name:      "consecutive spaces are collapsed",
			input:     "SET key hello  world",
			wantKey:   "key",
			wantValue: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := p.ParseCommand(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			key := cmd.Args[0]
			value := strings.Join(cmd.Args[1:], " ")

			if key != tt.wantKey {
				t.Errorf("key: got %q, want %q", key, tt.wantKey)
			}
			if value != tt.wantValue {
				t.Errorf("reconstructed value: got %q, want %q", value, tt.wantValue)
			}
		})
	}
}
