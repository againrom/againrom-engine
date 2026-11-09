package ui

import (
	"bytes"
	"image"
	"os"
	"testing"
	"time"
)

// undockRig drives one editor's pointer with a clock that moves a second per
// sample, so no two presses pair as a double click.
type undockRig struct {
	t   *testing.T
	e   *MapEditor
	now int64
}

func (r *undockRig) at(p image.Point, down bool) {
	r.now += 2
	r.e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: down}, time.Unix(r.now, 0))
}

func (r *undockRig) click(p image.Point) {
	r.at(p, true)
	r.at(p, false)
}

// drag presses at from, moves in two samples to from+d and releases.
func (r *undockRig) drag(from, d image.Point) {
	r.at(from, true)
	r.at(from.Add(image.Pt(d.X/2, d.Y/2)), true)
	r.at(from.Add(d), true)
	r.at(from.Add(d), false)
}

// panelWindow is the popup id and the rectangle of side panel i's window,
// docked or floating.
func panelWindow(t *testing.T, e *MapEditor, i int) (editorPopupID, popupLayout) {
	t.Helper()
	id := popTree
	s, ok := e.treeSpec()
	if i == panelRight {
		id = popObject
		s, _, ok = e.objectSpec()
	}
	if !ok {
		t.Fatalf("panel %d has no window", i)
	}
	return id, layoutPopup(s, e.font, e.bodyFont())
}

// undockRigFor opens an editor of size at lang with side panel i docked
// restored and its window hosted; ok is false where the window has no room
// for the panel.
func undockRigFor(t *testing.T, size image.Point, lang string, i int) (*undockRig, bool) {
	t.Helper()
	e, _, _ := panelsEditor(t, size.X, size.Y, lang)
	if i == panelRight {
		e.Select(7, false)
		if !e.panelOpen(panelRight) {
			_ = e.Key(sidePanels[panelLeft].key)
		}
	}
	if !e.panelOpen(i) {
		return nil, false
	}
	return &undockRig{t: t, e: e}, true
}

// pullOut drags side panel i's title row by d and returns the title press.
func (r *undockRig) pullOut(i int, d image.Point) image.Point {
	_, l := panelWindow(r.t, r.e, i)
	at := image.Pt(l.Title.Min.X+10, center(l.Title).Y)
	r.drag(at, d)
	return at
}

// A drag of a side panel's title row out of the panel undocks it: the panel's
// content becomes a window that follows the drag, the map takes the panel's
// room, and the window then moves and resizes as a floating popup does.
func TestMapEditorSidePanelUndocksByTitleDrag(t *testing.T) {
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			for _, i := range []int{panelLeft, panelRight} {
				name := size.String() + lang + sidePanels[i].name
				r, ok := undockRigFor(t, size, lang, i)
				if !ok {
					t.Logf("%s: the window has no room for the panel", name)
					continue
				}
				e := r.e
				before, mapBefore := e.sidePanelRect(i), e.mapArea()
				// A press that travels less than the pull distance stays docked.
				_, l := panelWindow(t, e, i)
				at := image.Pt(l.Title.Min.X+10, center(l.Title).Y)
				r.drag(at, image.Pt(2, 1))
				if e.panelFloats(i) || !e.panelOpen(i) {
					t.Fatalf("%s: a click on the title row undocked the panel", name)
				}
				d := image.Pt(40, 30)
				if i == panelRight {
					d.X = -40
				}
				r.pullOut(i, d)
				if !e.panelFloats(i) || e.panelOpen(i) {
					t.Fatalf("%s: the title drag did not undock the panel", name)
				}
				if !e.sidePanelRect(i).Empty() {
					t.Errorf("%s: the panel's place still shows %v", name, e.sidePanelRect(i))
				}
				if got := e.mapArea(); got.Dx() < mapLeastWidth || i == panelLeft && got.Min.X != 0 || i == panelRight && got.Max.X != size.X {
					t.Errorf("%s: the map is %v, was %v beside a panel %v", name, got, mapBefore, before)
				}
				id, w := panelWindow(t, e, i)
				if w.Rect.Min != before.Min.Add(d) || w.Rect.Size() != before.Size() {
					t.Errorf("%s: the window is %v, want %v moved by %v", name, w.Rect, before, d)
				}
				// The window moves by its title row.
				p := image.Pt(w.Title.Min.X+10, center(w.Title).Y)
				r.drag(p, image.Pt(-10, 5))
				if _, m := panelWindow(t, e, i); m.Rect.Min != w.Rect.Min.Add(image.Pt(-10, 5)) {
					t.Errorf("%s: the window moved to %v from %v", name, m.Rect.Min, w.Rect.Min)
				}
				// And resizes by its border.
				_, w = panelWindow(t, e, i)
				edge := image.Pt(w.Rect.Max.X-1, center(w.Rect).Y)
				if e.resizeEdgeAt(id, edge)&edgeRight == 0 {
					t.Fatalf("%s: the window's right border does not grab at %v", name, edge)
				}
				r.drag(edge, image.Pt(-20, 0))
				if _, m := panelWindow(t, e, i); m.Rect.Dx() != w.Rect.Dx()-20 || m.Rect.Min != w.Rect.Min {
					t.Errorf("%s: the window resized to %v from %v", name, m.Rect, w.Rect)
				}
			}
		}
	}
}

