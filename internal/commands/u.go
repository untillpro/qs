package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/untillpro/qs/git"
	"github.com/untillpro/qs/vcs"
)

func U(cfgStatus vcs.CfgStatus, cfgUpload vcs.CfgUpload, args []string) {
	var response string

	globalConfig()
	git.Status(cfgStatus)

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
}
