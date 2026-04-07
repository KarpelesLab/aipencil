package pattern

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var paramRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// SubstituteParams replaces {{paramName}} placeholders in a JSON structure
// with values from the params map.
func SubstituteParams(data []byte, params map[string]any) ([]byte, error) {
	s := string(data)
	s = paramRegex.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-2]
		if val, ok := params[name]; ok {
			return formatParam(val)
		}
		return match // leave unreplaced if no value
	})
	return []byte(s), nil
}

func formatParam(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

// EvalCondition evaluates simple conditions for the "if" field.
// Supports: "param == 'value'", "param != 'value'", "param" (truthy check).
// Supports && (AND) and || (OR) to combine conditions.
func EvalCondition(expr string, params map[string]any) bool {
	expr = strings.TrimSpace(expr)

	// Handle || (OR) — lowest precedence
	if parts := strings.SplitN(expr, "||", 2); len(parts) == 2 {
		return EvalCondition(parts[0], params) || EvalCondition(parts[1], params)
	}

	// Handle && (AND)
	if parts := strings.SplitN(expr, "&&", 2); len(parts) == 2 {
		return EvalCondition(parts[0], params) && EvalCondition(parts[1], params)
	}

	// Check for == or !=
	for _, op := range []string{"!=", "=="} {
		if idx := strings.Index(expr, op); idx >= 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])
			right = strings.Trim(right, "'\"")

			leftVal := resolveParam(left, params)
			if op == "==" {
				return leftVal == right
			}
			return leftVal != right
		}
	}

	// Truthy check: non-empty and non-"false"
	val := resolveParam(expr, params)
	return val != "" && val != "false" && val != "0"
}

func resolveParam(name string, params map[string]any) string {
	if val, ok := params[name]; ok {
		return formatParam(val)
	}
	return ""
}
