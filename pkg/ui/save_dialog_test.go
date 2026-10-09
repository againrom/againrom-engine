package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

type saveDialogSpy struct {
	requests                  []SaveRequest
	commits                   []bool
	lists                     []string
	directories               map[string]SaveDirectory
	paths, existing           []string
	prepareError, commitError error
}

func saveChooserStyleApp(t *testing.T, count int) (*App, *saveDialogSpy, []*image.RGBA) {
	t.Helper()
	entries := make([]SaveEntry, count)
	for i := range entries {
		entries[i] = SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("detail-%02d", i)}
	}
	spy := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}}
	a := newSaveDialogApp(t, spy, ScreenTown)
	a.SetWords(AuthoredWords(), solidFont15(), nil)
	art := &DialogFrame{}
	for i := range art.Pieces {
		art.Pieces[i] = image.NewRGBA(image.Rect(0, 0, 48, 48))
		draw.Draw(art.Pieces[i], art.Pieces[i].Bounds(), image.NewUniform(color.RGBA{91, 53, 27, 255}), image.Point{}, draw.Src)
	}
	a.SetGameMenuArt(art)
	frames := loadScrollTestFrames()
	a.SetCutsceneScrollArt(frames)
	return a, spy, frames
}

func saveChooserStyleFrame(t *testing.T, a *App) *image.RGBA {
	t.Helper()
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" || pix.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatal("Save style frame", note, err)
	}
	return pix
}

// TestSaveDialogLoadStyleHasLiteralRowsAndBar: the save chooser draws its
// rows and bar with the shared list builder: seven rows of font height plus
// 4 at (38,88), the bar at the list's right edge bound to the selection.
func TestSaveDialogLoadStyleHasLiteralRowsAndBar(t *testing.T) {
	a, spy, _ := saveChooserStyleApp(t, 9)
	check := func(sel int) {
		t.Helper()
		calls := recordWidgets(t, func() { saveChooserStyleFrame(t, a) })
		var lists, bars int
		for _, c := range calls {
			switch c.kind {
			case widgetListBox:
				lists++
				if c.rect != image.Rect(38, 88, 578, 223) {
					t.Errorf("Save list at %v", c.rect)
				}
			case widgetVScrollBar:
				bars++
				if b := c.state.(vScrollBar); c.rect != image.Rect(578, 88, 602, 223) || b.Pos != sel || b.Count != 9 {
					t.Errorf("Save bar %v at %d of %d, want %d of 9", c.rect, b.Pos, b.Count, sel)
				}
			}
		}
		if lists != 1 || bars != 1 {
			t.Fatalf("Save drew %d lists and %d bars", lists, bars)
		}
	}
	check(0)
	if err := a.HeadlessSaveSelect(8); err != nil {
		t.Fatal(err)
	}
	check(8)
	if top, count := a.flow.saveDialog.list.Visible(); top != 2 || count != 7 || a.flow.saveDialog.list.Selection() != 8 || a.flow.saveDialog.request.Name != "slot-08" {
		t.Fatal("Save style changed window or selected exact name")
	}
	mustSaveAction(t, a, "save")
	if len(spy.requests) != 1 || spy.requests[0].Name != "slot-08" || !reflect.DeepEqual(spy.commits, []bool{false}) {
		t.Fatal("Save style changed the activated target", spy.requests, spy.commits)
	}
}

func TestSaveDialogLoadStyleEmptyRowsAndFallbackStayInert(t *testing.T) {
	a, spy, frames := saveChooserStyleApp(t, 1)
	for _, damage := range []string{"missing", "nil", "empty"} {
		for _, index := range []int{7, 18, 22} {
			t.Run(fmt.Sprintf("frame%d-%s", index, damage), func(t *testing.T) {
				a.SetCutsceneScrollArt(nil)
				fallback := saveChooserStyleFrame(t, a)
				broken := append([]*image.RGBA(nil), frames...)
				switch damage {
				case "missing":
					broken = broken[:index]
				case "nil":
					broken[index] = nil
				case "empty":
					broken[index] = image.NewRGBA(image.Rectangle{})
				}
				a.SetCutsceneScrollArt(broken)
				if !bytes.Equal(saveChooserStyleFrame(t, a).Pix, fallback.Pix) {
					t.Fatal("Save skin did not retain complete missing-art fallback")
				}
			})
		}
	}
	a.SetCutsceneScrollArt(frames)
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, 100, 200); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.saveDialog.request.Name != "save" || len(spy.requests) != 0 {
		t.Fatal("empty painted Save row became actionable")
	}
}

