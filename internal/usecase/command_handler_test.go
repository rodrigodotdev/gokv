package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
	"github.com/rodrigodotdev/gokv/internal/domain/repository"
)

// --- Mock implementations ---

// mockStore implements KeyValueStore with an in-memory map.
type mockStore struct {
	data    map[string]string
	expires map[string]int // seconds remaining, -1 means no expiry
}

func newMockStore() *mockStore {
	return &mockStore{
		data:    make(map[string]string),
		expires: make(map[string]int),
	}
}

func (m *mockStore) Set(_ context.Context, key, value string) {
	m.data[key] = value
}

func (m *mockStore) Get(_ context.Context, key string) (string, bool) {
	v, ok := m.data[key]
	return v, ok
}

func (m *mockStore) Del(_ context.Context, key string) int {
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		delete(m.expires, key)
		return 1
	}
	return 0
}

func (m *mockStore) Expire(_ context.Context, key string, seconds int) bool {
	if _, ok := m.data[key]; !ok {
		return false
	}
	m.expires[key] = seconds
	return true
}

func (m *mockStore) TTL(_ context.Context, key string) int64 {
	if _, ok := m.data[key]; !ok {
		return -2
	}
	if sec, ok := m.expires[key]; ok {
		return int64(sec)
	}
	return -1
}

func (m *mockStore) Persist(_ context.Context, key string) bool {
	if _, ok := m.data[key]; !ok {
		return false
	}
	if _, hasExpiry := m.expires[key]; !hasExpiry {
		return false
	}
	delete(m.expires, key)
	return true
}

func (m *mockStore) Keys(_ context.Context, pattern string) []string {
	var keys []string
	for k := range m.data {
		matched, _ := filepath.Match(pattern, k)
		if matched {
			keys = append(keys, k)
		}
	}
	return keys
}

func (m *mockStore) Exists(_ context.Context, key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *mockStore) Size(_ context.Context) int {
	return len(m.data)
}

// mockPersistence implements domain.PersistenceRepository.
type mockPersistence struct {
	commands []string // track persisted commands as "CMD arg1 arg2 ..."
}

func newMockPersistence() *mockPersistence {
	return &mockPersistence{}
}

func (m *mockPersistence) Append(_ context.Context, cmd *command.Command) error {
	entry := cmd.Type.String()
	if len(cmd.Args) > 0 {
		entry += " " + strings.Join(cmd.Args, " ")
	}
	m.commands = append(m.commands, entry)
	return nil
}

func (m *mockPersistence) Replay(_ context.Context, handler repository.CommandHandler) error {
	return nil
}

func (m *mockPersistence) Close() error {
	return nil
}

// mockStats implements StatsCollector.
type mockStats struct {
	commands int64
}

func (m *mockStats) IncrementCommands() {
	m.commands++
}

func (m *mockStats) GetInfo(_ context.Context) string {
	return fmt.Sprintf("total_commands_processed:%d", m.commands)
}

// --- Helper ---

func newTestHandler() (*CommandHandler, *mockStore, *mockPersistence, *mockStats) {
	store := newMockStore()
	persist := newMockPersistence()
	stats := &mockStats{}
	handler := NewCommandHandler(store, persist, stats)
	return handler, store, persist, stats
}

// assertResultError checks that result carries an error wrapping wantErr.
func assertResultError(t *testing.T, result domain.Result, wantErr error) {
	t.Helper()
	if result.Err() == nil {
		t.Fatalf("expected error containing %v, got nil", wantErr)
	}
	if !errors.Is(result.Err(), wantErr) {
		t.Fatalf("expected error wrapping %v, got %v", wantErr, result.Err())
	}
}

// assertResultValue checks that result carries the expected value.
func assertResultValue(t *testing.T, result domain.Result, wantVal any) {
	t.Helper()
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	val, ok := result.Val()
	if !ok {
		t.Fatal("expected a value, got nil result")
	}
	if val != wantVal {
		t.Fatalf("expected value %v, got %v", wantVal, val)
	}
}

