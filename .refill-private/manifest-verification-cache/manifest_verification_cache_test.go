package manifest_verification_cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"caption-release-gate/internal/audit"
	"caption-release-gate/internal/caption"
	"caption-release-gate/internal/store/sqlite"
	"caption-release-gate/internal/workflow"
)

func TestManifestVerificationCacheCannotReuseAcrossSignedPayloads(t *testing.T) {
	ctx := context.Background()
	repo, err := sqlite.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("打开存储: %v", err)
	}
	defer repo.Close()

	auditService, err := audit.NewService("release-gate", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("创建审计服务: %v", err)
	}
	now := time.Date(2026, 8, 27, 9, 30, 0, 0, time.UTC)
	manifest, err := auditService.Issue("pkg_cache", "rev_original", "frozen_digest", "rules_digest", now, now)
	if err != nil {
		t.Fatalf("签发清单: %v", err)
	}

	pkg := caption.CaptionPackage{
		PackageID:         manifest.PackageID,
		Status:            caption.StatusFrozen,
		CurrentRevisionID: manifest.RevisionID,
		FrozenDigest:      manifest.FrozenDigest,
		Version:           7,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	body, err := json.Marshal(pkg)
	if err != nil {
		t.Fatalf("编码包: %v", err)
	}
	if _, err := repo.DB().ExecContext(ctx,
		`INSERT INTO packages(package_id, version, status, updated_at, body) VALUES(?, ?, ?, ?, ?)`,
		pkg.PackageID, pkg.Version, pkg.Status, now.Format(time.RFC3339Nano), body,
	); err != nil {
		t.Fatalf("写入包: %v", err)
	}

	service := workflow.NewService(repo, auditService)
	first, err := service.VerifyManifest(ctx, manifest)
	if err != nil {
		t.Fatalf("首次验证: %v", err)
	}
	if result := first.(audit.VerificationResult); !result.Valid || result.Code != "valid" {
		t.Fatalf("原始清单应有效，得到 %#v", result)
	}

	tampered := manifest
	tampered.RevisionID = "rev_tampered"
	second, err := service.VerifyManifest(ctx, tampered)
	if err != nil {
		t.Fatalf("篡改后验证: %v", err)
	}
	if result := second.(audit.VerificationResult); result.Valid || result.Code != "tampered" {
		t.Fatalf("修改受签名保护的 revisionId 后应为 tampered，得到 %#v", result)
	}
}
