package game

import (
	"fmt"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// ModCheckLines is the report of the mods a run loaded: one line for the
// set, one per mod in load order with its settings, the rules the mods set,
// a summary per data kind they hold and the install files they serve.
// againrom -check prints each line after "againrom: " and the map editor's
// overview of a mod shows them.
func ModCheckLines(res modrt.Result) []string {
	lines := []string{fmt.Sprintf("%d mod(s) active on %s, mod-set %s", len(res.Ordered), res.Set.Base, res.Set.Digest())}
	for i, m := range res.Ordered {
		entry := res.Set.Mods[i]
		var settings []string
		for _, v := range entry.Settings {
			settings = append(settings, v.Key+"="+v.Value.String())
		}
		lines = append(lines, fmt.Sprintf("mod %d: %s %s %q applies-to=%s settings=%s dir=%s", i+1,
			m.Manifest.ID, m.Manifest.Version, m.Manifest.Title, strings.Join(m.Manifest.AppliesTo, ","),
			strings.Join(settings, ","), m.Dir))
	}
	lines = append(lines, fmt.Sprintf("rules skill_cap=%d", res.Rules.SkillCap()))
	for _, k := range mod.DataKinds() {
		if line := k.Summary(res.Data); line != "" {
			lines = append(lines, line)
		}
	}
	if !res.Files.Empty() {
		var free []string
		for _, m := range res.Set.Mods {
			if m.StateFree {
				free = append(free, m.ID)
			}
		}
		lines = append(lines, fmt.Sprintf("install files=%d state-free=%s (a mod with no rule, no data file and no text file; the others are in a save's mod mark)", len(res.Files.Files), strings.Join(free, ",")))
	}
	return lines
}
