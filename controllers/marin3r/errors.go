package controllers

import "fmt"

const (
	// RevisionTaintedError indicates that a revision
	// cannot be publisehd
	RevisionTaintedError ControllerErrorType = "RevisionTainted"

	// AllRevisionsTaintedError indicates that there are no
	// revisisions suitable for publishing as they are all tainted
	AllRevisionsTaintedError ControllerErrorType = "AllRevisionsTainted"

	// UnknownError is used for non specific errors that don't
	// require special treatment or are yet unknown
	UnknownError ControllerErrorType = "Unknown"
)

type ControllerErrorType string

// CacheError custom error types for envoyconfig controller
type ControllerError struct {
	ErrorType     ControllerErrorType
	ReconcileTask string
	Message       string
}

// NewCacheError returns a new cacheErrorType object
func NewControllerError(t ControllerErrorType, rt string, msg string) ControllerError {
	return ControllerError{ErrorType: t, ReconcileTask: rt, Message: msg}
}

func (e ControllerError) Error() string {
	return fmt.Sprintf("[%s] %s", e.ReconcileTask, e.Message)
}
