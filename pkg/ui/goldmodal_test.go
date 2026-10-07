package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

// purseApp is an open map with entity 5 selected and a two-cell pack: one item
// and a purse of 2500 (cell 1).
func purseApp(t *testing.T) (*App, *Viewer) {
	t.Helper()
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, wornBoxFixtureH)
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	v.sel = selection{5}
	v.SetInventorySubject(InventorySubject{
		ID:        5,
		Pack:      []*image.RGBA{solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff}), solidPic(8, 8, color.RGBA{G: 0xff, A: 0xff})},
		PackCount: []uint32{1, 2500},
		PackPurse: []bool{false, true},
	})
	if _, n, ok := v.goldPurse(); !ok || n != 2500 {
		t.Fatalf("setup: purse = %d, %v", n, ok)
	}
	return a, v
}

type purseGesture struct {
	t *testing.T
	a *App
}

func (g purseGesture) press(x, y int, shift bool) {
	g.a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true, ShiftHeld: shift,
		Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true, Shift: shift}}, commandFrozen)
}

func (g purseGesture) move(x, y int, shift bool) {
	g.a.step(appInput{CursorX: x, CursorY: y, ShiftHeld: shift,
		Viewer: Input{CursorX: x, CursorY: y, PrimaryDown: true, Shift: shift}}, commandFrozen)
}

func (g purseGesture) release(x, y int) {
	g.a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true,
		Viewer: Input{CursorX: x, CursorY: y}}, commandFrozen)
}

func (g purseGesture) key(in appInput) {
	g.a.step(in, commandFrozen)
}

func (g purseGesture) doubleClick(x, y int) {
	g.press(x, y, false)
	g.release(x, y)
	g.press(x, y, false)
	g.release(x, y)
}

func (g purseGesture) ground() (int, int) {
	x, y, err := g.a.HeadlessGroundPoint()
	if err != nil {
		g.t.Fatal(err)
	}
	return x, y
}

func purseCenter(t *testing.T, v *Viewer) (int, int) {
	t.Helper()
	return packCellCenter(t, v, 1)
}

// The purse drag reserves one thousand in the preview without touching the
// stored subject, and a release on the ground spends that share as a request
// at the released cell. No item request is raised.
func TestPurseDragReservesPreviewAndRequestsOnGround(t *testing.T) {
	a, v := purseApp(t)
	g := purseGesture{t, a}
	cx, cy := purseCenter(t, v)
	g.press(cx, cy, false)
	g.move(cx+3*TapSlop, cy, false)
	if v.GoldHeld() != 1000 || v.invSubject.PackCount[1] != 2500 || v.previewSubject().PackCount[1] != 1500 {
		t.Fatalf("held %d stored %d preview %d, want 1000 2500 1500", v.GoldHeld(), v.invSubject.PackCount[1], v.previewSubject().PackCount[1])
	}
	gx, gy := g.ground()
	g.release(gx, gy)
	req, ok := v.TakeGoldDrop()
	if !ok || req.Amount != 1000 || req.AtSubject {
		t.Fatalf("request %+v, %v, want 1000 at the released cell", req, ok)
	}
	if wx, wy, err := a.HeadlessDropCell(gx, gy); err != nil || req.X != int32(wx) || req.Y != int32(wy) {
		t.Fatalf("request cell (%d,%d), release resolves to (%d,%d), %v", req.X, req.Y, wx, wy, err)
	}
	if v.GoldHeld() != 0 {
		t.Fatalf("held %d after the release", v.GoldHeld())
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Fatalf("the purse raised an equip request for %d", idx)
	}
	if _, idx, _, _, ok := v.TakeInventoryDrop(); ok {
		t.Fatalf("the purse raised an item drop for %d", idx)
	}
}

