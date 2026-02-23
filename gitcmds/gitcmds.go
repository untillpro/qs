package gitcmds

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	goGitPkg "github.com/go-git/go-git/v5"
	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"github.com/untillpro/qs/internal/jira"
	notesPkg "github.com/untillpro/qs/internal/notes"
	"github.com/untillpro/qs/utils"
)

const (
	mimm              = "-m"
	slash             = "/"
	caret             = "\n"
	caretByte         = '\n'
	git               = "git"
	push              = "push"
	pull              = "pull"
	fetch             = "fetch"
	branch            = "branch"
	origin            = "origin"
	originSlash       = "origin/"
	httppref          = "https"
	pushYes           = "y"
	MsgPreCommitError = "Attempt to commit too"
	MsgCommitForNotes = "Commit for keeping notes in branch"
	oneSpace          = " "
	err128            = "128"

	repoNotFound            = "git repo name not found"
	userNotFound            = "git user name not found"
	ErrAlreadyForkedMsg     = "you are in fork already\nExecute 'qs dev [branch name]' to create dev branch"
	ErrMsgPRNotesImpossible = "pull request without comments is impossible"
	DefaultCommitMessage    = "wip"

	IssuePRTtilePrefix = "Resolves issue"
	IssueSign          = "Resolves #"

	minPRTitleLength              = 8
	minRepoNameLength             = 4
	bashFilePerm      os.FileMode = 0644

	issuelineLength  = 5
	issuelinePosOrg  = 4
	issuelinePosRepo = 3
)

func CheckIfGitRepo(wd string) (bool, error) {
	_, err := GitStatus(wd)
	if err != nil {
		if strings.Contains(err.Error(), err128) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GitStatus(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("git", "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			err = fmt.Errorf("%s: %w", strings.TrimSpace(stderr), err)
		}
	}

	return stdout, err
}

// ChangedFilesExist s.e.
func ChangedFilesExist(wd string) (string, bool, error) {
	files, err := GitStatus(wd)
	uncommitedFiles := strings.TrimSpace(files)

	return uncommitedFiles, len(uncommitedFiles) > 0, err
}

// Stash stashes uncommitted changes
func Stash(wd string) error {
	_, stderr, err := new(exec.PipedExec).
		Command("git", "stash").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git stash failed: %w - %s", err, stderr)
	}

	return nil
}

// Unstash pops the latest stash if stash entries exist
func Unstash(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "stash", "list").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to check stash entries: %w", err)
	}

	if len(strings.TrimSpace(stdout)) == 0 {
		return nil
	}

	_, stderr, err = new(exec.PipedExec).
		Command(git, "stash", "pop").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git stash pop failed: %w", err)
	}

	return nil
}

func HaveUncommittedChanges(wd string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "status", "--porcelain").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return false, errors.New(stderr)
		}

		return false, fmt.Errorf("failed to check if there are uncommitted changes: %w", err)
	}

	return len(stdout) > 0, nil
}

func CheckoutOnBranch(wd, branchName string) error {
	_, stderr, err := new(exec.PipedExec).
		Command(git, "checkout", branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to checkout on %s: %w", branchName, err)
	}

	return nil
}

func GetBranchesWithRemoteTracking(wd, remoteName string) ([]string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "branch", "-vv").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return nil, errors.New(stderr)
		}

		return nil, fmt.Errorf("failed to get list of rtBranchLines with remote tracking: %w", err)
	}
	// No remote tracking branches found
	if len(stdout) == 0 {
		return nil, nil
	}

	mainBranchName, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	// remote tracking branch suffixes:
	// - gone - branch is deleted on remote
	// - ahead \d+ - branch is ahead of remote by d commits, could be ahead by merge commit
	rtBranchPattern := fmt.Sprintf(`\[%s/([A-Za-z0-9\-_\.\/]+):? ?(gone)?(ahead \d+)?\]`, remoteName)
	re := regexp.MustCompile(rtBranchPattern)

	rtBranchLines := strings.Split(strings.TrimSpace(stdout), caret)
	branchesWithRemoteTracking := make([]string, 0, len(rtBranchLines))
	for _, rtBranchLine := range rtBranchLines {
		matches := re.FindStringSubmatch(rtBranchLine)
		if matches == nil {
			continue
		}

		branchName := matches[1]
		// exclude the main branch from the list
		if branchName == mainBranchName {
			continue
		}
		branchesWithRemoteTracking = append(branchesWithRemoteTracking, branchName)
	}

	return branchesWithRemoteTracking, nil
}

