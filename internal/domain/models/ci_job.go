package models

// CITriggerType represents the type of event that triggered the CI job
type CITriggerType string

const (
	CITriggerTypePush        CITriggerType = "push"
	CITriggerTypeTag         CITriggerType = "tag"
	CITriggerTypePullRequest CITriggerType = "pull_request"
	CITriggerTypeManual      CITriggerType = "manual"
)

// CIRefType represents the type of git reference
type CIRefType string

const (
	CIRefTypeBranch CIRefType = "branch"
	CIRefTypeTag    CIRefType = "tag"
)
