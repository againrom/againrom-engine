package ui

import (
	"image"
	"image/color"
	"slices"
	"testing"
	"time"
)

// The inventory window: the composition (AC-7, AC-8, AC-12) and the toggle's
// whole open/closed behaviour (AC-6), each read straight off the acceptance
// table in docs/0110-inventory/spec.md.

// solidPic builds a synthetic w x h picture filled with one colour — a
// picture whose IDENTITY the composition can be asked about without decoding
// anything: these tests never open a game install (AGENTS.md rule 2).
func solidPic(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// invCellHasArt reports whether any pixel strictly inside box differs from
// the cell's own ground colour — the composition's own witness for "this
// cell drew a picture" without decoding what the picture was. It skips the
// one-pixel border, which is invCellBorder rather than invCellFill on every
// cell alike and would say nothing about whether a picture was drawn.
func invCellHasArt(img *image.RGBA, box image.Rectangle) bool {
	for y := box.Min.Y + 1; y < box.Max.Y-1; y++ {
		for x := box.Min.X + 1; x < box.Max.X-1; x++ {
			if img.RGBAAt(x, y) != invCellFill {
				return true
			}
		}
	}
	return false
}

// packBarFixture is a pack bar's rectangle and cell count at the shipped
// window size, plus its cells taken back to the composed picture's OWN origin
// — so a composition test can address a cell without a viewer, and addresses
// it through the very geometry the hit test uses (hud.go's packCellRects).
func packBarFixture(t *testing.T) (image.Rectangle, int, []image.Rectangle) {
	t.Helper()
	bar, cols, ok := packBarRect(image.Pt(MenuWindowW, MenuWindowH))
	if !ok {
		t.Fatalf("setup: no pack bar fits %dx%d", MenuWindowW, MenuWindowH)
	}
	cells := packCellRects(bar, cols)
	for i := range cells {
		cells[i] = cells[i].Sub(bar.Min)
	}
	return bar, cols, cells
}

// packOf is an InventorySubject carrying pics as its pack, in order — the
// slice the array used to be (0140), written once here so no test below builds
// one by hand.
func packOf(id uint32, pics ...*image.RGBA) InventorySubject {
	return InventorySubject{ID: id, Pack: pics, PackCount: make([]uint32, len(pics))}
}

func TestRenderWorn(t *testing.T) {
	t.Run("an entirely empty subject still composes (P-2)", func(t *testing.T) {
		img := RenderWorn(InventorySubject{})
		if img == nil {
			t.Fatal("RenderWorn returned nil for the zero subject")
		}
		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			t.Fatalf("composed a box with no area: %v", b)
		}
	})

	// AC-7: "twelve cells are drawn, the first holding that code's icon and
	// the other eleven none." pkg/ui receives an already-painted figure (plan
	// D-7) and cannot witness the layer count itself, so this covers exactly
	// the part this tier owns.
	t.Run("one occupied slot: that cell alone carries art (AC-7)", func(t *testing.T) {
		var s InventorySubject
		s.ID = 1
		s.Slots[0] = solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})

		img := RenderWorn(s)
		slots := wornSlotRects()

		if !invCellHasArt(img, slots[0]) {
			t.Error("the occupied first slot drew no icon")
		}
		for i := 1; i < len(slots); i++ {
			if invCellHasArt(img, slots[i]) {
				t.Errorf("empty slot %d drew something", i)
			}
		}
	})

	// AC-8: "a code whose icon the archive lacks... it composes, that cell is
	// empty, every other cell is unchanged." A nil picture is how an unread
	// address arrives at this tier (fence).
	t.Run("a nil slot picture is a drawn absence, every other cell unchanged (AC-8)", func(t *testing.T) {
		iconA := solidPic(8, 8, color.RGBA{G: 0xff, A: 0xff})
		iconC := solidPic(8, 8, color.RGBA{B: 0xff, A: 0xff})

		var s InventorySubject
		s.ID = 1
		s.Slots[2] = iconA
		s.Slots[5] = nil // the unread address under test
		s.Slots[7] = iconC

		img := RenderWorn(s) //
		slots := wornSlotRects()

		if !invCellHasArt(img, slots[2]) {
			t.Error("slot 2's icon did not draw")
		}
		if invCellHasArt(img, slots[5]) {
			t.Error("the nil-picture slot drew something")
		}
		if !invCellHasArt(img, slots[7]) {
			t.Error("slot 7's icon did not draw — a neighbouring nil picture changed it")
		}
	})

	// Same input, same picture — the composition is a pure function of the
	// subject and nothing else.
	t.Run("deterministic: the same subject composes the same pixels", func(t *testing.T) {
		s := InventorySubject{ID: 3, Slots: [12]*image.RGBA{0: solidPic(4, 4, color.RGBA{R: 1, A: 1})}}
		a, b := RenderWorn(s), RenderWorn(s)
		if !slices.Equal(a.Pix, b.Pix) || a.Bounds() != b.Bounds() {
			t.Error("two compositions of the same subject produced different pixels")
		}
	})

	// Every one of the twelve cells is inside the box that is composed for it —
	// the one thing that can silently go wrong when a grid is rewrapped, and it
	// was rewrapped from three columns to six when the figure left.
	t.Run("all twelve cells lie inside the box", func(t *testing.T) {
		box := image.Rect(0, 0, wornBoxSize().X, wornBoxSize().Y)
		for i, r := range wornSlotRects() {
			if !r.In(box) {
				t.Errorf("slot %d at %v is not inside the worn box %v", i, r, box)
			}
		}
	})
}

