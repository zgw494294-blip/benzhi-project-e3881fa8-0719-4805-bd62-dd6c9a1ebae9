package replacement_history_loss_test

import (
	"context"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestReplacementCheckPreservesResolvedParentFindings(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	auditService, err := audit.NewService("private-test-v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo, auditService)

	pkg, err := service.CreatePackage(ctx, workflow.CreatePackageInput{
		ProgramTitle:   "历史问题追溯测试",
		LanguageTag:    "zh-CN",
		FrameRate:      "25",
		TimecodeMode:   "non_drop",
		CreatedBy:      "editor-a",
		IdempotencyKey: "private-create-001",
	})
	if err != nil {
		t.Fatal(err)
	}

	firstRevision, err := service.ImportRevision(ctx, workflow.ImportRevisionInput{
		PackageID:       pkg.PackageID,
		SourceName:      "first.json",
		SubmittedBy:     "editor-a",
		ExpectedVersion: 1,
		IdempotencyKey:  "private-import-001",
		Cues: []caption.CueInput{{
			CueID:         "cue-001",
			Sequence:      1,
			StartTimecode: "00:00:01:00",
			EndTimecode:   "00:00:01:12",
			Lines:         []string{"短"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	firstCheck, err := service.RunChecks(ctx, workflow.RunCheckInput{
		PackageID:       pkg.PackageID,
		ActorID:         "editor-a",
		ExpectedVersion: 2,
		IdempotencyKey:  "private-check-001",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(firstCheck.Findings) != 1 {
		t.Fatalf("前置检查应产生一个问题，实际为 %d", len(firstCheck.Findings))
	}

	_, err = service.SubmitReplacement(ctx, workflow.ReplacementInput{
		PackageID:          pkg.PackageID,
		SourceName:         "replacement.json",
		SubmittedBy:        "editor-a",
		ResolvedFindingIDs: []string{firstCheck.Findings[0].FindingID},
		ExpectedVersion:    3,
		IdempotencyKey:     "private-replace-001",
		Cues: []caption.CueInput{{
			CueID:         "cue-001",
			Sequence:      1,
			StartTimecode: "00:00:01:00",
			EndTimecode:   "00:00:03:00",
			Lines:         []string{"已完成整改"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.RunChecks(ctx, workflow.RunCheckInput{
		PackageID:       pkg.PackageID,
		ActorID:         "editor-a",
		ExpectedVersion: 4,
		IdempotencyKey:  "private-check-002",
	})
	if err != nil {
		t.Fatal(err)
	}

	historical, err := repo.LoadFindings(ctx, firstRevision.RevisionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(historical) != 1 {
		t.Fatalf("父修订已解决问题发生数据丢失：期望 1 条，实际 %d", len(historical))
	}
	if historical[0].Status != caption.FindingResolved {
		t.Fatalf("父修订问题状态错误：%s", historical[0].Status)
	}
	if historical[0].ResolvedByRevisionID == "" {
		t.Fatal("父修订问题缺少 resolvedByRevisionId")
	}
}
