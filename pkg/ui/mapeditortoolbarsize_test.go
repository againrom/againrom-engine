package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"testing"
)

// dockedToolbarHigh is the docked toolbar row's height in a window width
// pixels wide: 36-pixel buttons where the 782-pixel row fits, else 22-pixel
// ones, with 3 pixels above and below them.
func dockedToolbarHigh(width int) int {
	if width >= 782 {
		return 42
	}
	return 28
}

// toolbarButtons are the toolbar's entries that are buttons, in table order:
// the rows that stand in a menu only are left out.
func toolbarButtons(e *MapEditor) []toolEntry {
	var out []toolEntry
	all := e.toolbarEntries()
	for _, b := range all {
		if b.onToolbar() {
			out = append(out, b)
		}
	}
	return out
}

// toolbarFloatGrid is each button's column and row in the floating window,
// as the owner's picture lays it out: rows 0 to 5 hold the six layer
// switches on the left and the six tools on the right, then one button per
// row on the left.
var toolbarFloatGrid = map[string]image.Point{
	"border": {0, 0}, "areas": {0, 1}, "drops": {0, 2}, "footprints": {0, 3}, "issues": {0, 4}, "heightlabels": {0, 5},
	"passview": {0, 6}, "playertint": {0, 7}, "grid": {0, 8}, "fit": {0, 9}, "report": {0, 10}, "objectspanel": {0, 11}, "propertiespanel": {0, 12},
	"select": {1, 0}, "brush.terrain": {1, 1}, "brush.height": {1, 2}, "brush.objects": {1, 3}, "brush.passability": {1, 4}, "drawarea": {1, 5},
	"brush.bridge": {1, 6},
}

// scaledIconMiss is the first inked pixel of icon ic that pic does not show
// as a scale by scale block of its ink, the bitmap's top-left corner at
// origin; empty when every inked pixel shows so.
func scaledIconMiss(pic *image.RGBA, origin image.Point, ic *editorIcon, scale int, disabled bool) string {
	for y, row := range ic.rows {
		for x := 0; x < len(row); x++ {
			want, ok := iconInk[row[x]]
			if !ok {
				continue
			}
			if disabled {
				want = iconDisabledInk
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					p := origin.Add(image.Pt(x*scale+dx, y*scale+dy))
					if got := pic.RGBAAt(p.X, p.Y); got != want {
						return fmt.Sprintf("bitmap pixel (%d,%d) shows %v at %v, want %v", x, y, got, p, want)
					}
				}
			}
		}
	}
	return ""
}

// sameRectPixels reports whether rectangle a of picture pa shows what
// rectangle b of picture pb shows.
func sameRectPixels(pa *image.RGBA, a image.Rectangle, pb *image.RGBA, b image.Rectangle) bool {
	if a.Size() != b.Size() {
		return false
	}
	for y := 0; y < a.Dy(); y++ {
		for x := 0; x < a.Dx(); x++ {
			if pa.RGBAAt(a.Min.X+x, a.Min.Y+y) != pb.RGBAAt(b.Min.X+x, b.Min.Y+y) {
				return false
			}
		}
	}
	return true
}

// toolbarFloating is the floating toolbar's window, when the toolbar floats.
func toolbarFloating(e *MapEditor) (popupLayout, bool) {
	s, ok := e.movableSpec(popToolbar)
	if !ok {
		return popupLayout{}, false
	}
	return layoutPopup(s, e.font, e.bodyFont()), true
}

// toolbarGripPoint is a point on the docked toolbar's grip, which spans 4 to
// 12 pixels from the row's left end.
func toolbarGripPoint(e *MapEditor) image.Point {
	return image.Pt(8, center(e.toolbarRect()).Y)
}

// toolEffect is what a toolbar button changes: the view switches, the tool,
// the panels, the report, the list and the message.
func toolEffect(e *MapEditor) string {
	return fmt.Sprintf("view=%+v tool=%s panels=%v/%v/%v report=%v list=%q message=%q",
		e.view, e.currentTool(), e.panelOn(panelLeft), e.panelOn(panelRight), e.panelOn(panelBottom), e.report.open, e.listKind(), e.message)
}

