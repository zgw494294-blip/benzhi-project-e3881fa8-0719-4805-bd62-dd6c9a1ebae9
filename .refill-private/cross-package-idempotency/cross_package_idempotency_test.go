package cross_package_idempotency_test

import (
	"context"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestIdempotencyReplayCannotCrossPackageBoundary(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	auditService, err := audit.NewService("scope-test", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo, auditService)
	create := func(title, key string) caption.CaptionPackage {
		pkg, createErr := service.CreatePackage(ctx, workflow.CreatePackageInput{
			ProgramTitle: title, LanguageTag: "zh-CN", FrameRate: "25",
			TimecodeMode: "non_drop", CreatedBy: "editor.scope", IdempotencyKey: key,
		})
		if createErr != nil {
			t.Fatal(createErr)
		}
		return pkg
	}
	first := create("包 A", "create-scope-a")
	second := create("包 B", "create-scope-b")
	cues := []caption.CueInput{{
		CueID: "cue-1", Sequence: 1, StartTimecode: "00:00:01:00",
		EndTimecode: "00:00:03:00", Lines: []string{"同一幂等键不得跨包复用"},
	}}
	sharedKey := "shared-import-key"
	if _, err := service.ImportRevision(ctx, workflow.ImportRevisionInput{
		PackageID: first.PackageID, SourceName: "a.json", SubmittedBy: "editor.scope",
		Cues: cues, ExpectedVersion: first.Version, IdempotencyKey: sharedKey,
	}); err != nil {
		t.Fatal(err)
	}
	replayed, err := service.ImportRevision(ctx, workflow.ImportRevisionInput{
		PackageID: second.PackageID, SourceName: "b.json", SubmittedBy: "editor.scope",
		Cues: cues, ExpectedVersion: second.Version, IdempotencyKey: sharedKey,
	})
	if err == nil {
		t.Fatalf("跨包复用 idempotencyKey 应被拒绝，却返回了 packageId=%q 的修订", replayed.PackageID)
	}
	loaded, loadErr := repo.LoadPackage(ctx, second.PackageID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if loaded.Status != caption.StatusDraft || loaded.CurrentRevisionID != "" {
		t.Fatalf("目标包不应被幂等缓存污染: %+v", loaded)
	}
}
