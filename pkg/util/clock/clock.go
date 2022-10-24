package clock

import "time"

// Clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

type Real struct{}

// Now returns the current time
func (Real) Now() time.Time { return time.Now() }

type Test struct {
	now time.Time
}

func NewTest(t time.Time) Test { return Test{t} }

// Now returns the the current time
func (tc Test) Now() time.Time { return tc.now }
