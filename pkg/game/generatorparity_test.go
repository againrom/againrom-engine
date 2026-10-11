package game

import (
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"testing"
)

// generatorParityWaiver is a field one generator description sets and the
// other does not: the field's path, the description that lacks it and why
// that game has no such field. A path covers every field under it.
type generatorParityWaiver struct {
	path, lacking, reason string
}

// generatorParityWaivers are every field the two generator descriptions do
// not share. A field one description gains is a field the other must state
// or waive here.
var generatorParityWaivers = []generatorParityWaiver{
	{"detail.plate-seam", "rom2", "no second-game claim names a strip at the skill column's left edge beside the attribute plate (R2-ENGINE-286)"},
	{"detail.doll-seam", "rom2", "no second-game claim names a strip at the skill column's right edge beside the inventory (R2-ENGINE-286)"},
	{"detail.focus", "rom2", "the second game's detail page has no keyboard focus walk (DIV-2771)"},
	{"detail.keys.focus-keys", "rom2", "the second game's detail keys are Enter and Escape only (DIV-2771)"},
	{"pre-create.keys.focus-keys", "rom2", "the second game's pre-create keys are Enter, Escape and typing (DIV-2768)"},
	{"pre-create.keys.double-click-forward", "rom2", "no second-game claim names a picture double click (R2-ENGINE-283)"},
	{"detail.message", "rom2", "the second game's Accept refuses no name, so it has no message strip (R2-ENGINE-283)"},
	{"detail.refusals.empty-slot", "rom2", "the second game's Accept refuses no name (R2-ENGINE-283)"},
	{"detail.refusals.reserved-slot", "rom2", "the second game's Accept refuses no name (R2-ENGINE-283)"},
	{"detail.refusals.reserved", "rom2", "the second game's Accept refuses no name (R2-ENGINE-283)"},
	{"detail.repeat", "rom2", "no second-game claim names a held attribute button repeat"},
	{"pre-create.sparkle.gap-steps", "rom2", "the second game's sparkle waits a drawn time, not a step count (R2-ENGINE-284)"},
	{"pre-create.sparkle.tour", "rom2", "the second game's sparkle runs inside a drawn control, not along a tour (R2-ENGINE-284)"},
	{"detail.default-skill", "rom1", "the first game's skill row opens on the party skill slot, not a described default"},
	{"hover", "rom1", "the first game's generator hover help draws as every first-game hover help draws (MENU-128)"},
	{"pre-create.heroes[].template", "rom1", "the first game's pictures take npc.reg archetypes, not template rows (DIV-1277)"},
	{"pre-create.heroes[].neighbour", "rom1", "the first game's pictures share no art with a neighbour (TOWN-521)"},
	{"pre-create.heroes[].art[].size", "rom1", "the first game's hero art takes its installed size (TOWN-521)"},
	{"pre-create.heroes[].art[].when.neighbour-hovered", "rom1", "the first game's pictures share no art with a neighbour (TOWN-521)"},
	{"pre-create.heroes[].art[].when.neighbour-selected", "rom1", "the first game's pictures share no art with a neighbour (TOWN-521)"},
	{"pre-create.loops", "rom1", "the first game's pre-create has no torch loops (TOWN-216)"},
	{"pre-create.sparkle.gap-ms", "rom1", "the first game's sparkle waits a step count (DIV-1276)"},
	{"pre-create.sparkle.gap-draw-divisor", "rom1", "the first game's sparkle waits a step count (DIV-1276)"},
	{"pre-create.sparkle.within", "rom1", "the first game's sparkle runs along a tour (DIV-1276)"},
	{"tips.select[].section", "rom1", "the first game's tips are whole files, not town.txt sections (TOWN-518)"},
	{"tips.fighter.section", "rom1", "the first game's tips are whole files, not town.txt sections (TOWN-187)"},
	{"tips.mage.section", "rom1", "the first game's tips are whole files, not town.txt sections (TOWN-187)"},
	{"tips.after.section", "rom1", "the first game's tips are whole files, not town.txt sections (TOWN-187)"},
}

// generatorFieldPaths are the field paths a description sets: dotted object
// keys, "[]" for an array's members, without cite lists and without the
// campaign adapter's part, which no builder reads.
func generatorFieldPaths(data []byte) (map[string]bool, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	var walk func(v any, path string)
	walk = func(v any, path string) {
		switch v := v.(type) {
		case map[string]any:
			for k, x := range v {
				if k == "cite" || path == "" && k == "campaign" {
					continue
				}
				p := k
				if path != "" {
					p = path + "." + k
				}
				out[p] = true
				walk(x, p)
			}
		case []any:
			for _, x := range v {
				walk(x, path+"[]")
			}
		}
	}
	walk(v, "")
	return out, nil
}

