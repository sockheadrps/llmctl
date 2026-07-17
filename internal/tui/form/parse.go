package form

import (
	"fmt"
	"strconv"
	"strings"
)

func splitCLIArgs(argsStr string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	started := false

	flush := func() {
		if started {
			tokens = append(tokens, cur.String())
			cur.Reset()
			started = false
		}
	}

	for _, r := range argsStr {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			started = true
		case r == '\\' && !inSingle:
			escaped = true
			started = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			started = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			started = true
		case !inSingle && !inDouble && (r == ' ' || r == '\t' || r == '\n' || r == '\r'):
			flush()
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	flush()
	return tokens
}

func isNumericToken(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	dotSeen := false
	digitSeen := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digitSeen = true
		case s[i] == '.':
			if dotSeen {
				return false
			}
			dotSeen = true
		default:
			return false
		}
	}
	return digitSeen
}

// ParseProfileArgs parses a space-separated CLI arg string (for example copied
// from export) and returns a map of field-index -> value. labelCount is the
// number of fields in the target form, defaultFlag maps a field index to its
// default llama-server CLI flag, flashIndex is the toggle row index, and
// extraArgsIndex is where any unrecognized arguments are stored.
func ParseProfileArgs(argsStr string, labelCount int, defaultFlag func(int) string, flashIndex int, extraArgsIndex int) (map[int]string, []string) {
	tokens := splitCLIArgs(argsStr)
	result := make(map[int]string)
	extra := make([]string, 0)
	sawRelevantFlag := false

	flagToField := make(map[string]int)
	for i := 0; i < labelCount; i++ {
		if f := defaultFlag(i); f != "" {
			flagToField[f] = i
		}
	}

	isKnownFlag := func(tok string) bool {
		if tok == "--flash-attn" {
			return true
		}
		if strings.HasPrefix(tok, "--no-") {
			return true
		}
		if eqIdx := strings.Index(tok, "="); eqIdx > 0 && strings.HasPrefix(tok, "-") {
			tok = tok[:eqIdx]
		}
		_, ok := flagToField[tok]
		return ok
	}

	consumeValue := func(i int) (string, bool, int) {
		if i+1 >= len(tokens) {
			return "", false, i
		}
		next := tokens[i+1]
		if isKnownFlag(next) {
			return "", false, i
		}
		if strings.HasPrefix(next, "-") && !isNumericToken(next) {
			return "", false, i
		}
		return next, true, i + 1
	}

	skipModelSource := func(i int) int {
		if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
			return i + 2
		}
		return i + 1
	}

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		// Drop the leading binary token from a pasted full command line.
		if i == 0 && !strings.HasPrefix(tok, "-") {
			i++
			continue
		}

		if tok == "--model" || tok == "-m" || tok == "-hf" {
			i = skipModelSource(i)
			continue
		}

		if tok == "llama-server" || tok == "llama-server.exe" {
			i++
			continue
		}

		// --flash-attn[=value]
		if tok == "--flash-attn" || strings.HasPrefix(tok, "--flash-attn=") {
			sawRelevantFlag = true
			val := "on"
			if eqIdx := strings.Index(tok, "="); eqIdx > 0 {
				raw := strings.ToLower(strings.TrimSpace(tok[eqIdx+1:]))
				if raw == "off" || raw == "false" || raw == "0" {
					val = "off"
				}
				result[flashIndex] = val
				i++
				continue
			}
			if next, ok, nextIdx := consumeValue(i); ok {
				raw := strings.ToLower(strings.TrimSpace(next))
				if raw == "off" || raw == "false" || raw == "0" {
					val = "off"
				}
				result[flashIndex] = val
				i = nextIdx + 1
				continue
			}
			result[flashIndex] = val
			i++
			continue
		}

		// --flag=value
		if eqIdx := strings.Index(tok, "="); eqIdx > 0 && strings.HasPrefix(tok, "-") {
			flagPart := tok[:eqIdx]
			if fieldIdx, ok := flagToField[flagPart]; ok {
				sawRelevantFlag = true
				result[fieldIdx] = tok[eqIdx+1:]
			} else {
				extra = append(extra, tok)
			}
			i++
			continue
		}

		// --no-flag -> bool false
		if strings.HasPrefix(tok, "--no-") {
			posFlag := "--" + tok[len("--no-"):]
			if fieldIdx, ok := flagToField[posFlag]; ok {
				sawRelevantFlag = true
				result[fieldIdx] = "false"
			} else {
				extra = append(extra, tok)
			}
			i++
			continue
		}

		// --flag value or bare --flag (= true)
		if strings.HasPrefix(tok, "-") {
			if fieldIdx, ok := flagToField[tok]; ok {
				sawRelevantFlag = true
				if next, ok, nextIdx := consumeValue(i); ok {
					result[fieldIdx] = next
					i = nextIdx + 1
				} else {
					result[fieldIdx] = "true"
					i++
				}
				continue
			}
		}

		if !sawRelevantFlag && !strings.HasPrefix(tok, "-") {
			i++
			continue
		}

		extra = append(extra, tok)
		i++
	}
	if len(extra) > 0 {
		result[extraArgsIndex] = strings.Join(extra, " ")
	}
	return result, extra
}

// ParseIntOrZero returns 0 for a blank field.
func ParseIntOrZero(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// ParsePort validates a positive port number.
func ParsePort(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("port must be a positive integer")
	}
	return n, nil
}

// ParseBoolPtr returns nil for a blank field.
func ParseBoolPtr(s string) (*bool, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ParseReasoning validates the reasoning mode.
func ParseReasoning(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "on", "off", "auto":
		return s, nil
	default:
		return "", fmt.Errorf("reasoning must be on, off, or auto")
	}
}

// ParseIntPtr returns nil for a blank field.
func ParseIntPtr(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ParseFloatPtr returns nil for a blank field.
func ParseFloatPtr(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
