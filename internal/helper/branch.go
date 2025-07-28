package helper

import (
	"errors"
	"strings"

	"github.com/untillpro/qs/internal/notes"
)

func GetBranchName(ignoreEmptyArg bool, args ...string) (branch string, comments []string, err error) {

	args = clearEmptyArgs(args)
	if len(args) == 0 {
		if ignoreEmptyArg {
			return "", []string{}, nil
		}

		return "", []string{}, errors.New("Need branch name for dev")
	}

	newArgs := splitQuotedArgs(args...)
	comments = make([]string, 0, len(newArgs)+1) // 1 for json notes
	comments = append(comments, newArgs...)
	for i, arg := range newArgs {
		arg = strings.TrimSpace(arg)
		if i == 0 {
			branch = arg
			continue
		}
		if i == len(newArgs)-1 {
			// Retrieve taskID from url and add it first to branch name
			url := arg
			topicID := GetTaskIDFromURL(url)
			if topicID == arg {
				branch = branch + msymbol + topicID
			} else {
				branch = topicID + msymbol + branch
			}
			break
		}
		branch = branch + "-" + arg
	}
	branch = CleanArgFromSpecSymbols(branch)
	// Prepare new notes
	notesObj, err := notes.Serialize("", "", notes.BranchTypeDev)
	if err != nil {
		return "", []string{}, err
	}
	comments = append(comments, notesObj)

	return branch, comments, nil
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
		subargs := strings.Split(arg, oneSpace)
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

func GetTaskIDFromURL(url string) string {
	var entry string
	str := strings.Split(url, "/")
	if len(str) > 0 {
		entry = str[len(str)-1]
	}
	entry = strings.ReplaceAll(entry, "#", "")
	entry = strings.ReplaceAll(entry, "!", "")
	return strings.TrimSpace(entry)
}
