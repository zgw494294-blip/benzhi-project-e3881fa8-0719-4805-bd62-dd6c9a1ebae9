package shared_detail_context_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/workflow"
)

type observingContext struct {
	context.Context
	onDone func()
}

func (c observingContext) Done() <-chan struct{} {
	c.onDone()
	return c.Context.Done()
}

type controlledRepository struct {
	loadCount   atomic.Int32
	firstLoaded chan struct{}
	secondRead  chan struct{}
	firstOnce   sync.Once
	secondOnce  sync.Once
}

func newControlledRepository() *controlledRepository {
	return &controlledRepository{
		firstLoaded: make(chan struct{}),
		secondRead:  make(chan struct{}),
	}
}

func (r *controlledRepository) Close() error { return nil }

func (r *controlledRepository) LoadPackage(ctx context.Context, packageID string) (caption.CaptionPackage, error) {
	if r.loadCount.Add(1) == 1 {
		r.firstOnce.Do(func() { close(r.firstLoaded) })
		<-ctx.Done()
		return caption.CaptionPackage{}, ctx.Err()
	}
	r.secondOnce.Do(func() { close(r.secondRead) })
	return caption.CaptionPackage{PackageID: packageID, Status: caption.StatusDraft, Version: 1}, nil
}

func (r *controlledRepository) ListPackages(context.Context) ([]caption.CaptionPackage, error) {
	return nil, nil
}

func (r *controlledRepository) LoadRevision(context.Context, string) (caption.CaptionRevision, error) {
	return caption.CaptionRevision{}, workflow.ErrNotFound
}

func (r *controlledRepository) ListRevisions(context.Context, string) ([]caption.CaptionRevision, error) {
	return []caption.CaptionRevision{}, nil
}

func (r *controlledRepository) LoadFindings(context.Context, string) ([]caption.QualityFinding, error) {
	return []caption.QualityFinding{}, nil
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

func (r *controlledRepository) LoadEvents(context.Context, string) ([]audit.Event, error) {
	return []audit.Event{}, nil
}

func (r *controlledRepository) LoadIdempotency(context.Context, string) (workflow.IdempotencyRecord, error) {
	return workflow.IdempotencyRecord{}, workflow.ErrNotFound
}

func (r *controlledRepository) Commit(context.Context, workflow.Commit) error { return nil }

func TestSharedDetailReadDoesNotInheritLeaderCancellation(t *testing.T) {
	repo := newControlledRepository()
	service := workflow.NewService(repo, nil)
	firstContext, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()

	firstResult := make(chan error, 1)
	go func() {
		_, err := service.GetPackage(firstContext, "pkg-shared-context")
		firstResult <- err
	}()
	<-repo.firstLoaded

	secondBase, cancelSecond := context.WithCancel(context.Background())
	defer cancelSecond()
	secondWaiting := make(chan struct{})
	var waitingOnce sync.Once
	secondContext := observingContext{
		Context: secondBase,
		onDone:  func() { waitingOnce.Do(func() { close(secondWaiting) }) },
	}
	secondResult := make(chan error, 1)
	go func() {
		_, err := service.GetPackage(secondContext, "pkg-shared-context")
		secondResult <- err
	}()

	select {
	case <-secondWaiting:
	case <-repo.secondRead:
	}
	cancelFirst()

	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("首个请求应按自身取消结束，实际错误为 %v", err)
	}
	select {
	case err := <-secondResult:
		if err != nil {
			t.Fatalf("仍存活的次请求继承了首请求的取消错误: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("仍存活的次请求未能独立完成")
	}
}