// Shift takes the whole available balance, and a share never exceeds it.
func TestPurseDragWithShiftTakesTheBalanceAndShareIsBounded(t *testing.T) {
	a, v := purseApp(t)
	g := purseGesture{t, a}
	cx, cy := purseCenter(t, v)
	g.press(cx, cy, true)
	g.move(cx+3*TapSlop, cy, true)
	if v.GoldHeld() != 2500 || v.previewSubject().PackCount[1] != 0 {
		t.Fatalf("shift held %d preview %d, want 2500 and 0", v.GoldHeld(), v.previewSubject().PackCount[1])
	}
	gx, gy := g.ground()
	g.release(gx, gy)
	if req, ok := v.TakeGoldDrop(); !ok || req.Amount != 2500 {
		t.Fatalf("shift request %+v, %v", req, ok)
	}

	v.SetInventorySubject(InventorySubject{ID: 5, Pack: v.invSubject.Pack, PackCount: []uint32{1, 70}, PackPurse: []bool{false, true}})
	g.press(cx, cy, false)
	g.move(cx+3*TapSlop, cy, false)
	if v.GoldHeld() != 70 {
		t.Fatalf("a purse of 70 held %d", v.GoldHeld())
	}
}

// A release anywhere but the ground returns the share: no request, no held
// amount and no item action.
func TestPurseDragReleasedOffTheGroundRequestsNothing(t *testing.T) {
	for _, name := range []string{"doll box", "origin cell", "pack bar"} {
		t.Run(name, func(t *testing.T) {
			a, v := purseApp(t)
			g := purseGesture{t, a}
			cx, cy := purseCenter(t, v)
			box, ok := v.heldItemBox()
			if !ok {
				t.Fatal("setup: no doll box")
			}
			at := map[string]image.Point{
				"doll box":    box.Min.Add(box.Max).Div(2),
				"origin cell": image.Pt(cx, cy),
				"pack bar":    image.Pt(cx+1, cy),
			}[name]
			g.press(cx, cy, false)
			g.move(cx+3*TapSlop, cy, false)
			g.release(at.X, at.Y)
			if req, ok := v.TakeGoldDrop(); ok {
				t.Errorf("request %+v", req)
			}
			if v.GoldHeld() != 0 {
				t.Errorf("held %d", v.GoldHeld())
			}
			if idx, ok := v.TakeInventoryEquip(); ok {
				t.Errorf("equip request %d", idx)
			}
			if v.goldModalOpen() {
				t.Error("the release opened the editor")
			}
		})
	}
}

// A double-click on the purse opens the editor over the literal text 0 and
// raises no equip request; a double-click on an item still equips.
func TestPurseDoubleClickOpensTheEditorAndItemDoubleClickStillEquips(t *testing.T) {
	a, v := purseApp(t)
	g := purseGesture{t, a}
	cx, cy := purseCenter(t, v)
	g.doubleClick(cx, cy)
	got, open := a.HeadlessGold()
	if !open || !got.Open || got.Text != "0" || got.Caret != 0 || got.SelStart != 0 || got.SelEnd != 0 || got.Focus != "edit" {
		t.Fatalf("editor state %+v, %v", got, open)
	}
	if idx, ok := v.TakeInventoryEquip(); ok {
		t.Fatalf("the purse raised an equip request for %d", idx)
	}
	g.key(appInput{Escape: true})
	if _, open := a.HeadlessGold(); open {
		t.Fatal("Escape did not close the editor")
	}

	ix, iy := packCellCenter(t, v, 0)
	g.doubleClick(ix, iy)
	if idx, ok := v.TakeInventoryEquip(); !ok || idx != 0 {
		t.Fatalf("item double-click equip = (%d, %v)", idx, ok)
	}
	if _, open := a.HeadlessGold(); open {
		t.Fatal("an item double-click opened the editor")
	}
}

func openPurseEditor(t *testing.T) (*App, *Viewer, purseGesture) {
	t.Helper()
	a, v := purseApp(t)
	g := purseGesture{t, a}
	cx, cy := purseCenter(t, v)
	g.doubleClick(cx, cy)
	if !v.goldModalOpen() {
		t.Fatal("setup: the editor did not open")
	}
	return a, v, g
}

func typeGold(g purseGesture, s string) {
	for _, r := range s {
		g.key(appInput{Typed: string(r)})
	}
}

