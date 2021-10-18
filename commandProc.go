package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	cobra "github.com/spf13/cobra"
	qc "github.com/untillpro/gochips"
	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

const (
	utilityName = "qs"                //root command name
	utilityDesc = "Quick git wrapper" //root command description
	msymbol     = "-"

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

	devParam      = "dev"
	devParamDesc  = "Create developer branch"
	devConfirm    = "Dev branch '$reponame' will be created. Yes/No? "
	devNeedToFork = "You are in $org/$repo repo\nExecute 'qs fork' first"
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
func BuildCommandProcessor() *commandProcessor {
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
		Use: pushParam,

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
	var cmd = &cobra.Command{
		Use:   devParam,
		Short: devParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
			remoteURL := strings.TrimSpace(git.GetRemoteUpstreamURL())
			repo, org := git.GetRepoAndOrgName()
			if len(remoteURL) == 0 {
				if git.IsMainOrg() {
					errMsg := strings.ReplaceAll(devNeedToFork, "$repo", repo)
					errMsg = strings.ReplaceAll(errMsg, "$org", org)
					fmt.Println(errMsg)
					return
				}
				git.MakeUpstream(repo)
			}
			branch := getBranchName(args...)
			devMsg := strings.ReplaceAll(devConfirm, "$reponame", branch)
			fmt.Print(devMsg)
			var response string
			fmt.Scanln(&response)
			switch response {
			case pushYes:
				git.Dev(branch)
			default:
				fmt.Print(pushFail)
			}
		},
	}
	cp.rootcmd.AddCommand(cmd)
	return cp
}

func (cp *commandProcessor) addForkBranch() *commandProcessor {
	var cmd = &cobra.Command{
		Use:   forkParam,
		Short: forkParamDesc,
		Run: func(cmd *cobra.Command, args []string) {
			globalConfig()
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
	if string(entry[0]) == "#" {
		entry = entry[1:]
	}
	if string(entry[0]) == "!" {
		entry = entry[1:]
	}
	return entry
}

func getBranchName(args ...string) string {

	if len(args) == 0 {
		fmt.Println("Need branch name for dev")
		os.Exit(1)
	}

	var arg string
	for i, ar := range args {
		if i == len(args)-1 {
			// Retrieve taskID from url and add it first to branch name
			url := ar
			topicid := getTaskIDFromURL(url)
			if strings.TrimSpace(topicid) == strings.TrimSpace(ar) {
				arg = arg + msymbol + topicid
			} else {
				arg = topicid + msymbol + arg
			}
			break
		}
		if i == 0 {
			arg = strings.TrimSpace(ar)
		} else {
			arg = arg + "-" + strings.TrimSpace(ar)
		}
	}
	// Clean branch name from bad symbols
	var symbol string
	replaceToMinus := []string{" ", ",", ";", ".", ":", "?", "!"}
	for _, symbol = range replaceToMinus {
		arg = strings.ReplaceAll(arg, symbol, "-")
	}
	replaceToNone := []string{"/", "\\", "(", ")", "{", "}", "[", "]", "'", "\""}
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
