package ui

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type spySaveSeams struct {
	saved   []bool
	loaded  []string
	entries []SaveEntry
	failure error
	toTown  bool
}

func (s *spySaveSeams) install(a *App) {
	a.SetSaveSeams(func(onMap bool) (string, error) {
		s.saved = append(s.saved, onMap)
		return "generated.ags", s.failure
	}, func() []SaveEntry { return s.entries }, func(name string) (MapOpener, bool, error) {
		s.loaded = append(s.loaded, name)
		return nil, s.toTown, s.failure
	})
}

func TestFunctionKeysHavePhysicalBindingsAndKeepDiagnostics(t *testing.T) {
	bindings := bindingSource(t)
	for field, key := range map[string]string{
		"SaveGame": "F2", "LoadGame": "F3", "QuickLoad": "F9", "Grid": "F10", "Readout": "F11",
	} {
		want := "inpututil.IsKeyJustPressed(ebiten.Key" + key + ")"
		if bindings[field] != want {
			t.Errorf("binding %s = %q, want %q", field, bindings[field], want)
		}
	}
	for field, suffix := range map[string]string{"QuickSave": " && !shiftHeld", "Reveal": " && shiftHeld"} {
		want := "inpututil.IsKeyJustPressed(ebiten.KeyF4)" + suffix
		if bindings[field] != want {
			t.Errorf("binding %s = %q, want %q", field, bindings[field], want)
		}
	}
	if bindings["ShiftHeld"] != "shiftHeld" {
		t.Errorf("ShiftHeld = %q, want the shared Shift level", bindings["ShiftHeld"])
	}
	if got, want := bindingLocalSource(t, "shiftHeld"), "ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)"; got != want {
		t.Errorf("shiftHeld = %q, want %q", got, want)
	}
	for field, expr := range bindings {
		if strings.Contains(expr, "ebiten.KeyF2)") && field != "SaveGame" ||
			strings.Contains(expr, "ebiten.KeyF3)") && field != "LoadGame" ||
			strings.Contains(expr, "ebiten.KeyF4)") && field != "QuickSave" && field != "Reveal" ||
			strings.Contains(expr, "ebiten.KeyF9)") && field != "QuickLoad" {
			t.Errorf("duplicate original key binding: %s = %s", field, expr)
		}
	}
}

func bindingLocalSource(t *testing.T, name string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "app.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	var out string
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "readAppInput" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			id, ok := assign.Lhs[0].(*ast.Ident)
			if ok && id.Name == name {
				lo, hi := fset.Position(assign.Rhs[0].Pos()).Offset, fset.Position(assign.Rhs[0].End()).Offset
				out = string(src[lo:hi])
			}
			return true
		})
	}
	return out
}

