package history_cache_staleness_test

import (
	"context"
	"testing"
	"time"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestHistoryCacheRefreshesAfterCommittedEvent(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	auditService, err := audit.NewService("private-test-issuer", []byte("private-test-secret-key"))
	if err != nil {
		t.Fatalf("create audit service: %v", err)
	}
	service := workflow.NewService(repo, auditService)
	defer service.Close()
	service.SetClock(func() time.Time {
		return time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	})

	pkg, err := service.CreatePackage(ctx, workflow.CreatePackageInput{
		ProgramTitle:   "缓存失效复现",
		LanguageTag:    "zh-CN",
		FrameRate:      "25",
		TimecodeMode:   "non_drop",
		CreatedBy:      "editor.private",
		IdempotencyKey: "private-create-history-cache",
	})
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	initialValue, err := service.History(ctx, pkg.PackageID)
	if err != nil {
		t.Fatalf("load initial history: %v", err)
	}
	initial := initialValue.([]audit.Event)
	if len(initial) != 1 {
		t.Fatalf("initial history length = %d, want 1", len(initial))
	}

	_, err = service.ImportRevision(ctx, workflow.ImportRevisionInput{
		PackageID:       pkg.PackageID,
		SourceName:      "private-v1.json",
		SubmittedBy:     "editor.private",
		ExpectedVersion: 1,
		IdempotencyKey:  "private-import-history-cache",
		Cues: []caption.CueInput{{
			CueID:         "cue-private-1",
			Sequence:      1,
			StartTimecode: "00:00:01:00",
			EndTimecode:   "00:00:03:00",
			Lines:         []string{"缓存之后提交的新字幕"},
		}},
	})
	if err != nil {
		t.Fatalf("import revision: %v", err)
	}

	latestValue, err := service.History(ctx, pkg.PackageID)
	if err != nil {
		t.Fatalf("load history after commit: %v", err)
	}
	latest := latestValue.([]audit.Event)
	if len(latest) != 2 {
		t.Fatalf("history after committed event has %d entries, want 2", len(latest))
	}
	if latest[1].Type != "revision.imported" {
		t.Fatalf("latest event type = %q, want revision.imported", latest[1].Type)
	}
}
