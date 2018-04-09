package main

// IVCS is an simplified interface to version control system
type IVCS interface {
	upload()
	download()
	gui()
}
