package detached_command_context_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestCanceledCreateDoesNotCommit(t *testing.T) {
	repo, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatalf("open repository: %v", err)
	}
	defer repo.Close()
	auditService, err := audit.NewService("private-test", []byte("private-test-signing-secret"))
	if err != nil {
		t.Fatalf("create audit service: %v", err)
	}
	service := workflow.NewService(repo, auditService)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	created, err := service.CreatePackage(ctx, workflow.CreatePackageInput{
		ProgramTitle:   "取消后的建档",
		LanguageTag:    "zh-CN",
		FrameRate:      "25",
		TimecodeMode:   "non_drop",
		CreatedBy:      "editor-private",
		IdempotencyKey: "cancel-create-private-001",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled create unexpectedly succeeded and persisted package %q: %v", created.PackageID, err)
	}

	packages, err := repo.ListPackages(context.Background())
	if err != nil {
		t.Fatalf("list packages: %v", err)
	}
	if len(packages) != 0 {
		t.Fatalf("canceled create left %d persisted package(s)", len(packages))
	}
}