// The pack bar's own composition: 0112's and 0138's pack criteria, moved from
// the window's fixed cell block to the bar that replaced it (0140).
func TestRenderPackBar(t *testing.T) {
	bar, cols, cells := packBarFixture(t)

	t.Run("an entirely empty subject still composes (P-2)", func(t *testing.T) {
		img := renderPackBar(InventorySubject{}, 0, cols, bar, nil)
		wantSize := image.Pt(bar.Dx()+sidebarWidth, bar.Dy())
		if img == nil || img.Bounds().Size() != wantSize {
			var got image.Point
			if img != nil {
				got = img.Bounds().Size()
			}
			t.Fatalf("renderPackBar size = %v, want full-frame-width layer %v", got, wantSize)
		}
		for i, box := range cells {
			if invCellHasArt(img, box) {
				t.Errorf("cell %d is not empty on a subject carrying nothing", i)
			}
		}
	})

	t.Run("an 80x80 base icon reaches every edge of its decoded cell", func(t *testing.T) {
		want := color.RGBA{R: 71, G: 73, B: 79, A: 0xff}
		img := renderPackBar(packOf(14, solidPic(80, 80, want)), 0, cols, bar, nil)
		box := cells[0]
		for _, p := range []image.Point{
			box.Min,
			{X: box.Max.X - 1, Y: box.Min.Y},
			{X: box.Min.X, Y: box.Max.Y - 1},
			box.Max.Sub(image.Pt(1, 1)),
		} {
			if got := img.RGBAAt(p.X, p.Y); got != want {
				t.Errorf("full-size icon pixel %v = %+v, want %+v", p, got, want)
			}
		}
	})

	// 0112 AC-9, first clause: a subject carrying a pack picture composes
	// differently from the same subject without one.
	t.Run("a pack picture composes differently from the same subject without one (0112 AC-9)", func(t *testing.T) {
		withPack := packOf(4, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff}))
		without := InventorySubject{ID: 4}

		gotWith := renderPackBar(withPack, 0, cols, bar, nil)
		gotWithout := renderPackBar(without, 0, cols, bar, nil)
		if slices.Equal(gotWith.Pix, gotWithout.Pix) {
			t.Error("a subject carrying a pack picture composed identically to the same subject without one")
		}
	})

	// 0112 AC-9, third clause: a nil pack entry is still an empty cell, and
	// its nil-ness does not disturb its neighbours.
	t.Run("a nil pack entry is a drawn absence, every other cell unchanged (0112 AC-9)", func(t *testing.T) {
		s := packOf(10,
			nil,
			solidPic(8, 8, color.RGBA{G: 0xff, A: 0xff}),
			nil, // the unread address under test
			solidPic(8, 8, color.RGBA{B: 0xff, A: 0xff}),
		)

		img := renderPackBar(s, 0, cols, bar, nil)

		if !invCellHasArt(img, cells[1]) {
			t.Error("cell 1's picture did not draw")
		}
		if invCellHasArt(img, cells[2]) {
			t.Error("the nil-picture cell drew something")
		}
		if !invCellHasArt(img, cells[3]) {
			t.Error("cell 3's picture did not draw — a neighbouring nil picture changed it")
		}
	})

	// AC-11: "the cell differs from the same cell at count 1." numeralFont
	// (numeral_test.go, this package) is reused rather than a second synthetic
	// font invented for one more piece of drawn text.
	t.Run("a pack element's count above 1 composes differently from the same element at 1 (0138 AC-11)", func(t *testing.T) {
		f := numeralFont()
		three := packOf(11, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff}))
		three.PackCount[0] = 3
		one := packOf(11, three.Pack[0])
		one.PackCount[0] = 1

		gotThree := renderPackBar(three, 0, cols, bar, f)
		gotOne := renderPackBar(one, 0, cols, bar, f)
		if slices.Equal(gotThree.Pix, gotOne.Pix) {
			t.Error("a pack cell at count 3 composed identically to the same cell at count 1")
		}
	})

	t.Run("a counted subject with no font, and the zero subject with a font, both compose (0138 AC-12)", func(t *testing.T) {
		s := packOf(12, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff}))
		s.PackCount[0] = 3

		if renderPackBar(s, 0, cols, bar, nil) == nil {
			t.Error("a counted subject with no font composed nothing")
		}
		if renderPackBar(InventorySubject{}, 0, cols, bar, numeralFont()) == nil {
			t.Error("the zero subject with a font composed nothing")
		}
	})

	// 0140: a PackCount shorter than Pack — which every hand-built fixture in
	// this package produces — reads as no count rather than as a panic.
	t.Run("a pack with no counts at all composes", func(t *testing.T) {
		s := InventorySubject{ID: 13, Pack: []*image.RGBA{solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})}}
		if renderPackBar(s, 0, cols, bar, numeralFont()) == nil {
			t.Error("a pack carrying pictures but no counts composed nothing")
		}
	})
}

