package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"caption-release-gate/internal/caption"
)

type FrozenMaterial struct {
	PackageID           string                   `json:"packageId"`
	RevisionID          string                   `json:"revisionId"`
	Cues                []caption.CaptionCue     `json:"cues"`
	Findings            []caption.QualityFinding `json:"findings"`
	Review              caption.ReviewDecision   `json:"review"`
	RuleSetDigest       string                   `json:"ruleSetDigest"`
	PreviousEventDigest string                   `json:"previousEventDigest"`
}

func FrozenDigest(material FrozenMaterial) (string, error) {
	copyValue := material
	copyValue.Cues = append([]caption.CaptionCue(nil), material.Cues...)
	copyValue.Findings = append([]caption.QualityFinding(nil), material.Findings...)
	sort.Slice(copyValue.Cues, func(i, j int) bool {
		if copyValue.Cues[i].Sequence == copyValue.Cues[j].Sequence {
			return copyValue.Cues[i].CueID < copyValue.Cues[j].CueID
		}
		return copyValue.Cues[i].Sequence < copyValue.Cues[j].Sequence
	})
	sort.Slice(copyValue.Findings, func(i, j int) bool { return copyValue.Findings[i].FindingID < copyValue.Findings[j].FindingID })
	encoded, err := json.Marshal(copyValue)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func StableJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}
