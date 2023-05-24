package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	cobra "github.com/spf13/cobra"
	qc "github.com/untillpro/gochips"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

const (
	maxDevBranchName = 50
	maxFileSize      = 100000
	maxFileQty       = 200

	utilityName = "qs"                //root command name
	utilityDesc = "Quick git wrapper" //root command description
	msymbol     = "-"
	devider     = "\n------------------------------------------"

	pushParam        = "u"
	pushParamDesc    = "Upload sources to repo"
	ignorehook       = "--ignore-hook"
	pushConfirm      = "\n*** Changes shown above will be uploaded to repository"
	pushFail         = "Ok, see you"
	pushYes          = "y"
	pushMessageWord  = "message"
	pushMessageParam = "m"
	pushMsgComment   = `Use the given string as the commit message. If multiple -m options are given
 their values are concatenated as separate paragraphs`

	delBranchConfirm      = "\n*** Branches shown above will be deleted from your forked repository, 'y': agree>"
	delBranchNothing      = "\n*** There are no remote branches to delete."
	delLocalBranchConfirm = "\n*** Branches shown above are unused local branches. Delete them all? 'y': agree>"

	pullParam     = "d"
	pullParamDesc = "Download sources from repo"

	releaseParam     = "r"
	releaseParamDesc = "Create a release"

	guiParam     = "g"
	guiParamDesc = "Show GUI"

	verboseWord  = "verbose"
	verboseParam = "v"
	verboseDesc  = "verbose output"

	forkParam     = "fork"
	forkParamDesc = "Fork original repo"

	devParam               = "dev"
	devDelParam            = "d"
	devDelParamFull        = "delete"
	ignorehookDelParam     = "i"
	ignorehookDelParamFull = "ignore-hook"
	prdraftParam           = "d"
	prdraftParamFull       = "draft"

	prParam        = "pr"
	prParamDesc    = "Make pull request"
	prMergeParam   = "merge"
	errMsgPRUnkown = "Unkown pr arguments"
	prConfirm      = "Pull request with title '$prname' will be created :Continue(y/n)?"

	devDelMsgComment        = "Deletes all merged branches from forked repository"
	devIgnoreHookMsgComment = "Ignore creating local hook"
	prdraftMsgComment       = "Create draft of pull request"
	devParamDesc            = "Create developer branch"
	devConfirm              = "Dev branch '$reponame' will be created. Continue(y/n)? "
	errMsgModFiles          = "You have modified files. Please first commit & push them."

	confMsgModFiles1      = "You have modified files: "
	confMsgModFiles2      = "All will be kept not commted. Continue(y/n)?"
	errMsgPRNotesNotFound = "Comments for Pull request not found. Please add comments manually:"
)

var verbose bool

func globalConfig() {
	qc.IsVerbose = verbose
}

type commandProcessor struct {
	cfgStatus vcs.CfgStatus
	rootcmd   *cobra.Command
}

// BuildCommandProcessor s.e.
func buildCommandProcessor() *commandProcessor {
	cp := commandProcessor{}
	return cp.setRootCmd()
}

func (cp *commandProcessor) setRootCmd() *commandProcessor {
	cp.rootcmd = &cobra.Command{
		Use:   utilityName,
		Short: utilityDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Status(cp.cfgStatus)
		},
	}
	cp.rootcmd.PersistentFlags().BoolVarP(&verbose, verboseWord, verboseParam, false, verboseDesc)
	return cp
}

