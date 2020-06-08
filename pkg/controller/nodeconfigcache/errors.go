package nodeconfigcache

import "fmt"

const (
	// RevisionTaintedError indicates that a revision
	// cannot be publisehd
	RevisionTaintedError cacheErrorType = "RevisionTainted"

	// AllRevisionsTaintedError indicates that there are no
	// revisisions suitable for publishing as they are all tainted
	AllRevisionsTaintedError cacheErrorType = "AllRevisionsTainted"

	// UnknownError is used for non specific errors that don't
	// require special treatment or are yet unkown
	UnknownError cacheErrorType = "Unknown"
)

type cacheErrorType string

// CacheError custom error types for nodeconfigcache controller
type cacheError struct {
	ErrorType     cacheErrorType
	ReconcileTask string
	Message       string
}

// NewCacheError returns a new cacheErrorType object
func newCacheError(t cacheErrorType, rt string, msg string) cacheError {
	return cacheError{ErrorType: t, ReconcileTask: rt, Message: msg}
}

func (e cacheError) Error() string {
	return fmt.Sprintf("[%s] %s", e.ReconcileTask, e.Message)
}
