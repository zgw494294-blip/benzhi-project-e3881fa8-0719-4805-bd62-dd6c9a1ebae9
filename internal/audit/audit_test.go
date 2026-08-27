package audit

import (
	"testing"
	"time"

	"caption-release-gate/internal/caption"
)

func TestEventChainDetectsMutation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first, err := NewEvent("pkg", 1, "created", "editor", now, map[string]string{"a": "b"}, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEvent("pkg", 2, "imported", "editor", now, map[string]int{"count": 2}, first.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyChain([]Event{first, second}); err != nil {
		t.Fatal(err)
	}
	second.ActorID = "intruder"
	if err := VerifyChain([]Event{first, second}); err == nil {
		t.Fatal("篡改后应校验失败")
	}
}

func TestManifestVerification(t *testing.T) {
	signer, err := NewSigner("local-v1", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	manifest, err := signer.Issue("pkg", "rev", "frozen", "rules", now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := signer.Verify(manifest); err != nil {
		t.Fatal(err)
	}
	manifest.FrozenDigest = "changed"
	if err := signer.Verify(manifest); err != ErrManifestTampered {
		t.Fatalf("预期篡改错误，实际 %v", err)
	}
	unknown := manifest
	unknown.IssuedBy = "another"
	if result := signer.Result(unknown, ""); result.Code != "unknown_issuer" {
		t.Fatalf("错误结果: %+v", result)
	}
	_ = caption.StatusDraft
}
