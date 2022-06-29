package main

func main() {

	cmdproc := buildCommandProcessor().
		addUpdateCmd().
		addDownloadCmd().
		addReleaseCmd().
		addGUICmd().
		addForkBranch().
		addDevBranch().
		addPr()
	cmdproc.Execute()
}