func (cp *commandProcessor) addUpdateCmd() *commandProcessor {
	var cfgUpload vcs.CfgUpload
	var response string

	var uploadCmd = &cobra.Command{
		Use:   pushParam,
		Short: pushParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Status(cp.cfgStatus)

			files := git.GetFilesForCommit()
			if len(files) == 0 {
				fmt.Println("There is nothing to commit")
				return
			}

			params := []string{}
			params = append(params, cfgUpload.Message...)

			bNeedConfirmCommitComment := false
			if len(params) == 1 {
				if strings.Compare(git.PushDefaultMsg, params[0]) == 0 {
					branch, _ := getBranchName(true, args...)
					if len(branch) > 3 {
						cfgUpload.Message = []string{branch}
					}
					isMainOrg := git.IsBranchInMain()
					if isMainOrg {
						fmt.Println("This is not user fork")
					}
					curBranch := git.GetCurrentBranchName()
					isMainBranch := (curBranch == "main") || (curBranch == "master")
					if isMainOrg || isMainBranch {
						bNeedConfirmCommitComment = true
						cmtmsg := strings.TrimSpace(cfgUpload.Message[0])
						if strings.Compare(git.PushDefaultMsg, cmtmsg) == 0 {
							if isMainBranch {
								fmt.Println("You are in branch:", curBranch)
							} else {
								fmt.Println("You are not in Fork")
							}
							fmt.Println("Empty commit. Please enter commit manually:")
							scanner := bufio.NewScanner(os.Stdin)
							scanner.Scan()
							prcommit := scanner.Text()
							prcommit = strings.TrimSpace(prcommit)
							if len(prcommit) < 5 {
								fmt.Println("----  Too short comment not allowed! ---")
								return
							}
							cfgUpload.Message[0] = prcommit
						}
					} else {
						cfgUpload.Message = []string{"misc"}
					}
				}
			}
			if len(args) > 0 {
				if args[0] == "i" {
					git.Upload(cfgUpload)
					return
				}
			}
			if !bNeedConfirmCommitComment {
				git.Upload(cfgUpload)
				return
			}
			pushConfirm := pushConfirm + " with comment: \n\n'" + cfgUpload.Message[0] + "'\n\n'y': agree, 'g': show GUI >"
			fmt.Print(pushConfirm)
			fmt.Scanln(&response)
			switch response {
			case pushYes:
				git.Upload(cfgUpload)
			case guiParam:
				git.Gui()
			default:
				fmt.Print(pushFail)
			}

		},
	}

	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{git.PushDefaultMsg}, pushMsgComment)
	cp.rootcmd.AddCommand(uploadCmd)
	return cp
}

