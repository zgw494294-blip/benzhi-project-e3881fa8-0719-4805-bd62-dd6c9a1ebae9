package sqlite

import (
	"context"
	"testing"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/workflow"
)

func TestRepositoryWorkflowPersistence(t *testing.T) {
	ctx := context.Background()
	repo, err := Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	auditService, err := audit.NewService("test-v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := workflow.NewService(repo, auditService)
	pkg, err := service.CreatePackage(ctx, workflow.CreatePackageInput{ProgramTitle: "测试节目", LanguageTag: "zh-CN", FrameRate: "25", TimecodeMode: "non_drop", CreatedBy: "editor", IdempotencyKey: "create-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.CreatePackage(ctx, workflow.CreatePackageInput{IdempotencyKey: "create-key-001"})
	if err != nil {
		t.Fatal(err)
	}
	if replayed.PackageID != pkg.PackageID {
		t.Fatal("幂等响应未复用")
	}
	loaded, err := repo.LoadPackage(ctx, pkg.PackageID)
	if err != nil || loaded.Version != 1 {
		t.Fatalf("持久化读取失败: %v", err)
	}
}
