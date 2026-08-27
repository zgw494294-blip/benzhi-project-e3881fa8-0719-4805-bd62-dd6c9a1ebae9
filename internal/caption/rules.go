package caption

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"unicode/utf8"
)

const RuleVersion = "caption-qc/1.0.0"

const (
	RuleOverlap    = "TIMING_OVERLAP"
	RuleGap        = "TIMING_MIN_GAP"
	RuleDuration   = "DISPLAY_DURATION"
	RuleCPS        = "CHARACTERS_PER_SECOND"
	RuleLineLength = "LINE_LENGTH"
	RuleEmpty      = "EMPTY_TEXT"
	RuleOrder      = "READING_ORDER"
)

type RuleConfig struct {
	MinimumGapFrames int64   `json:"minimumGapFrames"`
	MinimumDuration  float64 `json:"minimumDurationSeconds"`
	MaximumDuration  float64 `json:"maximumDurationSeconds"`
	MaximumCPS       float64 `json:"maximumCharactersPerSecond"`
	MaximumLineRunes int     `json:"maximumLineRunes"`
}

func DefaultRuleConfig() RuleConfig {
	return RuleConfig{MinimumGapFrames: 2, MinimumDuration: 1.0, MaximumDuration: 7.0, MaximumCPS: 20, MaximumLineRunes: 42}
}

func RuleSetDigest(config RuleConfig) string {
	payload := fmt.Sprintf("%s|%d|%.3f|%.3f|%.3f|%d", RuleVersion, config.MinimumGapFrames, config.MinimumDuration, config.MaximumDuration, config.MaximumCPS, config.MaximumLineRunes)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func RunQualityChecks(revision CaptionRevision, rate FrameRate, config RuleConfig) ([]QualityFinding, CheckSummary) {
	findings := make([]QualityFinding, 0)
	byTimeline := append([]CaptionCue(nil), revision.Cues...)
	sort.Slice(byTimeline, func(i, j int) bool {
		if byTimeline[i].StartFrame == byTimeline[j].StartFrame {
			return byTimeline[i].CueID < byTimeline[j].CueID
		}
		return byTimeline[i].StartFrame < byTimeline[j].StartFrame
	})
	for index, cue := range revision.Cues {
		if cue.NormalizedText == "" {
			findings = append(findings, newFinding(revision.RevisionID, RuleEmpty, SeverityBlocking, []string{cue.CueID}, "字幕文本为空"))
		}
		duration := FrameDurationSeconds(cue.EndFrame-cue.StartFrame, rate)
		if duration < config.MinimumDuration {
			findings = append(findings, newFinding(revision.RevisionID, RuleDuration, SeverityWarning, []string{cue.CueID}, fmt.Sprintf("显示时长 %.2f 秒，短于 %.2f 秒", duration, config.MinimumDuration)))
		}
		if duration > config.MaximumDuration {
			findings = append(findings, newFinding(revision.RevisionID, RuleDuration, SeverityWarning, []string{cue.CueID}, fmt.Sprintf("显示时长 %.2f 秒，长于 %.2f 秒", duration, config.MaximumDuration)))
		}
		if duration > 0 {
			cps := float64(utf8.RuneCountInString(cue.NormalizedText)) / duration
			if cps > config.MaximumCPS {
				findings = append(findings, newFinding(revision.RevisionID, RuleCPS, SeverityWarning, []string{cue.CueID}, fmt.Sprintf("每秒字符数 %.1f，超过 %.1f", cps, config.MaximumCPS)))
			}
		}
		for lineNumber, line := range cue.Lines {
			length := utf8.RuneCountInString(line)
			if length > config.MaximumLineRunes {
				findings = append(findings, newFinding(revision.RevisionID, RuleLineLength, SeverityWarning, []string{cue.CueID}, fmt.Sprintf("第 %d 行长度 %d，超过 %d", lineNumber+1, length, config.MaximumLineRunes)))
			}
		}
		if index > 0 {
			previous := revision.Cues[index-1]
			if cue.StartFrame < previous.StartFrame || cue.Sequence <= previous.Sequence {
				findings = append(findings, newFinding(revision.RevisionID, RuleOrder, SeverityBlocking, []string{previous.CueID, cue.CueID}, "序号顺序与时间轴阅读顺序不一致"))
			}
		}
	}
	for index := 1; index < len(byTimeline); index++ {
		previous, current := byTimeline[index-1], byTimeline[index]
		gap := current.StartFrame - previous.EndFrame
		if gap < 0 {
			findings = append(findings, newFinding(revision.RevisionID, RuleOverlap, SeverityBlocking, []string{previous.CueID, current.CueID}, fmt.Sprintf("时间重叠 %d 帧", -gap)))
		} else if gap < config.MinimumGapFrames {
			findings = append(findings, newFinding(revision.RevisionID, RuleGap, SeverityWarning, []string{previous.CueID, current.CueID}, fmt.Sprintf("间隔 %d 帧，少于 %d 帧", gap, config.MinimumGapFrames)))
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleCode == findings[j].RuleCode {
			return findings[i].FindingID < findings[j].FindingID
		}
		return findings[i].RuleCode < findings[j].RuleCode
	})
	summary := CheckSummary{RevisionID: revision.RevisionID, RuleVersion: RuleVersion, RuleDigest: RuleSetDigest(config), Total: len(findings)}
	for _, finding := range findings {
		if finding.Severity == SeverityBlocking {
			summary.Blocking++
		} else {
			summary.Warnings++
		}
	}
	return findings, summary
}

func newFinding(revisionID, code string, severity FindingSeverity, cueIDs []string, message string) QualityFinding {
	seed := revisionID + "|" + code + "|" + fmt.Sprint(cueIDs) + "|" + message
	sum := sha256.Sum256([]byte(seed))
	return QualityFinding{FindingID: "finding_" + hex.EncodeToString(sum[:8]), RevisionID: revisionID, CueIDs: append([]string(nil), cueIDs...), RuleCode: code, RuleVersion: RuleVersion, Severity: severity, Message: message, Status: FindingOpen}
}
