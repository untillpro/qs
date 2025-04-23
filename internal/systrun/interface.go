package systrun

type ISystemTest interface {
	// AddCase adds a test case to the system test
	AddCase(tc TestCase)
	// Run executes all test cases in the system test
	Run() error
	// GetClonePath returns the path to the cloned repository
	GetClonePath() string
}
