package caption

import "strings"

func EnsureMutable(pkg CaptionPackage) error {
	if pkg.Status == StatusFrozen || pkg.Status == StatusReleased {
		return ErrInvalidState
	}
	return nil
}

func CanImportRevision(pkg CaptionPackage) error {
	if err := EnsureMutable(pkg); err != nil {
		return err
	}
	switch pkg.Status {
	case StatusDraft, StatusEditing, StatusRemediation, StatusReviewReady, StatusApproved:
		return nil
	default:
		return ErrInvalidState
	}
}

func RequestFindingException(finding *QualityFinding, reason string) error {
	if finding.Status != FindingOpen {
		return ErrInvalidState
	}
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 8 {
		return Invalid("reason", "例外理由至少需要 8 个字符")
	}
	finding.Status = FindingException
	finding.ExceptionReason = reason
	return nil
}

func ResolveFindings(findings []QualityFinding, findingIDs []string, revisionID string) error {
	if revisionID == "" {
		return Invalid("revisionId", "不能为空")
	}
	wanted := make(map[string]struct{}, len(findingIDs))
	for _, id := range findingIDs {
		wanted[id] = struct{}{}
	}
	for index := range findings {
		if _, ok := wanted[findings[index].FindingID]; !ok {
			continue
		}
		if findings[index].Status == FindingAccepted || findings[index].Status == FindingResolved {
			return ErrInvalidState
		}
		findings[index].Status = FindingResolved
		findings[index].ResolvedByRevisionID = revisionID
		delete(wanted, findings[index].FindingID)
	}
	if len(wanted) != 0 {
		return Invalid("findingIds", "包含不存在的问题")
	}
	return nil
}

func ValidateReview(pkg CaptionPackage, revision CaptionRevision, findings []QualityFinding, reviewer string, outcome ReviewOutcome, acceptedIDs []string) error {
	if pkg.Status != StatusReviewReady && pkg.Status != StatusRemediation {
		return ErrInvalidState
	}
	reviewer = strings.TrimSpace(reviewer)
	if reviewer == "" {
		return Invalid("reviewerId", "不能为空")
	}
	if reviewer == revision.SubmittedBy {
		return ErrActorNotIndependent
	}
	if outcome != ReviewApproved && outcome != ReviewReturned {
		return Invalid("outcome", "必须为 approved 或 returned")
	}
	accepted := make(map[string]struct{}, len(acceptedIDs))
	for _, id := range acceptedIDs {
		accepted[id] = struct{}{}
	}
	for _, finding := range findings {
		_, willAccept := accepted[finding.FindingID]
		if willAccept && finding.Status != FindingException {
			return Invalid("acceptedExceptionIds", "只能接受待审例外")
		}
		if outcome == ReviewApproved && finding.Severity == SeverityBlocking && finding.Status != FindingResolved && finding.Status != FindingAccepted && !willAccept {
			return ErrBlockingFindings
		}
		if outcome == ReviewApproved && finding.Status == FindingOpen {
			return Invalid("findings", "存在尚未处置的问题")
		}
	}
	return nil
}

func ApplyAcceptedExceptions(findings []QualityFinding, acceptedIDs []string) []QualityFinding {
	accepted := make(map[string]struct{}, len(acceptedIDs))
	for _, id := range acceptedIDs {
		accepted[id] = struct{}{}
	}
	result := append([]QualityFinding(nil), findings...)
	for index := range result {
		if _, ok := accepted[result[index].FindingID]; ok {
			result[index].Status = FindingAccepted
		}
	}
	return result
}

func CanFreeze(pkg CaptionPackage, review *ReviewDecision) error {
	if pkg.Status != StatusApproved || review == nil || review.Outcome != ReviewApproved {
		return ErrInvalidState
	}
	if review.RevisionID != pkg.CurrentRevisionID {
		return ErrInvalidState
	}
	return nil
}

func CanIssue(pkg CaptionPackage) error {
	if pkg.Status != StatusFrozen || pkg.FrozenDigest == "" {
		return ErrInvalidState
	}
	return nil
}

func ReopenFromFrozen(pkg *CaptionPackage) error {
	if pkg.Status != StatusFrozen && pkg.Status != StatusReleased {
		return ErrInvalidState
	}
	pkg.Status = StatusEditing
	pkg.FrozenDigest = ""
	return nil
}
