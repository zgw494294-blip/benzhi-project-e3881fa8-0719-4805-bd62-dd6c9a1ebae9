package concurrentreviewscratch_test

import (
	"context"
	"sync"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

type controlledRepository struct {
	packages       map[string]caption.CaptionPackage
	revisions      map[string]caption.CaptionRevision
	findings       map[string][]caption.QualityFinding
	packageAStaged chan struct{}
	releaseA       chan struct{}
	commits        chan workflow.Commit
	stagedOnce     sync.Once
}

func (r *controlledRepository) Close() error { return nil }

func (r *controlledRepository) LoadPackage(_ context.Context, packageID string) (caption.CaptionPackage, error) {
	return r.packages[packageID], nil
}

func (r *controlledRepository) ListPackages(context.Context) ([]caption.CaptionPackage, error) {
	return nil, nil
}

func (r *controlledRepository) LoadRevision(_ context.Context, revisionID string) (caption.CaptionRevision, error) {
	return r.revisions[revisionID], nil
}

func (r *controlledRepository) ListRevisions(context.Context, string) ([]caption.CaptionRevision, error) {
	return nil, nil
}

func (r *controlledRepository) LoadFindings(_ context.Context, revisionID string) ([]caption.QualityFinding, error) {
	return append([]caption.QualityFinding(nil), r.findings[revisionID]...), nil
}

func (r *controlledRepository) LoadSummary(context.Context, string) (caption.CheckSummary, error) {
	return caption.CheckSummary{}, workflow.ErrNotFound
}

func (r *controlledRepository) LoadLatestReview(context.Context, string) (caption.ReviewDecision, error) {
	return caption.ReviewDecision{}, workflow.ErrNotFound
}

func (r *controlledRepository) LoadManifest(context.Context, string) (caption.ReleaseManifest, error) {
	return caption.ReleaseManifest{}, workflow.ErrNotFound
}

func (r *controlledRepository) LoadEvents(_ context.Context, packageID string) ([]audit.Event, error) {
	if packageID == "pkg-a" {
		r.stagedOnce.Do(func() { close(r.packageAStaged) })
		<-r.releaseA
	}
	return nil, nil
}

func (r *controlledRepository) LoadIdempotency(context.Context, string) (workflow.IdempotencyRecord, error) {
	return workflow.IdempotencyRecord{}, workflow.ErrNotFound
}

func (r *controlledRepository) Commit(_ context.Context, commit workflow.Commit) error {
	copied := commit
	copied.Findings = append([]caption.QualityFinding(nil), commit.Findings...)
	r.commits <- copied
	return nil
}

func TestConcurrentReviewsKeepFindingStateIsolated(t *testing.T) {
	repo := newControlledRepository()
	auditService, err := audit.NewService("test-issuer", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo, auditService)

	aDone := make(chan error, 1)
	go func() {
		_, reviewErr := service.Review(context.Background(), reviewInput("pkg-a", "finding-a", "review-a-key"))
		aDone <- reviewErr
	}()

	<-repo.packageAStaged
	if _, err := service.Review(context.Background(), reviewInput("pkg-b", "finding-b", "review-b-key")); err != nil {
		t.Fatalf("package B review failed: %v", err)
	}
	bCommit := <-repo.commits
	close(repo.releaseA)
	if err := <-aDone; err != nil {
		t.Fatalf("package A review failed: %v", err)
	}
	aCommit := <-repo.commits

	if bCommit.Package.PackageID != "pkg-b" || aCommit.Package.PackageID != "pkg-a" {
		t.Fatalf("controlled commit order changed: first=%s second=%s", bCommit.Package.PackageID, aCommit.Package.PackageID)
	}
	if len(aCommit.Findings) != 1 || aCommit.Findings[0].RevisionID != "rev-a" || aCommit.Findings[0].FindingID != "finding-a" {
		t.Fatalf("package A commit reused package B findings: %+v", aCommit.Findings)
	}
}

func newControlledRepository() *controlledRepository {
	packages := map[string]caption.CaptionPackage{}
	revisions := map[string]caption.CaptionRevision{}
	findings := map[string][]caption.QualityFinding{}
	for _, suffix := range []string{"a", "b"} {
		packageID := "pkg-" + suffix
		revisionID := "rev-" + suffix
		findingID := "finding-" + suffix
		packages[packageID] = caption.CaptionPackage{PackageID: packageID, CurrentRevisionID: revisionID, Status: caption.StatusReviewReady, Version: 4}
		revisions[revisionID] = caption.CaptionRevision{RevisionID: revisionID, PackageID: packageID, SubmittedBy: "editor-" + suffix}
		findings[revisionID] = []caption.QualityFinding{{FindingID: findingID, RevisionID: revisionID, Status: caption.FindingException, Severity: caption.SeverityWarning}}
	}
	return &controlledRepository{
		packages:       packages,
		revisions:      revisions,
		findings:       findings,
		packageAStaged: make(chan struct{}),
		releaseA:       make(chan struct{}),
		commits:        make(chan workflow.Commit, 2),
	}
}

func reviewInput(packageID, findingID, key string) workflow.ReviewInput {
	return workflow.ReviewInput{
		PackageID:            packageID,
		ReviewerID:           "reviewer-" + packageID,
		Outcome:              caption.ReviewApproved,
		AcceptedExceptionIDs: []string{findingID},
		ExpectedVersion:      4,
		IdempotencyKey:       key,
	}
}
