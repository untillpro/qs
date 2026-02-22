package gitcmds

type fileInfo struct {
	name         string
	sizeIncrease int64
}

type PRInfo struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