func (s *saveDialogSpy) install(a *App) {
	a.SetSaveDialogSeams(SaveDialogSeams{
		Directory: "saves",
		List: func(path string) (SaveDirectory, error) {
			s.lists = append(s.lists, path)
			if path == "unreadable" {
				return SaveDirectory{}, errors.New("cannot read folder")
			}
			if dir, ok := s.directories[path]; ok {
				return dir, nil
			}
			return SaveDirectory{Path: path}, nil
		},
		Prepare: func(r SaveRequest) (PreparedSave, error) {
			s.requests = append(s.requests, r)
			if s.prepareError != nil {
				return PreparedSave{}, s.prepareError
			}
			paths := s.paths
			if len(paths) == 0 {
				paths = []string{filepath.Join(r.Directory, r.Name+".sav")}
			}
			return PreparedSave{Paths: paths, Existing: s.existing,
				Commit: func(overwrite bool) ([]string, error) {
					s.commits = append(s.commits, overwrite)
					return paths, s.commitError
				}}, nil
		},
	})
}

func newSaveDialogApp(t *testing.T, spy *saveDialogSpy, back Screen) *App {
	t.Helper()
	a := newTestApp(t, nil, nil)
	spy.install(a)
	a.flow.openGameMenu(back)
	if err := a.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenSave {
		t.Fatalf("save action opened %s", a.Screen())
	}
	return a
}

func mustSaveAction(t *testing.T, a *App, action string) {
	t.Helper()
	if err := a.HeadlessSaveAction(action); err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}

func TestSaveDialogF2OpensWithoutWritingAndKeepsExplicitChoice(t *testing.T) {
	for _, onMap := range []bool{false, true} {
		t.Run(map[bool]string{false: "town", true: "mission"}[onMap], func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{Campaign: true} })
			if !onMap {
				h.a.flow.setScreen(ScreenTown)
			}
			s := &saveDialogSpy{}
			s.install(h.a)
			h.frame(appInput{SaveGame: true, Typed: "x", Enter: true})
			state, ok := h.a.HeadlessSaveState()
			if !ok || state.Request.OnMap != onMap || state.Request.Format != SaveSAV || state.Request.Name != "save" || len(s.requests) != 0 {
				t.Fatalf("opening save = %+v, %v, requests %v", state, ok, s.requests)
			}
			if err := h.a.HeadlessSaveEdit("", "explicit", SaveSAV); err != nil {
				t.Fatal(err)
			}
			s.prepareError = errors.New("current party invariant is inconsistent")
			if err := h.a.HeadlessSaveAction("save"); err == nil {
				t.Fatal("invalid current state was accepted")
			}
			state, _ = h.a.HeadlessSaveState()
			if len(s.requests) != 1 || s.requests[0].Format != SaveSAV || len(s.commits) != 0 || state.Request.Format != SaveSAV || state.Message != s.prepareError.Error() {
				t.Fatalf("explicit SAV silently changed: %+v / %+v", state, s)
			}
		})
	}
}