// The floating window's title row button closes it: the panel goes back to
// its minimized strip, and the strip's restore button docks it restored at
// its kept width.
func TestMapEditorFloatingPanelClosesToItsStrip(t *testing.T) {
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			for _, i := range []int{panelLeft, panelRight} {
				name := size.String() + lang + sidePanels[i].name
				r, ok := undockRigFor(t, size, lang, i)
				if !ok {
					continue
				}
				e := r.e
				width := e.sidePanelRect(i).Dx()
				r.pullOut(i, image.Pt(30, 30))
				btn, ok := e.FindControl(ControlQuery{Role: ControlPanelButton, N: i})
				if !ok || !e.panelFloats(i) {
					t.Fatalf("%s: no button on the floating window", name)
				}
				r.click(center(btn))
				if e.panelFloats(i) || e.panelOpen(i) {
					t.Fatalf("%s: the close button left the panel floating %v open %v", name, e.panelFloats(i), e.panelOpen(i))
				}
				if _, ok := e.stripSpec(i); !ok || e.sidePanelRect(i).Dx() != e.panelStrip(i) {
					t.Fatalf("%s: the closed panel shows no strip", name)
				}
				strip, ok := e.FindControl(ControlQuery{Role: ControlPanelButton, N: i})
				if !ok {
					t.Fatalf("%s: the strip has no restore button", name)
				}
				r.click(center(strip))
				if !e.panelOpen(i) || e.panelFloats(i) || e.sidePanelRect(i).Dx() != width {
					t.Errorf("%s: the restore button gave open %v floating %v width %d, want %d", name, e.panelOpen(i), e.panelFloats(i), e.sidePanelRect(i).Dx(), width)
				}
			}
		}
	}
}

// While a panel floats its toolbar switch and its key show it on, and a
// press of either closes the window to the strip.
func TestMapEditorFloatingPanelSwitchAndKeyCloseIt(t *testing.T) {
	on := func(e *MapEditor, key string) bool {
		for _, b := range e.toolbarEntries() {
			if b.key == key {
				return b.shown
			}
		}
		t.Fatalf("no toolbar entry %q", key)
		return false
	}
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			for _, i := range []int{panelLeft, panelRight} {
				name := size.String() + lang + sidePanels[i].name
				key := sidePanels[i].key
				for _, byKey := range []bool{true, false} {
					r, ok := undockRigFor(t, size, lang, i)
					if !ok {
						continue
					}
					e := r.e
					r.pullOut(i, image.Pt(30, 30))
					if !on(e, key) {
						t.Fatalf("%s: the switch shows a floating panel off", name)
					}
					if byKey {
						_ = e.Key(key)
					} else {
						for _, b := range e.toolbarEntries() {
							if b.key == key {
								r.click(center(b.rect))
							}
						}
					}
					if e.panelFloats(i) || e.panelOpen(i) || on(e, key) {
						t.Errorf("%s: key %v left floating %v open %v", name, byKey, e.panelFloats(i), e.panelOpen(i))
					}
				}
			}
		}
	}
}

