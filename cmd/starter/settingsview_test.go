package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ini"
)

// settingsFields are the fields and actions the settings view owns.
var (
	settingsFieldIDs = []fieldID{fVolume, fSaves, fMission, fSkill, fShopStockMultiplier, fExtra, fAgainrom}
	settingsActions  = []actionKind{aSound, aMovies, aVideo, aMarkers, aPicker, aChicken, aCloseOnPlay, aAcceptUnmarked}
)

func TestMainViewDrawsNoSettingsControl(t *testing.T) {
	a := rigWithLanguage(t, "english").app()
	for _, it := range a.layout() {
		for _, id := range settingsFieldIDs {
			if it.kind == iField && it.act == (action{aFocus, int(id)}) {
				t.Errorf("main view shows field %d", id)
			}
		}
		for _, k := range settingsActions {
			if it.act.kind == k {
				t.Errorf("main view shows action %d", k)
			}
		}
		if it.kind == iHeading && (it.text == a.t(cHeadParams) || it.text == a.t(cHeadAdvanced) || it.text == a.t(cHeadStarter)) {
			t.Errorf("main view shows the heading %q", it.text)
		}
	}
	found := false
	for _, it := range a.layout() {
		found = found || it.act.kind == aSettingsOpen
	}
	if !found {
		t.Fatal("main view has no Settings button")
	}
}

func TestSettingsViewShowsEveryFieldAndClosesWithoutChangingAValue(t *testing.T) {
	for _, closeBy := range []string{"button", "escape"} {
		a := rigWithLanguage(t, "russian").app()
		before := a.s
		cmdBefore, _ := a.command()
		openSettings(t, a)
		for _, id := range settingsFieldIDs {
			ok := false
			for _, it := range a.layout() {
				ok = ok || (it.kind == iField && it.act == (action{aFocus, int(id)}))
			}
			if !ok {
				t.Errorf("settings view lacks field %d", id)
			}
		}
		for _, k := range settingsActions {
			ok := false
			for _, it := range a.layout() {
				ok = ok || it.act.kind == k
			}
			if !ok {
				t.Errorf("settings view lacks action %d", k)
			}
		}
		for _, it := range a.layout() {
			if it.kind == iBaseRow || it.kind == iModRow {
				t.Errorf("settings view still shows the %q list", it.text)
			}
		}
		a.focus = fSkill
		if closeBy == "button" {
			do(t, a, action{kind: aSettingsClose})
		} else {
			a.escape()
		}
		if a.view != viewMain || a.focus != fNone {
			t.Fatalf("%s: view %v focus %v after closing", closeBy, a.view, a.focus)
		}
		if cmd, _ := a.command(); !reflect.DeepEqual(a.s, before) || cmd != cmdBefore {
			t.Errorf("%s: opening and closing changed the settings", closeBy)
		}
	}
}

// The same edits give the same settings file through the settings view as set
// directly on the stored settings.
func TestEditsThroughTheSettingsViewSaveTheSameFile(t *testing.T) {
	const start = "; hello\n[starter]\nlast-base = en\nfuture = 1\n[bases]\nen = {root}\n"
	r := newRig(t, start)
	a := r.app()
	openSettings(t, a)
	do(t, a, action{kind: aSound})
	typeInto(t, a, fVolume, "42")
	do(t, a, action{kind: aMovies})
	do(t, a, action{kind: aVideo})
	do(t, a, action{kind: aMarkers})
	typeInto(t, a, fSaves, "sv")
	typeInto(t, a, fMission, "7")
	do(t, a, action{kind: aPicker})
	do(t, a, action{kind: aChicken})
	typeInto(t, a, fSkill, "axe")
	typeInto(t, a, fExtra, "-x")
	do(t, a, action{kind: aCloseOnPlay})
	typeInto(t, a, fAgainrom, "g.exe")
	do(t, a, action{kind: aAcceptUnmarked})
	do(t, a, action{kind: aSettingsClose})
	do(t, a, action{kind: aSave})
	if a.statusBad {
		t.Fatalf("status %q", a.status)
	}
	got, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}

	f, _ := ini.Parse([]byte(strings.ReplaceAll(start, "{root}", r.root)))
	s := loadSettings(f)
	s.Sound, s.Volume, s.Movies, s.Video, s.Markers = soundOn, "42", false, video4x, true
	s.Saves, s.Mission, s.Picker, s.Chicken = "sv", "7", true, true
	s.Skill, s.Extra, s.CloseOnPlay, s.Againrom, s.AcceptUnmarked = "axe", "-x", true, "g.exe", true
	s.store(f)
	if string(got) != string(f.Bytes()) {
		t.Fatalf("saved file differs\n got:\n%s\nwant:\n%s", got, f.Bytes())
	}
}

func TestTabSkipsFieldsTheViewDoesNotShow(t *testing.T) {
	a := rigWithLanguage(t, "english").app()
	a.focus = fAddBase
	for i := 0; i < int(fieldCount); i++ {
		a.tab()
		if a.focus == fVolume || a.focus == fSaves || a.focus == fMission || a.focus == fSkill || a.focus == fExtra || a.focus == fAgainrom {
			t.Fatalf("main view tabbed to the settings field %d", a.focus)
		}
	}
	openSettings(t, a)
	for i := 0; i < int(fieldCount); i++ {
		a.tab()
		if a.focus == fAddBase || a.focus == fModsDir {
			t.Fatalf("settings view tabbed to the main field %d", a.focus)
		}
	}
}
