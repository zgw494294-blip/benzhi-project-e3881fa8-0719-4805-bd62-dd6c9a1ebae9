package tamper_classification_test

import (
	"testing"
	"time"

	"caption-release-gate/internal/audit"
)

func TestTamperedDigestIsNotMisclassifiedAsContentMismatch(t *testing.T) {
	signer, err := audit.NewSigner("classification-test", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	manifest, err := signer.Issue("pkg-1", "rev-1", "frozen-original", "rules-1", now, now)
	if err != nil {
		t.Fatal(err)
	}
	manifest.FrozenDigest = "forged-digest"
	result := signer.Result(manifest, "frozen-original")
	if result.Code != "tampered" {
		t.Fatalf("签名覆盖字段被篡改应报告 tampered，实际 code=%q message=%q", result.Code, result.Message)
	}
}
