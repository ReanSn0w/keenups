package nutserver

import (
	"fmt"
	"strings"
)

func splitLine(line string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuote := false
	escaped := false
	hasToken := false
	flush := func() {
		if hasToken {
			tokens = append(tokens, current.String())
			current.Reset()
			hasToken = false
		}
	}
	for _, r := range strings.TrimSpace(line) {
		if escaped {
			current.WriteRune(r)
			escaped = false
			hasToken = true
			continue
		}
		if r == '\\' && inQuote {
			escaped = true
			continue
		}
		if r == '"' {
			inQuote = !inQuote
			hasToken = true
			continue
		}
		if (r == ' ' || r == '\t') && !inQuote {
			flush()
			continue
		}
		current.WriteRune(r)
		hasToken = true
	}
	if escaped || inQuote {
		return nil, fmt.Errorf("unterminated quoted value")
	}
	flush()
	return tokens, nil
}

func quote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return "\"" + value + "\""
}
