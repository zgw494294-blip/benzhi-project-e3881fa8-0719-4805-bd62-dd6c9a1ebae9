package caption

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

type CueInput struct {
	CueID         string   `json:"cueId"`
	Sequence      int      `json:"sequence"`
	StartTimecode string   `json:"startTimecode"`
	EndTimecode   string   `json:"endTimecode"`
	Speaker       string   `json:"speaker"`
	Lines         []string `json:"lines"`
}

func NormalizeCues(inputs []CueInput, rate FrameRate, mode TimecodeMode) ([]CaptionCue, string, error) {
	if len(inputs) == 0 {
		return nil, "", Invalid("cues", "至少需要一条字幕")
	}
	seenIDs := make(map[string]struct{}, len(inputs))
	seenSequences := make(map[int]struct{}, len(inputs))
	cues := make([]CaptionCue, 0, len(inputs))
	for i, input := range inputs {
		input.CueID = strings.TrimSpace(input.CueID)
		if input.CueID == "" {
			return nil, "", Invalid("cueId", "不能为空")
		}
		if _, exists := seenIDs[input.CueID]; exists {
			return nil, "", ErrDuplicateCue
		}
		seenIDs[input.CueID] = struct{}{}
		if input.Sequence <= 0 {
			return nil, "", Invalid("sequence", "必须为正整数")
		}
		if _, exists := seenSequences[input.Sequence]; exists {
			return nil, "", Invalid("sequence", "序号重复")
		}
		seenSequences[input.Sequence] = struct{}{}
		start, startCode, err := ParseTimecode(input.StartTimecode, rate, mode)
		if err != nil {
			return nil, "", Invalid("cues", cueLocation(i, err.Error()))
		}
		end, endCode, err := ParseTimecode(input.EndTimecode, rate, mode)
		if err != nil {
			return nil, "", Invalid("cues", cueLocation(i, err.Error()))
		}
		if end <= start {
			return nil, "", Invalid("endTimecode", cueLocation(i, "出点必须晚于入点"))
		}
		lines := normalizeLines(input.Lines)
		text := strings.TrimSpace(strings.Join(lines, " "))
		cues = append(cues, CaptionCue{CueID: input.CueID, Sequence: input.Sequence, StartTimecode: startCode, EndTimecode: endCode, StartFrame: start, EndFrame: end, Speaker: strings.TrimSpace(input.Speaker), Lines: lines, NormalizedText: text})
	}
	sort.Slice(cues, func(i, j int) bool {
		if cues[i].Sequence == cues[j].Sequence {
			return cues[i].CueID < cues[j].CueID
		}
		return cues[i].Sequence < cues[j].Sequence
	})
	digest, err := CueDigest(cues)
	if err != nil {
		return nil, "", err
	}
	return cues, digest, nil
}

func cueLocation(index int, message string) string {
	return "第 " + strconvItoa(index+1) + " 条：" + message
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[position:])
}

func normalizeLines(lines []string) []string {
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Map(func(r rune) rune {
			if r == '\r' || r == '\n' {
				return -1
			}
			if unicode.IsSpace(r) {
				return ' '
			}
			return r
		}, strings.TrimSpace(line))
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func CueDigest(cues []CaptionCue) (string, error) {
	data, err := json.Marshal(cues)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