// 0140, the owner's own words: "the inventory scrolls with arrows on the
// right, potentially endlessly". The bar shows a WINDOW onto the pack, so the
// cell that draws element 0 at rest draws element 1 one press later — and the
// scroll can never run past the last element or before the first.
func TestThePackBarScrolls(t *testing.T) {
	bar, cols, cells := packBarFixture(t)

	// A pack two elements longer than the bar has cells, each a distinct
	// colour, so "which element is in cell 0" is answerable from the pixels.
	pics := make([]*image.RGBA, cols+2)
	for i := range pics {
		pics[i] = solidPic(8, 8, color.RGBA{R: uint8(i + 1), A: 0xff})
	}
	s := packOf(1, pics...)

	first := func(scroll int) color.RGBA {
		img := renderPackBar(s, scroll, cols, bar, nil)
		c := cells[0]
		return img.RGBAAt((c.Min.X+c.Max.X)/2, (c.Min.Y+c.Max.Y)/2)
	}
	if got, want := first(0), (color.RGBA{R: 1, A: 0xff}); got != want {
		t.Errorf("cell 0 at scroll 0 = %+v, want element 0's own colour %+v", got, want)
	}
	if got, want := first(1), (color.RGBA{R: 2, A: 0xff}); got != want {
		t.Errorf("cell 0 at scroll 1 = %+v, want element 1's own colour %+v", got, want)
	}

	if got := packMaxScroll(len(pics), cols); got != 2 {
		t.Errorf("packMaxScroll over %d elements in %d cells = %d, want 2", len(pics), cols, got)
	}
	if got := packMaxScroll(cols, cols); got != 0 {
		t.Errorf("packMaxScroll over a pack that fits = %d, want 0", got)
	}
	if got := packMaxScroll(0, cols); got != 0 {
		t.Errorf("packMaxScroll over an empty pack = %d, want 0", got)
	}
}

