package errors

import "fmt"

const (
	// UnknownError is used for non specific errors that don't
	// require special treatment or are yet unknown
	UnknownError ErrorReason = "Unknown"

	// MultipleMatchesForFilterError means that several revisions
	// match the provided filters when only one should
	MultipleMatchesForFilterError ErrorReason = "MultipleMatchesForFilter"

	// NoMatchesForFilterError means that no revision matches the provided filters
	NoMatchesForFilterError ErrorReason = "NoMatchesForFilter"
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

// IsNoMatchesForFilter returns true if the Reason field
// of an Error is a NoMatchesForFilterError. Returns false otherwise.
func IsNoMatchesForFilter(err error) bool {
	if ReasonForError(err) == NoMatchesForFilterError {
		return true
	}
	return false
}

// IsMultipleMatchesForFilter returns true if the Reason field of
// an Error is a MultipleRevisionsForFilterError. Returns false otherwise
func IsMultipleMatchesForFilter(err error) bool {
	if ReasonForError(err) == MultipleMatchesForFilterError {
		return true
	}
	return false
}
