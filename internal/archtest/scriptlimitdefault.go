package archtest

import (
	"regexp"
	"sort"
	"strings"
)

// firstProfile matches the first game's profile by name, not its constants
// such as FirstRegisters.
var firstProfile = regexp.MustCompile(`scriptprofile\.First\b`)

// firstScriptVariable matches a package-wide variable that holds the first game's profile.
var firstScriptVariable = regexp.MustCompile(`var script = scriptprofile\.First\b`)

// ScriptLimitDefaultViolations reports the sources that take script limits
// from the first game by name instead of the map's profile or the dialect
// table: the window model and the simulation fall back to scriptprofile.First,
// and the editor keeps no package-wide profile variable.
func ScriptLimitDefaultViolations(files map[string]string) []string {
	var out []string
	for name, src := range files {
		if (strings.HasPrefix(name, "pkg/ui/") || strings.HasPrefix(name, "pkg/sim/")) && firstProfile.MatchString(src) {
			out = append(out, name+": takes the first game's script limits by name; take the map's Profile or the dialect table")
		}
		if strings.HasPrefix(name, "pkg/mapedit/") && firstScriptVariable.MatchString(src) {
			out = append(out, name+": a package-wide script limit variable; take the editor's Profile")
		}
	}
	sort.Strings(out)
	return out
}
