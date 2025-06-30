package systrun

import "context"

// IExpectation is an interface for expectations in system tests
type IExpectation interface {
	// Check compares the current state of the system with the expected state
	Check(ctx context.Context) error
}
