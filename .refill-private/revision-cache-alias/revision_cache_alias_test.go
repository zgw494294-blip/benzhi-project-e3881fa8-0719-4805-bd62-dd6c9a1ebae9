package revision_cache_alias_test

import (
	"context"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestRevisionCacheReturnsIndependentNestedSlices(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	auditService, err := audit.NewService("test-v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo, auditService)
	pkg, err := service.CreatePackage(ctx, workflow.CreatePackageInput{
		ProgramTitle:   "缓存隔离测试",
		LanguageTag:    "zh-CN",
		FrameRate:      "25",
		TimecodeMode:   "non_drop",
		CreatedBy:      "editor-a",
		IdempotencyKey: "create-cache-isolation",
	})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := service.ImportRevision(ctx, workflow.ImportRevisionInput{
		PackageID:       pkg.PackageID,
		SourceName:      "episode.srt",
		SubmittedBy:     "editor-a",
		ExpectedVersion: pkg.Version,
		IdempotencyKey:  "import-cache-isolation",
		Cues: []caption.CueInput{{
			CueID:         "cue-1",
			Sequence:      1,
			StartTimecode: "00:00:01:00",
			EndTimecode:   "00:00:03:00",
			Lines:         []string{"原始字幕"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := repo.LoadRevision(ctx, revision.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	first.Cues[0].Lines[0] = "调用方局部修改"

	second, err := repo.LoadRevision(ctx, revision.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	if got := second.Cues[0].Lines[0]; got != "原始字幕" {
		t.Fatalf("%s: 第二次读取被首次返回值污染，got %q", t.Name(), got)
	}
}
