package caption

import (
	"fmt"
	"strconv"
	"strings"
)

type TimecodeMode string

const (
	TimecodeNonDrop TimecodeMode = "non_drop"
	TimecodeDrop    TimecodeMode = "drop_frame"
)

type FrameRate struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

var supportedRates = map[string]FrameRate{
	"24":     {Numerator: 24, Denominator: 1},
	"25":     {Numerator: 25, Denominator: 1},
	"30":     {Numerator: 30, Denominator: 1},
	"23.976": {Numerator: 24000, Denominator: 1001},
	"29.97":  {Numerator: 30000, Denominator: 1001},
}

func ParseFrameRate(value string) (FrameRate, error) {
	rate, ok := supportedRates[strings.TrimSpace(value)]
	if !ok {
		return FrameRate{}, Invalid("frameRate", "仅支持 24、25、30、23.976 或 29.97")
	}
	return rate, nil
}

func (r FrameRate) String() string {
	for name, candidate := range supportedRates {
		if r == candidate {
			return name
		}
	}
	return fmt.Sprintf("%d/%d", r.Numerator, r.Denominator)
}

func (r FrameRate) Nominal() int {
	if r.Numerator == 24000 {
		return 24
	}
	if r.Numerator == 30000 {
		return 30
	}
	return r.Numerator / r.Denominator
}

func (r FrameRate) FramesPerSecond() float64 {
	return float64(r.Numerator) / float64(r.Denominator)
}

func ValidateMode(rate FrameRate, mode TimecodeMode) error {
	if mode != TimecodeNonDrop && mode != TimecodeDrop {
		return Invalid("timecodeMode", "必须为 non_drop 或 drop_frame")
	}
	if mode == TimecodeDrop && rate != supportedRates["29.97"] {
		return Invalid("timecodeMode", "drop_frame 仅可用于 29.97 帧率")
	}
	return nil
}

func ParseTimecode(value string, rate FrameRate, mode TimecodeMode) (int64, string, error) {
	if err := ValidateMode(rate, mode); err != nil {
		return 0, "", err
	}
	separator := ":"
	if mode == TimecodeDrop {
		separator = ";"
	}
	normalized := strings.TrimSpace(value)
	if mode == TimecodeDrop {
		if len(normalized) != 11 || normalized[8] != ';' {
			return 0, "", Invalid("timecode", "drop_frame 格式应为 HH:MM:SS;FF")
		}
		normalized = normalized[:8] + ":" + normalized[9:]
	}
	parts := strings.Split(normalized, ":")
	if len(parts) != 4 {
		return 0, "", Invalid("timecode", "格式应为 HH:MM:SS:FF")
	}
	values := make([]int, 4)
	for i, part := range parts {
		if len(part) != 2 {
			return 0, "", Invalid("timecode", "每段必须为两位数字")
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0, "", Invalid("timecode", "包含非数字字符")
		}
		values[i] = n
	}
	hours, minutes, seconds, frames := values[0], values[1], values[2], values[3]
	nominal := rate.Nominal()
	if hours > 23 || minutes > 59 || seconds > 59 || frames >= nominal {
		return 0, "", Invalid("timecode", "时、分、秒或帧超出范围")
	}
	total := int64(((hours*60+minutes)*60+seconds)*nominal + frames)
	if mode == TimecodeDrop {
		dropped := 2 * (hours*60 + minutes - (hours*60+minutes)/10)
		if seconds == 0 && minutes%10 != 0 && frames < 2 {
			return 0, "", Invalid("timecode", "drop_frame 分钟边界不存在 00 和 01 帧")
		}
		total -= int64(dropped)
	}
	canonical := fmt.Sprintf("%02d:%02d:%02d%s%02d", hours, minutes, seconds, separator, frames)
	return total, canonical, nil
}

func FrameDurationSeconds(frames int64, rate FrameRate) float64 {
	return float64(frames) * float64(rate.Denominator) / float64(rate.Numerator)
}
