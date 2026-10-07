package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"io/fs"
	"strings"
	"testing"
)

// clRegistry is a synthetic registry carrying the four slot names this build
// wires, with the shipped hotspots, frame counts and periods for the two the
// cases below place or animate. It is not shipped art: what the real archives
// resolve to is pkg/game/cursorregistry_release_test.go's question.
func clRegistry() *CursorRegistry {
	return NewCursorRegistry([]CursorSlot{
		{Name: "default", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(5, 5), FrameCount: 1, PeriodMillis: 2000000000},
		{Name: "select", Frames: []*image.RGBA{{}}, Hotspot: image.Pt(3, 4), FrameCount: 1, PeriodMillis: 100},
		{Name: "wait", Frames: []*image.RGBA{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, Hotspot: image.Pt(16, 16), FrameCount: 10, PeriodMillis: 100},
		{Name: "attack", Frames: []*image.RGBA{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}}, Hotspot: image.Pt(3, 3), FrameCount: 10, PeriodMillis: 100},
	})
}

// TestEveryPathToAScreenRunsThatScreensTransition is B4's own witness, and the
// one that fails on the defect adversarial pass 1 returned the story for: the
// main menu reached by backing out of character generation drew `default`,
// because Escape restores f.chargenBack directly and toMenu — which carried the
// menu's `select` exit — is not on that path.
//
// It walks the paths, not the transitions. Each step is a production method a
// player's own key reaches, and the assertion is the cursor name afterwards.
//
// MUTATIONS THIS FAILS AGAINST: surfaceTransition gutted to a no-op; any
// screenExitCursor row changed or removed; setScreen not running the table;
// any one path restored to a bare f.screen assignment.
func TestEveryPathToAScreenRunsThatScreensTransition(t *testing.T) {
	f := newFlow(NewPicker(appRows(3)), nil)
	f.town = &stubTown{rows: []TownRow{{Text: "gates", Choosable: true}}}
	f.cursor.SetRegistry(clRegistry())

	want := func(step, name string) {
		t.Helper()
		if got := f.cursor.CurrentName(); got != name {
			t.Errorf("after %s: cursor = %q, want %q", step, got, name)
		}
	}

	f.toMenu()
	want("toMenu", "select")

	if !f.armChargen(NewChargen(chargenLegalSetup()), nil, ScreenMenu) {
		t.Fatal("armChargen refused a legal setup")
	}
	want("menu -> chargen", "default")

	f.escape()
	if f.screen != ScreenMenu {
		t.Fatalf("Escape from chargen landed on %v, want the menu", f.screen)
	}
	want("chargen -> Escape -> menu", "select")

	f.activateNewGame()
	if f.screen != ScreenPicker {
		t.Fatalf("activateNewGame landed on %v, want the picker", f.screen)
	}
	// The picker runs no transition of its own, so B2's persistence rule
	// leaves the menu's own cursor current. That is the behaviour, not an
	// omission: five of TOWN-372's seventeen routines set no cursor at all.
	want("menu -> picker", "select")

	if !f.armChargen(NewChargen(chargenLegalSetup()), nil, ScreenPicker) {
		t.Fatal("armChargen refused a legal setup from the picker")
	}
	f.escape()
	if f.screen != ScreenPicker {
		t.Fatalf("Escape from chargen landed on %v, want the picker it was armed from", f.screen)
	}
	want("picker -> chargen -> Escape -> picker", "default")

	if !f.showTown("") {
		t.Fatal("showTown refused a town that is installed")
	}
	want("town", "default")

	f.openGameMenu(ScreenTown)
	f.cursor.SetCursor("select") // whatever the menu screen leaves current
	f.closeGameMenu()
	if f.screen != ScreenTown {
		t.Fatalf("the in-game menu returned to %v, want the town it was opened from", f.screen)
	}
	want("town -> in-game menu -> return -> town", "default")

	f.openLoad(ScreenMenu)
	f.escape()
	if f.screen != ScreenMenu {
		t.Fatalf("Escape from the load window landed on %v, want the menu", f.screen)
	}
	want("load window -> Escape -> menu", "select")
}

// TestTheBootScreensTransitionRunsWhenTheRegistryArrives covers the one
// transition that cannot run at the moment its screen is reached: every
// SetCursor before a registry is installed is a no-op, and the front-end opens
// on the menu before App.SetCursorRegistry is called.
//
// MUTATION THIS FAILS AGAINST: cutting the SetCursorRegistry wire, which is the
// one statement that makes this story do anything at all.
func TestTheBootScreensTransitionRunsWhenTheRegistryArrives(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	if a.Screen() != ScreenMenu {
		t.Fatalf("a new App opens on %v, want the menu", a.Screen())
	}
	if got := a.flow.cursor.CurrentName(); got != "" {
		t.Fatalf("a cursor is current with no registry installed: %q", got)
	}
	a.SetCursorRegistry(clRegistry())
	if got := a.flow.cursor.CurrentName(); got != "select" {
		t.Fatalf("the boot menu's cursor = %q, want select (MENU-CURSOR-046)", got)
	}
}

