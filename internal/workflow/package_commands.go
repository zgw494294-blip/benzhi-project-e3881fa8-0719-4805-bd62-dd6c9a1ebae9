package workflow

import (
	"context"
	"errors"
	"strings"

	"caption-release-gate/internal/caption"
)

func (s *Service) CreatePackage(ctx context.Context, input CreatePackageInput) (caption.CaptionPackage, error) {
	const command = "create_package"
	var replayed caption.CaptionPackage
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, &replayed); ok || err != nil {
		return replayed, err
	}
	if len(strings.TrimSpace(input.IdempotencyKey)) < 8 {
		return caption.CaptionPackage{}, caption.Invalid("idempotencyKey", "至少需要 8 个字符")
	}
	rate, err := caption.ParseFrameRate(input.FrameRate)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	pkg, err := caption.NewPackage(caption.NewPackageInput{PackageID: newID("pkg"), ProgramTitle: input.ProgramTitle, LanguageTag: input.LanguageTag, FrameRate: rate, TimecodeMode: caption.TimecodeMode(input.TimecodeMode), CreatedBy: input.CreatedBy, Now: s.clock()})
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	event, err := s.audit.BuildEvent(pkg.PackageID, "package.created", pkg.CreatedBy, s.clock(), map[string]any{"programTitle": pkg.ProgramTitle, "languageTag": pkg.LanguageTag, "frameRate": pkg.FrameRate.String()}, nil)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, pkg)
	if err != nil {
		return caption.CaptionPackage{}, err
	}
	if err := s.repo.Commit(ctx, Commit{CreatePackage: true, Package: pkg, Event: event, Idempotency: idem}); err != nil {
		return caption.CaptionPackage{}, err
	}
	return pkg, nil
}

func (s *Service) ImportRevision(ctx context.Context, input ImportRevisionInput) (caption.CaptionRevision, error) {
	const command = "import_revision"
	var replayed caption.CaptionRevision
	if ok, err := s.replay(ctx, input.IdempotencyKey, command, &replayed); ok || err != nil {
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
	if pkg.Status == caption.StatusFrozen || pkg.Status == caption.StatusReleased {
		if err := caption.ReopenFromFrozen(&pkg); err != nil {
			return caption.CaptionRevision{}, err
		}
	}
	revision, err := caption.NewRevision(pkg, newID("rev"), input.SourceName, input.SubmittedBy, input.Cues, s.clock())
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	caption.Touch(&pkg, caption.StatusEditing, revision.RevisionID, s.clock())
	event, err := s.event(ctx, pkg.PackageID, "revision.imported", input.SubmittedBy, map[string]any{"revisionId": revision.RevisionID, "parentRevisionId": revision.ParentRevisionID, "cueCount": revision.CueCount, "contentDigest": revision.ContentDigest})
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	idem, err := record(input.IdempotencyKey, command, pkg.PackageID, revision)
	if err != nil {
		return caption.CaptionRevision{}, err
	}
	if err := s.repo.Commit(ctx, Commit{ExpectedVersion: input.ExpectedVersion, Package: pkg, Revision: &revision, Event: event, Idempotency: idem}); err != nil {
		return caption.CaptionRevision{}, err
	}
	return revision, nil
}

func (s *Service) GetPackage(ctx context.Context, packageID string) (PackageView, error) {
	pkg, err := s.repo.LoadPackage(ctx, packageID)
	if err != nil {
		return PackageView{}, err
	}
	view := PackageView{Package: pkg, Findings: []caption.QualityFinding{}, Revisions: []caption.CaptionRevision{}}
	view.Revisions, err = s.repo.ListRevisions(ctx, packageID)
	if err != nil {
		return PackageView{}, err
	}
	if pkg.CurrentRevisionID != "" {
		revision, loadErr := s.repo.LoadRevision(ctx, pkg.CurrentRevisionID)
		if loadErr != nil {
			return PackageView{}, loadErr
		}
		view.Revision = &revision
		view.Findings, err = s.repo.LoadFindings(ctx, revision.RevisionID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return PackageView{}, err
		}
		summary, summaryErr := s.repo.LoadSummary(ctx, revision.RevisionID)
		if summaryErr == nil {
			view.Summary = &summary
		} else if !errors.Is(summaryErr, ErrNotFound) {
			return PackageView{}, summaryErr
		}
	}
	review, reviewErr := s.repo.LoadLatestReview(ctx, packageID)
	if reviewErr == nil {
		view.Review = &review
	} else if !errors.Is(reviewErr, ErrNotFound) {
		return PackageView{}, reviewErr
	}
	manifest, manifestErr := s.repo.LoadManifest(ctx, packageID)
	if manifestErr == nil && manifest.RevisionID == pkg.CurrentRevisionID && manifest.FrozenDigest == pkg.FrozenDigest {
		view.Manifest = &manifest
	} else if !errors.Is(manifestErr, ErrNotFound) {
		if manifestErr != nil {
			return PackageView{}, manifestErr
		}
	}
	history, err := s.repo.LoadEvents(ctx, packageID)
	if err != nil {
		return PackageView{}, err
	}
	view.History = history
	return view, nil
}

func (s *Service) ListPackages(ctx context.Context) ([]caption.CaptionPackage, error) {
	return s.repo.ListPackages(ctx)
}