// At 1024x768 and 1920x1080, in English and in Russian, each docked toolbar
// button is 36 pixels square in a 42-pixel row, and its icon shows each
// bitmap pixel as a 2 by 2 block of the bitmap's colour.
func TestMapEditorToolbarDrawsIconsAtTwiceTheirSize(t *testing.T) {
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}} {
		for _, lang := range []string{"", "russian"} {
			name := size.String() + lang
			e, _, _ := panelsEditor(t, size.X, size.Y, lang)
			if got := e.toolbarRect().Dy(); got != 42 {
				t.Errorf("%s: the toolbar row is %d high, want 42", name, got)
			}
			pic := e.ChromeImage()
			for _, b := range toolbarButtons(e) {
				if b.rect.Dx() != 36 || b.rect.Dy() != 36 {
					t.Errorf("%s: %s is %v, want 36 square", name, b.key, b.rect)
					continue
				}
				if miss := scaledIconMiss(pic, b.rect.Min.Add(image.Pt(2, 2)), b.icon, 2, b.closed); miss != "" {
					t.Errorf("%s: %s: %s", name, b.key, miss)
				}
			}
		}
	}
}

// At 640x480, in English and in Russian, the docked row takes today's
// 22-pixel buttons: each lies whole inside the window, none overlaps another
// and the groups stand apart. The larger buttons start at 782 pixels wide.
func TestMapEditorToolbarTakesTodaysSizeWhereTheLargerDoesNotFit(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _, _ := panelsEditor(t, 640, 480, lang)
		window := image.Rect(0, 0, 640, 480)
		entries := toolbarButtons(e)
		var left, right image.Rectangle
		for i, b := range entries {
			if b.rect.Dx() != 22 || b.rect.Dy() != 22 || !b.rect.In(window) {
				t.Errorf("%q: %s at %v, want 22 square inside %v", lang, b.key, b.rect, window)
			}
			for _, o := range entries[i+1:] {
				if b.rect.Overlaps(o.rect) {
					t.Errorf("%q: %s at %v overlaps %s at %v", lang, b.key, b.rect, o.key, o.rect)
				}
			}
			if b.group == toolLeft {
				left = left.Union(b.rect)
			} else {
				right = right.Union(b.rect)
			}
		}
		if right.Min.X < left.Max.X+toolGroupGap {
			t.Errorf("%q: the groups %v and %v stand closer than %d", lang, left, right, toolGroupGap)
		}
		for _, c := range []struct{ width, side int }{{781, 22}, {782, 36}} {
			e.Layout(c.width, 600)
			if r, _ := e.ToolbarRect("select"); r.Dx() != c.side || e.toolbarRect().Dy() != dockedToolbarHigh(c.width) {
				t.Errorf("%q: at %d wide select is %v in a row %d high, want side %d", lang, c.width, r, e.toolbarRect().Dy(), c.side)
			}
		}
	}
}

