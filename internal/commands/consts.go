package commands

var verbose bool

const (
	maxDevBranchName = 50
	msymbol          = "-"
	devider          = "\n------------------------------------------"

	pushConfirm = "\n*** Changes shown above will be uploaded to repository(they are merged)"
	pushFail    = "Ok, see you"
	pushYes     = "y"

	delBranchConfirm      = "\n*** Branches shown above will be deleted from your forked repository, 'y': agree>"
	delBranchNothing      = "\n*** There are no remote branches to delete."
	delLocalBranchConfirm = "\n*** Branches shown above are unused local branches. Delete them all? 'y': agree>"
	delLocalBranchNothing = "\n*** There no unused local branches."

	guiParam               = "g"
	devDelParamFull        = "delete"
	ignorehookDelParamFull = "ignore-hook"
	prdraftParamFull       = "draft"
	noForkParamFull        = "no-fork"

	prMergeParam   = "merge"
	errMsgPRUnkown = "Unknown pr arguments"
	prConfirm      = "Pull request with title '$prname' will be created. Continue(y/n)?"

	devConfirm     = "Dev branch '$reponame' will be created. Continue(y/n)? "
	errMsgModFiles = "You have modified files. Please first commit & push them."

	confMsgModFiles1      = "You have modified files: "
	confMsgModFiles2      = "All will be kept not commted. Continue(y/n)?"
	errMsgPRNotesNotFound = "Comments for Pull request not found. Please add comments manually:"

	trueStr  = "true"
	oneSpace = " "
)
