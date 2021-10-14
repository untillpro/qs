package main

func main() {

	cmdproc := BuildCommandProcessor().
		addUpdateCmd().
		addDownloadCmd().
		addReleaseCmd().
		addGUICmd().
		addForkBranch().
		addDevBranch()
	cmdproc.Execute()
}
