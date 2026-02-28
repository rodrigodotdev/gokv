package protocol

import (
	"fmt"
	"testing"
)

func TestFormatResponse(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{
			name:  "string value returns as-is",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "int returns formatted number",
			input: 42,
			want:  "42",
		},
		{
			name:  "int64 returns formatted number",
			input: int64(123456789),
			want:  "123456789",
		},
		{
			name:  "bool true returns OK",
			input: true,
			want:  "OK",
		},
		{
			name:  "bool false returns ERR operation failed",
			input: false,
			want:  "ERR: operation failed",
		},
		{
			name:  "nil returns nil",
			input: nil,
			want:  "nil",
		},
		{
			name:  "error returns ERR message",
			input: fmt.Errorf("something went wrong"),
			want:  "ERR: something went wrong",
		},
		{
			name:  "string slice returns space-joined",
			input: []string{"key1", "key2", "key3"},
			want:  "key1 key2 key3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.FormatResponse(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatOK(t *testing.T) {
	f := NewFormatter()
	got := f.FormatOK()
	if got != "OK" {
		t.Errorf("FormatOK() = %q, want %q", got, "OK")
	}
}

func TestFormatNil(t *testing.T) {
	f := NewFormatter()
	got := f.FormatNil()
	if got != "nil" {
		t.Errorf("FormatNil() = %q, want %q", got, "nil")
	}
}

func TestFormatError(t *testing.T) {
	f := NewFormatter()
	got := f.FormatError("something broke")
	want := "ERR: something broke"
	if got != want {
		t.Errorf("FormatError() = %q, want %q", got, want)
	}
}
