package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// TestComposeScreenSelectsMenuComposer witnesses composeScreen's ScreenMenu
// case. The expectation is independent of the switch under test: it is
// a.assets.Compose(a.sel.State()) called separately in this test, the exact
// call composeScreen's own source makes — proving the two agree is what
// proves the switch reached that arm, not a hand-rederivation of menu
// composition itself.
func TestComposeScreenSelectsMenuComposer(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	got, err := a.composeScreen()
	if err != nil {
		t.Fatalf("composeScreen() on ScreenMenu: %v", err)
	}
	want := a.assets.Compose(a.sel.State())
	if !bytes.Equal(got.Pix, want.Pix) || got.Bounds() != want.Bounds() {
		t.Errorf("composeScreen() on ScreenMenu did not match a.assets.Compose(a.sel.State())")
	}
}

// chargenDispatchSetup is a minimal pre-create setup: no choices, no forward
// art, just enough that composeChargenScreen's own two refusals (nil model,
// nil PreCreate) do not fire. It never reaches DetailedStage.
func chargenDispatchSetup() ChargenSetup {
	return ChargenSetup{
		Title:     "t",
		PreCreate: &ChargenPreCreate{Art: &ChargenPresentation{}},
		Stats:     []ChargenStat{{Name: "A", Floor: 0, Ceiling: 10, Start: 3}},
		Cost:      triangular(10),
		Budget:    100,
		Confirm:   "go",
	}
}

// TestComposeScreenSelectsChargenComposer witnesses composeScreen's
// ScreenChargen case, armed through the production door OpenChargen. The
// expectation is ComposeChargenFrame(c) — the exported composer this story's
// contract names as one of the two zero-witness composers — called
// separately in this test with the same model, on the pre-create stage
// only.
func TestComposeScreenSelectsChargenComposer(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	c := NewChargen(chargenDispatchSetup())
	if err := a.OpenChargen(c, func(ChargenResult) (MapOpener, error) { return nil, nil }); err != nil {
		t.Fatalf("OpenChargen: %v", err)
	}
	if c.Stage() == DetailedStage {
		t.Fatal("fixture reached DetailedStage; this test must stay on the pre-create page (1022 owns the detailed page's geometry this cycle)")
	}
	got, err := a.composeScreen()
	if err != nil {
		t.Fatalf("composeScreen() on ScreenChargen: %v", err)
	}
	want := ComposeChargenFrame(c)
	if !bytes.Equal(got.Pix, want.Pix) || got.Bounds() != want.Bounds() {
		t.Errorf("composeScreen() on ScreenChargen did not match ComposeChargenFrame(c)")
	}
}

// TestComposeScreenSelectsTownComposer witnesses composeScreen's shared
// ScreenTown/ScreenGameMenu case reaching (*App).composeTownScreen, for a
// plain surface room (never the chargen page, never the town-square/shop
// rooms this same population already covers elsewhere). The expectation is
// ComposeTownScreen(t, msg) — the exported, App-free composer — called
// separately with the same town model and message.
func TestComposeScreenSelectsTownComposer(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	town := &fakeStatsSurfaceTown{}
	a.SetTown(town)
	if !a.flow.showTown("welcome") {
		t.Fatal("flow.showTown refused a non-nil town model")
	}

	for _, screen := range []Screen{ScreenTown, ScreenGameMenu} {
		a.flow.screen = screen
		got, err := a.composeScreen()
		if err != nil {
			t.Fatalf("composeScreen() on %v: %v", screen, err)
		}
		want, err := ComposeTownScreen(town, a.flow.msg)
		if err != nil {
			t.Fatalf("ComposeTownScreen: %v", err)
		}
		if !bytes.Equal(got.Pix, want.Pix) || got.Bounds() != want.Bounds() {
			t.Errorf("composeScreen() on %v did not match ComposeTownScreen(town, msg)", screen)
		}
	}
}

// TestDialogueOriginCentersDialogueInTheFrame pins dialogueOrigin's own
// arithmetic against hand-computed literals, not against a second
// evaluation of the same centering formula: the frame is 640x480
// (pkg/render/frame.W, .H) and a 200x120 dialogue centers at
// ((640-200)/2, (480-120)/2) = (220, 180), written here as the literal
// result of that arithmetic rather than as a repeated expression.
func TestDialogueOriginCentersDialogueInTheFrame(t *testing.T) {
	dialogue := image.NewRGBA(image.Rect(0, 0, 200, 120))
	x, y := dialogueOrigin(dialogue)
	if x != 220 || y != 180 {
		t.Errorf("dialogueOrigin(200x120 dialogue) = (%d,%d), want (220,180)", x, y)
	}

	// A second size, odd on both axes, to catch a formula that only agrees
	// with the first on even inputs (integer division rounds a 640-201
	// remainder down): (640-201)/2=219 (219.5 truncates), (480-121)/2=179.
	odd := image.NewRGBA(image.Rect(0, 0, 201, 121))
	x, y = dialogueOrigin(odd)
	if x != 219 || y != 179 {
		t.Errorf("dialogueOrigin(201x121 dialogue) = (%d,%d), want (219,179)", x, y)
	}
}

