package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/game"
)

const panelSettings = `[skill_cap]
label   = { en = "Skill cap", ru = "Предел навыка" }
type    = "int"
default = 150
min     = 100
max     = 150

[cloaks]
label   = "Cloaks for mages"
type    = "bool"
default = false

[mood]
label   = { en = "Mood" }
type    = "choice"
choices = ["calm", "grim", "wild"]
default = "calm"
`

// panelRig is a starter folder with one base, the mod skill-cap enabled and its
// settings declared.
func panelRig(t *testing.T, extraIni string) (*rig, *app) {
	t.Helper()
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n[mods]\nenabled = skill-cap\n"+extraIni)
	mods := filepath.Join(r.dir, "mods")
	writeModFolder(t, mods, "skill-cap", goodMod("skill-cap"))
	if err := os.WriteFile(filepath.Join(mods, "skill-cap", "settings.toml"), []byte(panelSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	return r, r.app()
}

func texts(a *app) []string {
	var out []string
	for _, it := range a.layout() {
		out = append(out, it.text)
	}
	return out
}

func containsText(a *app, want string) bool {
	for _, s := range texts(a) {
		if s == want {
			return true
		}
	}
	return false
}

func TestThePanelShowsTheEnabledModsKnobsWithTheirLabels(t *testing.T) {
	_, a := panelRig(t, "")
	for _, want := range []string{"Mod settings", "skill-cap (enabled)", "Skill cap (100..150)", "150", "Cloaks for mages", "off", "Mood", "calm", "Defaults"} {
		if !containsText(a, want) {
			t.Errorf("the panel lacks %q; it shows %q", want, texts(a))
		}
	}
}

func TestThePanelReadsTheBasesLanguage(t *testing.T) {
	r, a := panelRig(t, "")
	if a.lang() != "en" {
		t.Fatalf("lang %q", a.lang())
	}
	r.fakes.validRoots[r.root] = game.InstallInfo{Language: "russian"}
	a2 := r.app()
	if a2.lang() != "ru" || !containsText(a2, "Предел навыка (100..150)") {
		t.Fatalf("lang %q; the panel shows %q", a2.lang(), texts(a2))
	}
	// A mod with no Russian label falls back to its English one.
	if !containsText(a2, "Mood") {
		t.Fatalf("no English fallback: %q", texts(a2))
	}
}

func TestEditedSettingsGoToPlayAndCheck(t *testing.T) {
	r, a := panelRig(t, "")
	typeInto(t, a, fSetting0, "")
	for i := 0; i < 3; i++ {
		a.backspace()
	}
	a.typed("120")
	do(t, a, action{aSettingCycle, 1})
	do(t, a, action{aSettingCycle, 2})
	do(t, a, action{kind: aPlay})
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-mods", "skill-cap", "-mods-dir", filepath.Join(r.dir, "mods"),
		"-mod-setting", "skill-cap.skill_cap=120", "-mod-setting", "skill-cap.cloaks=true", "-mod-setting", "skill-cap.mood=grim"}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q\nwant    %q (status %q)", r.fakes.started, want, a.status)
	}
	do(t, a, action{kind: aCheck})
	if len(r.fakes.ran) == 0 || !reflect.DeepEqual(r.fakes.ran[len(r.fakes.ran)-1], append(want, "-check")) {
		t.Fatalf("check ran %q", r.fakes.ran)
	}
}

func TestAChoiceStepsThroughItsChoicesAndWraps(t *testing.T) {
	_, a := panelRig(t, "")
	var seen []string
	for i := 0; i < 4; i++ {
		do(t, a, action{aSettingCycle, 2})
		v, _ := a.s.modValue("skill-cap", "mood")
		seen = append(seen, v)
	}
	if !reflect.DeepEqual(seen, []string{"grim", "wild", "calm", "grim"}) {
		t.Fatalf("%v", seen)
	}
}

func TestARefusedValueStopsPlayAndCheckNamingTheModAndSetting(t *testing.T) {
	r, a := panelRig(t, "")
	typeInto(t, a, fSetting0, "")
	for i := 0; i < 3; i++ {
		a.backspace()
	}
	a.typed("99")
	a.play()
	if !a.statusBad || !strings.Contains(a.status, `mod "skill-cap" setting skill_cap: 99 is outside 100..150`) || len(r.fakes.started) != 0 {
		t.Fatalf("status %q started %q", a.status, r.fakes.started)
	}
	a.check()
	if !a.statusBad || !strings.Contains(a.status, "outside 100..150") || len(r.fakes.ran) != 1 {
		t.Fatalf("status %q ran %q", a.status, r.fakes.ran)
	}
	if line := texts(a); !containsText(a, `mod "skill-cap" setting skill_cap: 99 is outside 100..150`) {
		t.Fatalf("the command line does not show the refusal: %q", line)
	}
	for i := 0; i < 2; i++ {
		a.backspace()
	}
	a.typed("abc")
	a.play()
	if !strings.Contains(a.status, `"abc" is not an integer`) {
		t.Fatalf("status %q", a.status)
	}
	a.backspace()
	a.backspace()
	a.backspace()
	a.play()
	// An empty box means the default applies, so nothing is passed.
	if len(r.fakes.started) != 1 || strings.Contains(strings.Join(r.fakes.started[0], " "), "-mod-setting") {
		t.Fatalf("an empty value was passed on: %q (status %q)", r.fakes.started, a.status)
	}
}

