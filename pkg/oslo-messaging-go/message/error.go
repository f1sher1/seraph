package message

import "fmt"

// Error captures the code and reason a channel or connection has been closed
// by the server.
type Error struct {
	Code    int    // constant code from the specification
	Reason  string // description of the error
	Server  bool   // true when initiated from the server, false when from this library
	Recover bool   // true when this error can be recovered by retrying later or with different parameters
}

func NewError(code int, text string, recover, server bool) *Error {
	return &Error{
		Code:    code,
		Reason:  text,
		Recover: recover,
		Server:  server,
	}
}

func (e Error) Error() string {
	return fmt.Sprintf("Exception (%d) Reason: %q", e.Code, e.Reason)
}
