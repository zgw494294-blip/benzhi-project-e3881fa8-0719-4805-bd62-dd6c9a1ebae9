package caption

import "time"

type PackageStatus string

const (
	StatusDraft       PackageStatus = "draft"
	StatusEditing     PackageStatus = "editing"
	StatusRemediation PackageStatus = "remediation"
	StatusReviewReady PackageStatus = "review_ready"
	StatusApproved    PackageStatus = "approved"
	StatusFrozen      PackageStatus = "frozen"
	StatusReleased    PackageStatus = "released"
)

type CaptionPackage struct {
	PackageID         string        `json:"packageId"`
	ProgramTitle      string        `json:"programTitle"`
	LanguageTag       string        `json:"languageTag"`
	FrameRate         FrameRate     `json:"frameRate"`
	TimecodeMode      TimecodeMode  `json:"timecodeMode"`
	Status            PackageStatus `json:"status"`
	CurrentRevisionID string        `json:"currentRevisionId,omitempty"`
	FrozenDigest      string        `json:"frozenDigest,omitempty"`
	Version           int64         `json:"version"`
	CreatedBy         string        `json:"createdBy"`
	CreatedAt         time.Time     `json:"createdAt"`
	UpdatedAt         time.Time     `json:"updatedAt"`
}

type CaptionRevision struct {
	RevisionID       string       `json:"revisionId"`
	PackageID        string       `json:"packageId"`
	ParentRevisionID string       `json:"parentRevisionId,omitempty"`
	SourceName       string       `json:"sourceName"`
	CueCount         int          `json:"cueCount"`
	Cues             []CaptionCue `json:"cues"`
	ContentDigest    string       `json:"contentDigest"`
	SubmittedBy      string       `json:"submittedBy"`
	SubmittedAt      time.Time    `json:"submittedAt"`
}

type CaptionCue struct {
	CueID          string   `json:"cueId"`
	Sequence       int      `json:"sequence"`
	StartTimecode  string   `json:"startTimecode"`
	EndTimecode    string   `json:"endTimecode"`
	StartFrame     int64    `json:"startFrame"`
	EndFrame       int64    `json:"endFrame"`
	Speaker        string   `json:"speaker,omitempty"`
	Lines          []string `json:"lines"`
	NormalizedText string   `json:"normalizedText"`
}

type FindingSeverity string
type FindingStatus string

const (
	SeverityBlocking FindingSeverity = "blocking"
	SeverityWarning  FindingSeverity = "warning"
	FindingOpen      FindingStatus   = "open"
	FindingException FindingStatus   = "exception_requested"
	FindingAccepted  FindingStatus   = "exception_accepted"
	FindingResolved  FindingStatus   = "resolved"
)

type QualityFinding struct {
	FindingID            string          `json:"findingId"`
	RevisionID           string          `json:"revisionId"`
	CueIDs               []string        `json:"cueIds"`
	RuleCode             string          `json:"ruleCode"`
	RuleVersion          string          `json:"ruleVersion"`
	Severity             FindingSeverity `json:"severity"`
	Message              string          `json:"message"`
	Status               FindingStatus   `json:"status"`
	ExceptionReason      string          `json:"exceptionReason,omitempty"`
	ResolvedByRevisionID string          `json:"resolvedByRevisionId,omitempty"`
}

type ReviewOutcome string

const (
	ReviewReturned ReviewOutcome = "returned"
	ReviewApproved ReviewOutcome = "approved"
)

type ReviewDecision struct {
	DecisionID           string        `json:"decisionId"`
	PackageID            string        `json:"packageId"`
	RevisionID           string        `json:"revisionId"`
	ReviewerID           string        `json:"reviewerId"`
	Outcome              ReviewOutcome `json:"outcome"`
	AcceptedExceptionIDs []string      `json:"acceptedExceptionIds"`
	Comment              string        `json:"comment"`
	DecidedAt            time.Time     `json:"decidedAt"`
}

type CheckSummary struct {
	RevisionID  string `json:"revisionId"`
	RuleVersion string `json:"ruleVersion"`
	RuleDigest  string `json:"ruleDigest"`
	Total       int    `json:"total"`
	Blocking    int    `json:"blocking"`
	Warnings    int    `json:"warnings"`
}

type ReleaseManifest struct {
	ManifestID      string    `json:"manifestId"`
	PackageID       string    `json:"packageId"`
	RevisionID      string    `json:"revisionId"`
	FrozenDigest    string    `json:"frozenDigest"`
	RuleSetDigest   string    `json:"ruleSetDigest"`
	ApprovedAt      time.Time `json:"approvedAt"`
	IssuedBy        string    `json:"issuedBy"`
	IssuedAt        time.Time `json:"issuedAt"`
	VerificationTag string    `json:"verificationTag"`
}