func TestTheSystemPointerIsHiddenExactlyWhenThisBuildDrawsOne(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(clRegistry())

	frame := func(step string, hidden bool) {
		t.Helper()
		a.flow.syncPointerMode()
		if got := a.flow.cursor.PointerHidden(); got != hidden {
			t.Errorf("after %s: the engine is told hidden=%v, want %v", step, got, hidden)
		}
		if got := a.flow.pointerWanted(); got != hidden {
			t.Errorf("after %s: pointerWanted = %v, want %v", step, got, hidden)
		}
	}

	// A viewer the camera has never stepped has no observed cursor position
	// (hasCursor), which is what mapCursorPresent and attackPointerPresent
	// both gate on, so the operating system's arrow is still showing here even
	// though a cursor IS current in the manager. Once the camera has stepped,
	// this build draws its own map cursor outside attack mode too (story
	// 1031); missioncursor_test.go carries that witness.
	if a.flow.cursor.CurrentName() == "" {
		t.Fatal("setup: no cursor is current on the map screen")
	}
	frame("entering the map", false)

	x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
	a.step(afHeld(x, y), atAt)
	if _, _, shown := v.attackPointerPresent(); !shown {
		t.Fatal("setup: the attack pointer is not shown with the mode up")
	}
	frame("raising attack mode on the map", true)

	a.step(atFrame(x, y), atAt)
	if _, _, shown := v.attackPointerPresent(); shown {
		t.Fatal("setup: the attack pointer is still shown with the mode down")
	}
	frame("lowering attack mode on the map", false)

	// Off the map, the manager's picture is what is drawn, so the system
	// pointer goes. This is the mirror state: before the fix the engine was
	// left Visible by the viewer's cache while the manager's said hidden, and
	// the player saw two pointers at once.
	a.flow.leaveMap()
	a.flow.toMenu()
	frame("returning to the menu", true)
}

// TestLeavingAttackModeLowersTheMapsOwnCursor is the reachable half of a
// comment that was false: a mission that ends sends the flow to the map list,
// which runs no transition, so a manager left on "attack" drew the animated
// sword over the list of maps.
func TestLeavingAttackModeLowersTheMapsOwnCursor(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(clRegistry())
	x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
	// Two frames: the manager's clock stands above the map arm's own popup
	// return, and the attack mode is raised below it, so the selection is one
	// frame behind the mode (advanceCursorManager's own header). Neither edge
	// is visible — the fallback picture is frame 0 of the same sheet.
	a.step(afHeld(x, y), atAt)
	a.step(afHeld(x, y), atAt)
	if got := a.flow.cursor.CurrentName(); got != "attack" {
		t.Fatalf("with the mode up the cursor is %q, want attack", got)
	}
	a.step(atFrame(x, y), atAt)
	a.step(atFrame(x, y), atAt)
	if got := a.flow.cursor.CurrentName(); got != "default" {
		t.Fatalf("with the mode down the cursor is %q, want the map's own default", got)
	}
}

// screenWrite is one syntactic site that writes a `screen` field, the form it
// is written in, and the function it sits in.
type screenWrite struct {
	form string
	fn   string
	pos  string
}

// screenWrites reports every site in one parsed file that can put a value into
// a `screen` field. THE FORMS ARE THE POINT: a scan that reads assignments
// alone is blind to the other three, and one of them is in production.
//
//   - assign: `x.screen = v`, and every compound and multi-value form of it.
//   - complit: `T{screen: v}`, which sets the field with no statement at all.
//   - incdec: `x.screen++`, which a Screen's integer base type permits.
//   - addr: `&x.screen`, a pointer alias that can be assigned through later,
//     out of this file and out of any scan's reach.
//
// It attributes each site to its enclosing top-level declaration rather than
// to the last FuncDecl an ast.Inspect happened to pass, so a package-level
// composite literal is reported as such instead of inheriting a name.
func screenWrites(fset *token.FileSet, file *ast.File) []screenWrite {
	var out []screenWrite
	for _, decl := range file.Decls {
		fn := "<package level>"
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fn = fd.Name.Name
		}
		add := func(form string, n ast.Node) {
			out = append(out, screenWrite{form: form, fn: fn, pos: fset.Position(n.Pos()).String()})
		}
		isScreenField := func(e ast.Expr) bool {
			sel, ok := e.(*ast.SelectorExpr)
			return ok && sel.Sel.Name == "screen"
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.AssignStmt:
				for _, lhs := range v.Lhs {
					if isScreenField(lhs) {
						add("assign", v)
					}
				}
			case *ast.CompositeLit:
				for _, el := range v.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "screen" {
						add("complit", v)
					}
				}
			case *ast.IncDecStmt:
				if isScreenField(v.X) {
					add("incdec", v)
				}
			case *ast.UnaryExpr:
				if v.Op == token.AND && isScreenField(v.X) {
					add("addr", v)
				}
			}
			return true
		})
	}
	return out
}