func TestSaveDialogOffersOnlySAV(t *testing.T) {
	for _, back := range []Screen{ScreenTown, ScreenMap} {
		a := newSaveDialogApp(t, &saveDialogSpy{}, back)
		before, _ := a.HeadlessSaveState()
		for _, format := range []SaveFormat{"AGS", "BOTH", "invalid"} {
			if err := a.HeadlessSaveEdit("changed", "changed", format); err == nil {
				t.Fatalf("%s accepts %s", back, format)
			}
			after, _ := a.HeadlessSaveState()
			if after.Request != before.Request {
				t.Fatalf("invalid format edited request: %+v", after.Request)
			}
		}
		if err := a.HeadlessSaveEdit("", "", SaveSAV); err != nil {
			t.Fatal(err)
		}
		for _, c := range a.flow.saveDialog.controls() {
			if name := saveControlName(c); name == "AGS" || name == "BOTH" {
				t.Fatalf("obsolete output control %s", name)
			}
		}
	}
}

func TestSaveDialogEditsUnicodeAndBrowsesDiskNames(t *testing.T) {
	child := filepath.Join("saves", "journey")
	s := &saveDialogSpy{directories: map[string]SaveDirectory{
		"saves": {Path: "saves", Directories: []string{"journey"}, Entries: []SaveEntry{{Name: "real-name.sav", Label: "not the filename"}}},
		child:   {Path: child, Entries: []SaveEntry{{Name: "under.ags"}}},
	}}
	a := newSaveDialogApp(t, s, ScreenTown)
	if err := a.HeadlessSaveSelect(1); err != nil {
		t.Fatal(err)
	}
	state, _ := a.HeadlessSaveState()
	if state.Request.Name != "real-name" || state.Request.Format != SaveSAV {
		t.Fatalf("row changed name/format: %+v", state)
	}
	if err := a.HeadlessSaveSelect(0); err != nil {
		t.Fatal(err)
	}
	state, _ = a.HeadlessSaveState()
	if state.Request.Directory != child {
		t.Fatalf("browse = %q", state.Request.Directory)
	}
	mustSaveAction(t, a, "up")
	if err := a.HeadlessSaveEdit("new-folder", "путь", SaveSAV); err != nil {
		t.Fatal(err)
	}
	a.flow.saveDialog.setFocus(saveNameControl)
	if err := a.HeadlessKey("home"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("right"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("delete"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessType("У", false); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	want := SaveRequest{Directory: "new-folder", Name: "пУть", Format: SaveSAV}
	if len(s.requests) != 1 || s.requests[0] != want || !reflect.DeepEqual(s.commits, []bool{false}) {
		t.Fatalf("typed save = %+v, commits %v; want %+v", s.requests, s.commits, want)
	}
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("saved to %s", a.Screen())
	}
	if err := a.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	state, _ = a.HeadlessSaveState()
	if state.Request.Directory != "new-folder" || state.Request.Name != "save" || state.Request.Format != SaveSAV {
		t.Fatalf("reopened SAVE did not remember only its successful folder: %+v", state.Request)
	}
	if err := a.HeadlessSaveEdit("cancelled-folder", "", ""); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "cancel")
	if err := a.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	state, _ = a.HeadlessSaveState()
	if state.Request.Directory != "new-folder" {
		t.Fatalf("cancel changed remembered folder: %+v", state.Request)
	}
}

func TestSaveDialogConfirmsExactTargetAndDefaultsToBack(t *testing.T) {
	s := &saveDialogSpy{paths: []string{`D:\other\slot.sav`}, existing: []string{`D:\other\slot.sav`}}
	a := newSaveDialogApp(t, s, ScreenTown)
	if err := a.HeadlessSaveEdit("", "slot", SaveSAV); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	h, ok := a.HeadlessSaveState()
	if !ok || !h.Confirmation || h.Focus != "cancel" || !reflect.DeepEqual(h.Paths, s.paths) || !reflect.DeepEqual(h.Existing, s.existing) || len(h.Entries) != 0 {
		t.Fatalf("exact target confirmation = %+v", h)
	}
	if err := a.HeadlessSaveAction("save"); err == nil {
		t.Fatal("save bypassed explicit overwrite")
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	h, _ = a.HeadlessSaveState()
	if h.Confirmation || len(s.commits) != 0 {
		t.Fatal("default Enter overwrote")
	}
	mustSaveAction(t, a, "save")
	mustSaveAction(t, a, "overwrite")
	if !reflect.DeepEqual(s.commits, []bool{true}) || a.Screen() != ScreenGameMenu {
		t.Fatalf("overwrite = %v on %s", s.commits, a.Screen())
	}
}

func TestSaveDialogCommitFailureDiscardsPreparedTargets(t *testing.T) {
	s := &saveDialogSpy{paths: []string{"slot.sav"}, existing: []string{"slot.sav"}, commitError: errors.New("the target changed")}
	a := newSaveDialogApp(t, s, ScreenTown)
	if err := a.HeadlessSaveEdit("failed-folder", "", ""); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	if err := a.HeadlessSaveAction("overwrite"); err == nil {
		t.Fatal("commit failure disappeared")
	}
	h, _ := a.HeadlessSaveState()
	if h.Confirmation || h.Message != "the target changed" {
		t.Fatalf("failure state = %+v", h)
	}
	if err := a.HeadlessSaveAction("overwrite"); err == nil {
		t.Fatal("stale prepared output remained active")
	}
	mustSaveAction(t, a, "save")
	if len(s.requests) != 2 {
		t.Fatalf("retry prepared %d times", len(s.requests))
	}
	mustSaveAction(t, a, "cancel")
	mustSaveAction(t, a, "cancel")
	if len(s.commits) != 1 {
		t.Fatal("cancel wrote a save")
	}
	if a.flow.saveDialogSeams.Directory != "saves" {
		t.Fatal("failed commit changed remembered folder")
	}
}

func TestSaveDialogDirectoryFailureClearsOldRowsAndKeyboardScrolls(t *testing.T) {
	entries := make([]SaveEntry, 10)
	for i := range entries {
		entries[i].Name = string(rune('a'+i)) + ".ags"
	}
	s := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}}
	a := newSaveDialogApp(t, s, ScreenTown)
	if err := a.HeadlessSaveEdit("unreadable", "", ""); err == nil {
		t.Fatal("directory failure disappeared")
	}
	h, _ := a.HeadlessSaveState()
	if len(h.Entries) != 0 || len(a.HeadlessRows()) != 0 || h.Message != "cannot read folder" {
		t.Fatalf("failed directory retained misleading rows: %+v", h)
	}
	if err := a.HeadlessSaveEdit("saves", "", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	top, _ := a.flow.saveDialog.list.Visible()
	if top <= 0 {
		t.Fatal("keyboard selection stayed outside the list window")
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	h, _ = a.HeadlessSaveState()
	if h.Request.Name != "i.ags" || len(s.requests) != 0 {
		t.Fatalf("scrolled selection = %+v", h)
	}
}

func TestSaveDialogPointerRequiresMatchingGestureAndEmptyRowsDoNothing(t *testing.T) {
	s := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: []SaveEntry{{Name: "one.ags"}}}}}
	a := newSaveDialogApp(t, s, ScreenTown)
	a.step(appInput{CursorX: 400, CursorY: 444, PrimaryReleased: true}, headlessNow)
	if len(s.requests) != 0 {
		t.Fatal("orphan release saved")
	}
	a.step(appInput{CursorX: 100, CursorY: 220, PrimaryPressed: true}, headlessNow)
	a.step(appInput{CursorX: 100, CursorY: 220, PrimaryReleased: true}, headlessNow)
	if a.flow.saveDialog.request.Name != "save" {
		t.Fatal("empty list space chose prior row")
	}
	a.step(appInput{CursorX: 100, CursorY: 100, PrimaryPressed: true}, headlessNow)
	a.step(appInput{CursorX: 100, CursorY: 100, PrimaryReleased: true}, headlessNow)
	if a.flow.saveDialog.request.Name != "one.ags" {
		t.Fatal("visible save row did not fill name")
	}
	a.step(appInput{CursorX: 400, CursorY: 444, PrimaryPressed: true}, headlessNow)
	a.step(appInput{CursorX: 520, CursorY: 444, PrimaryReleased: true}, headlessNow)
	if len(s.requests) != 0 || a.Screen() != ScreenSave {
		t.Fatal("cross-button release activated")
	}
}