func TestSavedSettingsAreStoredInTheModsSectionAndReadBack(t *testing.T) {
	r, a := panelRig(t, "[mod.other]\nkept = yes\n")
	typeInto(t, a, fSetting0, "")
	for i := 0; i < 3; i++ {
		a.backspace()
	}
	a.typed("125")
	do(t, a, action{aSettingCycle, 1})
	a.save()
	data, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{"[mod.skill-cap]\nskill_cap = 125\ncloaks = true\n", "[mod.other]\nkept = yes\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the ini lacks %q:\n%s", want, got)
		}
	}
	b := r.app()
	if v, _ := b.s.modValue("skill-cap", "skill_cap"); v != "125" {
		t.Fatalf("read back %q", v)
	}
	if !containsText(b, "125") {
		t.Fatalf("the panel does not show the stored value: %q", texts(b))
	}
	// Defaults forgets the mod's values and the next save removes them.
	do(t, b, action{kind: aSettingReset})
	if _, ok := b.s.modValue("skill-cap", "skill_cap"); ok || !containsText(b, "150") {
		t.Fatal("Defaults kept a stored value")
	}
	b.save()
	data, _ = os.ReadFile(r.ini)
	if strings.Contains(string(data), "skill_cap") || !strings.Contains(string(data), "kept = yes") {
		t.Fatalf("after Defaults:\n%s", data)
	}
}

func TestAStoredKeyTheModNoLongerDeclaresIsNotPassed(t *testing.T) {
	r, a := panelRig(t, "[mod.skill-cap]\nretired = 3\nskill_cap = 140\n")
	a.play()
	want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-mods", "skill-cap", "-mods-dir", filepath.Join(r.dir, "mods"),
		"-mod-setting", "skill-cap.skill_cap=140"}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("started %q (status %q)", r.fakes.started, a.status)
	}
}

func TestSettingsOfAModThatIsNotEnabledAreShownButNotPassed(t *testing.T) {
	r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n[mod.skill-cap]\nskill_cap = 110\n")
	mods := filepath.Join(r.dir, "mods")
	writeModFolder(t, mods, "skill-cap", goodMod("skill-cap"))
	os.WriteFile(filepath.Join(mods, "skill-cap", "settings.toml"), []byte(panelSettings), 0o644)
	a := r.app()
	if containsText(a, "Skill cap (100..150)") {
		t.Fatal("the panel names a mod that was not selected or enabled")
	}
	do(t, a, action{aToggleMod, 0})
	do(t, a, action{aToggleMod, 0})
	if !containsText(a, "skill-cap (not enabled)") || !containsText(a, "110") {
		t.Fatalf("panel %q", texts(a))
	}
	a.play()
	if len(r.fakes.started) != 1 || strings.Contains(strings.Join(r.fakes.started[0], " "), "-mod") {
		t.Fatalf("started %q", r.fakes.started)
	}
}

func TestABrokenSettingsFileIsShownAndRefusesTheLaunch(t *testing.T) {
	r, a := panelRig(t, "")
	os.WriteFile(filepath.Join(r.dir, "mods", "skill-cap", "settings.toml"), []byte("[a]\ntype = \"int\"\n"), 0o644)
	a.rescan()
	found := false
	for _, s := range texts(a) {
		found = found || strings.HasPrefix(s, "settings.toml: ")
	}
	if !found {
		t.Fatalf("panel %q", texts(a))
	}
	a.play()
	if !a.statusBad || !strings.Contains(a.status, "mod skill-cap:") || len(r.fakes.started) != 0 {
		t.Fatalf("status %q", a.status)
	}
}

func TestTabVisitsOnlyFieldsThatExist(t *testing.T) {
	_, a := panelRig(t, "")
	a.focus = fAgainrom
	a.tab()
	if a.focus != fSetting0 {
		t.Fatalf("focus %v, want the integer setting", a.focus)
	}
	a.tab()
	if a.focus != fAddBase {
		t.Fatalf("focus %v, want the first field again", a.focus)
	}
}