// Gui shows gui
func Gui(wd string) error {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "gui").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("git gui failed: %w", err)
	}
	printLn(stdout)

	return nil
}

func getFullRepoAndOrgName(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "config", "--local", "remote.origin.url").
		WorkingDir(wd).
		Command("sed", "s/\\.git$//").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get remote origin URL: %w", err)
	}

	return strings.TrimSuffix(strings.TrimSpace(stdout), slash), nil
}

// GetRepoAndOrgName - from .git/config
func GetRepoAndOrgName(wd string) (repo string, org string, err error) {
	repoURL, err := getFullRepoAndOrgName(wd)
	if err != nil {
		return "", "", err
	}

	org, repo, _, err = ParseGitRemoteURL(repoURL)
	if err != nil {
		return "", "", err
	}

	return
}

func GetMainBranch(wd string) (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		Command("grep", "-E", "(/main|/master)([^a-zA-Z0-9]|$)").
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get main branch: %w", err)
	}
	logger.Verbose(stdout)

	// Check if the output contains "main" or "master"
	mainBranchFound := strings.Contains(stdout, "/main")
	masterBranchFound := strings.Contains(stdout, "/master")

	switch {
	case mainBranchFound && masterBranchFound:
		return "", fmt.Errorf("both main and master branches found")
	case mainBranchFound:
		return "main", nil
	case masterBranchFound:
		return "master", nil
	}

	return "", errors.New("neither main nor master branches found")
}

func MakeUpstreamForBranch(wd string, parentRepo string) error {
	_, stderr, err := new(exec.PipedExec).
		Command(git, "remote", "add", "upstream", "https://github.com/"+parentRepo).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to add upstream remote: %w", err)
	}

	return nil
}

// SyncMainBranch syncs the local main branch with upstream and origin
// Flow:
// 1. If upstream exists: Pull from upstream/main to main with rebase
// 2. Pull from origin/main to main with rebase
// 3. Push to origin/main
// In single remote mode (no upstream), only syncs with origin
func SyncMainBranch(wd, mainBranch string, upstreamExists bool) error {
	if upstreamExists {
		stdout, stderr, err := new(exec.PipedExec).
			Command(git, pull, "--rebase", "upstream", mainBranch, "--no-edit").
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			if err := showWorkaroundIfConflict(wd, mainBranch, stderr); err != nil {
				return err
			}
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to pull from upstream/%s with rebase: %w", mainBranch, err)
		}
		logger.Verbose(stdout)
	}

	// Pull from origin to MainBranch with rebase
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, pull, "--rebase", "origin", mainBranch, "--no-edit").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		if err := showWorkaroundIfConflict(wd, mainBranch, stderr); err != nil {
			return err
		}
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to pull from origin/%s: %w with rebase", mainBranch, err)
	}
	logger.Verbose(stdout)

	// Push to origin from MainBranch
	err = utils.Retry(func() error {
		var pushErr error
		stdout, stderr, pushErr = new(exec.PipedExec).
			Command(git, push, "origin", mainBranch).
			WorkingDir(wd).
			RunToStrings()
		if pushErr != nil {
			logger.Verbose(stderr)

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to push to origin/%s: %w", mainBranch, pushErr)
		}

		return nil
	})
	logger.Verbose(stdout)

	return err
}

// checkAndShowFastForwardFailure checks if stderr contains fast-forward failure message,
// and if so, displays helpful instructions and returns true
func checkAndShowFastForwardFailure(stderr, mainBranch string) bool {
	if strings.Contains(stderr, "fatal: Not possible to fast-forward") ||
		strings.Contains(stderr, "not possible to fast-forward") {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf(MsgCannotFastForward+"\n", mainBranch, mainBranch)
		fmt.Println(MsgMainBranchDiverged)
		fmt.Println("\n" + MsgToFixRunCommands)
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println(MsgGitCheckoutMain)
		fmt.Println(MsgGitFetchUpstream)
		fmt.Println(MsgGitResetHardUpstream)
		fmt.Println(MsgGitPushOriginMainForce)
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println(MsgWarningOverwriteMainBranch)
		return true
	}
	return false
}

