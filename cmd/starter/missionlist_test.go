package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/witness"
	basepkg "againrom/pkg/base"
	"againrom/pkg/launch"
	"againrom/pkg/mod"
)

// writeMissionMod writes the mod id into dir with n guest missions, numbered
// from 25 and chained from campaign mission 20, titled in English and Russian
// through the mod's strings. No map file is written.
func writeMissionMod(t *testing.T, dir, id string, n int) {
	t.Helper()
	files := map[string]string{"mod.toml": goodMod(id)}
	var missions, en, ru strings.Builder
	for i := 0; i < n; i++ {
		after := 20
		if i > 0 {
			after = 24 + i
		}
		fmt.Fprintf(&missions, "[[mission]]\nkey = \"m%d\"\nnumber = %d\nmap = \"maps/m%d.alm\"\ntitle = \"m%d.title\"\nafter = %d\nguest = 22\nchapter = 30\n", i+1, 25+i, i+1, i+1, after)
		fmt.Fprintf(&en, "\"m%d.title\" = \"Ambush %d\"\n", i+1, i+1)
		fmt.Fprintf(&ru, "\"m%d.title\" = \"Засада %d\"\n", i+1, i+1)
	}
	if n > 0 {
		files["data/missions.toml"] = missions.String()
		files["text/en/strings.toml"] = en.String()
		files["text/ru/strings.toml"] = ru.String()
	}
	for name, body := range files {
		path := filepath.Join(dir, id, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// missionRig is a starter on a base of the language with the mod "ambush" of
// n missions in its mods folder, enabled when enabled is set.
func missionRig(t *testing.T, language string, n int, enabled bool) (*rig, *app) {
	t.Helper()
	r := rigWithLanguage(t, language)
	writeMissionMod(t, filepath.Join(r.dir, "mods"), "ambush", n)
	a := r.app()
	if enabled {
		do(t, a, action{aToggleMod, 0})
	}
	return r, a
}

// missionTexts are the rows of the mission list in view.
func missionTexts(a *app) []string {
	var out []string
	for _, it := range a.layout() {
		if it.act.kind == aSelectMission {
			out = append(out, it.text)
		}
	}
	return out
}

func hasAction(a *app, kind actionKind) bool {
	for _, it := range a.layout() {
		if it.act.kind == kind {
			return true
		}
	}
	return false
}

// The mission list shows the missions of the selected enabled mod by number
// and by the title of the starter's language, for a mod with none, one and
// / more than the list holds. With AGAINROM_STARTER_MOD_FRAMES set each window
// is written there as a PNG.
func TestMissionListShowsTheMissionsOfTheEnabledMod(t *testing.T) {
	for _, tc := range []struct{ language, lang, title string }{{"english", "en", "Ambush"}, {"russian", "ru", "Засада"}} {
		for _, n := range []int{0, 1, 12} {
			t.Run(fmt.Sprintf("%s/%d", tc.lang, n), func(t *testing.T) {
				_, a := missionRig(t, tc.language, n, true)
				if !containsText(a, caption(tc.lang, cHeadMissions)) {
					t.Fatalf("no mission heading in %q", texts(a))
				}
				var want []string
				for i := 0; i < min(n, missionRows); i++ {
					want = append(want, fmt.Sprintf("%d. %s %d", 25+i, tc.title, i+1))
				}
				if got := missionTexts(a); !reflect.DeepEqual(got, want) {
					t.Fatalf("mission rows %q, want %q; the window shows %q", got, want, texts(a))
				}
				if got := containsText(a, caption(tc.lang, cNoMissions)); got != (n == 0) {
					t.Fatalf("the no-missions line shown=%v with %d missions", got, n)
				}
				if got := hasAction(a, aStartMission); got != (n > 0) {
					t.Fatalf("the Start button shown=%v with %d missions", got, n)
				}
				if n > 0 && !containsText(a, caption(tc.lang, cStartMission)) {
					t.Fatalf("no Start caption in %q", texts(a))
				}
				if dir := os.Getenv("AGAINROM_STARTER_MOD_FRAMES"); dir != "" {
					witness.PNG(t, filepath.Join(dir, fmt.Sprintf("starter-missions-%s-%d.png", tc.lang, n)), a.render())
				}
				if n <= missionRows {
					return
				}
				if !containsText(a, caption(tc.lang, cSettingsInView, 1, missionRows, n)) {
					t.Fatalf("no rows-in-view note in %q", texts(a))
				}
				// The wheel over the list moves it to its last rows and no further.
				at := scrollLists[listMissions].area.Min.Add(image.Pt(20, 5))
				a.wheel(at, 100)
				rows := missionTexts(a)
				if last := fmt.Sprintf("%d. %s %d", 24+n, tc.title, n); len(rows) != missionRows || rows[len(rows)-1] != last {
					t.Fatalf("scrolled rows %q, want the last to be %q", rows, last)
				}
			})
		}
	}
}

// A mod that is listed and not enabled shows no missions, and says why.
func TestMissionListAsksToEnableTheMod(t *testing.T) {
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		_, a := missionRig(t, tc.language, 3, false)
		if containsText(a, caption(tc.lang, cHeadMissions)) || len(missionTexts(a)) != 0 {
			t.Fatalf("%s: with no mod selected the window shows %q", tc.lang, texts(a))
		}
		do(t, a, action{aToggleMod, 0})
		do(t, a, action{aToggleMod, 0})
		if !containsText(a, caption(tc.lang, cMissionsDisabled)) || len(missionTexts(a)) != 0 || hasAction(a, aStartMission) {
			t.Fatalf("%s: a mod that is not enabled shows %q", tc.lang, texts(a))
		}
	}
}

// A broken missions file is reported in the list's place.
func TestMissionListReportsAMissionsFileItCannotRead(t *testing.T) {
	r := rigWithLanguage(t, "english")
	writeMissionMod(t, filepath.Join(r.dir, "mods"), "ambush", 1)
	if err := os.WriteFile(filepath.Join(r.dir, "mods", "ambush", "data", "missions.toml"), []byte("[[mission]]\nbogus = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := r.app()
	do(t, a, action{aToggleMod, 0})
	found := false
	for _, s := range texts(a) {
		found = found || strings.HasPrefix(s, "missions.toml: ")
	}
	if !found || len(missionTexts(a)) != 0 || hasAction(a, aStartMission) {
		t.Fatalf("a broken missions file shows %q", texts(a))
	}
}

func iniText(t *testing.T, r *rig) string {
	t.Helper()
	data, err := os.ReadFile(r.ini)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Start starts the game with the enabled list and the chosen mission, Play
// with the enabled list and no mission of the list, and both write the
// enabled list into starter.ini with no Save settings press.
func TestStartNamesTheChosenMissionAndPlayNamesNone(t *testing.T) {
	r, a := missionRig(t, "english", 3, true)
	exe, mods := filepath.Join(r.dir, "againrom.exe"), filepath.Join(r.dir, "mods")
	if strings.Contains(iniText(t, r), "ambush") {
		t.Fatal("the fixture's settings file already names the mod")
	}
	do(t, a, action{kind: aStartMission})
	if len(r.fakes.started) != 0 || !a.statusBad || !strings.Contains(a.status, caption("en", cSelectMission)) {
		t.Fatalf("Start with no mission chosen: started %q, status %q", r.fakes.started, a.status)
	}
	do(t, a, action{aSelectMission, 1})
	do(t, a, action{kind: aStartMission})
	want := []string{exe, "-assets", r.root, "-mods", "ambush", "-mods-dir", mods, "-mission", "26"}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("Start started %q\nwant %q", r.fakes.started, want)
	}
	if a.statusBad || !strings.Contains(iniText(t, r), "enabled = ambush") {
		t.Fatalf("Start left status %q and settings\n%s", a.status, iniText(t, r))
	}

	// Play on a fresh starter: the enabled list, no mission, settings written.
	r, a = missionRig(t, "english", 3, true)
	exe, mods = filepath.Join(r.dir, "againrom.exe"), filepath.Join(r.dir, "mods")
	do(t, a, action{aSelectMission, 2})
	do(t, a, action{kind: aPlay})
	want = []string{exe, "-assets", r.root, "-mods", "ambush", "-mods-dir", mods}
	if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
		t.Fatalf("Play started %q\nwant %q", r.fakes.started, want)
	}
	if a.statusBad || !strings.Contains(iniText(t, r), "enabled = ambush") {
		t.Fatalf("Play left status %q and settings\n%s", a.status, iniText(t, r))
	}
	// The typed mission number is Play's as before; Start's mission is the
	// list's.
	a.s.Mission = "7"
	do(t, a, action{kind: aPlay})
	do(t, a, action{kind: aStartMission})
	if n := len(r.fakes.started); n != 3 || !slices.Equal(r.fakes.started[1][len(want):], []string{"-mission", "7"}) ||
		!slices.Equal(r.fakes.started[2][len(want):], []string{"-mission", "27"}) {
		t.Fatalf("with a typed mission number the starts were %q", r.fakes.started)
	}
}

// A refused start writes nothing: the settings file is the last accepted one.
func TestARefusedStartWritesNoSettings(t *testing.T) {
	r, a := missionRig(t, "english", 1, true)
	if err := os.Remove(filepath.Join(r.dir, "againrom.exe")); err != nil {
		t.Fatal(err)
	}
	do(t, a, action{kind: aPlay})
	if _, err := os.Stat(r.ini); err != nil {
		t.Fatal(err)
	}
	if !a.statusBad || strings.Contains(iniText(t, r), "ambush") || len(r.fakes.started) != 0 {
		t.Fatalf("a refused Play: status %q, settings\n%s", a.status, iniText(t, r))
	}
}

// A base of the second game lists a mod with no missions as one with none,
// and Play starts it with the enabled list.
func TestSecondGameBaseShowsAModWithNoMissions(t *testing.T) {
	for _, id := range []string{basepkg.ROM2EN, basepkg.ROM2RU} {
		r := newRig(t, "[starter]\nlast-base = en\n[bases]\nen = {root}\n")
		r.fakes.validRoots[r.root] = profileInfo(id, true)
		writeMissionMod(t, filepath.Join(r.dir, "mods"), "plain", 0)
		a := r.app()
		do(t, a, action{aToggleMod, 0})
		if !containsText(a, a.t(cNoMissions)) || len(missionTexts(a)) != 0 || hasAction(a, aStartMission) {
			t.Fatalf("%s: the window shows %q", id, texts(a))
		}
		do(t, a, action{kind: aPlay})
		want := []string{filepath.Join(r.dir, "againrom.exe"), "-assets", r.root, "-base", id, "-mods", "plain", "-mods-dir", filepath.Join(r.dir, "mods")}
		if len(r.fakes.started) != 1 || !reflect.DeepEqual(r.fakes.started[0], want) {
			t.Fatalf("%s: Play started %q\nwant %q", id, r.fakes.started, want)
		}
		if !strings.Contains(iniText(t, r), "enabled = plain") {
			t.Fatalf("%s: Play wrote\n%s", id, iniText(t, r))
		}
	}
}

// packInto packs the mod folder into the mods folder dir as mod-<id>.res.
func packInto(t *testing.T, folder, dir string) string {
	t.Helper()
	m, data, err := mod.Pack(folder)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, mod.PackedName(m.ID))
	if err := os.WriteFile(file, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return file
}

// A mod packed into the mods folder while the starter was out of focus is
// listed when its window regains the focus, with no Rescan press; the ticks,
// the order and the selections stay. The starter then starts the packed
// mod's mission.
func TestRegainingFocusListsAModPublishedMeanwhile(t *testing.T) {
	r := rigWithLanguage(t, "english")
	r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
	mods := launch.ModsFolder("", r.dir)
	writeMissionMod(t, mods, "ambush", 2)
	writeModFolder(t, mods, "zeta", goodMod("zeta"))
	a := r.app()
	a.setFocused(true)
	rowOf := func(id string) int {
		for i, m := range a.modRows() {
			if m.id == id {
				return i
			}
		}
		t.Fatalf("no mod row %s in %+v", id, a.modRows())
		return -1
	}
	do(t, a, action{aToggleMod, rowOf("zeta")})
	do(t, a, action{aToggleMod, rowOf("ambush")})
	do(t, a, action{aSelectMission, 1})
	type state struct {
		enabled    []string
		order      []string
		selMod     string
		selMission int
		scroll     [listCount]int
	}
	snapshot := func() state {
		s := state{enabled: slices.Clone(a.s.Enabled), selMod: a.selMod, selMission: a.selMission, scroll: a.scroll}
		for _, m := range a.modRows() {
			if m.enabled {
				s.order = append(s.order, fmt.Sprintf("%d.%s", m.order, m.id))
			}
		}
		return s
	}
	before := snapshot()

	source := t.TempDir()
	writeMissionMod(t, source, "newcomer", 1)
	file := packInto(t, filepath.Join(source, "newcomer"), mods)
	a.setFocused(true)
	if slices.ContainsFunc(a.modRows(), func(m modRow) bool { return m.id == "newcomer" }) {
		t.Fatal("the mods folder was read again while the window kept the focus")
	}
	a.setFocused(false)
	a.takeDirty()
	a.setFocused(true)
	row := rowOf("newcomer")
	if got := a.modRows()[row]; got.problem != "" || got.enabled || !a.dirty {
		t.Fatalf("the packed mod is listed as %+v, dirty=%v", got, a.dirty)
	}
	if got := snapshot(); !reflect.DeepEqual(got, before) {
		t.Fatalf("regaining the focus changed the state\n got %+v\nwant %+v", got, before)
	}

	// The packed mod's mission starts through the starter's one launch route.
	do(t, a, action{aToggleMod, rowOf("newcomer")})
	if got, want := missionTexts(a), []string{"25. Ambush 1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the packed mod's missions %q, want %q", got, want)
	}
	do(t, a, action{aSelectMission, 0})
	do(t, a, action{kind: aStartMission})
	if len(r.fakes.started) != 1 {
		t.Fatalf("Start: %q, status %q", r.fakes.started, a.status)
	}
	got := r.fakes.started[0]
	at := slices.Index(got, "-mods")
	if at < 0 || !slices.Contains(strings.Split(got[at+1], ","), "newcomer") || !slices.Equal(got[len(got)-2:], []string{"-mission", "25"}) ||
		!slices.Equal(got[at+2:at+4], []string{"-mods-dir", filepath.Dir(file)}) {
		t.Fatalf("Start started %q", got)
	}
}

// The window is 760 pixels high, so a screen 768 high shows it. The action
// buttons and the status line stand where they stood before the mission list:
// the status line ends at pixel 622. The mission list and the Check output
// share what is under it, and nothing reaches past the window.
func TestTheWindowFitsAScreen768High(t *testing.T) {
	if winH != 760 || starterWindow.H != 760 {
		t.Fatalf("the window is %d (%d) pixels high, want 760", winH, starterWindow.H)
	}
	for _, n := range []int{-1, 0, 12} {
		r := rigWithLanguage(t, "russian")
		if n >= 0 {
			writeMissionMod(t, filepath.Join(r.dir, "mods"), "ambush", n)
		}
		a := r.app()
		if n >= 0 {
			do(t, a, action{aToggleMod, 0})
		}
		a.fail(cCannotPlay, "status line")
		a.output = []string{"one", "two", "three", "four", "five", "six", "seven"}
		box, lines := a.outputBox()
		if want := map[bool]int{false: outputLines, true: 2}[n >= 0]; lines != want || box.Max.Y > winH || box.Min.Y < statusTop+16 {
			t.Fatalf("%d missions: the Check output holds %d lines in %v, want %d under the status line", n, lines, box, want)
		}
		status, play, shown := false, false, 0
		for _, it := range a.layout() {
			if it.rect.Max.Y > winH || it.rect.Max.X > winW {
				t.Errorf("%d missions: %q at %v is outside the window", n, it.text, it.rect)
			}
			switch {
			case it.text == a.status:
				status = true
				if it.rect.Min.Y != 606 || it.rect.Max.Y > 622 {
					t.Errorf("%d missions: the status line is at %v, want rows 606 to 622", n, it.rect)
				}
			case it.kind == iButton && it.act.kind == aPlay:
				play = true
				if it.rect != image.Rect(10, 550, 110, 578) {
					t.Errorf("%d missions: Play is at %v", n, it.rect)
				}
			case slices.Contains(a.output, it.text):
				shown++
				if !it.rect.In(box) {
					t.Errorf("%d missions: output line %q at %v is outside %v", n, it.text, it.rect, box)
				}
			case it.act.kind == aSelectMission:
				if it.rect.Min.Y < 622 || it.rect.Max.Y > box.Min.Y {
					t.Errorf("%d missions: mission row %q at %v is not between the status line and the output", n, it.text, it.rect)
				}
			}
		}
		if !status || !play || shown != lines {
			t.Fatalf("%d missions: status %v, Play %v, %d output lines of %d", n, status, play, shown, lines)
		}
	}
}

// A chosen mission is a mission of its mod: when that mod leaves the list and
// the panel falls to another mod with a mission of the same number, no row is
// marked and Start names no mission.
func TestAChosenMissionDoesNotMoveToAnotherMod(t *testing.T) {
	r := rigWithLanguage(t, "english")
	r.fakes.validRoots[r.root] = profileInfo(basepkg.ROM1EN, true)
	mods := launch.ModsFolder("", r.dir)
	writeMissionMod(t, mods, "ambush", 2)
	writeMissionMod(t, mods, "border", 2)
	a := r.app()
	rowOf := func(id string) int {
		for i, m := range a.modRows() {
			if m.id == id {
				return i
			}
		}
		t.Fatalf("no mod row %s in %+v", id, a.modRows())
		return -1
	}
	do(t, a, action{aToggleMod, rowOf("border")})
	do(t, a, action{aToggleMod, rowOf("ambush")})
	do(t, a, action{aSelectMission, 1})
	if n, ok := a.chosenMission(); !ok || n != 26 {
		t.Fatalf("chosen mission %d, %v", n, ok)
	}
	do(t, a, action{aToggleMod, rowOf("ambush")})
	if err := os.RemoveAll(filepath.Join(mods, "ambush")); err != nil {
		t.Fatal(err)
	}
	a.rescan()
	if id, ok := a.panelMod(); !ok || id != "border" || len(missionTexts(a)) != 2 {
		t.Fatalf("the panel shows %q with rows %q", id, missionTexts(a))
	}
	for _, it := range a.layout() {
		if it.act.kind == aSelectMission && it.checked {
			t.Fatalf("the mark moved to %q of another mod", it.text)
		}
	}
	if n, ok := a.chosenMission(); ok {
		t.Fatalf("Start would name mission %d of another mod", n)
	}
	do(t, a, action{kind: aStartMission})
	if len(r.fakes.started) != 0 {
		t.Fatalf("Start started %q", r.fakes.started)
	}
}
