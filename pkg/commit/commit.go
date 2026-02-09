package commit

import "strings"

// From: https://raw.githubusercontent.com/conventional-commits/conventionalcommits.org/refs/heads/master/content/v1.0.0/index.md
const ConventionalSpec = `Conventional Commits 1.0.0 Spec
Summary
The Conventional Commits specification is a lightweight convention on top of commit messages. It provides an easy set of rules for creating an explicit commit history; which makes it easier to write automated tools on top of. This convention dovetails with SemVer, by describing the features, fixes, and breaking changes made in commit messages.

The commit message should be structured as follows:

<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]

The commit contains the following structural elements, to communicate intent to the consumers of your library:
- fix: a commit of the type fix patches a bug in your codebase (correlates with PATCH in Semantic Versioning).
- feat: a commit of the type feat introduces a new feature to the codebase (correlates with MINOR in Semantic Versioning).
- BREAKING CHANGE: a commit that has a footer BREAKING CHANGE:, or appends a ! after the type/scope, introduces a breaking API change (correlates with MAJOR in Semantic Versioning). A BREAKING CHANGE can be part of commits of any type.
- types other than fix and feat are allowed (e.g., build, chore, ci, docs, style, refactor, perf, test, and others).
- footers other than BREAKING CHANGE: may be provided and follow a convention similar to git trailer format.
- a scope may be provided to a commit's type and is contained within parenthesis, e.g., feat(parser): add ability to parse arrays.

Specification
1. Commits MUST be prefixed with a type, which consists of a noun (feat, fix, etc.), followed by the OPTIONAL scope, OPTIONAL !, and REQUIRED terminal colon and space.
2. The type feat MUST be used when a commit adds a new feature.
3. The type fix MUST be used when a commit represents a bug fix.
4. A scope MAY be provided after a type. A scope MUST consist of a noun describing a section of the codebase surrounded by parenthesis, e.g., fix(parser):
5. A description MUST immediately follow the colon and space after the type/scope prefix.
6. The description is a short summary of the code changes.
7. A longer commit body MAY be provided after the short description. The body MUST begin one blank line after the description.
8. A commit body is free-form and MAY consist of any number of newline separated paragraphs.
9. One or more footers MAY be provided one blank line after the body.
10. Each footer MUST consist of a word token, followed by either a : or # separator, followed by a string value (inspired by git trailer convention). A footer's token MUST use - in place of whitespace characters (e.g., Acked-by). An exception is made for BREAKING CHANGE which MAY also be used as a token.
11. A footer's value MAY contain spaces and newlines, and parsing MUST terminate when the next valid footer token/separator pair is observed.
12. Breaking changes MUST be indicated in the type/scope prefix of a commit, or as an entry in the footer.
13. If included as a footer, a breaking change MUST consist of the uppercase text BREAKING CHANGE, followed by a colon, space, and description.
14. If included in the type/scope prefix, breaking changes MUST be indicated by a ! immediately before the :. If ! is used, BREAKING CHANGE: MAY be omitted from the footer, and the commit description SHALL be used to describe the breaking change.
15. Types other than feat and fix MAY be used in your commit messages.
16. The units of information that make up Conventional Commits MUST NOT be treated as case sensitive by implementors, with the exception of BREAKING CHANGE which MUST be uppercase. BREAKING-CHANGE MUST be synonymous with BREAKING CHANGE, when used as a token in a footer.
`

const BodyLineWidth = 72

func WrapMessage(msg string, width int) string {
	paragraphs := strings.Split(msg, "\n\n")
	out := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			out = append(out, "")
			continue
		}
		run := strings.ReplaceAll(p, "\n", " ")
		var (
			line strings.Builder
			pos  int
		)
		for pos < len(run) {
			for pos < len(run) && run[pos] == ' ' {
				pos++
			}
			if pos >= len(run) {
				break
			}
			start := pos
			for pos < len(run) && run[pos] != ' ' {
				pos++
			}
			word := run[start:pos]
			newLen := line.Len()
			if newLen > 0 {
				newLen++
			}
			newLen += len(word)
			if newLen > width && line.Len() > 0 {
				lineStr := line.String()
				lastSent := -1
				for i := len(lineStr) - 1; i >= 0 && i >= len(lineStr)-width; i-- {
					if i > 0 && (lineStr[i] == '.' || lineStr[i] == '?' || lineStr[i] == '!') && lineStr[i-1] != '.' {
						if i+1 < len(lineStr) && lineStr[i+1] == ' ' {
							lastSent = i + 2
							break
						}
						if i+1 >= len(lineStr) {
							lastSent = i + 1
							break
						}
					}
				}
				var breakAt int
				if lastSent > 0 && lastSent <= len(lineStr) {
					breakAt = lastSent
				} else {
					lastSpace := strings.LastIndex(lineStr, " ")
					if lastSpace > 0 {
						breakAt = lastSpace + 1
					} else {
						breakAt = len(lineStr)
					}
				}
				out = append(out, strings.TrimSpace(lineStr[:breakAt]))
				line.Reset()
				line.WriteString(strings.TrimLeft(lineStr[breakAt:], " "))
			}
			if line.Len() > 0 {
				line.WriteByte(' ')
			}
			line.WriteString(word)
		}
		if line.Len() > 0 {
			out = append(out, line.String())
		}
	}
	result := strings.Join(out, "\n")
	firstBreak := strings.Index(result, "\n")
	if firstBreak == -1 {
		return result
	}
	rest := strings.TrimLeft(result[firstBreak+1:], "\n")
	if strings.TrimSpace(rest) == "" {
		return result[:firstBreak]
	}
	return result[:firstBreak] + "\n\n" + rest
}
