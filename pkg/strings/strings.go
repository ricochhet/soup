package strings

import (
	"bytes"
	"regexp"
	"strings"
)

func Bytes(str string, pad int) []byte {
	tmp := []byte(str)
	tmp = append(tmp, bytes.Repeat([]byte{0}, pad-(len(tmp)%pad))...)

	return tmp
}

func Match(data []byte, str string) bool { return regexp.MustCompile(str).Match(data) }
func Length(str string) int {
	re := regexp.MustCompile(`[\p{Han}\p{Katakana}\p{Hiragana}\p{Hangul}]`)
	return len(re.ReplaceAllString(str, "ab"))
}

func Strip(str string, value byte) string {
	if len(str) > 0 && str[0] == value {
		return str[1:]
	}

	return str
}

func Replace(str string, replace map[string]string) string {
	for k, v := range replace {
		str = strings.ReplaceAll(str, k, v)
	}

	return str
}

func Pad(str string) []byte {
	s := []byte(str)
	s = append(s, bytes.Repeat([]byte{0}, 2-(len(s)%2))...)

	return s
}

func StartsWithAny(str string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}

	return false
}

func EndsWithAny(str string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(str, suffix) {
			return true
		}
	}

	return false
}

func Box(str []string, draw func(string)) {
	var longest int

	for _, s := range str {
		if l := Length(s); l > longest {
			longest = l
		}
	}

	line := strings.Repeat("-", longest)
	draw("┌─" + line + "─┐")

	for _, txt := range str {
		spaceSize := longest - Length(txt)
		draw("│ " + txt + strings.Repeat(" ", spaceSize) + " │")
	}

	draw("└─" + line + "─┘")
}