// showWorkaroundIfConflict shows workaround instructions in case of merge conflict during rebase
func showWorkaroundIfConflict(wd, mainBranch, stderr string) error {
	if strings.Contains(stderr, "could not apply") {
		// Abort the rebase
		_, _, _ = new(exec.PipedExec).
			Command("git", "rebase", "--abort").
			WorkingDir(wd).RunToStrings()
		// Provide instructions to reset and force-push
		fmt.Printf(MsgConflictDetected+"\n\n", mainBranch, mainBranch, mainBranch)
		fmt.Println(MsgGitCheckoutMain)
		fmt.Println(MsgGitFetchUpstream)
		fmt.Println(MsgGitResetHardUpstream)
		fmt.Println(MsgGitPushOriginMainForce)
		fmt.Println()
		fmt.Println(MsgWarningOverwriteMainBranch)
		fmt.Println()

		return fmt.Errorf("unable to rebase on upstream/%s", mainBranch)
	}

	return nil
}

// GetBranchTypeByName returns branch type based on branch name
func GetBranchTypeByName(branchName string) notesPkg.BranchType {
	switch {
	case strings.HasSuffix(branchName, "-dev"):
		return notesPkg.BranchTypeDev
	case strings.HasSuffix(branchName, "-pr"):
		return notesPkg.BranchTypePr
	default:
		return notesPkg.BranchTypeUnknown
	}
}

// GetBranchType returns branch type based on notes or branch name
func GetBranchType(wd string) (string, notesPkg.BranchType, error) {
	currentBranchName, err := GetCurrentBranchName(wd)
	if err != nil {
		return "", notesPkg.BranchTypeUnknown, err
	}

	notes, _, err := GetNotes(wd, currentBranchName)
	if err != nil {
		logger.Verbose(err)
	}

	if len(notes) > 0 {
		notesObj, ok := notesPkg.Deserialize(notes)
		if !ok {
			if isOldStyledBranch(notes) {
				return currentBranchName, notesPkg.BranchTypeDev, nil
			}
		}

		if notesObj != nil {
			return currentBranchName, notesObj.BranchType, nil
		}
	}

	return currentBranchName, GetBranchTypeByName(currentBranchName), nil
}

// isOldStyledBranch checks if branch is old styled
func isOldStyledBranch(notes []string) bool {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, IssuePRTtilePrefix) || strings.Contains(s, IssueSign) {
				return true
			}
		}
	}

	return false
}

// GetParentRepoName - parent repo of forked
func GetParentRepoName(wd string) (name string, err error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return "", err
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command("gh", "api", "repos/"+org+slash+repo, "--jq", ".parent.full_name").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get parent repo name: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

// IsBranchInMain Is my branch in main org?
func IsBranchInMain(wd string) (bool, error) {
	repo, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return false, err
	}
	parent, err := GetParentRepoName(wd)

	return (parent == org+slash+repo) || (strings.TrimSpace(parent) == ""), err
}

// GetMergedBranchList returns merged user's branch list
func GetMergedBranchList(wd string) (brlist []string, err error) {
	mbrlist := []string{}
	_, org, err := GetRepoAndOrgName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetRepoAndOrgName failed: %w", err)
	}

	repo, err := GetParentRepoName(wd)
	if err != nil {
		return nil, fmt.Errorf("GetParentRepoName failed: %w", err)
	}

	stdout, _, err := new(exec.PipedExec).
		Command("gh", "pr", "list", "-L", "200", "--state", "merged", "--author", org, "--repo", repo).
		WorkingDir(wd).
		Command("gawk", "-F:", "{print $2}").
		Command("gawk", "/MERGED/{print $1}").
		RunToStrings()
	if err != nil {
		return []string{}, err
	}

	mbrlistraw := strings.Split(stdout, caret)
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	curbr, err := GetCurrentBranchName(wd)
	if err != nil {
		return nil, err
	}

	for _, mbranchstr := range mbrlistraw {
		arrstr := strings.TrimSpace(mbranchstr)
		if (strings.TrimSpace(arrstr) != "") && !strings.Contains(arrstr, curbr) && !strings.Contains(arrstr, mainBranch) {
			mbrlist = append(mbrlist, arrstr)
		}
	}
	_, _, err = new(exec.PipedExec).
		Command(git, "remote", "prune", origin).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}

	stdout, _, err = new(exec.PipedExec).
		Command(git, branch, "-r").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		return nil, err
	}
	mybrlist := strings.Split(stdout, caret)

	for _, mybranch := range mybrlist {
		mybranch := strings.TrimSpace(mybranch)
		mybranch = strings.ReplaceAll(strings.TrimSpace(mybranch), originSlash, "")
		mybranch = strings.TrimSpace(mybranch)
		bfound := false
		if !strings.Contains(mybranch, mainBranch) && !strings.Contains(mybranch, "HEAD") {
			for _, mbranch := range mbrlist {
				mbranch = strings.ReplaceAll(strings.TrimSpace(mbranch), "MERGED", "")
				mbranch = strings.TrimSpace(mbranch)
				if mybranch == mbranch {
					bfound = true
					break
				}
			}
		}
		if bfound {
			// delete branch in fork
			brlist = append(brlist, mybranch)
		}
	}

	return brlist, nil
}

