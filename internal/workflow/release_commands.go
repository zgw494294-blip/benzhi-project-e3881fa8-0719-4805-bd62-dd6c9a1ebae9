package workflow

import (
	"context"
	"strings"

	"caption-release-gate/internal/caption"
)

func (s *Service) Review(ctx context.Context, input ReviewInput) (caption.ReviewDecision, error) {
	const command = "review"
	var replayed caption.ReviewDecision
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.ReviewerID, input.IdempotencyKey); err != nil {
		return caption.ReviewDecision{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return caption.ReviewDecision{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return caption.ReviewDecision{}, err
	}
	revision, err := s.repo.LoadRevision(ctx, pkg.CurrentRevisionID)
	if err != nil {
		return caption.ReviewDecision{}, err
	}
	findings, err := s.repo.LoadFindings(ctx, revision.RevisionID)
	if err != nil && len(findings) == 0 {
		return caption.ReviewDecision{}, err
	}
	if err := caption.ValidateReview(pkg, revision, findings, input.ReviewerID, input.Outcome, input.AcceptedExceptionIDs); err != nil {
		return caption.ReviewDecision{}, err
	}
	findings = caption.ApplyAcceptedExceptions(findings, input.AcceptedExceptionIDs)
	decision := caption.ReviewDecision{DecisionID: newID("decision"), PackageID: pkg.PackageID, RevisionID: revision.RevisionID, ReviewerID: strings.TrimSpace(input.ReviewerID), Outcome: input.Outcome, AcceptedExceptionIDs: append([]string(nil), input.AcceptedExceptionIDs...), Comment: strings.TrimSpace(input.Comment), DecidedAt: s.clock()}
	status := caption.StatusRemediation
	if input.Outcome == caption.ReviewApproved {
		status = caption.StatusApproved
	}
	caption.Touch(&pkg, status, "", s.clock())
	event, err := s.event(ctx, pkg.PackageID, "review."+string(input.Outcome), input.ReviewerID, decision)
	if err != nil {
		return caption.ReviewDecision{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, decision)
	if err != nil {
		return caption.ReviewDecision{}, err
	}
	if err := s.commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Findings: findings, ReplaceFindings: true, Review: &decision, Event: event, Idempotency: idem}); err != nil {
		return caption.ReviewDecision{}, err
	}
	return decision, nil
}

func (s *Service) Freeze(ctx context.Context, input FreezeInput) (caption.CaptionPackage, error) {
	const command = "freeze"
	var replayed caption.CaptionPackage
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.ActorID, input.IdempotencyKey); err != nil {
		return caption.CaptionPackage{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return caption.CaptionPackage{}, err
	}
	review, err := s.repo.LoadLatestReview(ctx, pkg.PackageID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	if err := caption.CanFreeze(pkg, &review); err != nil {
		return caption.CaptionPackage{}, err
	}
	revision, err := s.repo.LoadRevision(ctx, pkg.CurrentRevisionID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	findings, err := s.repo.LoadFindings(ctx, revision.RevisionID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	summary, err := s.repo.LoadSummary(ctx, revision.RevisionID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	history, err := s.repo.LoadEvents(ctx, pkg.PackageID)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	digest, err := s.audit.Freeze(revision, findings, review, summary.RuleDigest, history)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	pkg.FrozenDigest = digest
	caption.Touch(&pkg, caption.StatusFrozen, "", s.clock())
	event, err := s.audit.BuildEvent(pkg.PackageID, "package.frozen", input.ActorID, s.clock(), map[string]string{"frozenDigest": digest, "revisionId": revision.RevisionID}, history)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, pkg)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	if err := s.commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Event: event, Idempotency: idem}); err != nil {
		return caption.CaptionPackage{}, err
	}
	return pkg, nil
}

func (s *Service) IssueManifest(ctx context.Context, input IssueInput) (caption.ReleaseManifest, error) {
	const command = "issue_manifest"
	var replayed caption.ReleaseManifest
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, &replayed); ok || err != nil {
		return replayed, err
	}
	if err := validateCommand(input.PackageID, input.IssuedBy, input.IdempotencyKey); err != nil {
		return caption.ReleaseManifest{}, err
	}
	pkg, err := s.repo.LoadPackage(ctx, input.PackageID)
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	if err := checkVersion(pkg, input.ExpectedVersion); err != nil {
		return caption.ReleaseManifest{}, err
	}
	if err := caption.CanIssue(pkg); err != nil {
		return caption.ReleaseManifest{}, err
	}
	review, err := s.repo.LoadLatestReview(ctx, pkg.PackageID)
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	summary, err := s.repo.LoadSummary(ctx, pkg.CurrentRevisionID)
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	manifest, err := s.audit.Issue(pkg.PackageID, pkg.CurrentRevisionID, pkg.FrozenDigest, summary.RuleDigest, review.DecidedAt, s.clock())
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	caption.Touch(&pkg, caption.StatusReleased, "", s.clock())
	event, err := s.event(ctx, pkg.PackageID, "manifest.issued", input.IssuedBy, map[string]string{"manifestId": manifest.ManifestID, "verificationTag": manifest.VerificationTag})
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, manifest)
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	if err := s.commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Manifest: &manifest, Event: event, Idempotency: idem}); err != nil {
		return caption.ReleaseManifest{}, err
	}
	return manifest, nil
}

func (s *Service) VerifyManifest(ctx context.Context, manifest caption.ReleaseManifest) (auditResult any, err error) {
	pkg, loadErr := s.repo.LoadPackage(ctx, manifest.PackageID)
	expected := ""
	if loadErr == nil {
		expected = pkg.FrozenDigest
	} else if loadErr != ErrNotFound {
		return nil, loadErr
	}
	return s.audit.Verify(manifest, expected), nil
}

func (s *Service) History(ctx context.Context, packageID string) (any, error) {
	s.historyMu.RLock()
	cached, ok := s.historyCache[packageID]
	s.historyMu.RUnlock()
	if ok {
		return cloneEvents(cached), nil
	}
	events, err := s.repo.LoadEvents(ctx, packageID)
	if err != nil {
		return nil, err
	}
	if err := s.audit.VerifyHistory(events); err != nil {
		return nil, err
	}
	s.historyMu.Lock()
	s.historyCache[packageID] = cloneEvents(events)
	s.historyMu.Unlock()
	return cloneEvents(events), nil
}
