package workflow

import (
	"context"

	"caption-release-gate/internal/caption"
)

type CheckResult struct {
	Package  caption.CaptionPackage   `json:"package"`
	Summary  caption.CheckSummary     `json:"summary"`
	Findings []caption.QualityFinding `json:"findings"`
}

func (s *Service) RunChecks(ctx context.Context, input RunCheckInput) (CheckResult, error) {
	const command = "run_checks"
	var replayed CheckResult
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, input.PackageID, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.ActorID, input.IdempotencyKey); err != nil {
		return CheckResult{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return CheckResult{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return CheckResult{}, err
	}
	if pkg.Status != caption.StatusEditing {
		return CheckResult{}, caption.ErrInvalidState
	}
	revision, err := s.repo.LoadRevision(ctx, pkg.CurrentRevisionID)
	if err != nil {
		return CheckResult{}, err
	}
	findings, summary := caption.RunQualityChecks(revision, pkg.FrameRate, caption.DefaultRuleConfig())
	status := caption.StatusReviewReady
	if len(findings) > 0 {
		status = caption.StatusRemediation
	}
	caption.Touch(&pkg, status, "", s.clock())
	result := CheckResult{Package: pkg, Summary: summary, Findings: findings}
	event, err := s.event(ctx, pkg.PackageID, "quality.checked", input.ActorID, summary)
	if err != nil {
		return CheckResult{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, result)
	if err != nil {
		return CheckResult{}, err
	}
	if err := s.repo.Commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Findings: findings, ReplaceFindings: true, Summary: &summary, Event: event, Idempotency: idem}); err != nil {
		return CheckResult{}, err
	}
	return result, nil
}

func (s *Service) RequestException(ctx context.Context, input ExceptionInput) (caption.QualityFinding, error) {
	const command = "request_exception"
	var replayed caption.QualityFinding
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, input.PackageID, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.ActorID, input.IdempotencyKey); err != nil {
		return caption.QualityFinding{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return caption.QualityFinding{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return caption.QualityFinding{}, err
	}
	if pkg.Status != caption.StatusRemediation {
		return caption.QualityFinding{}, caption.ErrInvalidState
	}
	findings, err := s.repo.LoadFindings(ctx, pkg.CurrentRevisionID)
	if err != nil {
		return caption.QualityFinding{}, err
	}
	index := -1
	for i := range findings {
		if findings[i].FindingID == input.FindingID {
			index = i
			break
		}
	}
	if index < 0 {
		return caption.QualityFinding{}, ErrNotFound
	}
	if err := caption.RequestFindingException(&findings[index], input.Reason); err != nil {
		return caption.QualityFinding{}, err
	}
	allAddressed := true
	for _, finding := range findings {
		if finding.Status == caption.FindingOpen {
			allAddressed = false
		}
	}
	status := caption.StatusRemediation
	if allAddressed {
		status = caption.StatusReviewReady
	}
	caption.Touch(&pkg, status, "", s.clock())
	result := findings[index]
	event, err := s.event(ctx, pkg.PackageID, "finding.exception_requested", input.ActorID, map[string]string{"findingId": result.FindingID, "reason": result.ExceptionReason})
	if err != nil {
		return caption.QualityFinding{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, result)
	if err != nil {
		return caption.QualityFinding{}, err
	}
	if err := s.repo.Commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Findings: findings, ReplaceFindings: true, Event: event, Idempotency: idem}); err != nil {
		return caption.QualityFinding{}, err
	}
	return result, nil
}

func (s *Service) SubmitReplacement(ctx context.Context, input ReplacementInput) (caption.CaptionRevision, error) {
	const command = "submit_replacement"
	var replayed caption.CaptionRevision
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, input.PackageID, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.SubmittedBy, input.IdempotencyKey); err != nil {
		return caption.CaptionRevision{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return caption.CaptionRevision{}, err
	}
	if pkg.Status != caption.StatusRemediation && pkg.Status != caption.StatusReviewReady {
		return caption.CaptionRevision{}, caption.ErrInvalidState
	}
	oldRevisionID := pkg.CurrentRevisionID
	revision, err := caption.NewRevision(pkg, newID("rev"), input.SourceName, input.SubmittedBy, input.Cues, s.clock())
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	findings, err := s.repo.LoadFindings(ctx, oldRevisionID)
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	if err := caption.ResolveFindings(findings, input.ResolvedFindingIDs, revision.RevisionID); err != nil {
		return caption.CaptionRevision{}, err
	}
	caption.Touch(&pkg, caption.StatusEditing, revision.RevisionID, s.clock())
	event, err := s.event(ctx, pkg.PackageID, "revision.replacement_submitted", input.SubmittedBy, map[string]any{"revisionId": revision.RevisionID, "parentRevisionId": revision.ParentRevisionID, "resolvedFindingIds": input.ResolvedFindingIDs})
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, revision)
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	if err := s.repo.Commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Revision: &revision, Findings: findings, ReplaceFindings: true, Event: event, Idempotency: idem}); err != nil {
		return caption.CaptionRevision{}, err
	}
	return revision, nil
}
