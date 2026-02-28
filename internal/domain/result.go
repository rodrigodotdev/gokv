package domain

// Result represents the outcome of a command execution.
// It carries either a value or an error, allowing the adapter layer
// to format the response according to the protocol.
type Result struct {
	value any
	err   error
}

// OK returns a result indicating success.
func OK() Result {
	return Result{value: "OK"}
}

// Nil returns a result indicating a nil/missing value.
func Nil() Result {
	return Result{}
}

// Value returns a result carrying a value.
func Value(v any) Result {
	return Result{value: v}
}

// Error returns a result carrying an error.
func Error(err error) Result {
	return Result{err: err}
}

// Err returns the error, if any.
func (r Result) Err() error {
	return r.err
}

// Val returns the value and whether it is present.
func (r Result) Val() (any, bool) {
	return r.value, r.value != nil
}

// IsNil returns true if the result carries no value and no error.
func (r Result) IsNil() bool {
	return r.value == nil && r.err == nil
}