func TestSaveDialogHoldsMissionClockThroughCancel(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{Campaign: true} })
	s := &saveDialogSpy{}
	s.install(h.a)
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("clock control = %d", got)
	}
	h.frame(appInput{SaveGame: true})
	if h.a.Screen() != ScreenSave || !h.a.flow.popupOpen() {
		t.Fatal("dialog did not raise the popup gate")
	}
	if got := h.run(haltRun, haltNeutral()); got != 0 {
		t.Fatalf("dialog advanced world %d ticks", got)
	}
	before := h.w.world
	h.frame(appInput{Escape: true})
	h.frame(appInput{Escape: true})
	if h.a.Screen() != ScreenMap || h.w.world != before {
		t.Fatal("cancel changed frozen mission")
	}
	if got := h.run(haltRun, haltNeutral()); got != haltRunTicks {
		t.Fatalf("resumed clock accumulated dialog debt: %d", got)
	}
	if len(s.requests) != 0 {
		t.Fatal("cancel prepared a save")
	}
}

func TestComposeScreenSelectsSaveDialogComposer(t *testing.T) {
	a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	got, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	want, err := a.composeSaveDialogScreen()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("save screen dispatch selected another composer")
	}
	if _, note, err := a.HeadlessFrame(); err != nil || note != "" {
		t.Fatalf("headless save frame: %q / %v", note, err)
	}
}