// The bar and the cursor agree about which ELEMENT a pixel names, at every
// scroll position — including "none at all" for a cell past the end of the
// pack, which most of the bar usually is.
func TestThePackBarsHitTestNamesTheElementUnderTheCursor(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	// The bar is drawn for the subject's own single selection and for nothing
	// else since 0140 (inventory.go's packBar), so a fixture that means to
	// press on one has to make that selection.
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	v.sel = selection{5}

	bar, cols, ok := packBarRect(image.Pt(v.frameW, v.frameH))
	if !ok {
		t.Fatalf("setup: no pack bar at %dx%d", v.frameW, v.frameH)
	}
	pics := make([]*image.RGBA, cols+2)
	for i := range pics {
		pics[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	v.SetInventorySubject(packOf(5, pics...))

	cells := packCellRects(bar, cols)
	mid := func(r image.Rectangle) (int, int) { return (r.Min.X + r.Max.X) / 2, (r.Min.Y + r.Max.Y) / 2 }

	x, y := mid(cells[0])
	if idx, ok := v.inventoryPackCellAt(x, y); !ok || idx != 0 {
		t.Errorf("cell 0 at rest names (%d,%v), want (0,true)", idx, ok)
	}
	if !v.ScrollPack(1) {
		t.Fatal("ScrollPack(1) reported no movement on a pack longer than the bar")
	}
	if idx, ok := v.inventoryPackCellAt(x, y); !ok || idx != 1 {
		t.Errorf("cell 0 after one scroll names (%d,%v), want (1,true)", idx, ok)
	}

	// The bar is drawn full width whatever is carried, so most of it is empty
	// cells for a short pack, and an empty cell names no element.
	v.packScroll = 0
	v.SetInventorySubject(packOf(5, pics[0], pics[1]))
	lx, ly := mid(cells[cols-1])
	if idx, ok := v.inventoryPackCellAt(lx, ly); ok {
		t.Errorf("a cell past the end of the pack named element %d, want none", idx)
	}
	v.SetInventorySubject(packOf(5, pics...))

	// And the scroll cannot run off either end.
	v.packScroll = 0
	if v.ScrollPack(-1) {
		t.Error("ScrollPack(-1) moved a bar already at the first element")
	}
	v.packScroll = packMaxScroll(len(pics), cols)
	if v.ScrollPack(1) {
		t.Error("ScrollPack(1) moved a bar already showing the last element")
	}
}

// inventoryTestApp parks a fresh App on the map screen over a synthetic
// viewer, the way appOnMapSeam and newOnMap do throughout this package's own
// tests, so a subtest that never touches the picker or the menu still starts
// from a state the front-end could really be in.
func inventoryTestApp(t *testing.T) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, time.Unix(1_700_000_000, 0))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: screen = %v, want ScreenMap", a.Screen())
	}
	return a
}