func (cp *commandProcessor) addDownloadCmd() *commandProcessor {
	var cfg vcs.CfgDownload
	var cmd = &cobra.Command{
		Use:   pullParam,
		Short: pullParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Download(cfg)
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addReleaseCmd() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   releaseParam,
		Short: releaseParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Release()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addGUICmd() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   guiParam,
		Short: guiParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Gui()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) Execute() {
	if cp.rootcmd == nil {
		return
	}
	if len(cp.rootcmd.Commands()) == 0 {
		return
	}
	err := cp.rootcmd.Execute()

	if err != nil {
		fmt.Println(err)
	}
}

func notCommitedRefused() bool {
	s, fileExists := git.ChangedFilesExist()
	if !fileExists {
		return false
	}
	fmt.Println(confMsgModFiles1)
	fmt.Println("------   " + s)
	fmt.Print(confMsgModFiles2)
	var response string
	fmt.Scanln(&response)
	return response != pushYes
}

func (cp *commandProcessor) addPr() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   prParam,
		Short: prParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()

			var prurl string
			bDirectPR := true
			if len(args) > 0 {
				if args[0] != prMergeParam {
					fmt.Println(errMsgPRUnkown)
					return
				}
				if len(args) > 1 {
					prurl = args[1]
				}
				bDirectPR = false
			}

			var err error
			if bDirectPR {

				if _, ok := git.ChangedFilesExist(); ok {
					fmt.Println(errMsgModFiles)
					return
				}

				notes, ok := git.GetNotes()
				if !ok {
					// Ask PR title
					fmt.Println(errMsgPRNotesNotFound)
					scanner := bufio.NewScanner(os.Stdin)
					scanner.Scan()
					prnotes := scanner.Text()
					prnotes = strings.TrimSpace(prnotes)
					notes = append(notes, prnotes)
				}
				var response string
				strnotes, _ := git.GetNoteAndURL(notes)
				if len(strnotes) > 0 {
					needDraft := false
					if cmd.Flag(prdraftParamFull).Value.String() == "true" {
						needDraft = true
					}

					prMsg := strings.ReplaceAll(prConfirm, "$prname", strnotes)
					fmt.Print(prMsg)
					fmt.Scanln(&response)
					switch response {
					case pushYes:
						err = git.MakePR(notes, needDraft)
					default:
						fmt.Print(pushFail)
					}
				}
			} else {
				err = git.MakePRMerge(prurl)
			}
			if err != nil {
				fmt.Println(err)
				return
			}
		},
	}
	cmd.Flags().BoolP(prdraftParamFull, prdraftParam, false, prdraftMsgComment)
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addDevBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   devParam,
		Short: devParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			// qs dev -d is running
			if cmd.Flag(devDelParamFull).Value.String() == "true" {
				cp.deleteBranches()
				return
			}
			var needAskHook bool = true
			if cmd.Flag(ignorehookDelParamFull).Value.String() == "true" {
				needAskHook = false
			}
			// qs dev is running
			var branch string
			var notes []string
			var response string

			if len(args) == 0 {
				clipargs := strings.TrimSpace(getArgStringFromClipboard())
				args = append(args, clipargs)
			}
			issueNum, ok := argContainsIssueLink(args...)
			if ok {
				fmt.Println("Dev branch for issue #" + strconv.Itoa(issueNum) + " will be created. Agree?(y/n)")
				fmt.Scanln(&response)
				if response == pushYes {
					// Remote developer branch, linked to issue is created
					branch, notes = git.DevIssue(issueNum)
				}
			} else {
				branch, notes = getBranchName(false, args...)

				devMsg := strings.ReplaceAll(devConfirm, "$reponame", branch)
				fmt.Print(devMsg)
				fmt.Scanln(&response)
			}
			remoteURL := git.GetRemoteUpstreamURL()
			switch response {
			case pushYes:
				// Remote developer branch, linked to issue is created
				if len(remoteURL) == 0 {
					git.DevShort(branch, notes)
				} else {
					git.Dev(branch, notes)
				}
			default:
				fmt.Print(pushFail)
			}

			// Create pre-commit hook to control commiting file size
			if needAskHook {
				setPreCommitHook()
			}
		},
	}
	cmd.Flags().BoolP(devDelParamFull, devDelParam, false, devDelMsgComment)
	cp.rootcmd.AddCommand(cmd)
	cmd.Flags().BoolP(ignorehookDelParamFull, ignorehookDelParam, false, devIgnoreHookMsgComment)
	cp.rootcmd.AddCommand(cmd)

	return cp
}

func (cp *commandProcessor) addForkBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   forkParam,
		Short: forkParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()

			if notCommitedRefused() {
				return
			}

			repo, err := git.Fork()
			if err != nil {
				fmt.Println(err)
				return
			}
			git.MakeUpstream(repo)
			git.PopStashedFiles()
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func getTaskIDFromURL(url string) string {
	var entry string
	str := strings.Split(url, "/")
	if len(str) > 0 {
		entry = str[len(str)-1]
	}
	entry = strings.ReplaceAll(entry, "#", "")
	entry = strings.ReplaceAll(entry, "!", "")
	return strings.TrimSpace(entry)
}

func argContainsIssueLink(args ...string) (IssueNum int, ok bool) {
	ok = false
	if len(args) != 1 {
		return
	}
	url := args[0]
	parentrepo := git.GetParentRepoName()
	if strings.Contains(url, parentrepo+"/issues") {
		segments := strings.Split(url, "/")
		strIssueNum := segments[len(segments)-1]
		i, err := strconv.Atoi(strIssueNum)
		if err != nil {
			return
		}
		return i, true
	}
	return
}

