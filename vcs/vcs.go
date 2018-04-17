package vcs

func detect() IVCS {
	return NewVCSGit()
}

// Upload sources
func Upload(uploadCmdMessage []string) {
	detect().Upload(uploadCmdMessage)
}

// Status detects VCS and runs its status
func Status() {
	detect().Status()
}