// A drag of six pixels or more from the docked toolbar's grip floats it with
// its drag strip under the pointer, and a click does not. While it floats
// its row is gone, the side panels and the map start under the top bar, its
// buttons stand in the picture's grid, look, tip and press as the docked
// ones do, and the window moves by its strip; its close button docks it.
func TestMapEditorToolbarFloatsByGripDragAndDocksByClose(t *testing.T) {
	keys := []string{"grid", "border", "brush.terrain", "heightlabels", "objectspanel", "select"}
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			name := size.String() + lang
			e, _, _ := panelsEditor(t, size.X, size.Y, lang)
			r := &undockRig{t: t, e: e}
			dockedPic := e.ChromeImage()
			docked := map[string]toolEntry{}
			tips := map[string]string{}
			for _, b := range toolbarButtons(e) {
				docked[b.key] = b
				tips[b.key], _, _ = e.toolTip(center(b.rect))
			}
			mapBefore, barBefore := e.mapArea(), e.toolbarRect()
			grip := toolbarGripPoint(e)
			r.click(grip)
			r.drag(grip, image.Pt(3, 2))
			if _, ok := toolbarFloating(e); ok || e.toolbarRect() != barBefore {
				t.Fatalf("%s: a click or a short press on the grip floated the toolbar", name)
			}
			d := image.Pt(120, 70)
			r.drag(grip, d)
			l, ok := toolbarFloating(e)
			if !ok {
				t.Fatalf("%s: the grip drag did not float the toolbar", name)
			}
			if at := grip.Add(d); !at.In(l.Title) || at.In(l.Close) {
				t.Errorf("%s: the pointer at %v is off the window's drag strip %v", name, at, l.Title)
			}
			if !l.Rect.In(image.Rect(0, 0, size.X, size.Y)) {
				t.Errorf("%s: the window %v passes the editor's edges", name, l.Rect)
			}
			top := e.barHeight()
			if e.toolbarRect().Dy() != 0 || e.mapArea().Min.Y != top || e.chromeHit(image.Pt(size.X/2, barBefore.Max.Y-2)) == popToolbar {
				t.Errorf("%s: floating, the row is %v and the map %v, want the map from %d", name, e.toolbarRect(), e.mapArea(), top)
			}
			for _, i := range []int{panelLeft, panelRight} {
				if p := e.sidePanelRect(i); !p.Empty() && p.Min.Y != top {
					t.Errorf("%s: panel %d at %v, want its top at %d", name, i, p, top)
				}
			}
			entries := toolbarButtons(e)
			side := entries[0].rect.Dx()
			origin, _ := e.ToolbarRect("border")
			pic := e.ChromeImage()
			for _, b := range entries {
				cell, ok := toolbarFloatGrid[b.key]
				if want := origin.Min.Add(cell.Mul(side + toolGap)); !ok || b.rect.Min != want || b.rect.Dx() != side || b.rect.Dy() != side || !b.rect.In(l.Rect) || b.rect.Overlaps(l.Title) {
					t.Errorf("%s: %s floats at %v, want cell %v at %v in %v", name, b.key, b.rect, cell, want, l.Rect)
				}
				if e.chromeHit(center(b.rect)) != popToolbar {
					t.Errorf("%s: %s floating does not hit the toolbar", name, b.key)
				}
				was := docked[b.key]
				if b.shown != was.shown || b.closed != was.closed {
					t.Errorf("%s: %s floating on %v disabled %v, docked %v %v", name, b.key, b.shown, b.closed, was.shown, was.closed)
				}
				if tip, _, _ := e.toolTip(center(b.rect)); tip != tips[b.key] {
					t.Errorf("%s: %s floating tips %q, docked %q", name, b.key, tip, tips[b.key])
				}
				if side == was.rect.Dx() && !sameRectPixels(pic, b.rect, dockedPic, was.rect) {
					t.Errorf("%s: %s draws otherwise floating at %v than docked at %v", name, b.key, b.rect, was.rect)
				}
			}
			// A press on a floating button does what the docked one does, each
			// on a fresh pair of editors: a palette a tool opens may cover the
			// floating window.
			for _, key := range keys {
				f, _, _ := panelsEditor(t, size.X, size.Y, lang)
				rf := &undockRig{t: t, e: f}
				rf.drag(toolbarGripPoint(f), d)
				g, _, _ := panelsEditor(t, size.X, size.Y, lang)
				rg := &undockRig{t: t, e: g}
				fr, _ := f.ToolbarRect(key)
				dr, _ := g.ToolbarRect(key)
				rf.click(center(fr))
				rg.click(center(dr))
				if got, want := toolEffect(f), toolEffect(g); got != want {
					t.Errorf("%s: %s floating gave %s, docked %s", name, key, got, want)
				}
			}
			// The window moves by its strip.
			l, _ = toolbarFloating(e)
			r.drag(image.Pt(l.Title.Min.X+4, center(l.Title).Y), image.Pt(-30, 20))
			if m, _ := toolbarFloating(e); m.Rect.Min != l.Rect.Min.Add(image.Pt(-30, 20)) {
				t.Errorf("%s: the window moved to %v from %v", name, m.Rect.Min, l.Rect.Min)
			}
			// Its close button docks it back into its row.
			m, _ := toolbarFloating(e)
			r.click(center(m.Close))
			if _, ok := toolbarFloating(e); ok || e.toolbarRect() != barBefore || e.mapArea() != mapBefore {
				t.Errorf("%s: the close button left the row %v and the map %v, want %v and %v", name, e.toolbarRect(), e.mapArea(), barBefore, mapBefore)
			}
			for _, b := range toolbarButtons(e) {
				if b.rect != docked[b.key].rect {
					t.Errorf("%s: %s docks back at %v, was %v", name, b.key, b.rect, docked[b.key].rect)
				}
			}
		}
	}
}

// The floating window takes the larger buttons where it fits under the top
// bar: the top bar, the drag strip, 6 pixels around the grid and 13 rows of
// 36-pixel buttons 2 pixels apart; a lower window takes today's buttons.
func TestMapEditorFloatingToolbarTakesTodaysSizeWhereTheLargerDoesNotFit(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _, _ := panelsEditor(t, 1024, 768, lang)
		r := &undockRig{t: t, e: e}
		r.drag(toolbarGripPoint(e), image.Pt(40, 40))
		l, ok := toolbarFloating(e)
		if !ok {
			t.Fatalf("%q: the grip drag did not float the toolbar", lang)
		}
		need := e.barHeight() + l.Title.Dy() + 2*6 + 13*36 + 12*2
		t.Logf("%q: the floating window takes 36-pixel buttons from %d pixels high", lang, need)
		for _, c := range []struct{ high, side int }{{need - 1, 22}, {need, 36}} {
			e.Layout(1024, c.high)
			if b, _ := e.ToolbarRect("select"); b.Dx() != c.side {
				t.Errorf("%q: %d high, the floating select is %v, want side %d", lang, c.high, b, c.side)
			}
		}
		e.Layout(640, 480)
		if l, _ := toolbarFloating(e); l.Rect.Dy() > 480-e.barHeight() {
			t.Errorf("%q: at 640x480 the floating window is %v", lang, l.Rect)
		}
	}
}

