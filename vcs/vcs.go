package vcs

func detect() IVCS {
	return NewVCSGit()
}

// Upload sources
func Upload() {
	detect().Upload()
}

// Status detects VCS and runs its status
func Status() {
	detect().Status()
}
