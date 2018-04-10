package main

// Detect detects VCS in current folder
func Detect() IVCS {
	return NewGit()
}