func TestSaveDialogDrawsFieldsAndTitleAtLiteralOrigins(t *testing.T) {
	a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
	font := chargenTestFont()
	a.SetWords(AuthoredWords(), font, nil)
	var pix *image.RGBA
	var err error
	calls := recordWidgets(t, func() { pix, err = a.composeSaveDialogScreen() })
	if err != nil {
		t.Fatal(err)
	}
	deleteButton, nameField := false, false
	for _, c := range calls {
		deleteButton = deleteButton || c.kind == widgetPushButton && c.rect == image.Rect(238, 432, 354, 458)
		nameField = nameField || c.kind == widgetEdit && c.rect == saveControlRect(saveNameControl)
	}
	if pix.RGBAAt(24, 12) != AuthoredDialogueLayout().Border || !nameField || !deleteButton {
		t.Fatal("panel, name field, or delete button moved")
	}
	expect := image.NewRGBA(pix.Bounds())
	font.Draw(expect, "Save game", 38, 23, AuthoredDialogueLayout().TextColor)
	for y := 23; y < 28; y++ {
		for x := 38; x < 65; x++ {
			if expect.RGBAAt(x, y).A != 0 && pix.RGBAAt(x, y) != expect.RGBAAt(x, y) {
				t.Fatalf("title pixel moved at %d,%d", x, y)
			}
		}
	}
}

func TestSaveDialogWrapPreservesWholePathsAndLocalizesMissionMeaning(t *testing.T) {
	path := `D:\` + strings.Repeat("long folder ", 400) + `\actual-target.sav`
	s := &saveDialogSpy{paths: []string{path}, existing: []string{path}}
	a := newSaveDialogApp(t, s, ScreenMap)
	font := chargenTestFont()
	font.Selector = text.SelectorConverting
	a.SetWords(AuthoredWords(), font, nil)
	if err := a.HeadlessSaveEdit("", "", SaveSAV); err != nil {
		t.Fatal(err)
	}
	w := a.flow.saveWords()
	if w.MapSAVDetail != "Продолжить эту карту с момента сохранения." {
		t.Fatalf("RU meaning = %+v", w)
	}
	mustSaveAction(t, a, "save")
	wrapped := a.saveWrapped(path, 548)
	if strings.Join(wrapped, "") != path || len(wrapped) < 18 {
		t.Fatal("confirmation lost exact path characters")
	}
	for i := 0; i < 100; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.saveDialog.confirmTop <= 0 {
		t.Fatal("long confirmation cannot scroll")
	}
}