// TestScreenIsAssignedOnlyBySetScreen closes the CLASS rather than the site.
// Adversarial pass 1 found one path to the main menu that ran no transition;
// the population at the time was fifteen assignments over three files, three of
// which restored a remembered screen. A per-path cursor pair is a rule every
// future path has to remember, so the assignment itself is the one place the
// table can be read from, and this is what keeps it that way.
//
// It scans this package's own production source. A test that asserted the four
// known paths instead would pass the day a fifth is written.
//
// WHAT THE CLASS IS NOW CLOSED AGAINST: assignment, composite literal,
// increment, and taking the field's address. An earlier revision inspected
// ast.AssignStmt alone, and newFlow's own `&flow{screen: ScreenMenu, ...}` was
// already invisible to it.
//
// WHAT IT IS STILL NOT CLOSED AGAINST: a write through a pointer obtained
// somewhere this scan cannot follow (reflection, unsafe, or an alias passed out
// of the package), and a write from outside package ui. The field is unexported
// and the addr form above is what would produce such an alias here, so the gap
// is reachable only by code this package does not contain today.
//
// NEWFLOW'S LITERAL IS ALLOWED BY NAME. It runs no transition, which is correct
// and not an exemption from the rule: at that moment no registry is installed,
// every SetCursor is a no-op, and App.SetCursorRegistry runs the current
// screen's entry when one arrives. TestTheBootScreensTransitionRunsWhenTheRegistryArrives
// is the witness for that repair.
//
// MUTATIONS THIS FAILS AGAINST: any path restored to a bare `f.screen =
// f.menuBack`; a second composite literal setting the field.
func TestScreenIsAssignedOnlyBySetScreen(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing this package: %v", err)
	}
	byForm := map[string]int{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, w := range screenWrites(fset, file) {
				byForm[w.form]++
				switch {
				case w.form == "assign" && w.fn == "setScreen":
				case w.form == "complit" && w.fn == "newFlow":
				default:
					t.Errorf("%s: %s writes .screen as a %s; every screen change goes through flow.setScreen so the surface transition (B4) cannot be skipped",
						w.pos, w.fn, w.form)
				}
			}
		}
	}
	if byForm["assign"] == 0 {
		t.Error("the scan found no assignment to .screen at all, so it is measuring nothing")
	}
	if byForm["complit"] == 0 {
		t.Error("the scan found no composite literal setting .screen, and newFlow's own is one")
	}
	// incdec and addr have population ZERO in this package and that is the
	// correct state, so there is no count to require here. What proves those
	// two arms of the scan are not dead is
	// TestTheScreenWriteScanSeesEveryFormItCloses below, which runs the same
	// matcher over source carrying one of each.
	if n := byForm["incdec"] + byForm["addr"]; n != 0 {
		t.Errorf("%d increment or address-of writes to .screen; both are reported above", n)
	}
}

