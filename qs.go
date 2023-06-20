package main

func main() {

	cmdproc := buildCommandProcessor().
		addUpdateCmd().
		addDownloadCmd().
		addReleaseCmd().
		addGUICmd().
		addForkBranch().
		addDevBranch().
		addPr().
		addUpgrade().
		addVersion()
	cmdproc.Execute()
}
