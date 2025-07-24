package cmdproc

const (
	msgOkSeeYou       = "Ok, see you"
	pushParamDesc    = "Upload sources to repo"
	pushMessageWord  = "message"
	pushMessageParam = "m"
	pushMsgComment   = `Use the given string as the commit message. If multiple -m options are given
 their values are concatenated as separate paragraphs`

	pullParamDesc = "Download sources from repo"

	releaseParamDesc = "Create a release"

	guiParamDesc = "Show GUI"

	forkParamDesc = "Fork original repo"

	devDelParam            = "d"
	devDelParamFull        = "delete"
	ignorehookDelParam     = "i"
	ignorehookDelParamFull = "ignore-hook"
	prdraftParam           = "d"
	prdraftParamFull       = "draft"
	noForkParam            = "n"
	noForkParamFull        = "no-fork"

	prParamDesc = "Make pull request"

	devDelMsgComment        = "Deletes all merged branches from forked repository"
	devIgnoreHookMsgComment = "Ignore creating local hook"
	devNoForkMsgComment     = "Allows to create branch in main repo"
	prdraftMsgComment       = "Create draft of pull request"
	devParamDesc            = "Create developer branch"
	upgradeParamDesc        = "Print command to upgrade qs"
	versionParamDesc        = "Print qs version"
)