// The profile file keeps whether the toolbar floats and its window's corner
// and size; docked, it keeps no toolbar entry. A file without the entry, or
// with a broken or out-of-range one, reads docked and the rest still reads.
func TestMapEditorFloatingToolbarProfileRoundTrip(t *testing.T) {
	for _, size := range panelSizes {
		name := size.String()
		e, _, saved := panelsEditor(t, size.X, size.Y, "")
		r := &undockRig{t: t, e: e}
		r.drag(toolbarGripPoint(e), image.Pt(100, 80))
		l, ok := toolbarFloating(e)
		if !ok || len(*saved) == 0 {
			t.Fatalf("%s: the grip drag floated %v and kept %d files", name, ok, len(*saved))
		}
		last := (*saved)[len(*saved)-1]
		var kept map[string]struct {
			Float *struct{ X, Y, Width, Height int }
		}
		if err := json.Unmarshal(last, &kept); err != nil {
			t.Fatal(err)
		}
		if f := kept["toolbar"].Float; f == nil || image.Rect(f.X, f.Y, f.X+f.Width, f.Y+f.Height) != l.Rect {
			t.Fatalf("%s: the file keeps %q, want the window %v", name, last, l.Rect)
		}
		f, _, _ := panelsEditor(t, size.X, size.Y, "")
		f.SetPanelProfile(last, nil, nil)
		if m, ok := toolbarFloating(f); !ok || m.Rect != l.Rect || f.toolbarRect().Dy() != 0 {
			t.Errorf("%s: read back floating %v at %v, want %v", name, ok, m.Rect, l.Rect)
		}
		var again []byte
		f.panels.save = func(b []byte) error { again = append([]byte(nil), b...); return nil }
		f.savePanels()
		if !bytes.Equal(again, last) {
			t.Errorf("%s: written again %q, was %q", name, again, last)
		}
		rf := &undockRig{t: t, e: f}
		m, _ := toolbarFloating(f)
		rf.click(center(m.Close))
		if _, ok := toolbarFloating(f); ok || bytes.Contains(again, []byte(`"toolbar"`)) {
			t.Errorf("%s: docked, floats %v and keeps %q", name, ok, again)
		}
		g, _, _ := panelsEditor(t, size.X, size.Y, "")
		for _, file := range []string{
			string(again),
			`{"left":{"minimized":false,"width":300}}`,
			`{"left":{"minimized":false,"width":300},"toolbar":"x"}`,
			`{"left":{"minimized":false,"width":300},"toolbar":{"float":"x"}}`,
			`{"left":{"minimized":false,"width":300},"toolbar":{"float":{"x":"a","y":2,"width":80,"height":300}}}`,
			`{"left":{"minimized":false,"width":300},"toolbar":{"float":{"x":1,"y":2,"width":0,"height":300}}}`,
			`{"left":{"minimized":false,"width":300},"toolbar":{"float":{"x":99999,"y":2,"width":80,"height":300}}}`,
			`{"left":{"minimized":false,"width":300},"toolbar":{"float":{"x":1,"y":2,"width":80,"height":99999}}}`,
		} {
			g.SetPanelProfile([]byte(file), nil, nil)
			if _, ok := toolbarFloating(g); ok || g.toolbarRect().Dy() == 0 {
				t.Errorf("%s: %s reads a floating toolbar", name, file)
			}
			if file != string(again) && g.panels.state[panelLeft] != (panelState{Restored: true, Width: 300}) {
				t.Errorf("%s: %s reads the left panel %+v", name, file, g.panels.state[panelLeft])
			}
		}
	}
}

// A drop-down with an icon, the icon painter's caller off the toolbar,
// draws its icon at one screen pixel per bitmap pixel, as it did.
func TestIconDropdownKeepsOnePixelPerBitmapPixel(t *testing.T) {
	pic := image.NewRGBA(image.Rect(0, 0, 40, 40))
	drawDropdown(pic, nil, nil, dropdown{Rect: image.Rect(5, 5, 27, 27), Icon: &iconGrid}, nil, image.Point{}, false)
	if miss := scaledIconMiss(pic, image.Pt(8, 8), &iconGrid, 1, false); miss != "" {
		t.Error(miss)
	}
	if scaledIconMiss(pic, image.Pt(7, 7), &iconGrid, 2, false) == "" {
		t.Error("the drop-down draws its icon at twice its size")
	}
}