// The profile file keeps whether a side panel floats and its window's corner
// and size; a file without them reads as a docked panel.
func TestMapEditorFloatingPanelProfileRoundTrip(t *testing.T) {
	for _, size := range panelSizes {
		for _, i := range []int{panelLeft, panelRight} {
			name := size.String() + sidePanels[i].name
			r, ok := undockRigFor(t, size, "", i)
			if !ok {
				continue
			}
			e := r.e
			var saved [][]byte
			e.panels.save = func(b []byte) error { saved = append(saved, append([]byte(nil), b...)); return nil }
			r.pullOut(i, image.Pt(25, 35))
			if len(saved) == 0 {
				t.Fatalf("%s: undocking and moving kept nothing", name)
			}
			last := saved[len(saved)-1]
			if !bytes.Contains(last, []byte(`"float":{`)) {
				t.Fatalf("%s: the saved state has no floating window: %q", name, last)
			}
			want := e.places[panelStoreRow(i)]
			f, _, _ := panelsEditor(t, size.X, size.Y, "")
			f.SetPanelProfile(last, nil, nil)
			if !f.panelFloats(i) || f.panelOpen(i) || f.places[panelStoreRow(i)] != want {
				t.Errorf("%s: read back floating %v place %+v, want %+v", name, f.panelFloats(i), f.places[panelStoreRow(i)], want)
			}
			if f.panelFloats(1 - i) {
				t.Errorf("%s: the other side floats", name)
			}
			// The same state written again is the same bytes.
			f.panels.save = func(b []byte) error { saved = append(saved, append([]byte(nil), b...)); return nil }
			f.savePanels()
			if again := saved[len(saved)-1]; !bytes.Equal(again, last) {
				t.Errorf("%s: written again %q, was %q", name, again, last)
			}
			// A file without the window reads docked or minimized, as before.
			f.SetPanelProfile([]byte(`{"left":{"minimized":false,"width":300},"right":{"minimized":true,"width":600}}`), nil, nil)
			if f.panelFloats(panelLeft) || f.panelFloats(panelRight) || f.panels.state[panelLeft] != (panelState{Restored: true, Width: 300}) {
				t.Errorf("%s: a file without a window reads %v floats %v", name, f.panels.state, f.panels.floats)
			}
			// A window that cannot be took is not floating.
			f.SetPanelProfile([]byte(`{"left":{"minimized":true,"width":300,"float":{"x":1,"y":2,"width":0,"height":5}}}`), nil, nil)
			if f.panelFloats(panelLeft) {
				t.Errorf("%s: an empty window floats", name)
			}
			f.SetPanelProfile(nil, os.ErrNotExist, nil)
			if f.panelFloats(panelLeft) || f.panelFloats(panelRight) {
				t.Errorf("%s: a missing file floats a panel", name)
			}
		}
	}
}

// A right panel the profile keeps floating stands at every window size, with
// its hint while nothing is selected; it moves, closes to its strip, and a
// window with no room for it docked leaves it on the strip.
func TestMapEditorFloatingRightPanelStandsWhereTheWindowHasNoRoom(t *testing.T) {
	for _, size := range panelSizes {
		for _, lang := range []string{"", "russian"} {
			name := size.String() + lang
			e, _, _ := panelsEditor(t, size.X, size.Y, lang)
			e.SetPanelProfile([]byte(`{"right":{"minimized":true,"width":520,"float":{"x":40,"y":120,"width":360,"height":300}}}`), nil, nil)
			e.Layout(size.X, size.Y)
			r := &undockRig{t: t, e: e}
			s, ok := e.propsSpec()
			if !ok || !e.panelFloats(panelRight) {
				t.Fatalf("%s: the floating right panel shows no window", name)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			if e.chromeHit(center(l.Rect)) != popProps {
				t.Fatalf("%s: a press inside the window does not reach it", name)
			}
			r.drag(image.Pt(l.Title.Min.X+10, center(l.Title).Y), image.Pt(15, 10))
			s, _ = e.propsSpec()
			if m := layoutPopup(s, e.font, e.bodyFont()); m.Rect.Min != l.Rect.Min.Add(image.Pt(15, 10)) {
				t.Errorf("%s: the hint window moved to %v from %v", name, m.Rect.Min, l.Rect.Min)
			}
			btn, _ := e.FindControl(ControlQuery{Role: ControlPanelButton, N: panelRight})
			r.click(center(btn))
			if e.panelFloats(panelRight) || e.panelOpen(panelRight) {
				t.Fatalf("%s: the close button left the panel on", name)
			}
			_ = e.Key(sidePanels[panelRight].key)
			if size.X < 1024 && (e.panelOpen(panelRight) || e.message != e.word(sidePanels[panelRight].narrowWord)) {
				t.Errorf("%s: a window with no room docked the panel, message %q", name, e.message)
			}
		}
	}
}
