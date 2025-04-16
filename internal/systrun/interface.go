package systrun

type ISystemTest interface {
	AddCase(tc TestCase)
	Run() error
	GetClonePath() string
}
