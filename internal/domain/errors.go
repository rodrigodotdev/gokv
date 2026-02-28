// Package domain defines the core types and errors shared across the
// key-value store's domain layer.
package domain

import "errors"

// Sentinel errors returned by command parsing and execution.
var (
	// ErrEmptyCommand indicates that an empty command was received.
	ErrEmptyCommand = errors.New("empty command")
	// ErrUnknownCommand indicates that the command type is not recognized.
	ErrUnknownCommand = errors.New("unknown command")
	// ErrWrongArgs indicates that the command was called with an incorrect
	// number of arguments.
	ErrWrongArgs = errors.New("wrong number of arguments")
	// ErrKeyNotFound indicates that the requested key does not exist in the store.
	ErrKeyNotFound = errors.New("key not found")
)
