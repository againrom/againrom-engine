package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	basepkg "againrom/pkg/base"
	"againrom/pkg/mod"
)

// servedBases are the profiles the starter serves, in the order baseRig lists
// them.
var servedBases = []string{basepkg.ROM1EN, basepkg.ROM1RU, basepkg.ROM2EN, basepkg.ROM2RU}

// baseRig is a rig whose settings list one exact install of each served
// profile, keyed by its id, with rom1-en selected, the enabled list and the
// mods written into the mods folder beside the starter.
func baseRig(t *testing.T, enabled []string, mods map[string]string) *rig {
	t.Helper()
	r := newRig(t, "")
	var ini strings.Builder
	ini.WriteString("[starter]\nlast-base = " + basepkg.ROM1EN + "\n[bases]\n")
	for _, id := range servedBases {
		root := filepath.Join(r.dir, id)
		r.fakes.validRoots[root] = profileInfo(id, true)
		fmt.Fprintf(&ini, "%s = %s\n", id, root)
	}
	fmt.Fprintf(&ini, "[mods]\nenabled = %s\n", strings.Join(enabled, ","))
	if err := os.WriteFile(r.ini, []byte(ini.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	for id, m := range mods {
		writeModFolder(t, filepath.Join(r.dir, "mods"), id, m)
	}
	return r
}

// manifest is a usable mod.toml of id with the extra lines.
func manifest(id string, lines ...string) string {
	return "id = \"" + id + "\"\ntitle = \"T " + id + "\"\nversion = \"1.0\"\n" + strings.Join(lines, "\n") + "\n"
}

// selectBase clicks the base row of profile id.
func selectBase(t *testing.T, a *app, id string) {
	t.Helper()
	for i, b := range a.s.Bases {
		if b.Key == id {
			do(t, a, action{aSelectBase, i})
			return
		}
	}
	t.Fatalf("no base %s", id)
}

// rowsByID are the mod rows by mod id.
func rowsByID(a *app) map[string]modRow {
	out := map[string]modRow{}
	for _, r := range a.modRows() {
		out[r.id] = r
	}
	return out
}

// enabledIDs are the ids of the enabled rows in list order.
func enabledIDs(a *app) []string {
	var out []string
	for _, r := range a.modRows() {
		if r.enabled {
			out = append(out, r.id)
		}
	}
	return out
}

// orderOf is mod.Order's load order of the enabled mods of the folder on base,
// read by the game's own reader.
func orderOf(t *testing.T, dir string, enabled []string, base string) []string {
	t.Helper()
	entries, err := mod.Resolve(dir, enabled)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range mod.Order(entries, base).Mods {
		out = append(out, e.Manifest.ID)
	}
	return out
}

// The enabled rows stand in mod.Order's load order on each served base, common
// mods first, whatever order they were ticked in; changing the base changes
// which rows are refused and the next mod after them.
func TestEnabledModsStandInTheGamesLoadOrderOnEveryBase(t *testing.T) {
	mods := map[string]string{
		"a-addon":  manifest("a-addon", `requires = ["b-core>=1.0"]`),
		"b-core":   manifest("b-core"),
		"c-first":  manifest("c-first", `applies-to = ["rom1"]`),
		"d-second": manifest("d-second", `applies-to = ["rom2-en", "rom2-ru"]`),
		"e-ru":     manifest("e-ru", `applies-to = ["rom1-ru", "rom2-ru"]`, `load-after = ["c-first", "d-second"]`),
		"f-late":   manifest("f-late", `load-after = ["c-first"]`),
		"g-spare":  manifest("g-spare", `applies-to = ["rom2-en"]`),
	}
	ticked := []string{"f-late", "e-ru", "d-second", "c-first", "b-core", "a-addon"}
	r := baseRig(t, ticked, mods)
	a := r.app()
	want := []string{"b-core", "a-addon", "c-first", "f-late", "d-second", "e-ru"}
	refused := map[string][]string{
		basepkg.ROM1EN: {"d-second", "e-ru", "g-spare"},
		basepkg.ROM1RU: {"d-second", "g-spare"},
		basepkg.ROM2EN: {"c-first", "e-ru"},
		basepkg.ROM2RU: {"c-first", "g-spare"},
	}
	for _, id := range servedBases {
		selectBase(t, a, id)
		if got := enabledIDs(a); !reflect.DeepEqual(got, want) || !reflect.DeepEqual(got, orderOf(t, filepath.Join(r.dir, "mods"), ticked, id)) {
			t.Errorf("%s: enabled rows %v, want %v (mod.Order)", id, got, want)
		}
		rows := a.modRows()
		if last := rows[len(rows)-1]; last.id != "g-spare" || last.enabled || last.order != 0 {
			t.Errorf("%s: the mod that is not enabled does not follow: %+v", id, last)
		}
		var bad []string
		for i, row := range rows {
			if row.enabled && row.order != i+1 {
				t.Errorf("%s: row %d is numbered %d", id, i, row.order)
			}
			if row.refusal != "" {
				bad = append(bad, row.id)
				if want := a.t(cRefusedBase, id); row.refusal != want || !strings.Contains(row.label, want) {
					t.Errorf("%s: row %s refusal %q label %q, want %q", id, row.id, row.refusal, row.label, want)
				}
			}
		}
		if !reflect.DeepEqual(bad, refused[id]) {
			t.Errorf("%s: refused rows %v, want %v", id, bad, refused[id])
		}
	}
}

// Every refusal mod.Order raises stands, worded by the caption table with the
// mod ids as data, on the row of the mod it names, before Play; Play refuses
// the set naming the first refused row, and a mod that is not enabled says it
// does not fit the base. Selecting the base where they fit clears those rows.
func TestEveryRefusalStandsOnTheRowOfItsMod(t *testing.T) {
	mods := map[string]string{
		"api2":     manifest("api2", "api = 2"),
		"clash":    manifest("clash", `conflicts = ["core"]`),
		"core":     manifest("core"),
		"loop-a":   manifest("loop-a", `load-after = ["loop-b"]`),
		"loop-b":   manifest("loop-b", `load-after = ["loop-a"]`),
		"needy":    manifest("needy", `requires = ["ghost"]`),
		"old":      manifest("old", `requires = ["core>=2"]`),
		"ru-only":  manifest("ru-only", `applies-to = ["rom1-ru"]`),
		"spare":    manifest("spare"),
		"spare-ru": manifest("spare-ru", `applies-to = ["rom1-ru"]`),
	}
	r := baseRig(t, []string{"api2", "clash", "core", "loop-a", "loop-b", "needy", "old", "ru-only"}, mods)
	a := r.app()
	want := map[string]string{
		"api2":     caption("en", cRefusedAPI, 2, mod.SupportedAPI),
		"clash":    caption("en", cRefusedConflict, "core"),
		"core":     "",
		"loop-a":   caption("en", cRefusedCycle, "loop-b"),
		"loop-b":   caption("en", cRefusedCycle, "loop-a"),
		"needy":    caption("en", cRefusedMissing, "ghost"),
		"old":      caption("en", cRefusedVersion, "core>=2", "1.0"),
		"ru-only":  caption("en", cRefusedBase, basepkg.ROM1EN),
		"spare":    "",
		"spare-ru": caption("en", cRefusedBase, basepkg.ROM1EN),
	}
	rows := rowsByID(a)
	for id, w := range want {
		if got := rows[id].refusal; got != w {
			t.Errorf("row %s refusal %q, want %q", id, got, w)
		}
	}
	if rows["spare-ru"].enabled || rows["spare-ru"].problem != "" {
		t.Errorf("the mod that is not enabled: %+v", rows["spare-ru"])
	}
	for _, it := range a.layout() {
		if it.kind == iModRow && strings.Contains(it.text, "T core") && it.tone != toneNormal {
			t.Errorf("the accepted row %q has tone %v", it.text, it.tone)
		}
		if it.kind == iModRow && strings.Contains(it.text, want["clash"]) && it.tone != toneBad {
			t.Errorf("the refused row %q has tone %v", it.text, it.tone)
		}
	}
	first := enabledIDs(a)[0]
	a.play()
	if len(r.fakes.started) != 0 || !a.statusBad || a.status != caption("en", cCannotPlay, caption("en", cModUnusable, first, want[first])) {
		t.Fatalf("started %v status %q", r.fakes.started, a.status)
	}
	// a refused mod that is not enabled is ticked like any other
	a.apply(action{aToggleMod, len(a.modRows()) - 1})
	if !a.s.modEnabled("spare-ru") {
		t.Fatalf("spare-ru was not ticked: %q", a.status)
	}
	selectBase(t, a, basepkg.ROM1RU)
	rows = rowsByID(a)
	if rows["ru-only"].refusal != "" || rows["spare-ru"].refusal != "" || rows["clash"].refusal != caption("ru", cRefusedConflict, "core") {
		t.Fatalf("on rom1-ru: %+v %+v %+v", rows["ru-only"], rows["spare-ru"], rows["clash"])
	}
}

// A mod that requires an enabled mod which cannot be used (its mod.toml cannot
// be read, or its folder is gone) is refused with the wording of an unusable
// mod, never as one that is not enabled; Play and Check name the unusable mod
// and its own error.
func TestRequirementOnAnEnabledUnusableModIsNotCalledNotEnabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		mods map[string]string
	}{
		{"manifest cannot be read", map[string]string{"good": manifest("good", `requires = ["broken"]`), "broken": "id = \"broken\"\n"}},
		{"folder is missing", map[string]string{"good": manifest("good", `requires = ["broken"]`)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := baseRig(t, []string{"good", "broken"}, tc.mods)
			a := r.app()
			for _, lang := range []struct{ base, code string }{{basepkg.ROM1EN, "en"}, {basepkg.ROM1RU, "ru"}} {
				selectBase(t, a, lang.base)
				rows := rowsByID(a)
				broken := rows["broken"]
				if !broken.enabled || broken.problem == "" {
					t.Fatalf("%s: broken row %+v", lang.code, broken)
				}
				want := caption(lang.code, cRefusedUnusable, "broken")
				if rows["good"].refusal != want || strings.Contains(rows["good"].refusal, caption(lang.code, cRefusedMissing, "broken")) {
					t.Errorf("%s: good refusal %q, want %q", lang.code, rows["good"].refusal, want)
				}
				tones := map[string]tone{}
				for _, it := range a.layout() {
					if it.kind != iModRow {
						continue
					}
					for _, id := range []string{"good", "broken"} {
						if it.act.i < len(a.modRows()) && a.modRows()[it.act.i].id == id {
							tones[id] = it.tone
						}
					}
				}
				if tones["good"] != toneBad || tones["broken"] != toneBad {
					t.Errorf("%s: tones %v", lang.code, tones)
				}
				wantErr := caption(lang.code, cModUnusable, "broken", broken.problem)
				a.play()
				if len(r.fakes.started) != 0 || !a.statusBad || a.status != caption(lang.code, cCannotPlay, wantErr) {
					t.Errorf("%s: play status %q, want %q", lang.code, a.status, caption(lang.code, cCannotPlay, wantErr))
				}
				ran := len(r.fakes.ran)
				a.check()
				if len(r.fakes.ran) != ran || !a.statusBad || a.status != caption(lang.code, cCannotCheck, wantErr) {
					t.Errorf("%s: check status %q, want %q", lang.code, a.status, caption(lang.code, cCannotCheck, wantErr))
				}
			}
		})
	}
}
