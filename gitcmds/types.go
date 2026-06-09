package gitcmds

type fileInfo struct {
	name         string
	sizeIncrease int64
}

type PRInfo struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// RepoContext holds repository state gathered upfront for Dev()
type RepoContext struct {
	Repo                  string
	Org                   string
	CurrentBranch         string
	MainBranch            string
	IsMain                bool
	HasUncommittedChanges bool
	UpstreamExists        bool
	ParentRepo            string
}