func TestFunctionKeyContextTableUsesOriginalActions(t *testing.T) {
	// Independent transcription of keyboard.tsv rows 12-15. Menu row order
	// and its empty-store Load disable do not supply this expectation.
	for _, tc := range []struct {
		name     string
		town     bool
		campaign bool
		input    appInput
		want     Screen
		page     gameMenuPage
		saves    []bool
	}{
		{"campaign F2", false, true, appInput{SaveGame: true}, ScreenGameMenu, gameMenuRoot, []bool{true}},
		{"campaign F3 empty store", false, true, appInput{LoadGame: true}, ScreenLoad, gameMenuRoot, nil},
		{"campaign simultaneous", false, true, appInput{SaveGame: true, LoadGame: true}, ScreenGameMenu, gameMenuRoot, []bool{true}},
		{"standalone F2", false, false, appInput{SaveGame: true}, ScreenMap, gameMenuRoot, nil},
		{"standalone F3", false, false, appInput{LoadGame: true}, ScreenGameMenu, gameMenuDiplomacyPage, nil},
		{"standalone simultaneous", false, false, appInput{SaveGame: true, LoadGame: true}, ScreenGameMenu, gameMenuDiplomacyPage, nil},
		{"town F2", true, false, appInput{SaveGame: true}, ScreenGameMenu, gameMenuRoot, []bool{false}},
		{"town F3 empty store", true, false, appInput{LoadGame: true}, ScreenLoad, gameMenuRoot, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHaltFix(t, haltOpts{})
			h.v.SetGameMenuContext(func() GameMenuContext {
				return GameMenuContext{Campaign: tc.campaign, Relations: []GameMenuRelation{{Slot: 3, State: "ally"}}}
			})
			if tc.town {
				h.a.SetTown(&stubTown{rows: []TownRow{{Text: "town", Choosable: true}}})
				if !h.a.flow.showTown("") {
					t.Fatal("town unavailable")
				}
			}
			spy := &spySaveSeams{}
			spy.install(h.a)
			h.frame(tc.input)
			if h.a.Screen() != tc.want || h.a.flow.menuPage != tc.page || !reflect.DeepEqual(spy.saved, tc.saves) {
				t.Fatalf("got screen/page/save = %v/%v/%v, want %v/%v/%v", h.a.Screen(), h.a.flow.menuPage, spy.saved, tc.want, tc.page, tc.saves)
			}
			if tc.want == ScreenLoad && (h.a.flow.msg != "no saved games" || h.a.flow.loadBack != ScreenGameMenu) {
				t.Fatalf("empty Load did not open with retained origin: %q / %v", h.a.flow.msg, h.a.flow.loadBack)
			}
			if len(tc.saves) > 0 && h.a.flow.msg != AuthoredWords().SaveAcknowledgement {
				t.Fatalf("Save lost its common acknowledgement: %q", h.a.flow.msg)
			}
			if tc.page == gameMenuDiplomacyPage && !reflect.DeepEqual(h.a.flow.menuContext.Relations, []GameMenuRelation{{Slot: 3, State: "ally"}}) {
				t.Fatal("Diplomacy did not read the live relation context")
			}
		})
	}
}

func TestFunctionKeysPreserveSaveFailureAndModalOwnership(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	spy := &spySaveSeams{failure: errors.New("disk unavailable")}
	spy.install(h.a)
	h.frame(appInput{SaveGame: true})
	if h.a.Screen() != ScreenGameMenu || h.a.flow.msg != "disk unavailable" || !h.v.menuUp {
		t.Fatalf("failed save lost its held error surface: %v/%q", h.a.Screen(), h.a.flow.msg)
	}
	h.frame(appInput{SaveGame: true, LoadGame: true})
	if len(spy.saved) != 1 || h.a.Screen() != ScreenGameMenu || h.a.flow.msg != "disk unavailable" {
		t.Fatal("a key behind the error/menu surface created another action")
	}
}

func TestFunctionKeysRespectFocusPopupTextAndTownDialogue(t *testing.T) {
	for _, key := range []appInput{{SaveGame: true}, {LoadGame: true}} {
		for _, blocked := range []string{"focus", "notice", "notice closes", "popup", "text", "documents", "menu", "picker", "town dialogue"} {
			t.Run(blocked, func(t *testing.T) {
				input := key
				h := newHaltFix(t, haltOpts{})
				spy := &spySaveSeams{}
				spy.install(h.a)
				switch blocked {
				case "focus":
					input.Unfocused = true
				case "notice":
					h.v.SetNotice("notice", NoticeDialogue)
				case "notice closes":
					h.v.SetNotice("notice", NoticeDialogue)
					input.Enter = true
				case "popup":
					h.v.menuUp = true
				case "text":
					h.a.flow.screen = ScreenChargen // character-name text owns keys
				case "documents":
					h.a.flow.screen = ScreenDocuments
				case "menu":
					h.a.flow.screen = ScreenMenu
				case "picker":
					h.a.flow.screen = ScreenPicker
				case "town dialogue":
					h.a.SetTown(&fakeTownDialogue{pic: image.NewRGBA(image.Rect(0, 0, 10, 10))})
					h.a.flow.showTown("")
				}
				before := h.a.Screen()
				h.frame(input)
				if h.a.Screen() != before || len(spy.saved) != 0 || h.a.flow.loadList != nil {
					t.Fatalf("%s failed to own the function key", blocked)
				}
			})
		}
	}
}