// Delete clears the prefilled 0, typed digits fill the field and Enter submits
// the amount at the selected owner's cell. The stored purse is untouched.
func TestPurseEditorSubmitsTheTypedAmountAtTheSubject(t *testing.T) {
	a, v, g := openPurseEditor(t)
	g.key(appInput{Delete: true})
	typeGold(g, "700")
	if got, _ := a.HeadlessGold(); got.Text != "700" || got.Caret != 3 {
		t.Fatalf("editor %+v, want text 700 caret 3", got)
	}
	g.key(appInput{Enter: true})
	req, ok := v.TakeGoldDrop()
	if !ok || req.Amount != 700 || !req.AtSubject {
		t.Fatalf("request %+v, %v", req, ok)
	}
	if _, open := a.HeadlessGold(); open {
		t.Fatal("the editor stayed open after Enter")
	}
	if v.invSubject.PackCount[1] != 2500 {
		t.Fatalf("the stored purse changed to %d", v.invSubject.PackCount[1])
	}
}

// A request above the purse takes the available balance; zero, malformed and
// refused text request nothing.
func TestPurseEditorClampsToTheBalanceAndRefusesUnparsableText(t *testing.T) {
	cases := []struct {
		name string
		text string
		want uint32
	}{
		{"above the purse", "99999999999", 2500},
		{"exact", "2500", 2500},
		{"fraction truncates", "12.9", 12},
		{"zero", "", 0},
		{"letters", "abc", 0},
		{"exponent", "1e3", 0},
		{"percent", "50%", 0},
		{"negative", "-5", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, v, g := openPurseEditor(t)
			g.key(appInput{Delete: true})
			typeGold(g, c.text)
			g.key(appInput{Enter: true})
			req, ok := v.TakeGoldDrop()
			if c.want == 0 && ok || c.want != 0 && (!ok || req.Amount != c.want) {
				t.Fatalf("text %q gave request %+v, %v, want %d", c.text, req, ok, c.want)
			}
			if _, open := a.HeadlessGold(); open {
				t.Fatal("the editor stayed open")
			}
		})
	}
}

// Cancel, Escape and a focused Cancel button's Enter all request nothing.
func TestPurseEditorCancelPathsRequestNothing(t *testing.T) {
	for name, close := range map[string]func(purseGesture, *App){
		"Escape": func(g purseGesture, a *App) { g.key(appInput{Escape: true}) },
		"Tab to Cancel then Enter": func(g purseGesture, a *App) {
			g.key(appInput{PaneMode: true})
			g.key(appInput{PaneMode: true})
			if got, _ := a.HeadlessGold(); got.Focus != "cancel" {
				g.t.Fatalf("focus %q after two Tabs", got.Focus)
			}
			g.key(appInput{Enter: true})
		},
		"Cancel click": func(g purseGesture, a *App) {
			x, y, err := a.HeadlessGoldControlPoint("cancel")
			if err != nil {
				g.t.Fatal(err)
			}
			g.press(x, y, false)
			g.release(x, y)
		},
	} {
		t.Run(name, func(t *testing.T) {
			a, v, g := openPurseEditor(t)
			g.key(appInput{Delete: true})
			typeGold(g, "700")
			close(g, a)
			if _, open := a.HeadlessGold(); open {
				t.Fatal("the editor stayed open")
			}
			if req, ok := v.TakeGoldDrop(); ok {
				t.Fatalf("request %+v", req)
			}
		})
	}
}

// The action button submits on a release over it, and Shift+Tab walks the
// focus ring backwards.
func TestPurseEditorActionClickAndFocusRing(t *testing.T) {
	a, v, g := openPurseEditor(t)
	g.key(appInput{Delete: true})
	typeGold(g, "5")
	g.key(appInput{PaneMode: true, ShiftHeld: true})
	if got, _ := a.HeadlessGold(); got.Focus != "cancel" {
		t.Fatalf("Shift+Tab from edit = %q, want cancel", got.Focus)
	}
	x, y, err := a.HeadlessGoldControlPoint("action")
	if err != nil {
		t.Fatal(err)
	}
	g.press(x, y, false)
	g.release(x, y)
	if req, ok := v.TakeGoldDrop(); !ok || req.Amount != 5 {
		t.Fatalf("action click request %+v, %v", req, ok)
	}
}

