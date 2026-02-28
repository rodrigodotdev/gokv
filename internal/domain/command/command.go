// Package command defines the command types and structures used to represent
// parsed key-value store operations.
package command

// Type represents the kind of command to execute.
type Type string

// Supported command types.
const (
	SET     Type = "SET"
	GET     Type = "GET"
	DEL     Type = "DEL"
	EXPIRE  Type = "EXPIRE"
	TTL     Type = "TTL"
	PERSIST Type = "PERSIST"
	QUIT    Type = "QUIT"

	KEYS   Type = "KEYS"
	EXISTS Type = "EXISTS"
	PING   Type = "PING"
	INFO   Type = "INFO"
)

// Command represents a parsed command with its type and arguments.
type Command struct {
	Type Type
	Args []string
}

// String returns the string representation of the command type.
func (t Type) String() string {
	return string(t)
}

// IsValid reports whether t is a recognized command type.
func (t Type) IsValid() bool {
	switch t {
	case SET, GET, DEL, EXPIRE, TTL, PERSIST, QUIT, KEYS, EXISTS, PING, INFO:
		return true
	default:
		return false
	}
}

// IsWriteCommand reports whether t mutates the store's state.
// Note: QUIT is a connection-control command and is classified as neither
// a write nor a read command.
func (t Type) IsWriteCommand() bool {
	switch t {
	case SET, DEL, EXPIRE, PERSIST:
		return true
	default:
		return false
	}
}

// IsReadCommand reports whether t only reads from the store.
// Note: QUIT is a connection-control command and is classified as neither
// a write nor a read command.
func (t Type) IsReadCommand() bool {
	switch t {
	case GET, TTL, KEYS, EXISTS, PING, INFO:
		return true
	default:
		return false
	}
}