// TestAppComposeTownRoomThreadsLiveCursorAndDragState witnesses
// (*App).composeTownRoom carrying the App's own live pointer state
// (a.townCursor, a.hasTownCursor) and its own held-item state
// (a.shopDragItemPresent, read from a.shopDragArmed/a.shopDragMoved/
// a.shopDragIcon) into the package-level composeTownRoom — state neither the
// package-level function nor ComposeTownScreen ever has, because a bare
// TownScreen carries none of a live App's interaction (composeTownRoom's
// own header). The fixture is the shop room with no art at all
// (TestComposeShopScreenDrawsTheDragIconUnderTheCursor's own fixture,
// shopscreen_test.go:238): with no furniture rect at that point, a drag icon
// under the cursor is the one pixel this frame can show, so the two
// composites must differ once the App is armed with a hover point and a
// held item.
func TestAppComposeTownRoomThreadsLiveCursorAndDragState(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	town := &fakeShopTown{}
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("flow.showTown refused a non-nil town model")
	}

	bare, err := a.composeTownRoom()
	if err != nil {
		t.Fatalf("composeTownRoom() with no cursor and no held item: %v", err)
	}

	icon := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			icon.SetRGBA(x, y, color.RGBA{R: 0xff, A: 0xff})
		}
	}
	hover := image.Pt(300, 100) // clear of every furniture rect at art == nil
	a.townCursor, a.hasTownCursor = hover, true
	a.shopDragArmed, a.shopDragMoved, a.shopDragIcon = true, TapSlop, icon

	armed, err := a.composeTownRoom()
	if err != nil {
		t.Fatalf("composeTownRoom() with cursor and held item armed: %v", err)
	}

	if bytes.Equal(bare.Pix, armed.Pix) {
		t.Fatal("composeTownRoom() drew the same frame armed as bare: a.townCursor/a.hasTownCursor/a.shopDragItemPresent() are not reaching the package-level composeTownRoom")
	}
	if got := armed.RGBAAt(hover.X, hover.Y); got.R != 0xff || got.A != 0xff {
		t.Errorf("(%d,%d) = %+v, want the held icon's own opaque red — the wrapper's own cursor/drag state must land at the wrapper's own cursor position", hover.X, hover.Y, got)
	}
}

// TestHeadlessFrameNoteNamesTheGameMenuOverlay witnesses HeadlessFrame's own
// note branch (headless.go:149): non-empty exactly on ScreenGameMenu, empty
// on every other composable screen. The expectation is the literal string
// HeadlessFrame's own source assigns, transcribed here rather than read back
// from a call to HeadlessFrame in the same test.
func TestHeadlessFrameNoteNamesTheGameMenuOverlay(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	town := &fakeStatsSurfaceTown{}
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("flow.showTown refused a non-nil town model")
	}

	const wantNote = "the in-game menu's dim and panel are not in this frame: paintGameMenu draws with ebiten's vector package, no CPU composite"

	a.flow.screen = ScreenTown
	if _, note, err := a.HeadlessFrame(); err != nil || note != "" {
		t.Errorf("HeadlessFrame() on ScreenTown: note = %q, err = %v; want empty note, no error", note, err)
	}

	a.flow.screen = ScreenGameMenu
	if _, note, err := a.HeadlessFrame(); err != nil || note != wantNote {
		t.Errorf("HeadlessFrame() on ScreenGameMenu: note = %q, err = %v; want %q, no error", note, err, wantNote)
	}
}

// TestRenderPanelDrawsAOnePixelBorderOnAFixedBox closes this story's second
// zero-witness composer (contract B2). Size is fixed on both axes (a
// non-zero Size means fixed, not fit-to-content — PanelLayout's own
// comment), so the box is exactly the 40x20 requested regardless of Rows:
// the assertion is that the panel frame's border occupies exactly the
// outermost ring and nothing more, colours chosen by this test and compared
// directly rather than read back from the layout that produced them.
func TestRenderPanelDrawsAOnePixelBorderOnAFixedBox(t *testing.T) {
	fill := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	border := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	layout := PanelLayout{
		Size:   image.Pt(40, 20),
		Fill:   fill,
		Border: border,
	}

	got := RenderPanel(layout, chargenTestFont(), PanelSubject{})
	if got == nil {
		t.Fatal("RenderPanel returned nil for a fixed non-zero Size")
	}
	if got.Bounds() != image.Rect(0, 0, 40, 20) {
		t.Fatalf("RenderPanel bounds = %v, want (0,0)-(40,20): a fixed Size must not be resized to content", got.Bounds())
	}
	corners := []image.Point{{0, 0}, {39, 0}, {0, 19}, {39, 19}}
	for _, p := range corners {
		if c := got.RGBAAt(p.X, p.Y); c != border {
			t.Errorf("RenderPanel corner %v = %#v, want the border colour %#v", p, c, border)
		}
	}
	interior := []image.Point{{1, 1}, {20, 10}, {38, 18}}
	for _, p := range interior {
		if c := got.RGBAAt(p.X, p.Y); c != fill {
			t.Errorf("RenderPanel interior %v = %#v, want the fill colour %#v: the border is one pixel wide, not two", p, c, fill)
		}
	}
}