// Text keys reach the edit only while it holds focus, and an unfocused window
// delivers nothing.
func TestPurseEditorGatesInputByFocus(t *testing.T) {
	a, v, g := openPurseEditor(t)
	g.key(appInput{Typed: "7", Unfocused: true})
	g.key(appInput{Delete: true, Unfocused: true})
	g.key(appInput{Enter: true, Unfocused: true})
	if got, _ := a.HeadlessGold(); !got.Open || got.Text != "0" {
		t.Fatalf("unfocused input changed the editor: %+v", got)
	}
	if _, ok := v.TakeGoldDrop(); ok {
		t.Fatal("unfocused Enter requested gold")
	}
	g.key(appInput{PaneMode: true}) // action button focused
	g.key(appInput{Typed: "9"})
	g.key(appInput{Delete: true})
	g.key(appInput{Backspace: true})
	if got, _ := a.HeadlessGold(); got.Text != "0" || got.Focus != "action" {
		t.Fatalf("keys reached the edit through a focused button: %+v", got)
	}
}

// With the editor open the map reads no key: Backspace clears no message line
// whether or not the edit takes it, and a typed digit moves no selection.
func TestPurseEditorBlocksMessageClearAndMapKeys(t *testing.T) {
	a, v, g := openPurseEditor(t)
	v.PostMessage("keep me", MessageWhite, time.Hour)
	g.key(appInput{Backspace: true}) // caret zero: the edit declines it
	if len(v.MessageLines()) == 0 {
		t.Fatal("an inert Backspace cleared the message line")
	}
	g.key(appInput{Delete: true})
	typeGold(g, "12")
	g.key(appInput{Backspace: true}) // the edit takes it
	if len(v.MessageLines()) == 0 {
		t.Fatal("an edit Backspace cleared the message line")
	}
	if got, _ := a.HeadlessGold(); got.Text != "1" {
		t.Fatalf("text %q after Backspace, want 1", got.Text)
	}
	if len(v.sel) != 1 || v.sel[0] != 5 {
		t.Fatalf("selection moved to %v", v.sel)
	}
	g.key(appInput{Escape: true})
	g.key(appInput{Backspace: true})
	if len(v.MessageLines()) != 0 {
		t.Fatal("Backspace no longer clears the message line once the editor is closed")
	}
}

// Backspace changes the text and caret but no selection endpoint, Delete acts
// on a positive selection first, and typed characters replace one.
func TestGoldEditSelectionSemantics(t *testing.T) {
	g := goldState{text: "12345"}
	g.caret, g.selStart, g.selEnd = 5, 0, 0
	g.move(2, true)
	g.move(1, true)
	if g.caret != 1 || g.selStart != 1 || g.selEnd != 5 {
		t.Fatalf("shift selection = caret %d [%d,%d), want caret 1 [1,5)", g.caret, g.selStart, g.selEnd)
	}
	g.caret = 3
	before := [2]int{g.selStart, g.selEnd}
	if !g.backspace() || g.text != "1245" || g.caret != 2 || [2]int{g.selStart, g.selEnd} != before {
		t.Fatalf("backspace over a selection: %+v", g)
	}
	g.caret = 0
	if g.backspace() || g.text != "1245" {
		t.Fatalf("backspace at caret zero changed the text: %+v", g)
	}
	g.selStart, g.selEnd, g.caret = 1, 3, 3
	g.deleteForward()
	if g.text != "15" || g.caret != 1 || g.selStart != 1 || g.selEnd != 1 {
		t.Fatalf("delete over a selection: %+v", g)
	}
	g.selStart, g.selEnd, g.caret = 0, 2, 2
	g.insert("9")
	if g.text != "9" || g.caret != 1 {
		t.Fatalf("a character over a selection: %+v", g)
	}
	g.move(0, false)
	g.insert("\x01\xff")
	if g.text != "9" {
		t.Fatalf("control and non-ASCII characters inserted: %q", g.text)
	}
	g.text = "123456789012345678901234"
	g.caret, g.selStart, g.selEnd = 24, 24, 24
	g.insert("5")
	if len(g.text) != goldTextLimit {
		t.Fatalf("text grew past the bound: %d", len(g.text))
	}
}

