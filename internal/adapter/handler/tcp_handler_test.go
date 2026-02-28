package handler

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/rodrigodotdev/gokv/internal/domain"
	"github.com/rodrigodotdev/gokv/internal/domain/command"
)

// --- Mock implementations ---

type mockCommandProcessor struct {
	handler func(ctx context.Context, cmd *command.Command) domain.Result
}

func (m *mockCommandProcessor) HandleCommand(ctx context.Context, cmd *command.Command) domain.Result {
	if m.handler != nil {
		return m.handler(ctx, cmd)
	}
	return domain.OK()
}

type mockParser struct {
	parseFunc func(input string) (*command.Command, error)
}

func (m *mockParser) ParseCommand(input string) (*command.Command, error) {
	if m.parseFunc != nil {
		return m.parseFunc(input)
	}
	return &command.Command{Type: command.PING, Args: []string{}}, nil
}

type mockFormatter struct{}

func (m *mockFormatter) FormatResponse(resp any) string {
	switch v := resp.(type) {
	case string:
		return "+" + v + "\r\n"
	default:
		_ = v
		return "+OK\r\n"
	}
}
func (m *mockFormatter) FormatOK() string              { return "+OK\r\n" }
func (m *mockFormatter) FormatNil() string             { return "$-1\r\n" }
func (m *mockFormatter) FormatError(msg string) string { return "-ERR " + msg + "\r\n" }

type mockConnTracker struct {
	incremented int
	decremented int
}

func (m *mockConnTracker) IncrementConnections() { m.incremented++ }
func (m *mockConnTracker) DecrementConnections() { m.decremented++ }

// --- Tests ---

func TestHandleConnectionPingResponse(t *testing.T) {
	t.Parallel()

	processor := &mockCommandProcessor{
		handler: func(_ context.Context, cmd *command.Command) domain.Result {
			if cmd.Type == command.PING {
				return domain.Value("PONG")
			}
			return domain.OK()
		},
	}

	parser := &mockParser{
		parseFunc: func(input string) (*command.Command, error) {
			return &command.Command{Type: command.PING, Args: []string{}}, nil
		},
	}

	tracker := &mockConnTracker{}
	h := NewTCPHandler(processor, parser, &mockFormatter{}, tracker, 5*time.Second)

	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.HandleConnection(server)
	}()

	// Send PING command
	_, err := client.Write([]byte("PING\r\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read response
	buf := make([]byte, 1024)
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "PONG") {
		t.Errorf("expected PONG in response, got %q", response)
	}

	// Close client to trigger handler exit
	_ = client.Close()
	<-done

	if tracker.incremented != 1 {
		t.Errorf("connections incremented = %d, want 1", tracker.incremented)
	}
	if tracker.decremented != 1 {
		t.Errorf("connections decremented = %d, want 1", tracker.decremented)
	}
}

func TestHandleConnectionQuitCommand(t *testing.T) {
	t.Parallel()

	processor := &mockCommandProcessor{
		handler: func(_ context.Context, cmd *command.Command) domain.Result {
			return domain.OK()
		},
	}

	parser := &mockParser{
		parseFunc: func(input string) (*command.Command, error) {
			trimmed := strings.TrimSpace(input)
			switch strings.ToUpper(trimmed) {
			case "QUIT":
				return &command.Command{Type: command.QUIT, Args: []string{}}, nil
			default:
				return &command.Command{Type: command.PING, Args: []string{}}, nil
			}
		},
	}

	tracker := &mockConnTracker{}
	h := NewTCPHandler(processor, parser, &mockFormatter{}, tracker, 5*time.Second)

	client, server := net.Pipe()
	defer func() { _ = client.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.HandleConnection(server)
	}()

	// Send QUIT command
	_, err := client.Write([]byte("QUIT\r\n"))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	// Read OK response
	buf := make([]byte, 1024)
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "OK") {
		t.Errorf("expected OK in response, got %q", response)
	}

	// Wait for handler to exit
	select {
	case <-done:
		// Handler exited as expected
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not exit after QUIT")
	}
}

func TestHandleConnectionTracksConnections(t *testing.T) {
	t.Parallel()

	tracker := &mockConnTracker{}
	h := NewTCPHandler(
		&mockCommandProcessor{},
		&mockParser{},
		&mockFormatter{},
		tracker,
		5*time.Second,
	)

	client, server := net.Pipe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.HandleConnection(server)
	}()

	// Close immediately
	_ = client.Close()
	<-done

	if tracker.incremented != 1 {
		t.Errorf("incremented = %d, want 1", tracker.incremented)
	}
	if tracker.decremented != 1 {
		t.Errorf("decremented = %d, want 1", tracker.decremented)
	}
}