// assertResultNil checks that result is a nil result (no value, no error).
func assertResultNil(t *testing.T, result domain.Result) {
	t.Helper()
	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}
	if !result.IsNil() {
		t.Fatal("expected nil result, got a value")
	}
}

// assertPersisted checks whether commands were persisted (or not).
func assertPersisted(t *testing.T, persist *mockPersistence, want bool) {
	t.Helper()
	if want && len(persist.commands) == 0 {
		t.Fatal("expected command to be persisted, but nothing was persisted")
	}
	if !want && len(persist.commands) > 0 {
		t.Fatalf("expected no persistence, but got %v", persist.commands)
	}
}

// runCommand is a convenience for executing a command against a test handler.
func runCommand(handler *CommandHandler, cmdType command.Type, args []string) domain.Result {
	cmd := &command.Command{Type: cmdType, Args: args}
	return handler.HandleCommand(context.Background(), cmd)
}

// --- Tests ---

func TestHandleCommand_SET(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setup       func(*mockStore)
		wantErr     error
		wantVal     any
		wantStored  map[string]string // expected key-value pairs in store after command
		wantPersist bool
	}{
		{
			name:        "SET with key and value returns OK",
			args:        []string{"mykey", "myvalue"},
			wantVal:     "OK",
			wantStored:  map[string]string{"mykey": "myvalue"},
			wantPersist: true,
		},
		{
			name:        "SET with key and multi-word value joins args",
			args:        []string{"greeting", "hello", "world"},
			wantVal:     "OK",
			wantStored:  map[string]string{"greeting": "hello world"},
			wantPersist: true,
		},
		{
			name:    "SET with fewer than 2 args returns ErrWrongArgs",
			args:    []string{"onlykey"},
			wantErr: domain.ErrWrongArgs,
		},
		{
			name:    "SET with no args returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, persist, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.SET, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			assertResultValue(t, result, tt.wantVal)

			for k, v := range tt.wantStored {
				got, exists := store.data[k]
				if !exists {
					t.Fatalf("expected key %q to be stored", k)
				}
				if got != v {
					t.Fatalf("expected store[%q] = %q, got %q", k, v, got)
				}
			}

			assertPersisted(t, persist, tt.wantPersist)
		})
	}
}