// DeleteBranchesRemote delete branch list
func DeleteBranchesRemote(wd string, brs []string) error {
	if len(brs) == 0 {
		return nil
	}

	for _, br := range brs {
		err := utils.Retry(func() error {
			_, _, deleteErr := new(exec.PipedExec).
				Command(git, push, origin, ":"+br).
				WorkingDir(wd).
				RunToStrings()
			return deleteErr
		})
		if err != nil {
			return fmt.Errorf("branch %s was not deleted: %w", br, err)
		}

		fmt.Printf("Branch %s deleted\n", br)
	}

	return nil
}

func PullUpstream(wd string) error {
	mainBranch, err := GetMainBranch(wd)
	if err != nil {
		return fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}

	stdout, stderr, err := new(exec.PipedExec).
		Command(git, pull, "--ff-only", "upstream", mainBranch, "--no-edit").
		WorkingDir(wd).
		RunToStrings()

	if err != nil {
		// Check if fast-forward failed
		if checkAndShowFastForwardFailure(stderr, mainBranch) {
			return fmt.Errorf("cannot fast-forward merge upstream/%s", mainBranch)
		}

		// If upstream doesn't exist, try to create it
		parentRepoName, err := GetParentRepoName(wd)
		if err != nil {
			return fmt.Errorf("GetParentRepoName failed: %w", err)
		}

		return MakeUpstreamForBranch(wd, parentRepoName)
	}

	logger.Verbose(stdout)
	return nil
}

func GetNoteAndURL(notes []string) (note string, url string) {
	for _, s := range notes {
		s = strings.TrimSpace(s)
		if len(s) > 0 {
			if strings.Contains(s, httppref) {
				url = s
				if len(note) > 0 {
					break
				}
			} else {
				if note == "" {
					note = s
				} else {
					note = note + oneSpace + s
				}
				if strings.Contains(strings.ToLower(s), strings.ToLower(IssuePRTtilePrefix)) {
					break
				}
			}
		}
	}
	return note, url
}

// GetFilesForCommit shows list of file names, ready for commit
func GetFilesForCommit(wd string) []string {
	stdout, _, _ := new(exec.PipedExec).
		Command(git, "status", "-s").
		WorkingDir(wd).
		RunToStrings()
	ss := strings.Split(stdout, caret)
	var strs []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			strs = append(strs, s)
		}
	}
	return strs
}

func HasRemote(wd, remoteName string) (bool, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command(git, "remote").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return false, errors.New(stderr)
		}

		return false, fmt.Errorf("failed to list git remotes: %w: %s", err, stderr)
	}

	remotes := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, remote := range remotes {
		if strings.TrimSpace(remote) == remoteName {
			return true, nil
		}
	}

	return false, nil
}

func GetCurrentBranchName(wd string) (string, error) {
	branchName, stderr, err := new(exec.PipedExec).
		Command(git, branch, "--show-current").
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return "", errors.New(stderr)
		}

		return "", fmt.Errorf("failed to get current branch name: %w", err)
	}

	return strings.TrimSpace(branchName), nil
}

func GetCurrentBranchInfo(wd string) (currentBranch, mainBranch string, isMain bool, err error) {
	currentBranch, err = GetCurrentBranchName(wd)
	if err != nil {
		return "", "", false, err
	}
	logger.Verbose("Current branch: " + currentBranch)

	mainBranch, err = GetMainBranch(wd)
	if err != nil {
		return "", "", false, fmt.Errorf(errMsgFailedToGetMainBranch, err)
	}
	logger.Verbose("Main branch: " + mainBranch)

	return currentBranch, mainBranch, strings.EqualFold(currentBranch, mainBranch), nil
}

