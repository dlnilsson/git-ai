package util

import "strings"

func SplitArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	return strings.Fields(raw)
}

func AddModelArg(args []string, model string) []string {
	if len(args) == 0 {
		return []string{"-m", model}
	}
	out := make([]string, 0, len(args)+2)
	if args[0] == "exec" {
		out = append(out, args[0], "-m", model)
		out = append(out, args[1:]...)
		return out
	}
	out = append(out, args...)
	out = append(out, "-m", model)
	return out
}

func ModelInList(name string, list []string) bool {
	for _, item := range list {
		if item == name {
			return true
		}
	}
	return false
}

func ToInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func ExtractJSONField(raw string, keys []string) string {
	for _, key := range keys {
		var (
			needle = `"` + key + `":`
			idx    = strings.Index(raw, needle)
		)
		if idx == -1 {
			continue
		}
		rest := raw[idx+len(needle):]
		rest = strings.TrimLeft(rest, " \n\r\t")
		if strings.HasPrefix(rest, "\"") {
			rest = rest[1:]
			out := strings.Builder{}
			escaped := false
			for _, r := range rest {
				if escaped {
					out.WriteRune(r)
					escaped = false
					continue
				}
				if r == '\\' {
					escaped = true
					continue
				}
				if r == '"' {
					return out.String()
				}
				out.WriteRune(r)
			}
		}
	}
	return ""
}