func TestHudSwitches(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	at := func(i int) time.Time { return base.Add(time.Duration(i) * 100 * time.Millisecond) }

	// newOnMap gives a fresh mission with two assembled characters, entity 5
	// and entity 6, and a subject naming entity 5 — the fixture every subtest
	// below starts from. The layout is stated because every box down here is
	// refused outright in a window with no room for it (hud.go); wornBoxFixtureH
	// and not MenuWindowH itself, since the worn box is one of the four this
	// test walks (viewportfixture_test.go).
	newOnMap := func(t *testing.T) (*App, *Viewer) {
		t.Helper()
		a := inventoryTestApp(t)
		v := a.flow.viewer
		layoutViewport(v, MenuWindowW, wornBoxFixtureH)
		v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}, {ID: 6, Cell: image.Pt(2, 2)}})
		v.SetInventorySubject(InventorySubject{ID: 5})
		return a, v
	}

	// The owner's own reading of "display settings": everything is shown until
	// he turns it off. Stored inverted on the viewer, so this is also the
	// assertion that a fresh Viewer needs no constructor statement to get there.
	t.Run("all four are on at the start", func(t *testing.T) {
		_, v := newOnMap(t)
		for p := hudPanel(0); p < hudPanelCount; p++ {
			if !v.hudShown(p) {
				t.Errorf("switch %q is off on a fresh viewer, want on", hudToggleLabels[p])
			}
		}
	})

	// The key flips its own switch and nothing else's — the whole of what a
	// press does now, in both directions and whatever is selected.
	t.Run("the keys flip their own switch, ungated", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			in    appInput
			panel hudPanel
		}{
			{"I switches the pack", appInput{Inventory: true}, hudPanelPack},
			{"E switches the worn set", appInput{Worn: true}, hudPanelWorn},
		} {
			t.Run(tc.name, func(t *testing.T) {
				a, v := newOnMap(t)
				v.sel = nil // ungated: nothing selected is still a switch that moves

				a.step(tc.in, at(1))
				if v.hudShown(tc.panel) {
					t.Fatal("one press left the switch on")
				}
				for p := hudPanel(0); p < hudPanelCount; p++ {
					if p != tc.panel && !v.hudShown(p) {
						t.Errorf("the press also switched %q off", hudToggleLabels[p])
					}
				}

				a.step(tc.in, at(2))
				if !v.hudShown(tc.panel) {
					t.Error("the second press did not switch it back on")
				}
			})
		}
	})

	// The eligibility clause, unchanged in effect and moved into each box's own
	// gate: the pack bar and the worn box are DRAWN for exactly the subject's
	// own single selection, whatever the switches say.
	t.Run("the subject's boxes are drawn only for its own single selection", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			sel  selection
			want bool
		}{
			{"nothing selected", nil, false},
			{"a different single character", selection{6}, false},
			{"two selected, its subject among them", selection{5, 6}, false},
			{"an id the snapshot no longer holds", selection{9}, false},
			{"its own subject alone", selection{5}, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, v := newOnMap(t)
				v.sel = tc.sel

				if _, _, ok := v.packBar(); ok != tc.want {
					t.Errorf("packBar drawn=%v, want %v", ok, tc.want)
				}
				if _, ok := v.wornBox(); ok != tc.want {
					t.Errorf("wornBox drawn=%v, want %v", ok, tc.want)
				}
			})
		}
	})

	// A switched-off box is not drawn even for the one selection that entitles
	// it — the half of the gate the selection does not decide.
	//
	// THE DOLL IS NOT ONE OF THESE ANY MORE AT THIS FIXTURE'S FRAME (story
	// 1036 round 2): characterPaneModeFlag is true whenever the frame has
	// room for the figure box beside the character panel, which
	// wornBoxFixtureH does — the same condition every shipped mission frame
	// satisfies — and `DIV-217`'s own directive is that there is nothing
	// left there to toggle, so dollBox stops reading hudPanelDoll in that
	// case. TestEverySwitchReachesItsOwnBoxAndNoOther carries the same
	// exclusion and asserts the inertness explicitly.
	t.Run("a switched-off box is not drawn for any selection", func(t *testing.T) {
		_, v := newOnMap(t)
		v.sel = selection{5}

		v.toggleHudPanel(hudPanelPack)
		if _, _, ok := v.packBar(); ok {
			t.Error("the pack bar is drawn with its own switch off")
		}
		v.toggleHudPanel(hudPanelWorn)
		if _, ok := v.wornBox(); ok {
			t.Error("the worn box is drawn with its own switch off")
		}
	})

	// The doll follows every selected unit; only worn gear and the pack require
	// a matching inventory subject.
	t.Run("the doll follows any selected unit", func(t *testing.T) {
		_, v := newOnMap(t)

		v.sel = nil
		if _, ok := v.dollBox(); ok {
			t.Error("a doll box stands with nothing selected")
		}

		v.sel = selection{6}
		if _, ok := v.dollBox(); !ok {
			t.Error("no doll box for a selected non-subject ally")
		}

		v.sel = selection{5}
		if _, ok := v.dollBox(); !ok {
			t.Error("no doll box for the persistent hero subject")
		}
	})

	// Cancel is cancel again. The window this replaced took Escape before even
	// the notice; a setting is not unset by leaving.
	t.Run("cancel does not touch a switch", func(t *testing.T) {
		a, v := newOnMap(t)
		a.step(appInput{Escape: true}, at(1))
		for p := hudPanel(0); p < hudPanelCount; p++ {
			if !v.hudShown(p) {
				t.Errorf("Escape switched %q off", hudToggleLabels[p])
			}
		}
	})
}

// EVERY SWITCH REACHES ITS OWN BOX AND ONLY ITS OWN, over a viewer on which all
// four are drawn at once.
func TestEverySwitchReachesItsOwnBoxAndNoOther(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, wornBoxFixtureH)
	// A font and a panel layout, because two of the four are gated on a font
	// and two are placed above the unit panel: a viewer without them draws a
	// different set of boxes than the running game does, and this test is about
	// the set.
	v.SetFont(panelFont())
	v.SetPanelLayout(AuthoredPanelLayout())
	s := panelSubjectFixture()
	v.SetEntities([]MapEntity{{ID: s.ID, Cell: image.Pt(3, 3), Name: s.Name,
		HP: s.HP, MaxHP: s.MaxHP, Combat: s.Combat, Char: s.Char}})
	v.sel = selection{s.ID}
	v.SetInventorySubject(packOf(s.ID, solidPic(8, 8, color.RGBA{R: 0xff, A: 0xff})))
	v.SetSpellbook(s.ID, []SpellEntry{{ID: 1, Name: "Fire Arrow"}})

	drawn := func() [hudPanelCount]bool {
		var out [hudPanelCount]bool
		_, _, out[hudPanelPack] = v.packBar()
		_, _, out[hudPanelBook] = v.spellbookBar()
		_, out[hudPanelDoll] = v.dollBox()
		_, out[hudPanelWorn] = v.wornBox()
		return out
	}

	all := drawn()
	for p := hudPanel(0); p < hudPanelCount; p++ {
		if !all[p] {
			t.Fatalf("setup: %q is not drawn with every switch on, so switching it off "+
				"would prove nothing", hudToggleLabels[p])
		}
	}

	for p := hudPanel(0); p < hudPanelCount; p++ {
		v.toggleHudPanel(p)
		got := drawn()
		for q := hudPanel(0); q < hudPanelCount; q++ {
			want := q != p
			if got[q] != want {
				t.Errorf("with %q switched off, %q is drawn=%v, want %v",
					hudToggleLabels[p], hudToggleLabels[q], got[q], want)
			}
		}
		v.toggleHudPanel(p)
		if drawn() != all {
			t.Errorf("switching %q back on did not restore every box", hudToggleLabels[p])
		}
	}
}

