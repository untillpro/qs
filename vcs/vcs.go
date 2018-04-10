package vcs

func detect() IVCS {
	return NewVCSGit()
}

// Upload sources
func Upload() {
	detect().Upload()
}
