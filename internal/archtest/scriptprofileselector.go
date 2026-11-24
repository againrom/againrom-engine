package archtest

import (
	"sort"
	"strings"
)

// ScriptProfileSelectorViolations reports the sources (path to text) that
// declare a script profile selector other than scriptprofile.For, or that
// call one. The selector takes the operation vocabulary and the extended flag.
func ScriptProfileSelectorViolations(files map[string]string) []string {
	var out []string
	for name, src := range files {
		if strings.HasPrefix(name, "pkg/scriptprofile/") {
			if strings.Contains(src, "func Of(") || strings.Contains(src, "func ForOps(") {
				out = append(out, name+": a second script profile selector is declared")
			}
			continue
		}
		// The needles are split so this file does not match itself.
		if strings.Contains(src, "scriptprofile"+".Of(") || strings.Contains(src, "scriptprofile"+".ForOps(") {
			out = append(out, name+": calls a retired script profile selector")
		}
		// The game and the editor window take a map's limits from its editor's Profile, not a selector.
		if (strings.HasPrefix(name, "pkg/game/") || strings.HasPrefix(name, "pkg/ui/")) && strings.Contains(src, "scriptprofile"+".For(") {
			out = append(out, name+": selects a script profile; take the map's Profile")
		}
	}
	sort.Strings(out)
	return out
}
