package main

func main() {

	cmdproc := buildCommandProcessor().
		addUpdateCmd().
		addDownloadCmd().
		addReleaseCmd().
		addGUICmd().
		addForkBranch().
		addDevBranch()
	cmdproc.Execute()
}
