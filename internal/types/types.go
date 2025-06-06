package types

type BranchType int

const (
	BranchTypeUnknown BranchType = iota
	BranchTypeDev
	BranchTypePr
)
