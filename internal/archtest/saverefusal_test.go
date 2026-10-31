package archtest

import (
	"os"
	"testing"
)

func TestCheckSaveRefusal(t *testing.T) {
	cases := []struct {
		name, file, src string
		bad             bool
	}{
		{"a table row", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) townSavePoint(f *FrontEnd) error { return firstRefusals.refuse(refusedNoGame) }\n", false},
		{"a built refusal", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) townSavePoint(f *FrontEnd) error { return errors.New(\"no\") }\n", true},
		{"a formatted refusal", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) exportable(s Snapshot) error { return fmt.Errorf(\"no %d\", 1) }\n", true},
		{"a row outside the table", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) exportable(s Snapshot) error { return saveRefusal{reasonWord: \"w\"} }\n", true},
		{"an error elsewhere", "pkg/game/a.go",
			"package game\nfunc load() error { return errors.New(\"no\") }\n", false},
	}
	for _, tc := range cases {
		_, vs := checkSaveRefusal(map[string]string{tc.file: tc.src})
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestSaveRefusalIsOneTable(t *testing.T) {
	files := saveRefusalSources(t)
	rules, vs := checkSaveRefusal(files)
	if rules < 6 {
		t.Fatalf("save point rule methods read = %d, want at least 6 (three per game); the scan guards nothing", rules)
	}
	for _, v := range vs {
		t.Error(v)
	}
}

// One planted instance per shape the census names, and the forms the scan
// leaves alone.
func TestCheckSavePoint(t *testing.T) {
	const ui = "package ui\n"
	cases := []struct {
		name, file, src string
		bad             bool
	}{
		{"town predicate call", "pkg/ui/menurows.go", ui + "func (s gameMenuRowState) holds() bool { return townCanSave(s.town) }\n", true},
		{"menu save point call", "pkg/ui/save.go", ui + "func (f *flow) open() bool { return f.atGameSavePoint() }\n", true},
		{"refusal display call", "pkg/ui/functionkeys.go", ui + "func (a *App) keys() bool { return a.flow.showSaveRefusal(false) }\n", true},
		{"CanSave assertion", "pkg/ui/town.go", ui + "func can(t TownScreen) bool { s, ok := t.(interface{ CanSave() bool }); return ok && s.CanSave() }\n", true},
		{"admission request", "pkg/ui/save.go", ui + "func (f *flow) ask() { f.saveDialogSeams.Prepare(SaveRequest{Admission: true}) }\n", true},
		{"refusal line write", "pkg/ui/save.go", ui + "func (f *flow) refuse(m string) { f.saveRefused = m }\n", true},
		{"Save key over a popup", "pkg/ui/functionkeys.go", ui + "func (a *App) keys(in appInput) bool { return in.SaveGame && a.flow.popupOpen() }\n", true},
		{"quick key over a town dialogue", "pkg/ui/functionkeys.go", ui + "func (a *App) keys(in appInput) bool { _, open := townDialogue(a.flow.town); return in.QuickSave && !open }\n", true},
		{"menu save over the mission shop", "pkg/ui/save.go", ui + "func (f *flow) act(a gameMenuAction) bool { return a == gameMenuSave && !missionShopOpen(f.town) }\n", true},
		{"autosave under a notice", "pkg/ui/timedautosave.go", ui + "func (a *App) poll() { a.flow.gameOptions.Autosave.Poll(nil, true, !a.flow.viewer.NoticeOpen()) }\n", true},
		{"autosave beside a dialogue", "pkg/ui/timedautosave.go", ui + "func (a *App) poll() { _, d := a.currentDialoguePointerOwner(); a.flow.gameOptions.Autosave.Poll(nil, true, !d) }\n", true},
		{"a literal deciding a save", "pkg/ui/app.go", ui + "func (a *App) step(in appInput) { go func() { _ = in.SaveGame && a.flow.popupOpen() }() }\n", true},
		{"the gate itself", "pkg/ui/savepoint.go", ui + "func (f *flow) savePoint(k saveKind) bool { _ = f.viewer.popupOpen(); f.saveRefused = \"\"; return townCanSave(f.town) && f.showSaveRefusal(true) }\n", false},
		{"a popup test without a save input", "pkg/ui/viewer.go", ui + "func (v *Viewer) idle() bool { return !v.popupOpen() }\n", false},
		{"a save input in a literal beside a popup test", "pkg/ui/app.go", ui + "func (a *App) step(in appInput) { defer func() { _ = in.SaveGame }(); _ = a.flow.popupOpen() }\n", false},
		{"a test file", "pkg/ui/save_test.go", ui + "func open(f *flow) bool { return townCanSave(f.town) }\n", false},
		{"another package", "pkg/game/townhost.go", "package game\nfunc can(t TownScreen) bool { return townCanSave(t) }\n", false},
	}
	for _, tc := range cases {
		_, vs := checkSavePoint(map[string]string{tc.file: tc.src})
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestSavePointIsOneGate(t *testing.T) {
	files := saveRefusalSources(t)
	asks, vs := checkSavePoint(files)
	if asks < 8 {
		t.Fatalf("savePoint calls read = %d, want at least 8 (menu action, menu row, dialog, page, quick keys, Load key, Save key, autosave); the scan guards nothing", asks)
	}
	for _, v := range vs {
		t.Error(v)
	}
}

func saveRefusalSources(t *testing.T) map[string]string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	return files
}
