package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"caption-release-gate/internal/caption"
)

const ManifestVersion = "release-manifest/1"

var (
	ErrManifestTampered = errors.New("清单内容或签名已被篡改")
	ErrUnknownIssuer    = errors.New("未知签发标识")
)

type Signer struct {
	issuer string
	secret []byte
}

func NewSigner(issuer string, secret []byte) (*Signer, error) {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return nil, fmt.Errorf("签发标识不能为空")
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("签发密钥至少需要 16 字节")
	}
	return &Signer{issuer: issuer, secret: append([]byte(nil), secret...)}, nil
}

func (s *Signer) Issuer() string { return s.issuer }

func (s *Signer) Issue(packageID, revisionID, frozenDigest, ruleDigest string, approvedAt, issuedAt time.Time) (caption.ReleaseManifest, error) {
	if packageID == "" || revisionID == "" || frozenDigest == "" || ruleDigest == "" {
		return caption.ReleaseManifest{}, fmt.Errorf("签发材料不完整")
	}
	manifest := caption.ReleaseManifest{ManifestID: manifestID(packageID, frozenDigest), PackageID: packageID, RevisionID: revisionID, FrozenDigest: frozenDigest, RuleSetDigest: ruleDigest, ApprovedAt: approvedAt.UTC(), IssuedBy: s.issuer, IssuedAt: issuedAt.UTC()}
	tag, err := s.tag(manifest)
	if err != nil {
		return caption.ReleaseManifest{}, err
	}
	manifest.VerificationTag = tag
	return manifest, nil
}

func (s *Signer) Verify(manifest caption.ReleaseManifest) error {
	if manifest.IssuedBy != s.issuer {
		return ErrUnknownIssuer
	}
	provided := manifest.VerificationTag
	expected, err := s.tag(manifest)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return ErrManifestTampered
	}
	return nil
}

func (s *Signer) tag(manifest caption.ReleaseManifest) (string, error) {
	manifest.VerificationTag = ""
	payload := struct {
		Version  string                  `json:"version"`
		Manifest caption.ReleaseManifest `json:"manifest"`
	}{ManifestVersion, manifest}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write(encoded)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func manifestID(packageID, digest string) string {
	sum := sha256.Sum256([]byte(ManifestVersion + "|" + packageID + "|" + digest))
	return "manifest_" + base64.RawURLEncoding.EncodeToString(sum[:12])
}

type VerificationResult struct {
	Valid   bool   `json:"valid"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Signer) Result(manifest caption.ReleaseManifest, expectedDigest string) VerificationResult {
	if manifest.IssuedBy != s.issuer {
		return VerificationResult{Code: "unknown_issuer", Message: ErrUnknownIssuer.Error()}
	}
	if expectedDigest != "" && manifest.FrozenDigest != expectedDigest {
		return VerificationResult{Code: "content_mismatch", Message: "清单摘要与已冻结内容不一致"}
	}
	if err := s.Verify(manifest); err != nil {
		return VerificationResult{Code: "tampered", Message: err.Error()}
	}
	return VerificationResult{Valid: true, Code: "valid", Message: "清单签名与冻结内容均有效"}
}