func TestParseGoldAmount(t *testing.T) {
	for text, want := range map[string]uint32{
		"700": 700, "+5": 5, " 7 ": 7, "12.9": 12, "0": 0, "": 0, ".": 0, ".5": 0, "-1": 0, "1e3": 0,
		"abc": 0, "5%": 0, "1,000": 0, "4294967295": math.MaxUint32, "99999999999999999999": math.MaxUint32,
	} {
		if got := parseGoldAmount(text); got != want {
			t.Errorf("parseGoldAmount(%q) = %d, want %d", text, got, want)
		}
	}
}

// A save point's gesture cancel drops the editor, the held share and the drag
// that carried it, and the release that follows requests nothing.
func TestPurseGestureCancelSuppressesTheLaterRelease(t *testing.T) {
	a, v := purseApp(t)
	g := purseGesture{t, a}
	cx, cy := purseCenter(t, v)
	g.press(cx, cy, false)
	g.move(cx+3*TapSlop, cy, false)
	if v.GoldHeld() == 0 {
		t.Fatal("setup: nothing held")
	}
	v.cancelPointerGesture()
	if v.GoldHeld() != 0 || v.dragActive {
		t.Fatalf("held %d drag %v after the cancel", v.GoldHeld(), v.dragActive)
	}
	gx, gy := g.ground()
	g.release(gx, gy)
	if req, ok := v.TakeGoldDrop(); ok {
		t.Fatalf("the release after the cancel requested %+v", req)
	}

	g.doubleClick(cx, cy)
	if !v.goldModalOpen() {
		t.Fatal("setup: editor closed")
	}
	v.cancelPointerGesture()
	if v.goldModalOpen() {
		t.Fatal("the cancel left the editor open")
	}
}

// The editor draws: its composed picture is the original's rectangle and shows
// the text, so a frame with the editor open differs from one without.
func TestPurseEditorPictureTracksTheDraft(t *testing.T) {
	a, v, g := openPurseEditor(t)
	_ = a
	before, at, ok := v.goldModalPresent()
	if !ok || before.Bounds().Dx() != 296 || before.Bounds().Dy() != 168 || at != image.Pt(100, v.frameH-200) {
		t.Fatalf("editor picture %v at %v, %v", before.Bounds(), at, ok)
	}
	first := append([]byte(nil), before.Pix...)
	g.key(appInput{PaneMode: true})
	after, _, _ := v.goldModalPresent()
	same := len(first) == len(after.Pix)
	for i := 0; same && i < len(first); i++ {
		same = first[i] == after.Pix[i]
	}
	if same {
		t.Fatal("moving focus did not change the drawn editor")
	}
}

// Alt+Backspace reaches no GUI child, the open editor's edit included
// (MENU-082, MENU-088). Plain Backspace is the loss control.
func TestPurseEditorIgnoresAltBackspace(t *testing.T) {
	a, v, g := openPurseEditor(t)
	v.PostMessage("keep me", MessageWhite, time.Hour)
	typeGold(g, "12") // text 120, caret 2
	if got, _ := a.HeadlessGold(); got.Text != "120" || got.Caret != 2 {
		t.Fatalf("setup: editor %+v", got)
	}
	if err := a.HeadlessKey("alt-backspace"); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.HeadlessGold(); got.Text != "120" || got.Caret != 2 || !got.Open {
		t.Fatalf("Alt+Backspace changed the editor: %+v", got)
	}
	if len(v.MessageLines()) == 0 {
		t.Fatal("Alt+Backspace cleared the message line")
	}
	if err := a.HeadlessKey("backspace"); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.HeadlessGold(); got.Text != "10" || got.Caret != 1 {
		t.Fatalf("control: Backspace left %+v, want text 10 caret 1", got)
	}
}
