package utils

import (
	"fmt"
)

// Output 表示输出格式。
type Output int

const (
	// TextOutput 表示输出将以文本格式呈现。这是默认值。
	TextOutput Output = iota
	// JSONOutput 表示输出将以JSON格式呈现。
	JSONOutput
	// RawJSONOutput 表示输出将以原始JSON格式呈现。
	RawJSONOutput
)

const (
	textOutputString    = "text"
	jsonOutputString    = "json"
	jsonRawOutputString = "raw-json"
)

// Parse 从字符串解析输出格式。
func Parse(s string) (Output, error) {
	switch s {
	case textOutputString:
		return TextOutput, nil
	case jsonOutputString:
		return JSONOutput, nil
	case jsonRawOutputString:
		return RawJSONOutput, nil
	default:
		return TextOutput, fmt.Errorf("invalid output %q", s)
	}
}

// String 返回输出格式的字符串表示。
func (o Output) String() string {
	switch o {
	case TextOutput:
		return textOutputString
	case JSONOutput:
		return jsonOutputString
	case RawJSONOutput:
		return jsonRawOutputString
	default:
		return ""
	}
}
