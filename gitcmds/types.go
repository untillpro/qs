package gitcmds

type fileStatus int

const (
	fileStatusUntracked fileStatus = iota
	fileStatusAdded
	fileStatusModified
	fileStatusDeleted
	fileStatusRenamed
)

type FileInfo struct {
	status       fileStatus
	name         string
	oldName      string
	sizeIncrease int64
}