func TestCommandPanelSlotNeverTogglesADisplaySwitch(t *testing.T) {
	newOnMap := func(t *testing.T) *Viewer {
		t.Helper()
		v := inventoryTestApp(t).flow.viewer
		layoutViewport(v, MenuWindowW, MenuWindowH)
		v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
		v.sel = selection{5}
		v.SetInventorySubject(InventorySubject{ID: 5})
		return v
	}

	v := newOnMap(t)
	bar, ok := v.hudToggleBar()
	if !ok {
		t.Fatal("setup: no control panel in a 1280x960 window")
	}

	before := v.hudHidden
	for i, cell := range commandCellRects(bar) {
		mid := image.Pt((cell.Min.X+cell.Max.X)/2, (cell.Min.Y+cell.Max.Y)/2)
		ords, run := v.command(appInput{CursorX: mid.X, CursorY: mid.Y, PrimaryPressed: true})
		if run || len(ords) != 0 {
			t.Errorf("cell %d: the press reached the map (orders %v, run %v)", i, ords, run)
		}
		if v.hudHidden != before {
			t.Errorf("cell %d: the press changed a display switch (%v, want %v)", i, v.hudHidden, before)
		}
	}

	// The panel's own frame: swallowed, and it means nothing.
	corner := image.Pt(bar.Min.X, bar.Min.Y)
	if ords, run := v.command(appInput{CursorX: corner.X, CursorY: corner.Y, PrimaryPressed: true}); run || len(ords) != 0 {
		t.Errorf("a press on the panel's frame reached the map (orders %v, run %v)", ords, run)
	}
	if v.hudHidden != before {
		t.Error("a press on the panel's frame flipped a switch")
	}

	// And one pixel outside it is the map's, exactly as before this panel
	// existed — this is furniture, not a modal.
	out := image.Pt(bar.Max.X+1, bar.Min.Y-1)
	if v.commandPanelCaptures(out.X, out.Y) {
		t.Error("a pixel outside the panel was captured by it")
	}
}

func TestInventoryStateIsFreshAtEachMissionOpen(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	at := func(i int) time.Time { return base.Add(time.Duration(i) * 100 * time.Millisecond) }

	a := inventoryTestApp(t)
	v1 := a.flow.viewer
	v1.SetEntities([]MapEntity{{ID: 5}})
	v1.sel = selection{5}
	v1.SetInventorySubject(InventorySubject{ID: 5})

	a.step(appInput{Inventory: true}, at(1))
	if v1.hudShown(hudPanelPack) {
		t.Fatal("setup: the first mission's pack switch did not go off")
	}

	// Leave the map the way a won or lost mission's own notice would —
	// straight through flow.leaveMap.
	a.flow.leaveMap()
	a.flow.screen = ScreenPicker

	a.step(appInput{Enter: true}, at(2))
	if a.Screen() != ScreenMap {
		t.Fatalf("setup: second load did not reach the map screen")
	}
	v2 := a.flow.viewer
	if v2 == v1 {
		t.Fatal("setup: the second load returned the same viewer as the first")
	}
	if !v2.hudShown(hudPanelPack) {
		t.Error("the switch the player set in the first mission survived into the second")
	}
	if v2.invHasSubject {
		t.Error("the first mission's subject is still held at the start of a second mission")
	}
}