func TestFunctionKeyNavigationConsumesSameFrameSideInput(t *testing.T) {
	for _, key := range []appInput{{SaveGame: true}, {LoadGame: true}} {
		h := newHaltFix(t, haltOpts{})
		spy := &spySaveSeams{}
		spy.install(h.a)
		key.Grid, key.Readout, key.LightStep, key.Pause = true, true, true, true
		key.PrimaryPressed, key.PrimaryReleased, key.Enter = true, true, true
		key.Typed = "s"
		key.Viewer = Input{PanRight: true}
		world, anim, cameraX, cameraY := h.w.world, h.v.AnimationCounter(), h.v.Camera().X, h.v.Camera().Y
		h.frame(key)
		if h.w.world != world || h.v.AnimationCounter() != anim || h.v.Camera().X != cameraX || h.v.Camera().Y != cameraY || h.v.GridOverlay() || h.a.flow.stopped {
			t.Fatal("function-key navigation leaked side input into the old map")
		}
		if len(spy.saved) > 1 || len(spy.loaded) != 0 {
			t.Fatal("the navigation frame also activated its new surface")
		}
	}
}

func TestFunctionKeyNavigationOwnsEarlierHeldMouseRelease(t *testing.T) {
	// Literal destination coordinates and effects, independent of menuRows.
	// Earlier-down is the returned review history; same-frame-down is its
	// passing control. Every new route must also permit the next fresh click.
	for _, tc := range []struct {
		name             string
		town, standalone bool
		save             bool
		x, y             int
	}{
		{"map Save", false, false, true, 200, 115},
		{"map Load", false, false, false, 200, 392},
		{"town Save", true, false, true, 200, 155},
		{"town Load", true, false, false, 200, 392},
		{"standalone Diplomacy", false, true, false, 200, 175},
	} {
		for _, downWithKey := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/earlier-down", true: "/same-frame-down"}[downWithKey], func(t *testing.T) {
				h := newHaltFix(t, haltOpts{})
				h.a.Layout(640, 480)
				if tc.town {
					h.a.SetTown(&stubTown{rows: []TownRow{{Text: "town", Choosable: true}}})
					h.a.flow.showTown("")
				}
				if tc.standalone {
					h.v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{} })
				}
				spy := &spySaveSeams{entries: []SaveEntry{{Name: "named.ags", Label: "selected save"}}, failure: errors.New("observed action")}
				spy.install(h.a)
				held := appInput{CursorX: 300, CursorY: 300, Viewer: Input{PrimaryDown: true, CursorX: 300, CursorY: 300}}
				if !downWithKey {
					press := held
					press.PrimaryPressed = true
					h.frame(press)
					h.frame(held)
				}
				key := held
				key.PrimaryPressed = downWithKey
				key.SaveGame, key.LoadGame = tc.save, !tc.save
				h.frame(key)
				wantScreen, wantPage := h.a.Screen(), h.a.flow.menuPage
				wantSaves := 0
				if tc.save {
					wantSaves = 1
				}
				h.frame(held)
				h.frame(held)
				release := appInput{PrimaryReleased: true, CursorX: tc.x, CursorY: tc.y, Viewer: Input{CursorX: tc.x, CursorY: tc.y}}
				h.frame(release)
				if len(spy.saved) != wantSaves || len(spy.loaded) != 0 || h.a.Screen() != wantScreen || h.a.flow.menuPage != wantPage {
					t.Fatalf("old-screen release activated destination: saves=%d want=%d loads=%d screen=%v page=%v", len(spy.saved), wantSaves, len(spy.loaded), h.a.Screen(), h.a.flow.menuPage)
				}
				press := release
				press.PrimaryPressed, press.PrimaryReleased, press.Viewer.PrimaryDown = true, false, true
				h.frame(press)
				h.frame(release)
				switch {
				case tc.save:
					if len(spy.saved) != 2 {
						t.Fatal("fresh Save click was suppressed")
					}
				case tc.standalone:
					if h.a.flow.menuPage != gameMenuRoot {
						t.Fatal("fresh Return click was suppressed")
					}
				default:
					if len(spy.loaded) != 1 {
						t.Fatal("fresh Load click was suppressed")
					}
				}
			})
		}
	}
}

