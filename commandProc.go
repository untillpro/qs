package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	cobra "github.com/spf13/cobra"
	"github.com/untillpro/gochips"
	qc "github.com/untillpro/gochips"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

const (
	maxDevBranchName = 50
	utilityName      = "qs"                //root command name
	utilityDesc      = "Quick git wrapper" //root command description
	msymbol          = "-"

	pushParam        = "u"
	pushParamDesc    = "Upload sources to repo"
	pushConfirm      = "\n*** Changes shown above will be uploaded to repository, 'y': agree, 'g': show GUI >"
	pushFail         = "Ok, see you"
	pushYes          = "y"
	pushMessageWord  = "message"
	pushMessageParam = "m"
	pushDefaultMsg   = "misc"
	pushMsgComment   = `Use the given string as the commit message. If multiple -m options are given
 their values are concatenated as separate paragraphs`

	delBranchConfirm = "\n*** Branches shown above will be deleted from your forked repository, 'y': agree>"
	delBranchNothing = "\n*** There are no branches to delete>"

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

	devParam        = "dev"
	devDelParam     = "d"
	devDelParamFull = "delete"

	devDelMsgComment = "Deletes all merged branches from forked repository"
	devParamDesc     = "Create developer branch"
	devConfirm       = "Dev branch '$reponame' will be created. Yes/No? "
	devNeedToFork    = "You are in $org/$repo repo\nExecute 'qs fork' first"

	errMsgModFiles = "You have modified files. Please commit all changes first!"
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
	var uploadCmd = &cobra.Command{
		Use:   pushParam,
		Short: pushParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			git.Status(cp.cfgStatus)
			if len(args) > 0 && args[0] == "i" {
				git.Upload(cfgUpload)
				return
			}
			fmt.Print(pushConfirm)
			var response string
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

	uploadCmd.Flags().StringSliceVarP(&cfgUpload.Message, pushMessageWord, pushMessageParam, []string{pushDefaultMsg}, pushMsgComment)
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
	cp.rootcmd.Execute()
}

func (cp *commandProcessor) addDevBranch() *commandProcessor {
	//var cfgUpload vcs.CfgUpload
	var cmd = &cobra.Command{
		Use:   devParam,
		Short: devParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			if cmd.Flag(devDelParamFull).Value.String() == "true" {
				cp.deleteBranches()
				return
			}

			if changedFilesExist() {
				fmt.Println(errMsgModFiles)
				return
			}
			remoteURL := strings.TrimSpace(git.GetRemoteUpstreamURL())
			branch := getBranchName(args...)
			fmt.Println("branch:", branch)
			devMsg := strings.ReplaceAll(devConfirm, "$reponame", branch)
			fmt.Print(devMsg)
			var response string
			fmt.Scanln(&response)
			switch response {
			case pushYes:
				if len(remoteURL) == 0 {
					git.DevShort(branch)
				} else {
					git.Dev(branch)
				}
			default:
				fmt.Print(pushFail)
			}
		},
	}
	cmd.Flags().BoolP(devDelParamFull, devDelParam, false, devDelMsgComment)
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func changedFilesExist() bool {
	stdouts, _, err := new(gochips.PipedExec).
		Command("git", "status", "-s").
		RunToStrings()
	gochips.ExitIfError(err)
	return len(strings.TrimSpace(stdouts)) > 0
}

func (cp *commandProcessor) addForkBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   forkParam,
		Short: forkParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			if changedFilesExist() {
				fmt.Println(errMsgModFiles)
				return
			}
			repo, err := git.Fork()
			if err != nil {
				fmt.Println(err)
				return
			}
			git.MakeUpstream(repo)
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

func getBranchName(args ...string) string {

	if len(args) == 0 {
		args = append(args, getArgStringFromClipboard())
	}
	if len(args) == 0 {
		fmt.Println("Need branch name for dev")
		os.Exit(1)
	}

	newargs := splitQuotedArgs(args...)
	var branch string
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
	return branch
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
	replaceToNone := []string{"&", "$", "@", "%", "/", "\\", "(", ")", "{", "}", "[", "]", "'", "\""}
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

	if len(lst) == 0 {
		fmt.Print(delBranchNothing)
		return
	}
	fmt.Println("------------------------------------------")
	for _, l := range lst {
		fmt.Println(l)
	}
	fmt.Println("------------------------------------------")

	fmt.Print(delBranchConfirm)
	var response string
	fmt.Scanln(&response)
	switch response {
	case pushYes:
		git.DeleteBranches(lst)
	default:
		fmt.Print(pushFail)
	}

}
