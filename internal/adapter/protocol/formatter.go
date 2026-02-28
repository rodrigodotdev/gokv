package protocol

import (
	"fmt"
	"strings"
)

// Formatter converts domain values into protocol response strings suitable
// for sending to the client over a TCP connection.
type Formatter struct{}

// NewFormatter returns a new Formatter.
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatResponse formats an arbitrary domain value into a protocol response
// string suitable for sending to the client.
func (f *Formatter) FormatResponse(resp any) string {
	switch v := resp.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return f.FormatOK()
		}
		return f.FormatError("operation failed")
	case []string:
		return strings.Join(v, " ")
	case nil:
		return f.FormatNil()
	case error:
		return f.FormatError(v.Error())
	default:
		return fmt.Sprintf("%v", resp)
	}
}

// FormatOK returns the standard success response string.
func (f *Formatter) FormatOK() string { return "OK" }

// FormatNil returns the standard nil/not-found response string.
func (f *Formatter) FormatNil() string { return "nil" }

// FormatError returns a protocol error response prefixed with "ERR: ".
func (f *Formatter) FormatError(msg string) string {
	return fmt.Sprintf("ERR: %s", msg)
}