// D-9: "one composed surface... a frame that changes none of them
// re-presents." The worn box's picture is rebuilt on a change to the subject
// and on no other frame.
func TestWornPresentRebuildsOnlyOnKeyChange(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, wornBoxFixtureH) // every box down here needs room (hud.go)
	v.SetEntities([]MapEntity{{ID: 5}})
	v.sel = selection{5}
	v.SetInventorySubject(InventorySubject{ID: 5})

	if _, _, ok := v.wornPresent(); !ok {
		t.Fatal("wornPresent answered false with the subject selected")
	}
	if v.invBuilds != 1 {
		t.Fatalf("invBuilds = %d after the first present, want 1", v.invBuilds)
	}

	if _, _, ok := v.wornPresent(); !ok {
		t.Fatal("second present answered false")
	}
	if v.invBuilds != 1 {
		t.Errorf("invBuilds = %d after an unchanged present, want 1 (no rebuild)", v.invBuilds)
	}

	v.SetInventorySubject(InventorySubject{ID: 5, Figure: solidPic(4, 4, color.RGBA{A: 1})})
	if _, _, ok := v.wornPresent(); !ok {
		t.Fatal("present answered false after a subject change")
	}
	if v.invBuilds != 2 {
		t.Errorf("invBuilds = %d after a changed subject, want 2", v.invBuilds)
	}
}

func TestPresentAnswersFalseWithNothingToDraw(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH) // so "false" is the state's doing and not the window size's

	if _, _, ok := v.wornPresent(); ok {
		t.Error("a worn box presented with no subject at all")
	}

	v.SetInventorySubject(InventorySubject{ID: 5})
	if _, _, ok := v.wornPresent(); ok {
		t.Error("a worn box presented for a subject that is not the selection")
	}
	if v.invBuilds != 0 {
		t.Errorf("invBuilds = %d though nothing was ever drawn", v.invBuilds)
	}
}

func TestInventoryTogglingIsInert(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	at := func(i int) time.Time { return base.Add(time.Duration(i) * 100 * time.Millisecond) }

	seams := &[]*mapSeam{}
	a := newTestApp(t, appRows(3), seamLoader(t, seams))
	a.flow.screen = ScreenPicker
	a.step(appInput{Enter: true}, at(0))
	v := a.flow.viewer
	v.SetEntities([]MapEntity{{ID: 7, Cell: image.Pt(2, 2)}})
	v.sel = selection{7}
	v.SetInventorySubject(InventorySubject{ID: 7})

	wantSel := slices.Clone(v.sel)

	for i, in := range []appInput{
		{Inventory: true}, // the pack switch off
		{Inventory: true}, // and back on
		{Worn: true},      // the worn switch off
		{Worn: true},      // and back on
		{Inventory: true}, // off again
	} {
		a.step(in, at(i+1))
	}

	if !slices.Equal(v.sel, wantSel) {
		t.Errorf("selection changed by a sequence of opens and closes: got %+v, want %+v", v.sel, wantSel)
	}
	s := (*seams)[0]
	if len(s.orders) != 0 {
		t.Errorf("flipping the switches issued %+v, want no orders", s.orders)
	}
	if len(s.blows) != 0 {
		t.Errorf("flipping the switches issued %+v, want no blows", s.blows)
	}
	if len(s.attacks) != 0 {
		t.Errorf("flipping the switches issued %+v, want no attacks", s.attacks)
	}
	// One tick per frame, exactly as a run with no switching at all would
	// advance: both keys are read INSIDE the map arm, below the statement that
	// calls tick(), so neither can skip a frame's advance the way a leaving
	// Escape or an open notice's own advance does.
	if s.ticks != 5 {
		t.Errorf("ticks = %d over 5 frames, want 5", s.ticks)
	}
}

// A REAL FIGURE AND A REAL ICON ARE MOSTLY TRANSPARENT — a body on an
// empty canvas, an object on an empty canvas — and the window they are
// drawn into is opaque.
//
// It failed exactly that way when the evidence stage first rendered a shipped
// figure, so the criterion is written from the defect and not from the
// requirement's wording.
func TestATransparentPictureLeavesTheGroundBeneathItOpaque(t *testing.T) {
	// A picture whose every pixel is transparent: the extreme of what a real
	// sheet's empty canvas is. Nothing of it may reach the surface.
	hollowIcon := image.NewRGBA(image.Rect(0, 0, 40, 40))

	var s InventorySubject
	s.ID = 7
	s.Slots[0] = hollowIcon

	if !slices.Equal(RenderWorn(s).Pix, RenderWorn(InventorySubject{ID: 7}).Pix) {
		t.Errorf("a wholly transparent icon changed the composed worn box; " +
			"the picture was written over the ground rather than composited onto it")
	}

	// And the whole surface stays opaque whatever it is handed, which is the
	// property a hole would break wherever it landed.
	opaque := func(name string, pic *image.RGBA) {
		t.Helper()
		for i := 3; i < len(pic.Pix); i += 4 {
			if pic.Pix[i] != 0xff {
				px := (i - 3) / 4
				t.Fatalf("%s: pixel (%d, %d) has alpha %d, want 255 — the box is a hole there",
					name, px%pic.Bounds().Dx(), px/pic.Bounds().Dx(), pic.Pix[i])
			}
		}
	}
	opaque("wholly transparent icons", RenderWorn(s))
	opaque("no icons at all", RenderWorn(InventorySubject{ID: 7}))
}