func getBranchName(ignoreEmptyArg bool, args ...string) (branch string, comments []string) {

	args = clearEmptyArgs(args)
	if len(args) == 0 {
		if ignoreEmptyArg {
			return "", []string{}
		}
		fmt.Println("Need branch name for dev")
		os.Exit(1)
	}

	newargs := splitQuotedArgs(args...)
	comments = newargs
	for i, arg := range newargs {
		arg = strings.TrimSpace(arg)
		if i == 0 {
			branch = arg
			continue
		}
		if i == len(newargs)-1 {
			// Retrieve taskID from url and add it first to branch name
			url := arg
			topicid := getTaskIDFromURL(url)
			if topicid == arg {
				branch = branch + msymbol + topicid
			} else {
				branch = topicid + msymbol + branch
			}
			break
		}
		branch = branch + "-" + arg
	}
	branch = cleanArgfromSpecSymbols(branch)
	return branch, comments
}

func clearEmptyArgs(args []string) (newargs []string) {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if len(arg) > 0 {
			newargs = append(newargs, arg)
		}
	}
	return
}

func splitQuotedArgs(args ...string) []string {
	var newargs []string
	for _, arg := range args {
		subargs := strings.Split(arg, " ")
		if len(subargs) == 0 {
			continue
		}
		for _, a := range subargs {
			if len(a) > 0 {
				newargs = append(newargs, a)
			}
		}
	}
	return newargs
}

func getArgStringFromClipboard() string {
	arg, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	args := strings.Split(arg, "\n")
	var newarg string
	for _, str := range args {
		newarg += str
		newarg += " "
	}
	return newarg
}

func cleanArgfromSpecSymbols(arg string) string {
	var symbol string
	replaceToMinus := []string{" ", ",", ";", ".", ":", "?", "!"}
	for _, symbol = range replaceToMinus {
		arg = strings.ReplaceAll(arg, symbol, "-")
	}
	replaceToNone := []string{"&", "$", "@", "%", "/", "\\", "(", ")", "{", "}", "[", "]", "<", ">", "'", "\""}
	for _, symbol = range replaceToNone {
		arg = strings.ReplaceAll(arg, symbol, "")
	}
	for string(arg[len(arg)-1]) == msymbol {
		arg = arg[:len(arg)-1]
	}
	for string(arg[0]) == msymbol {
		arg = arg[1:]
	}

	arg = deleteDupMinus(arg)
	if len(arg) > maxDevBranchName {
		arg = arg[:maxDevBranchName]
	}
	return arg
}

func deleteDupMinus(str string) string {
	var buf bytes.Buffer
	var pc rune
	for _, c := range str {
		if pc == c && string(c) == msymbol {
			continue
		}
		pc = c
		buf.WriteRune(c)
	}
	return buf.String()
}

func (cp *commandProcessor) deleteBranches() {
	lst, err := git.GetMergedBranchList()
	if err != nil {
		fmt.Println(err)
		return
	}

	var response string
	if len(lst) == 0 {
		fmt.Print(delBranchNothing)
	} else {
		fmt.Print(devider)
		for _, l := range lst {
			fmt.Print("\n" + l)
		}
		fmt.Print(devider)

		fmt.Print(delBranchConfirm)
		fmt.Scanln(&response)
		switch response {
		case pushYes:
			git.DeleteBranchesRemote(lst)
		default:
			fmt.Print(pushFail)
		}
	}
	fmt.Print("\nChecking if unused local branches exist...")
	var strs *[]string = git.GetGoneBranchesLocal()
	if len(*strs) == 0 {
		fmt.Print("\n***There no unused local branches.")
		return
	}
	fmt.Print(devider)
	for _, str := range *strs {
		if strings.TrimSpace(str) != "" {
			fmt.Print("\n" + str)
		}
	}
	fmt.Print(devider)
	fmt.Print(delLocalBranchConfirm)
	fmt.Scanln(&response)
	switch response {
	case pushYes:
		git.DeleteBranchesLocal(strs)
	default:
		fmt.Print(pushFail)
	}
}

func setPreCommitHook() {
	var response string
	if git.LocalPreCommitHookExist() {
		return
	}

	fmt.Print("\nGit pre-commit hook, preventing commit large files does not exist.\nDo you want to set hook(y/n)?")
	fmt.Scanln(&response)
	switch response {
	case pushYes:
		git.SetLocalPreCommitHook()
	default:
		return
	}
}
