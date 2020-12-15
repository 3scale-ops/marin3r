package errors

import "fmt"

const (
	// UnknownError is used for non specific errors that don't
	// require special treatment or are yet unknown
	UnknownError ErrorReason = "Unknown"

	// AllRevisionsTaintedError indicates that there are no
	// revisisions suitable for publishing as they are all tainted
	AllRevisionsTaintedError ErrorReason = "AllRevisionsTainted"

	// RollbackOccurredError indicates that the version that was
	// suposed to be published is tainted and a differente previous
	// version has been published.
	RollbackOccurredError ErrorReason = "RollbackOccurred"
)

// ErrorReason is an enum of possible errors for the reconciler
type ErrorReason string

// Error custom error types for envoyconfig controller
type Error struct {
	Reason  ErrorReason
	Method  string
	Message string
}

// New returns a new ErrorType struct
func New(t ErrorReason, method string, msg string) Error {
	return Error{Reason: t, Method: method, Message: msg}
}

func (e Error) Error() string {
	return fmt.Sprintf("error in %s: %s", e.Method, e.Message)
}

// ReasonForError returns the ErrorReason for a given error
func ReasonForError(err error) ErrorReason {
	switch t := err.(type) {
	case Error:
		return t.Reason
	}
	return UnknownError
}

// IsAllRevisionsTainted returns true if the Reason field
// of an Error is a AllRevisionsTaintedError. Returns false otherwise.
func IsAllRevisionsTainted(err error) bool {
	if ReasonForError(err) == AllRevisionsTaintedError {
		return true
	}
	return false
}

// IsRollbackOccurred returns true if the Reason field
// of an Error is a RollbackOccurredError. Returns false otherwise.
func IsRollbackOccurred(err error) bool {
	if ReasonForError(err) == RollbackOccurredError {
		return true
	}
	return false
}
