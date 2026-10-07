package gatedtests

import (
	"strings"
	"unicode"
)

// Subject turns a gated test's own function name into a lower-case,
// space-separated phrase for cmd/screencensus to print beside it — the
// test's own name is already a full-sentence description (project
// convention, see PROSE.md's headings rule and every TestXxx name in this
// module), so the subject is derived mechanically from it rather than
// hand-authored a second time, which would drift the moment a test is
// renamed. The leading "Test" is dropped; every other CamelCase boundary
// becomes a space.
func Subject(t Test) string {
	name := strings.TrimPrefix(t.Func, "Test")
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prevLower := unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (unicode.IsUpper(runes[i-1]) && nextLower) {
				b.WriteByte(' ')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
