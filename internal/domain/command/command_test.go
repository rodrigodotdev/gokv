package command

import "testing"

func TestType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"SET is valid", SET, true},
		{"GET is valid", GET, true},
		{"DEL is valid", DEL, true},
		{"EXPIRE is valid", EXPIRE, true},
		{"TTL is valid", TTL, true},
		{"PERSIST is valid", PERSIST, true},
		{"QUIT is valid", QUIT, true},
		{"KEYS is valid", KEYS, true},
		{"EXISTS is valid", EXISTS, true},
		{"PING is valid", PING, true},
		{"INFO is valid", INFO, true},
		{"empty string is invalid", Type(""), false},
		{"UNKNOWN is invalid", Type("UNKNOWN"), false},
		{"lowercase get is invalid", Type("get"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.typ.IsValid(); got != tt.want {
				t.Errorf("Type(%q).IsValid() = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestType_IsWriteCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"SET is write", SET, true},
		{"DEL is write", DEL, true},
		{"EXPIRE is write", EXPIRE, true},
		{"PERSIST is write", PERSIST, true},
		{"GET is not write", GET, false},
		{"TTL is not write", TTL, false},
		{"KEYS is not write", KEYS, false},
		{"EXISTS is not write", EXISTS, false},
		{"PING is not write", PING, false},
		{"INFO is not write", INFO, false},
		{"QUIT is not write", QUIT, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.typ.IsWriteCommand(); got != tt.want {
				t.Errorf("Type(%q).IsWriteCommand() = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestType_IsReadCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want bool
	}{
		{"GET is read", GET, true},
		{"TTL is read", TTL, true},
		{"KEYS is read", KEYS, true},
		{"EXISTS is read", EXISTS, true},
		{"PING is read", PING, true},
		{"INFO is read", INFO, true},
		{"SET is not read", SET, false},
		{"DEL is not read", DEL, false},
		{"EXPIRE is not read", EXPIRE, false},
		{"PERSIST is not read", PERSIST, false},
		{"QUIT is not read", QUIT, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.typ.IsReadCommand(); got != tt.want {
				t.Errorf("Type(%q).IsReadCommand() = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}

func TestType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  Type
		want string
	}{
		{"SET string", SET, "SET"},
		{"GET string", GET, "GET"},
		{"DEL string", DEL, "DEL"},
		{"EXPIRE string", EXPIRE, "EXPIRE"},
		{"TTL string", TTL, "TTL"},
		{"PERSIST string", PERSIST, "PERSIST"},
		{"QUIT string", QUIT, "QUIT"},
		{"KEYS string", KEYS, "KEYS"},
		{"EXISTS string", EXISTS, "EXISTS"},
		{"PING string", PING, "PING"},
		{"INFO string", INFO, "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.typ.String(); got != tt.want {
				t.Errorf("Type(%q).String() = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}
