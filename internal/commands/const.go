package commands

var Verbose bool

const (
	msymbol          = "-"
	devider          = "\n------------------------------------------"

	msgOkSeeYou = "Ok, see you"
	pushYes     = "y"

	delBranchConfirm      = "\n*** Branches shown above will be deleted from your forked repository, 'y': agree>"
	delBranchNothing      = "\n*** There are no remote branches to delete."
	delLocalBranchConfirm = "\n*** Branches shown above are unused local branches. Delete them all? 'y': agree>"
	delLocalBranchNothing = "\n*** There no unused local branches."

	devDelParamFull = "delete"
	noForkParamFull = "no-fork"

	devConfirm = "Dev branch '$reponame' will be created. Continue(y/n)? "

	confMsgModFiles1 = "You have modified files: "
	confMsgModFiles2 = "All will be kept not commted. Continue(y/n)?"

	trueStr               = "true"
	oneSpace              = " "
	EnvSkipQsVersionCheck = "QS_SKIP_QS_VERSION_CHECK"
)

const (
	CommandNameFork    = "fork"
	CommandNameDev     = "dev"
	CommandNameUpgrade = "upgrade"
	CommandNameVersion = "version"
	CommandNamePR      = "pr"
	CommandNameD       = "d"
	CommandNameU       = "u"
	CommandNameR       = "r"
	CommandNameG       = "g"
)
