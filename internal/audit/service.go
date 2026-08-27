package audit

import (
	"fmt"
	"sync"
	"time"

	"caption-release-gate/internal/caption"
)

type Service struct {
	signer *Signer

	verificationMu    sync.RWMutex
	verificationCache map[string]VerificationResult
}

func NewService(issuer string, secret []byte) (*Service, error) {
	signer, err := NewSigner(issuer, secret)
	if err != nil {
		return nil, err
	}
	return &Service{
		signer:            signer,
		verificationCache: make(map[string]VerificationResult),
	}, nil
}

func (s *Service) BuildEvent(packageID, eventType, actorID string, at time.Time, data any, history []Event) (Event, error) {
	sequence, previous, err := NextLink(history)
	if err != nil {
		return Event{}, fmt.Errorf("审计链校验失败: %w", err)
	}
	return NewEvent(packageID, sequence, eventType, actorID, at, data, previous)
}

func (s *Service) Freeze(revision caption.CaptionRevision, findings []caption.QualityFinding, review caption.ReviewDecision, ruleDigest string, history []Event) (string, error) {
	if revision.RevisionID == "" || review.DecisionID == "" {
		return "", fmt.Errorf("冻结材料缺少修订或复核决定")
	}
	if review.RevisionID != revision.RevisionID {
		return "", fmt.Errorf("复核决定与当前修订不一致")
	}
	previous := ""
	if len(history) > 0 {
		if err := VerifyChain(history); err != nil {
			return "", err
		}
		previous = history[len(history)-1].Digest
	}
	return FrozenDigest(FrozenMaterial{PackageID: revision.PackageID, RevisionID: revision.RevisionID, Cues: revision.Cues, Findings: findings, Review: review, RuleSetDigest: ruleDigest, PreviousEventDigest: previous})
}

func (s *Service) Issue(packageID, revisionID, frozenDigest, ruleDigest string, approvedAt, issuedAt time.Time) (caption.ReleaseManifest, error) {
	return s.signer.Issue(packageID, revisionID, frozenDigest, ruleDigest, approvedAt, issuedAt)
}

func (s *Service) Verify(manifest caption.ReleaseManifest, expectedDigest string) VerificationResult {
	s.verificationMu.RLock()
	result, ok := s.verificationCache[manifest.PackageID]
	s.verificationMu.RUnlock()
	if ok {
		return result
	}

	result = s.signer.Result(manifest, expectedDigest)
	s.verificationMu.Lock()
	s.verificationCache[manifest.PackageID] = result
	s.verificationMu.Unlock()
	return result
}

func (s *Service) VerifyHistory(events []Event) error { return VerifyChain(events) }
func (s *Service) Issuer() string                     { return s.signer.Issuer() }