// GetRemoteUrlByName retrieves the URL of a specified remote by its name
func GetRemoteUrlByName(wd string, remoteName string) (string, error) {
	repo, err := OpenGitRepository(wd)
	if err != nil {
		return "", err
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return "", fmt.Errorf("failed to get remotes: %w", err)
	}

	for _, remote := range remotes {
		if remote.Config().Name == remoteName {
			if len(remote.Config().URLs) > 0 {
				return remote.Config().URLs[0], nil
			}
			return "", fmt.Errorf("remote %s has no URLs configured", remoteName)
		}
	}

	return "", fmt.Errorf("remote %s not found", remoteName)
}

func printLn(stdout string) {
	if len(stdout) > 0 {
		fmt.Println(stdout)
	}
}

func GetIssueDescription(notes []string) (string, error) {
	var (
		description string
		err         error
	)

	if len(notes) == 0 {
		return "", err
	}

	notesObj, ok := notesPkg.Deserialize(notes)
	if !ok {
		return "", nil
	}

	switch {
	case len(notesObj.GithubIssueURL) > 0:
		description, err = GetGitHubIssueDescription(notesObj.GithubIssueURL)
	case len(notesObj.JiraTicketURL) > 0:
		var jiraTicketID string
		description, jiraTicketID, err = jira.GetJiraIssueTitle(notesObj.JiraTicketURL, "")
		if err != nil {
			return "", err
		}

		description = "[" + jiraTicketID + "] " + description
	default:
		// If no GitHub or Jira URL, use the description from notes (from qs dev {some text})
		description = notesObj.Description
	}
	if err != nil {
		return "", fmt.Errorf("error retrieving issue description: %w", err)
	}

	return description, nil
}

// OpenGitRepository opens a git repository at the specified directory
func OpenGitRepository(dir string) (*goGitPkg.Repository, error) {
	repo, err := goGitPkg.PlainOpenWithOptions(dir, &goGitPkg.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		if errors.Is(err, goGitPkg.ErrRepositoryNotExists) {
			return nil, errors.New("no .git directory found; please run this command inside a git repository")
		}

		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	return repo, nil
}

func RemoveBranch(wd, branchName string) error {
	// Delete branch locally
	_, stderr, err := new(exec.PipedExec).
		Command("git", "branch", "-D", branchName).
		WorkingDir(wd).
		RunToStrings()
	if err != nil {
		logger.Verbose(stderr)

		if len(stderr) > 0 {
			return errors.New(stderr)
		}

		return fmt.Errorf("failed to delete local branch %s: %w", branchName, err)
	}

	// Delete branch from origin
	return utils.Retry(func() error {
		_, stderr, err = new(exec.PipedExec).
			Command("git", "push", "origin", "--delete", branchName).
			WorkingDir(wd).
			RunToStrings()
		if err != nil {
			logger.Verbose(stderr)

			// If a branch does not exist on origin, we can ignore the error
			if strings.Contains(stderr, "remote ref does not exist") {
				return nil
			}

			if len(stderr) > 0 {
				return errors.New(stderr)
			}

			return fmt.Errorf("failed to delete remote branch %s: %w", branchName, err)
		}

		return nil
	})
}

// ParseGitRemoteURL extracts account, repository name, and token from a git remote URL.
// Handles HTTPS format (https://github.com/account/repo.git),
// and HTTPS with token (https://account:token@github.com/account/repo.git or https://oauth2:token@github.com/repo.git)
func ParseGitRemoteURL(remoteURL string) (account, repo string, token string, err error) {
	if remoteURL == "" {
		return "", "", "", errors.New("remote URL is empty")
	}

	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	// Handle HTTPS format: https://github.com/account/repo
	// or https://account:token@github.com/account/repo
	if strings.HasPrefix(remoteURL, "http") {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
		}

		// Extract token if present in the userinfo section
		if u.User != nil {
			password, hasPassword := u.User.Password()
			if hasPassword {
				token = password
			}
		}

		// Remove leading '/' if any
		path := strings.TrimPrefix(u.Path, slash)
		pathParts := strings.Split(path, slash)

		if len(pathParts) < 2 {
			return "", "", "", fmt.Errorf("invalid repository path in URL: %s", remoteURL)
		}

		// If no token was found or username is oauth2 (common for token auth without real account name)
		// use the first path component as account
		if u.User != nil && u.User.Username() == "oauth2" {
			account = pathParts[0]
		} else if u.User != nil && u.User.Username() != "" {
			account = u.User.Username()
		} else {
			account = pathParts[0]
		}

		return account, pathParts[1], token, nil
	}

	return "", "", "", fmt.Errorf("unsupported git URL format: %s", remoteURL)
}