func TestHandleCommand_GET(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantErr error
		wantVal any
		wantNil bool
	}{
		{
			name: "GET existing key returns the value",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantVal: "myvalue",
		},
		{
			name:    "GET non-existent key returns Nil",
			args:    []string{"missing"},
			wantNil: true,
		},
		{
			name:    "GET with no args returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
		{
			name:    "GET with too many args returns ErrWrongArgs",
			args:    []string{"key1", "key2"},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.GET, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			if tt.wantNil {
				assertResultNil(t, result)
				return
			}

			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_DEL(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		setup           func(*mockStore)
		wantErr         error
		wantVal         any
		wantPersist     bool
		wantPersistCmds []string // if set, verify exact persisted commands
	}{
		{
			name: "DEL existing key returns count 1",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantVal:         1,
			wantPersist:     true,
			wantPersistCmds: []string{"DEL mykey"},
		},
		{
			name:        "DEL non-existent key returns count 0",
			args:        []string{"missing"},
			wantVal:     0,
			wantPersist: false,
		},
		{
			name: "DEL multiple keys only persists deleted ones",
			args: []string{"exists1", "missing", "exists2"},
			setup: func(s *mockStore) {
				s.data["exists1"] = "v1"
				s.data["exists2"] = "v2"
			},
			wantVal:         2,
			wantPersist:     true,
			wantPersistCmds: []string{"DEL exists1", "DEL exists2"},
		},
		{
			name:    "DEL with no args returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, persist, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.DEL, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			assertResultValue(t, result, tt.wantVal)
			assertPersisted(t, persist, tt.wantPersist)

			if tt.wantPersistCmds != nil {
				if len(persist.commands) != len(tt.wantPersistCmds) {
					t.Fatalf("expected %d persisted commands, got %d: %v",
						len(tt.wantPersistCmds), len(persist.commands), persist.commands)
				}
				for i, want := range tt.wantPersistCmds {
					if persist.commands[i] != want {
						t.Errorf("persisted[%d] = %q, want %q", i, persist.commands[i], want)
					}
				}
			}
		})
	}
}

func TestHandleCommand_EXPIRE(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantErr error
		wantVal any
	}{
		{
			name: "EXPIRE with valid args on existing key returns true",
			args: []string{"mykey", "60"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantVal: true,
		},
		{
			name:    "EXPIRE on non-existent key returns false",
			args:    []string{"missing", "60"},
			wantVal: false,
		},
		{
			name: "EXPIRE with invalid seconds returns error",
			args: []string{"mykey", "notanumber"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantErr: fmt.Errorf("invalid expiration time"),
		},
		{
			name:    "EXPIRE with wrong arg count (1 arg) returns ErrWrongArgs",
			args:    []string{"mykey"},
			wantErr: domain.ErrWrongArgs,
		},
		{
			name:    "EXPIRE with wrong arg count (0 args) returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.EXPIRE, tt.args)

			if tt.wantErr != nil {
				if result.Err() == nil {
					t.Fatalf("expected an error, got nil")
				}
				if errors.Is(tt.wantErr, domain.ErrWrongArgs) {
					assertResultError(t, result, domain.ErrWrongArgs)
				} else if !strings.Contains(result.Err().Error(), "invalid expiration time") {
					t.Fatalf("expected error containing 'invalid expiration time', got %v", result.Err())
				}
				return
			}

			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_TTL(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantErr error
		wantVal any
	}{
		{
			name: "TTL returns expiry seconds for key with TTL set",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
				s.expires["mykey"] = 120
			},
			wantVal: int64(120),
		},
		{
			name: "TTL returns -1 for key without expiry",
			args: []string{"noexpiry"},
			setup: func(s *mockStore) {
				s.data["noexpiry"] = "value"
			},
			wantVal: int64(-1),
		},
		{
			name:    "TTL returns -2 for non-existent key",
			args:    []string{"missing"},
			wantVal: int64(-2),
		},
		{
			name:    "TTL with wrong arg count returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.TTL, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_PERSIST(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantErr error
		wantVal any
	}{
		{
			name: "PERSIST key with TTL returns true",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
				s.expires["mykey"] = 60
			},
			wantVal: true,
		},
		{
			name: "PERSIST key without TTL returns false",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantVal: false,
		},
		{
			name:    "PERSIST non-existent key returns false",
			args:    []string{"missing"},
			wantVal: false,
		},
		{
			name:    "PERSIST with wrong arg count returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.PERSIST, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_KEYS(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantNil bool
		wantLen int
	}{
		{
			name: "KEYS with matching pattern returns keys",
			args: []string{"user:*"},
			setup: func(s *mockStore) {
				s.data["user:1"] = "alice"
				s.data["user:2"] = "bob"
				s.data["session:1"] = "xyz"
			},
			wantLen: 2,
		},
		{
			name: "KEYS with wildcard returns all keys",
			args: []string{"*"},
			setup: func(s *mockStore) {
				s.data["a"] = "1"
				s.data["b"] = "2"
			},
			wantLen: 2,
		},
		{
			name:    "KEYS with no matches returns Nil",
			args:    []string{"nonexistent:*"},
			wantNil: true,
		},
		{
			name: "KEYS with no args uses wildcard pattern",
			args: []string{},
			setup: func(s *mockStore) {
				s.data["x"] = "1"
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.KEYS, tt.args)

			if result.Err() != nil {
				t.Fatalf("unexpected error: %v", result.Err())
			}

			if tt.wantNil {
				assertResultNil(t, result)
				return
			}

			val, ok := result.Val()
			if !ok {
				t.Fatal("expected a value, got nil result")
			}

			keys, isSlice := val.([]string)
			if !isSlice {
				t.Fatalf("expected []string, got %T", val)
			}
			if len(keys) != tt.wantLen {
				t.Fatalf("expected %d keys, got %d: %v", tt.wantLen, len(keys), keys)
			}
		})
	}
}

func TestHandleCommand_EXISTS(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*mockStore)
		wantErr error
		wantVal any
	}{
		{
			name: "EXISTS returns 1 for existing key",
			args: []string{"mykey"},
			setup: func(s *mockStore) {
				s.data["mykey"] = "myvalue"
			},
			wantVal: 1,
		},
		{
			name:    "EXISTS returns 0 for missing key",
			args:    []string{"missing"},
			wantVal: 0,
		},
		{
			name: "EXISTS with multiple keys returns count of existing",
			args: []string{"a", "b", "c"},
			setup: func(s *mockStore) {
				s.data["a"] = "1"
				s.data["c"] = "3"
			},
			wantVal: 2,
		},
		{
			name: "EXISTS with duplicate existing keys counts each",
			args: []string{"x", "x"},
			setup: func(s *mockStore) {
				s.data["x"] = "1"
			},
			wantVal: 2,
		},
		{
			name:    "EXISTS with wrong arg count returns ErrWrongArgs",
			args:    []string{},
			wantErr: domain.ErrWrongArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, store, _, _ := newTestHandler()

			if tt.setup != nil {
				tt.setup(store)
			}

			result := runCommand(handler, command.EXISTS, tt.args)

			if tt.wantErr != nil {
				assertResultError(t, result, tt.wantErr)
				return
			}

			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_PING(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantVal any
	}{
		{
			name:    "PING with no args returns PONG",
			args:    []string{},
			wantVal: "PONG",
		},
		{
			name:    "PING with args returns joined args",
			args:    []string{"hello", "world"},
			wantVal: "hello world",
		},
		{
			name:    "PING with single arg returns that arg",
			args:    []string{"hi"},
			wantVal: "hi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _, _, _ := newTestHandler()

			result := runCommand(handler, command.PING, tt.args)
			assertResultValue(t, result, tt.wantVal)
		})
	}
}

func TestHandleCommand_INFO(t *testing.T) {
	handler, _, _, _ := newTestHandler()

	result := runCommand(handler, command.INFO, []string{})

	if result.Err() != nil {
		t.Fatalf("unexpected error: %v", result.Err())
	}

	val, ok := result.Val()
	if !ok {
		t.Fatal("expected a value, got nil result")
	}

	info, isString := val.(string)
	if !isString {
		t.Fatalf("expected string, got %T", val)
	}
	if !strings.Contains(info, "total_commands_processed") {
		t.Fatalf("expected info to contain 'total_commands_processed', got %q", info)
	}
}

func TestHandleCommand_UnknownCommand(t *testing.T) {
	handler, _, _, _ := newTestHandler()

	result := runCommand(handler, command.Type("FOOBAR"), []string{})
	assertResultError(t, result, domain.ErrUnknownCommand)
}

func TestHandleCommand_StatsIncremented(t *testing.T) {
	handler, _, _, stats := newTestHandler()

	for range 3 {
		runCommand(handler, command.PING, []string{})
	}

	if stats.commands != 3 {
		t.Fatalf("expected stats.commands to be 3, got %d", stats.commands)
	}
}

func TestHandleCommandSetCancelledContext(t *testing.T) {
	t.Parallel()

	store := newMockStore()
	persist := newMockPersistence()
	stats := &mockStats{}
	handler := NewCommandHandler(store, persist, stats)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result := handler.HandleCommand(ctx, &command.Command{
		Type: command.SET,
		Args: []string{"key", "value"},
	})

	// After context cancellation, store should not have the key
	if _, exists := store.data["key"]; exists {
		t.Error("key should not exist in store after cancelled context")
	}

	// Persistence should not have been called
	if len(persist.commands) > 0 {
		t.Error("persist should not have been called after cancelled context")
	}

	// Result should indicate error, not OK
	if result.Err() == nil {
		t.Error("expected error result on cancelled context, got nil error")
	}
}