// TestTheScreenWriteScanSeesEveryFormItCloses is the witness for the scan
// itself. Two of the four forms it closes have population zero in production,
// so a widening that matched nothing would leave TestScreenIsAssignedOnlyBySetScreen
// passing exactly as it does now. The same matcher is run over source that
// carries one site of each form and must report all four.
//
// MUTATION THIS FAILS AGAINST: deleting any one arm of screenWrites.
func TestTheScreenWriteScanSeesEveryFormItCloses(t *testing.T) {
	const src = `package p

type s struct{ screen int }

func newS() *s { return &s{screen: 1} }

func (x *s) set(v int) { x.screen = v }

func (x *s) bump() { x.screen++ }

func (x *s) alias() *int { return &x.screen }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatalf("parsing the fixture: %v", err)
	}
	got := map[string]string{}
	for _, w := range screenWrites(fset, file) {
		got[w.form] = w.fn
	}
	for form, fn := range map[string]string{"assign": "set", "complit": "newS", "incdec": "bump", "addr": "alias"} {
		if got[form] != fn {
			t.Errorf("the scan reports the %s form in %q, want %q; that arm sees nothing and closes nothing", form, got[form], fn)
		}
	}
	if len(got) != 4 {
		t.Errorf("the scan reported %d forms over the fixture, want 4: %v", len(got), got)
	}
}

func TestCursorPictureIsPlacedByItsHotspot(t *testing.T) {
	f := newFlow(NewPicker(nil), nil)
	f.cursor.SetRegistry(clRegistry())

	tip := image.Pt(100, 50)
	if _, _, ok := f.cursorPresent(tip); ok {
		t.Fatal("a picture is present with no cursor set")
	}
	f.cursor.SetCursor("default") // hotspot (5,5)
	_, at, ok := f.cursorPresent(tip)
	if !ok {
		t.Fatal("no picture with a cursor set")
	}
	if at != image.Pt(95, 45) {
		t.Errorf("origin = %v, want (95,45): the (5,5) hotspot pixel lands on the cursor point", at)
	}
	f.cursor.SetCursor("select") // hotspot (3,4)
	if _, at, _ = f.cursorPresent(tip); at != image.Pt(97, 46) {
		t.Errorf("origin = %v, want (97,46): each registration carries its own hotspot", at)
	}
}

func TestEveryAnimationFrameReachesTheUpload(t *testing.T) {
	var tex cursorTexture
	a, b := &image.RGBA{}, &image.RGBA{}
	if !tex.stale(a) {
		t.Fatal("the first picture of a session is not stale, so nothing is ever uploaded")
	}
	tex.src = a
	if tex.stale(a) {
		t.Error("the picture already uploaded is stale, so the texture is rewritten every frame")
	}
	if !tex.stale(b) {
		t.Fatal("a different picture is not stale, so a new animation frame never reaches the screen")
	}

	// And over the manager's own counter: ten frames at a 100ms period, each
	// one a picture the draw path has not uploaded yet.
	m := NewCursorManager()
	m.SetRegistry(clRegistry())
	m.SetCursor("attack")
	var run cursorTexture
	uploads := 0
	for now := int64(0); now < 1000; now += 150 {
		m.Advance(now)
		pic, _, ok := m.Current()
		if !ok {
			t.Fatal("Current() = not ok while attack is the current cursor")
		}
		if run.stale(pic) {
			uploads++
			run.src = pic
		}
	}
	if uploads < 6 {
		t.Errorf("%d of 7 frames past the period would have been uploaded; the animation is standing still", uploads)
	}
}

// TestTheMapsAnimatedPointerIsPlacedByTheRegistrationsOwnHotspot puts an
// assertion on the arm of attackPointerPresent that the real game runs.
//
// WHY IT IS SEPARATE FROM pointer_test.go. atOnMap builds an App whose manager
// has no registry, so CursorManager.Current answers not-ok and every case in
// that file falls through to the story-0080 arm. Adversarial pass 2 measured
// it: a panic in the manager arm left the whole package at exit 0. The two arms
// are indistinguishable to pointer_test.go's origin assertion because
// AttackPointerHotspot and the attack registration's hotspot are the same
// (3,3), so which arm answered is established here by the PICTURE, and only
// then does the origin assertion measure the manager arm's own subtraction.
//
// The hotspot (3,3), the ten frames and the 100ms period are
// SPR16A-CURSOR-046's attack row; clRegistry carries the same values.
//
// MUTATIONS THIS FAILS AGAINST: the manager arm dropping the hotspot, written
// so it compiles (`pic, _, ok := ...` and `image.Pt(v.cursorX, v.cursorY)`) —
// that mutation leaves every other test in this package at exit 0; and the
// manager arm removed altogether, which hands back the still picture
// SetAttackPointer supplied.
func TestTheMapsAnimatedPointerIsPlacedByTheRegistrationsOwnHotspot(t *testing.T) {
	a, v, _ := atOnMap(t)
	a.SetCursorRegistry(clRegistry())
	still := ptPic()
	v.SetAttackPointer(still)

	x, y := ptHover(a, v, atEmptyCol, atEmptyRow)
	// Two frames: the manager's clock stands above the map arm's own popup
	// return and the mode is raised below it, so the selection is one frame
	// behind the mode (advanceCursorManager's own header).
	a.step(afHeld(x, y), atAt)
	a.step(afHeld(x, y), atAt)
	if got := a.flow.cursor.CurrentName(); got != "attack" {
		t.Fatalf("with the mode up the manager's cursor is %q, want attack", got)
	}

	pic, at, ok := v.attackPointerPresent()
	if !ok {
		t.Fatal("no pointer with the mode up and a registry installed")
	}
	if pic == still {
		t.Fatal("the pointer is the still story-0080 picture, so the manager arm was not taken and the origin below measures the fallback")
	}
	want, hot, _ := a.flow.cursor.Current()
	if pic != want {
		t.Errorf("the pointer is %v, want the manager's own current frame %v", pic, want)
	}
	if hot != image.Pt(3, 3) {
		t.Fatalf("setup: the attack registration's hotspot is %v, want (3,3) per SPR16A-CURSOR-046", hot)
	}
	if want := image.Pt(x-3, y-3); at != want {
		t.Errorf("the pointer goes at %v, want %v: the cursor less the attack registration's own hotspot (3,3)", at, want)
	}
}
