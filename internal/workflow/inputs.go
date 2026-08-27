package workflow

import (
	"time"

	"caption-release-gate/internal/caption"
)

type CreatePackageInput struct {
	ProgramTitle   string `json:"programTitle"`
	LanguageTag    string `json:"languageTag"`
	FrameRate      string `json:"frameRate"`
	TimecodeMode   string `json:"timecodeMode"`
	CreatedBy      string `json:"createdBy"`
	IdempotencyKey string `json:"idempotencyKey"`
}

type ImportRevisionInput struct {
	PackageID       string             `json:"packageId"`
	SourceName      string             `json:"sourceName"`
	SubmittedBy     string             `json:"submittedBy"`
	Cues            []caption.CueInput `json:"cues"`
	ExpectedVersion int64              `json:"expectedVersion"`
	IdempotencyKey  string             `json:"idempotencyKey"`
}

type RunCheckInput struct {
	PackageID       string `json:"packageId"`
	ActorID         string `json:"actorId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type ExceptionInput struct {
	PackageID       string `json:"packageId"`
	FindingID       string `json:"findingId"`
	Reason          string `json:"reason"`
	ActorID         string `json:"actorId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type ReplacementInput struct {
	PackageID          string             `json:"packageId"`
	SourceName         string             `json:"sourceName"`
	SubmittedBy        string             `json:"submittedBy"`
	ResolvedFindingIDs []string           `json:"resolvedFindingIds"`
	Cues               []caption.CueInput `json:"cues"`
	ExpectedVersion    int64              `json:"expectedVersion"`
	IdempotencyKey     string             `json:"idempotencyKey"`
}

type ReviewInput struct {
	PackageID            string                `json:"packageId"`
	ReviewerID           string                `json:"reviewerId"`
	Outcome              caption.ReviewOutcome `json:"outcome"`
	AcceptedExceptionIDs []string              `json:"acceptedExceptionIds"`
	Comment              string                `json:"comment"`
	ExpectedVersion      int64                 `json:"expectedVersion"`
	IdempotencyKey       string                `json:"idempotencyKey"`
}

type FreezeInput struct {
	PackageID       string `json:"packageId"`
	ActorID         string `json:"actorId"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type IssueInput struct {
	PackageID       string `json:"packageId"`
	IssuedBy        string `json:"issuedBy"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
}

type PackageView struct {
	Package   caption.CaptionPackage    `json:"package"`
	Revision  *caption.CaptionRevision  `json:"revision,omitempty"`
	Revisions []caption.CaptionRevision `json:"revisions"`
	Findings  []caption.QualityFinding  `json:"findings"`
	Summary   *caption.CheckSummary     `json:"checkSummary,omitempty"`
	Review    *caption.ReviewDecision   `json:"review,omitempty"`
	Manifest  *caption.ReleaseManifest  `json:"manifest,omitempty"`
	History   any                       `json:"history"`
}

type Clock func() time.Time
