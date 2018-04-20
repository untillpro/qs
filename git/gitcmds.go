package git

import (
	"os"

	"github.com/untillpro/qg/vcs"

	u "github.com/untillpro/qg/utils"
)

// Status shows git repo status
func Status(cfg vcs.CfgStatus) {
	err := new(u.PipedExec).
		Command("git", "remote", "-v").
		Command("grep", "fetch").
		Run(os.Stdout, os.Stdout)
	if nil != err {
		return
	}
	new(u.PipedExec).
		Command("git", "status", "-s", "-b", "-uall").
		Run(os.Stdout, os.Stdout)

}

// Upload upload sources to git repo
func Upload(cfg vcs.CfgUpload) {
	new(u.PipedExec).
		Command("git", "add", ".").
		Run(os.Stdout, os.Stdout)

	params := []string{"commit", "-a"}
	for _, m := range cfg.Message {
		params = append(params, "-m", m)
	}

	new(u.PipedExec).
		Command("git", params...).
		Run(os.Stdout, os.Stdout)

	new(u.PipedExec).
		Command("git", "push").
		Run(os.Stdout, os.Stdout)
}

// Download sources from git repo
func Download(cfg vcs.CfgDownload) {
	new(u.PipedExec).
		Command("git", "pull").
		Run(os.Stdout, os.Stdout)
}

// Gui shows gui
func Gui() {
	new(u.PipedExec).
		Command("git", "gui").
		Run(os.Stdout, os.Stdout)
}
