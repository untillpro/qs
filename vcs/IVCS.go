package vcs

// IVCS is a simplified interface to version control system
type IVCS interface {
	Upload()
	Download()
	Gui()
}