// toggleHudPanel changes no selection and keeps the same camera object. Pack
// and Book may change that camera's live ViewH and clamp its origin at a map
// edge; viewport_panels_hotfix_test.go owns those exact effects.
func TestToggleHudPanelKeepsSelectionAndCameraIdentity(t *testing.T) {
	for _, sel := range []selection{nil, {5}, {6}, {5, 6}, {9}} {
		v := inventoryTestApp(t).flow.viewer
		v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}, {ID: 6, Cell: image.Pt(2, 2)}})
		v.SetInventorySubject(InventorySubject{ID: 5})
		v.sel = slices.Clone(sel)
		cam := v.cam

		for p := hudPanel(0); p < hudPanelCount; p++ {
			before := v.hudShown(p)
			v.toggleHudPanel(p)
			if v.hudShown(p) == before {
				t.Errorf("selection %v: switch %q did not move", sel, hudToggleLabels[p])
			}
		}
		if !slices.Equal(v.sel, sel) {
			t.Errorf("selection %v: toggling moved it to %v", sel, v.sel)
		}
		if v.cam != cam {
			t.Errorf("selection %v: toggling replaced the camera", sel)
		}
	}

	// A panel index outside the four is a no-op rather than a panic — the
	// bound this method states in as many words.
	v := inventoryTestApp(t).flow.viewer
	before := v.hudHidden
	v.toggleHudPanel(-1)
	v.toggleHudPanel(hudPanelCount)
	if v.hudHidden != before {
		t.Error("an out-of-range panel moved a switch")
	}
}

// The wheel over the pack bar scrolls the pack one element per notch and
// leaves the map zoom alone; off the bar it still zooms (owner).
func TestTheWheelOverThePackScrollsItInsteadOfZooming(t *testing.T) {
	a := inventoryTestApp(t)
	v := a.flow.viewer
	layoutViewport(v, MenuWindowW, MenuWindowH)
	v.SetEntities([]MapEntity{{ID: 5, Cell: image.Pt(1, 1)}})
	v.sel = selection{5}
	bar, cols, ok := packBarRect(image.Pt(v.frameW, v.frameH))
	if !ok {
		t.Fatalf("setup: no pack bar at %dx%d", v.frameW, v.frameH)
	}
	pics := make([]*image.RGBA, cols+2)
	for i := range pics {
		pics[i] = solidPic(4, 4, color.RGBA{A: 0xff})
	}
	v.SetInventorySubject(packOf(5, pics...))
	if _, _, shown := v.packBar(); !shown {
		t.Fatal("setup: the pack bar is not shown")
	}
	c := packCellRects(bar, cols)[1]
	x, y := (c.Min.X+c.Max.X)/2, (c.Min.Y+c.Max.Y)/2

	zoom := v.cam.Zoom
	if err := a.HeadlessPointer("wheel-down", x, y); err != nil {
		t.Fatal(err)
	}
	if got := v.packScrollAt(cols); got != 1 || v.cam.Zoom != zoom {
		t.Fatalf("wheel-down over the pack: scroll %d zoom %v, want 1 and %v", got, v.cam.Zoom, zoom)
	}
	if err := a.HeadlessPointer("wheel-up", x, y); err != nil {
		t.Fatal(err)
	}
	if got := v.packScrollAt(cols); got != 0 || v.cam.Zoom != zoom {
		t.Fatalf("wheel-up over the pack: scroll %d zoom %v, want 0 and %v", got, v.cam.Zoom, zoom)
	}
	if err := a.HeadlessPointer("wheel-up", x, bar.Min.Y-40); err != nil {
		t.Fatal(err)
	}
	if v.packScrollAt(cols) != 0 || v.cam.Zoom == zoom {
		t.Fatalf("wheel over the map: scroll %d zoom %v, want 0 and a changed zoom", v.packScrollAt(cols), v.cam.Zoom)
	}
}
