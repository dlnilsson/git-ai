package providers

import "strings"

// FormatCommand renders a shell-safe command line with all arguments quoted as
// needed for display in dry-run mode and error messages.
func FormatCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(name))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return joinCommand(parts)
}

// FormatCommandWithStdin renders a reusable shell snippet that pipes the given
// stdin into the command.
func FormatCommandWithStdin(stdin string, name string, args []string) string {
	return "printf '%s' " + shellQuote(stdin) + " | " + FormatCommand(name, args)
}

func shellQuote(s string) string {
	if s != "" && strings.IndexFunc(s, func(r rune) bool {
		switch {
		case r >= 'a' && r <= 'z':
			return false
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case strings.ContainsRune("@%_+=:,./-", r):
			return false
		default:
			return true
		}
	}) == -1 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func joinCommand(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += " " + parts[i]
	}
	return out
}