func TestFunctionKeyLoadIdleThenImmediateEscapeOrReturnOwesNoTime(t *testing.T) {
	for _, escape := range []bool{true, false} {
		h := newHaltFix(t, haltOpts{})
		spy := &spySaveSeams{}
		spy.install(h.a)
		h.frame(appInput{LoadGame: true})
		if h.a.Screen() != ScreenLoad {
			t.Fatal("F3 did not open Load")
		}
		world, anim := h.w.world, h.v.AnimationCounter()
		for i := 0; i < 310; i++ { // 31 seconds, not one missed ordinary tick
			h.frame(haltNeutral())
		}
		h.frame(appInput{Escape: true}) // Load -> existing menu, no neutral frame
		if escape {
			h.frame(appInput{Escape: true})
		} else {
			// Independently use the root's final Return accelerator, not a row
			// selected from menuRows under test.
			h.frame(appInput{Typed: "r"})
		}
		if h.a.Screen() != ScreenMap || h.w.world != world || h.v.AnimationCounter() != anim {
			t.Fatalf("escape=%v repaid time during dismissal: world=%d anim=%d", escape, h.w.world-world, h.v.AnimationCounter()-anim)
		}
		h.frame(haltNeutral())
		// One ordinary 100-ms frame may earn at most two 62-ms sim ticks
		// and one 100-ms ambient frame, never the 31-second hidden interval.
		if got := h.w.world - world; got < 1 || got > 2 {
			t.Fatalf("escape=%v resumed with %d sim ticks, want 1..2", escape, got)
		}
		if got := h.v.AnimationCounter() - anim; got > 1 {
			t.Fatalf("escape=%v resumed with %d ambient ticks, want <=1", escape, got)
		}
	}
}

func TestFunctionKeyLoadFailureAndSuccessKeepExistingLoadRoute(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	h.a.SetTown(&stubTown{rows: []TownRow{{Text: "town", Choosable: true}}})
	spy := &spySaveSeams{entries: []SaveEntry{{Name: "named.ags", Label: "selected save"}}, failure: errors.New("bad save")}
	spy.install(h.a)
	h.frame(appInput{LoadGame: true})
	h.frame(appInput{Enter: true})
	if h.a.Screen() != ScreenLoad || h.a.flow.msg != "named.ags: bad save" {
		t.Fatalf("load refusal lost its file and retry surface: %v %q", h.a.Screen(), h.a.flow.msg)
	}
	spy.failure, spy.toTown = nil, true
	h.frame(appInput{Enter: true})
	if h.a.Screen() != ScreenTown || !reflect.DeepEqual(spy.loaded, []string{"named.ags", "named.ags"}) {
		t.Fatal("successful F3 Load did not use the common load destination")
	}
}

func TestFunctionKeysAreAvailableThroughHeadlessPhysicalVocabulary(t *testing.T) {
	h := newHaltFix(t, haltOpts{})
	spy := &spySaveSeams{}
	spy.install(h.a)
	if err := h.a.HeadlessKey("f2"); err != nil || len(spy.saved) != 1 {
		t.Fatalf("headless F2: saves=%v err=%v", spy.saved, err)
	}
	h.a.step(appInput{Escape: true}, time.Unix(100, 0))
	if err := h.a.HeadlessKey("f3"); err != nil || h.a.Screen() != ScreenLoad {
		t.Fatalf("headless F3: screen=%v err=%v", h.a.Screen(), err)
	}
}
