package domain

import (
	"errors"
	"testing"
)

func TestOK(t *testing.T) {
	t.Parallel()

	r := OK()

	val, ok := r.Val()
	if !ok {
		t.Error("OK().Val() should not be nil")
	}
	if val != "OK" {
		t.Errorf("OK().Val() = %v, want %q", val, "OK")
	}
	if r.Err() != nil {
		t.Errorf("OK().Err() = %v, want nil", r.Err())
	}
	if r.IsNil() {
		t.Error("OK().IsNil() = true, want false")
	}
}

func TestNil(t *testing.T) {
	t.Parallel()

	r := Nil()

	val, ok := r.Val()
	if ok {
		t.Error("Nil().Val() second return should be false")
	}
	if val != nil {
		t.Errorf("Nil().Val() = %v, want nil", val)
	}
	if r.Err() != nil {
		t.Errorf("Nil().Err() = %v, want nil", r.Err())
	}
	if !r.IsNil() {
		t.Error("Nil().IsNil() = false, want true")
	}
}

func TestValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  any
	}{
		{"string value", "hello"},
		{"int value", 42},
		{"bool value", true},
		{"string slice value", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := Value(tt.val)

			val, ok := r.Val()
			if !ok {
				t.Error("Value().Val() second return should be true")
			}
			if r.Err() != nil {
				t.Errorf("Value().Err() = %v, want nil", r.Err())
			}
			if r.IsNil() {
				t.Error("Value().IsNil() = true, want false")
			}

			// Type-specific value comparison.
			switch expected := tt.val.(type) {
			case string:
				if val != expected {
					t.Errorf("Val() = %v, want %v", val, expected)
				}
			case int:
				if val != expected {
					t.Errorf("Val() = %v, want %v", val, expected)
				}
			case bool:
				if val != expected {
					t.Errorf("Val() = %v, want %v", val, expected)
				}
			case []string:
				got, ok := val.([]string)
				if !ok {
					t.Fatalf("Val() type = %T, want []string", val)
				}
				if len(got) != len(expected) {
					t.Fatalf("Val() len = %d, want %d", len(got), len(expected))
				}
				for i := range expected {
					if got[i] != expected[i] {
						t.Errorf("Val()[%d] = %q, want %q", i, got[i], expected[i])
					}
				}
			}
		})
	}
}

func TestError(t *testing.T) {
	t.Parallel()

	err := errors.New("something went wrong")
	r := Error(err)

	val, ok := r.Val()
	if ok {
		t.Error("Error().Val() second return should be false")
	}
	if val != nil {
		t.Errorf("Error().Val() = %v, want nil", val)
	}
	if r.Err() == nil {
		t.Fatal("Error().Err() = nil, want error")
	}
	if !errors.Is(r.Err(), err) {
		t.Errorf("Error().Err() = %v, want %v", r.Err(), err)
	}
	if r.IsNil() {
		t.Error("Error().IsNil() = true, want false")
	}
}

func TestIsNil_FalseForNonNilResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result Result
	}{
		{"OK result", OK()},
		{"Value result", Value("data")},
		{"Error result", Error(errors.New("err"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.result.IsNil() {
				t.Errorf("%s: IsNil() = true, want false", tt.name)
			}
		})
	}
}