// generatorParityGaps are the fields one description sets and the other
// neither sets nor waives, and the waivers no gap uses.
func generatorParityGaps(rom1, rom2 []byte) ([]string, error) {
	sets := map[string]map[string]bool{}
	for name, data := range map[string][]byte{"rom1": rom1, "rom2": rom2} {
		paths, err := generatorFieldPaths(data)
		if err != nil {
			return nil, err
		}
		sets[name] = paths
	}
	waived := func(path, lacking string) (int, bool) {
		for i, w := range generatorParityWaivers {
			if w.lacking == lacking && (path == w.path || strings.HasPrefix(path, w.path+".") || strings.HasPrefix(path, w.path+"[]")) {
				return i, true
			}
		}
		return -1, false
	}
	used := make([]bool, len(generatorParityWaivers))
	var gaps []string
	for _, pair := range [][2]string{{"rom1", "rom2"}, {"rom2", "rom1"}} {
		for path := range sets[pair[0]] {
			if sets[pair[1]][path] {
				continue
			}
			if i, ok := waived(path, pair[1]); ok {
				used[i] = true
				continue
			}
			gaps = append(gaps, pair[1]+" lacks "+path)
		}
	}
	for i, w := range generatorParityWaivers {
		if !used[i] {
			gaps = append(gaps, "waiver "+w.path+" for "+w.lacking+" covers no gap")
		}
	}
	sort.Strings(gaps)
	return gaps, nil
}

// The two generator descriptions set the same presentation fields: a field
// one sets and the other lacks fails unless the waiver table states why the
// other game has no such field, and a waiver that covers no gap fails too.
func TestGeneratorDescriptionsShareEveryField(t *testing.T) {
	gaps, err := generatorParityGaps(rom1GeneratorJSON, rom2GeneratorJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range gaps {
		t.Error(g)
	}
	for _, w := range generatorParityWaivers {
		if w.reason == "" || !slices.Contains([]string{"rom1", "rom2"}, w.lacking) {
			t.Errorf("waiver %s: lacking %q, reason %q", w.path, w.lacking, w.reason)
		}
	}
}

// The guard's loss control: the second description without any one of the
// fields it shares with the first fails the guard.
func TestGeneratorParityGuardCatchesARemovedField(t *testing.T) {
	removals := []struct {
		name   string
		remove func(d map[string]any)
	}{
		{"detail.card-offset", func(d map[string]any) { delete(d["detail"].(map[string]any), "card-offset") }},
		{"detail.card-seam", func(d map[string]any) { delete(d["detail"].(map[string]any), "card-seam") }},
		{"detail.restore", func(d map[string]any) { delete(d["detail"].(map[string]any), "restore") }},
		{"detail.stats.value-font", func(d map[string]any) {
			delete(d["detail"].(map[string]any)["stats"].(map[string]any), "value-font")
		}},
		{"detail.stats.value-shadow", func(d map[string]any) {
			delete(d["detail"].(map[string]any)["stats"].(map[string]any), "value-shadow")
		}},
		{"detail.stats.pool-at", func(d map[string]any) {
			delete(d["detail"].(map[string]any)["stats"].(map[string]any), "pool-at")
		}},
		{"pre-create.levels[].tooltip", func(d map[string]any) {
			for _, l := range d["pre-create"].(map[string]any)["levels"].([]any) {
				delete(l.(map[string]any), "tooltip")
			}
		}},
		{"pre-create.name.tooltip", func(d map[string]any) {
			delete(d["pre-create"].(map[string]any)["name"].(map[string]any), "tooltip")
		}},
	}
	for _, r := range removals {
		var d map[string]any
		if err := json.Unmarshal(rom2GeneratorJSON, &d); err != nil {
			t.Fatal(err)
		}
		r.remove(d)
		data, err := json.Marshal(d)
		if err != nil {
			t.Fatal(err)
		}
		gaps, err := generatorParityGaps(rom1GeneratorJSON, data)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(gaps, "rom2 lacks "+r.name) {
			t.Errorf("without %s the guard reports %v", r.name, gaps)
		}
	}
}
