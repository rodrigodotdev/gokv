package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

// respEntry builds the RESP-like serialization for a single command.
// e.g. respEntry("SET", "key", "value") => "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
func respEntry(parts ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(parts))
	for _, p := range parts {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(p), p)
	}
	return b.String()
}

// assertCommands compares two command slices element-by-element, reporting
// Type and Args mismatches via t.Errorf and length mismatches via t.Fatalf.
func assertCommands(t *testing.T, got, want []command.Command) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d commands, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i, w := range want {
		if got[i].Type != w.Type {
			t.Errorf("cmd[%d].Type = %q, want %q", i, got[i].Type, w.Type)
		}
		if len(got[i].Args) != len(w.Args) {
			t.Errorf("cmd[%d].Args length = %d, want %d\ngot:  %v\nwant: %v", i, len(got[i].Args), len(w.Args), got[i].Args, w.Args)
			continue
		}
		for j, a := range w.Args {
			if got[i].Args[j] != a {
				t.Errorf("cmd[%d].Args[%d] = %q, want %q", i, j, got[i].Args[j], a)
			}
		}
	}
}

func TestAppend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		commands    []command.Command
		wantContent string
		wantErr     bool
		cancelCtx   bool
	}{
		{
			name: "writes command with no args in RESP format",
			commands: []command.Command{
				{Type: command.SET, Args: nil},
			},
			wantContent: respEntry("SET"),
		},
		{
			name: "writes command with args in RESP format",
			commands: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
			},
			wantContent: respEntry("SET", "key1", "value1"),
		},
		{
			name: "handles values with spaces correctly",
			commands: []command.Command{
				{Type: command.SET, Args: []string{"greeting", "hello world"}},
			},
			wantContent: respEntry("SET", "greeting", "hello world"),
		},
		{
			name: "cancelled context returns context error",
			commands: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
			},
			cancelCtx: true,
			wantErr:   true,
		},
		{
			name: "multiple appends produce multiple RESP entries",
			commands: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
				{Type: command.DEL, Args: []string{"key2"}},
				{Type: command.SET, Args: []string{"key3", "value3"}},
			},
			wantContent: respEntry("SET", "key1", "value1") +
				respEntry("DEL", "key2") +
				respEntry("SET", "key3", "value3"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(t.TempDir(), "test.aof")
			aof, err := NewAOF(filePath)
			if err != nil {
				t.Fatalf("NewAOF() error = %v", err)
			}
			defer func() { _ = aof.Close() }()

			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			var lastErr error
			for i := range tt.commands {
				if err := aof.Append(ctx, &tt.commands[i]); err != nil {
					lastErr = err
					break
				}
			}

			if tt.wantErr {
				if lastErr == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if lastErr != nil {
				t.Fatalf("Append() unexpected error = %v", lastErr)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("reading AOF file: %v", err)
			}

			got := string(data)
			if got != tt.wantContent {
				t.Errorf("file content mismatch\ngot:  %q\nwant: %q", got, tt.wantContent)
			}
		})
	}
}

func TestReplay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fileContent string
		createFile  bool
		cancelCtx   bool
		wantCmds    []command.Command
		wantErr     bool
	}{
		{
			name:       "calls handler for each write command",
			createFile: true,
			fileContent: respEntry("SET", "key1", "value1") +
				respEntry("DEL", "key2") +
				respEntry("EXPIRE", "key3", "100") +
				respEntry("PERSIST", "key4"),
			wantCmds: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
				{Type: command.DEL, Args: []string{"key2"}},
				{Type: command.EXPIRE, Args: []string{"key3", "100"}},
				{Type: command.PERSIST, Args: []string{"key4"}},
			},
		},
		{
			name:       "skips non-write commands",
			createFile: true,
			fileContent: respEntry("GET", "key1") +
				respEntry("SET", "key2", "value2") +
				respEntry("PING") +
				respEntry("TTL", "key3") +
				respEntry("DEL", "key4") +
				respEntry("KEYS", "*") +
				respEntry("EXISTS", "key5") +
				respEntry("INFO"),
			wantCmds: []command.Command{
				{Type: command.SET, Args: []string{"key2", "value2"}},
				{Type: command.DEL, Args: []string{"key4"}},
			},
		},
		{
			name:        "empty file returns no commands",
			createFile:  true,
			fileContent: "",
			wantCmds:    nil,
		},
		{
			name:       "non-existent file returns nil",
			createFile: false,
			wantCmds:   nil,
		},
		{
			name:        "cancelled context returns context error",
			createFile:  true,
			fileContent: respEntry("SET", "key1", "value1"),
			cancelCtx:   true,
			wantErr:     true,
		},
		{
			name:        "handles values with spaces",
			createFile:  true,
			fileContent: respEntry("SET", "greeting", "hello world"),
			wantCmds: []command.Command{
				{Type: command.SET, Args: []string{"greeting", "hello world"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.aof")

			if tt.createFile {
				if err := os.WriteFile(filePath, []byte(tt.fileContent), 0o644); err != nil {
					t.Fatalf("writing test AOF file: %v", err)
				}
			}

			aof, err := NewAOF(filePath)
			if err != nil {
				t.Fatalf("NewAOF() error = %v", err)
			}
			defer func() { _ = aof.Close() }()

			ctx := context.Background()
			if tt.cancelCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			var got []command.Command
			handler := func(_ context.Context, cmd *command.Command) {
				got = append(got, *cmd)
			}

			err = aof.Replay(ctx, handler)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Replay() unexpected error = %v", err)
			}

			assertCommands(t, got, tt.wantCmds)
		})
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "does not error on valid file"},
		{name: "can be called on a freshly created AOF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(t.TempDir(), "test.aof")
			aof, err := NewAOF(filePath)
			if err != nil {
				t.Fatalf("NewAOF() error = %v", err)
			}

			if err := aof.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}
		})
	}
}

func TestAOFProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     AOFProviderConfig
		wantNil bool
		wantErr bool
	}{
		{
			name: "enabled false returns nil nil",
			cfg: AOFProviderConfig{
				Enabled:  false,
				FilePath: "irrelevant.aof",
			},
			wantNil: true,
		},
		{
			name: "enabled true creates the AOF",
			cfg: AOFProviderConfig{
				Enabled: true,
				// FilePath set in test body
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.cfg
			if cfg.Enabled {
				cfg.FilePath = filepath.Join(t.TempDir(), "provider.aof")
			}

			repo, err := NewAOFProvider(cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewAOFProvider() error = %v", err)
			}

			if tt.wantNil {
				if repo != nil {
					t.Fatalf("expected nil repository, got %v", repo)
				}
				return
			}

			if repo == nil {
				t.Fatal("expected non-nil repository, got nil")
			}
			defer func() { _ = repo.Close() }()
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		appends  []command.Command
		wantCmds []command.Command
	}{
		{
			name: "append close and replay reads back correctly",
			appends: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
				{Type: command.DEL, Args: []string{"key2"}},
				{Type: command.EXPIRE, Args: []string{"key3", "60"}},
				{Type: command.SET, Args: []string{"key4", "value4"}},
				{Type: command.PERSIST, Args: []string{"key5"}},
			},
			wantCmds: []command.Command{
				{Type: command.SET, Args: []string{"key1", "value1"}},
				{Type: command.DEL, Args: []string{"key2"}},
				{Type: command.EXPIRE, Args: []string{"key3", "60"}},
				{Type: command.SET, Args: []string{"key4", "value4"}},
				{Type: command.PERSIST, Args: []string{"key5"}},
			},
		},
		{
			name: "values with spaces survive round trip",
			appends: []command.Command{
				{Type: command.SET, Args: []string{"greeting", "hello world"}},
				{Type: command.SET, Args: []string{"msg", "foo bar baz"}},
			},
			wantCmds: []command.Command{
				{Type: command.SET, Args: []string{"greeting", "hello world"}},
				{Type: command.SET, Args: []string{"msg", "foo bar baz"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filePath := filepath.Join(t.TempDir(), "roundtrip.aof")
			ctx := context.Background()

			// Phase 1: Append and close
			aof1, err := NewAOF(filePath)
			if err != nil {
				t.Fatalf("NewAOF() for writing: %v", err)
			}
			for i := range tt.appends {
				if err := aof1.Append(ctx, &tt.appends[i]); err != nil {
					t.Fatalf("Append(%s %v) error = %v", tt.appends[i].Type, tt.appends[i].Args, err)
				}
			}
			if err := aof1.Close(); err != nil {
				t.Fatalf("Close() after writing: %v", err)
			}

			// Phase 2: Open a new AOF on the same file and replay
			aof2, err := NewAOF(filePath)
			if err != nil {
				t.Fatalf("NewAOF() for reading: %v", err)
			}
			defer func() { _ = aof2.Close() }()

			var got []command.Command
			handler := func(_ context.Context, cmd *command.Command) {
				got = append(got, *cmd)
			}

			if err := aof2.Replay(ctx, handler); err != nil {
				t.Fatalf("Replay() error = %v", err)
			}

			assertCommands(t, got, tt.wantCmds)
		})
	}
}
