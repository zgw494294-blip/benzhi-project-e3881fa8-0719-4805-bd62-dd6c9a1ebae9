package caption

import "testing"

func TestNormalizeAndRules(t *testing.T) {
	rate, err := ParseFrameRate("25")
	if err != nil {
		t.Fatal(err)
	}
	cues, _, err := NormalizeCues([]CueInput{
		{CueID: "a", Sequence: 1, StartTimecode: "00:00:01:00", EndTimecode: "00:00:01:10", Lines: []string{"短字幕"}},
		{CueID: "b", Sequence: 2, StartTimecode: "00:00:01:09", EndTimecode: "00:00:03:00", Lines: []string{"下一条"}},
	}, rate, TimecodeNonDrop)
	if err != nil {
		t.Fatal(err)
	}
	revision := CaptionRevision{RevisionID: "r1", Cues: cues}
	findings, summary := RunQualityChecks(revision, rate, DefaultRuleConfig())
	if summary.Total == 0 || len(findings) == 0 {
		t.Fatal("预期生成质量问题")
	}
	foundOverlap := false
	for _, finding := range findings {
		if finding.RuleCode == RuleOverlap {
			foundOverlap = true
		}
	}
	if !foundOverlap {
		t.Fatal("缺少重叠问题")
	}
}

func TestDropFrameBoundary(t *testing.T) {
	rate, _ := ParseFrameRate("29.97")
	if _, _, err := ParseTimecode("00:01:00;00", rate, TimecodeDrop); err == nil {
		t.Fatal("应拒绝不存在的 drop frame")
	}
	if _, normalized, err := ParseTimecode("00:10:00;00", rate, TimecodeDrop); err != nil || normalized != "00:10:00;00" {
		t.Fatalf("十分钟边界应有效: %v", err)
	}
}
