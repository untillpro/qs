package commands

var Verbose bool

const (
	msgOkSeeYou = "Ok, see you"
	pushYes     = "y"

	devDelParamFull = "delete"
	noForkParamFull = "no-fork"

	devConfirm = "Dev branch '$reponame' will be created. Continue(y/n)? "

	confMsgModFiles1 = "You have modified files: "
	confMsgModFiles2 = "All will be kept not commted. Continue(y/n)?"

	trueStr                 = "true"
	oneSpace                = " "
	EnvSkipQsVersionCheck   = "QS_SKIP_QS_VERSION_CHECK"
	minimumCommitMessageLen = 8
	fetch                   = "fetch"
	origin                  = "origin"
	git                     = "git"
	refsNotes               = "refs/notes/*:refs/notes/*"
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
