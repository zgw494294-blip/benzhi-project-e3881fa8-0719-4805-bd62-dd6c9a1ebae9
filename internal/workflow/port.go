package workflow

import (
	"context"
	"encoding/json"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
)

type IdempotencyRecord struct {
	Key       string          `json:"key"`
	Command   string          `json:"command"`
	PackageID string          `json:"packageId"`
	Response  json.RawMessage `json:"response"`
}

type Commit struct {
	CreatePackage   bool
	ExpectedVersion int64
	Package         caption.CaptionPackage
	Revision        *caption.CaptionRevision
	Findings        []caption.QualityFinding
	ReplaceFindings bool
	Summary         *caption.CheckSummary
	Review          *caption.ReviewDecision
	Manifest        *caption.ReleaseManifest
	Event           audit.Event
	Idempotency     IdempotencyRecord
}

type Repository interface {
	Close() error
	LoadPackage(context.Context, string) (caption.CaptionPackage, error)
	ListPackages(context.Context) ([]caption.CaptionPackage, error)
	LoadRevision(context.Context, string) (caption.CaptionRevision, error)
	ListRevisions(context.Context, string) ([]caption.CaptionRevision, error)
	LoadFindings(context.Context, string) ([]caption.QualityFinding, error)
	LoadSummary(context.Context, string) (caption.CheckSummary, error)
	LoadLatestReview(context.Context, string) (caption.ReviewDecision, error)
	LoadManifest(context.Context, string) (caption.ReleaseManifest, error)
	LoadEvents(context.Context, string) ([]audit.Event, error)
	LoadIdempotency(context.Context, string) (IdempotencyRecord, error)
	Commit(context.Context, Commit) error
}
