package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// frozenModLines is the -check mod report as main.go wrote it before the
// report moved to game.ModCheckLines, kept to hold its bytes.
func frozenModLines(res modrt.Result) []string {
	lines := []string{fmt.Sprintf("againrom: %d mod(s) active on %s, mod-set %s", len(res.Ordered), res.Set.Base, res.Set.Digest())}
	for i, m := range res.Ordered {
		entry := res.Set.Mods[i]
		var settings []string
		for _, v := range entry.Settings {
			settings = append(settings, v.Key+"="+v.Value.String())
		}
		lines = append(lines, fmt.Sprintf("againrom: mod %d: %s %s %q applies-to=%s settings=%s dir=%s", i+1,
			m.Manifest.ID, m.Manifest.Version, m.Manifest.Title, strings.Join(m.Manifest.AppliesTo, ","),
			strings.Join(settings, ","), m.Dir))
	}
	lines = append(lines, fmt.Sprintf("againrom: rules skill_cap=%d", res.Rules.SkillCap()))
	for _, k := range mod.DataKinds() {
		if line := k.Summary(res.Data); line != "" {
			lines = append(lines, "againrom: "+line)
		}
	}
	if !res.Files.Empty() {
		var free []string
		for _, m := range res.Set.Mods {
			if m.StateFree {
				free = append(free, m.ID)
			}
		}
		lines = append(lines, fmt.Sprintf("againrom: install files=%d state-free=%s (a mod with no rule, no data file and no text file; the others are in a save's mod mark)", len(res.Files.Files), strings.Join(free, ",")))
	}
	return lines
}

// -check prints the same bytes for every fixture mod, alone and a set that
// loads together, on both bases. A mod that declares missions gets the map
// and pages folder cmd/modmission would copy out of an install.
func TestModLinesKeepTheirBytesOnTheFixtures(t *testing.T) {
	warnings := map[string][]string{
		"bone-pike":   {`mod "bone-pike": data/items.toml:7: sprite assets/sprites/bone-pike.png uses an old path; use assets/items/<name>.png`},
		"heavy-armor": {`mod "heavy-armor": data/items.toml:19: sprite assets/sprites/gambeson.png uses an old path; use assets/items/<name>.png`},
		"war-hammer-body": {
			`mod "war-hammer-body": data/bodies.toml:5: heroes bodies/hammer2h-heroes.png uses an old path; use assets/bodies/<name>.png`,
			`mod "war-hammer-body": data/bodies.toml:6: heroes_l bodies/hammer2h-heroes_l.png uses an old path; use assets/bodies/<name>.png`,
			`mod "war-hammer-body": data/weapon-bodies.toml: use data/wpnbodies.toml for weapon bodies`,
		},
	}
	entries, err := os.ReadDir(exampleModsDir)
	if err != nil {
		t.Fatal(err)
	}
	modsDir := t.TempDir()
	if err := os.CopyFS(modsDir, os.DirFS(exampleModsDir)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ids = append(ids, e.Name())
		if _, err := os.Stat(filepath.Join(modsDir, e.Name(), "data", "missions.toml")); err != nil {
			continue
		}
		for _, lang := range []string{"en", "ru"} {
			place := mod.PlaceLanguageMission(lang, mod.MissionStem("10"), false)
			at := filepath.Join(modsDir, e.Name(), filepath.FromSlash(place.Map))
			if err := os.MkdirAll(filepath.Join(modsDir, e.Name(), filepath.FromSlash(place.Pages(lang))), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(at, synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"}), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(ids) != 18 {
		t.Fatalf("%d fixture mods, want 18", len(ids))
	}
	sets := [][]string{}
	for _, id := range ids {
		sets = append(sets, []string{id})
	}
	sets = append(sets, []string{"fire-ball-tune", "skill-cap", "east-tavern", "war-glaive"})
	var refused []string
	for _, base := range []string{"rom1-en", "rom1-ru"} {
		for _, set := range sets {
			mods, err := mod.Resolve(modsDir, set)
			if err != nil {
				t.Fatalf("%v: %v", set, err)
			}
			res, err := modrt.Load(mods, base, nil, modrt.Options{})
			if err != nil {
				// The run refuses the set and -check prints no mod line.
				refused = append(refused, base+" "+strings.Join(set, ","))
				continue
			}
			var unchanged []string
			if want := warnings[strings.Join(set, ",")]; !slices.Equal(res.Warnings, want) {
				t.Errorf("%s %v warnings %q, want %q", base, set, res.Warnings, want)
			}
			for _, line := range modLines(res) {
				if strings.HasPrefix(line, "againrom: warning:") {
					t.Logf("%s %s %s", base, strings.Join(set, ","), line)
					continue
				}
				unchanged = append(unchanged, line)
			}
			if got, want := unchanged, frozenModLines(res); !slices.Equal(got, want) {
				t.Errorf("%s %v:\n%q\nwant\n%q", base, set, got, want)
			}
		}
	}
	if want := []string{"rom1-en east-town", "rom1-ru east-town"}; !slices.Equal(refused, want) {
		t.Fatalf("refused %q, want %q", refused, want)
	}
}
