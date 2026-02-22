package cmdproc

const (
	msgOkSeeYou      = "Ok, see you"
	pushParamDesc    = "Upload sources to repo"
	pushMessageWord  = "message"
	pushMessageParam = "m"
	pushMsgComment   = "Use the given string as the commit message"

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

	prParamDesc = "Make pull request"

	devDelMsgComment        = "Deletes all merged branches from forked repository"
	devIgnoreHookMsgComment = "Ignore creating local hook"
	prdraftMsgComment       = "Create draft of pull request"
	devParamDesc            = "Create developer branch"
	upgradeParamDesc        = "Print command to upgrade qs"
	versionParamDesc        = "Print qs version"
)

var requiredCommands = []string{"grep", "sed", "jq", "gawk", "wc", "curl", "chmod"}