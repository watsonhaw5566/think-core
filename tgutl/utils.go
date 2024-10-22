package tgutl

import (
	"strings"
)

// SubStringLast 截取某个字符串之后的字符串
func SubStringLast(str string, substr string) string {
	index := strings.Index(str, substr)
	if index < 0 {
		return ""
	}
	return str[index+len(substr):]
}
