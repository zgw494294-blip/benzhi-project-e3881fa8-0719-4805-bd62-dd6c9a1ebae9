package caption

import (
	"strings"
	"time"
)

type NewPackageInput struct {
	PackageID    string
	ProgramTitle string
	LanguageTag  string
	FrameRate    FrameRate
	TimecodeMode TimecodeMode
	CreatedBy    string
	Now          time.Time
}

func NewPackage(input NewPackageInput) (CaptionPackage, error) {
	if strings.TrimSpace(input.PackageID) == "" {
		return CaptionPackage{}, Invalid("packageId", "不能为空")
	}
	if strings.TrimSpace(input.ProgramTitle) == "" {
		return CaptionPackage{}, Invalid("programTitle", "不能为空")
	}
	if strings.TrimSpace(input.LanguageTag) == "" {
		return CaptionPackage{}, Invalid("languageTag", "不能为空")
	}
	if strings.TrimSpace(input.CreatedBy) == "" {
		return CaptionPackage{}, Invalid("createdBy", "不能为空")
	}
	if err := ValidateMode(input.FrameRate, input.TimecodeMode); err != nil {
		return CaptionPackage{}, err
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return CaptionPackage{PackageID: strings.TrimSpace(input.PackageID), ProgramTitle: strings.TrimSpace(input.ProgramTitle), LanguageTag: strings.TrimSpace(input.LanguageTag), FrameRate: input.FrameRate, TimecodeMode: input.TimecodeMode, Status: StatusDraft, Version: 1, CreatedBy: strings.TrimSpace(input.CreatedBy), CreatedAt: now, UpdatedAt: now}, nil
}

func NewRevision(pkg CaptionPackage, revisionID, sourceName, submitter string, inputs []CueInput, now time.Time) (CaptionRevision, error) {
	if err := CanImportRevision(pkg); err != nil {
		return CaptionRevision{}, err
	}
	if strings.TrimSpace(revisionID) == "" {
		return CaptionRevision{}, Invalid("revisionId", "不能为空")
	}
	if strings.TrimSpace(sourceName) == "" {
		return CaptionRevision{}, Invalid("sourceName", "不能为空")
	}
	if strings.TrimSpace(submitter) == "" {
		return CaptionRevision{}, Invalid("submittedBy", "不能为空")
	}
	cues, digest, err := NormalizeCues(inputs, pkg.FrameRate, pkg.TimecodeMode)
	if err != nil {
		return CaptionRevision{}, err
	}
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return CaptionRevision{RevisionID: strings.TrimSpace(revisionID), PackageID: pkg.PackageID, ParentRevisionID: pkg.CurrentRevisionID, SourceName: strings.TrimSpace(sourceName), CueCount: len(cues), Cues: cues, ContentDigest: digest, SubmittedBy: strings.TrimSpace(submitter), SubmittedAt: now}, nil
}

func Touch(pkg *CaptionPackage, status PackageStatus, revisionID string, now time.Time) {
	pkg.Status = status
	if revisionID != "" {
		pkg.CurrentRevisionID = revisionID
	}
	pkg.Version++
	pkg.UpdatedAt = now.UTC()
}
