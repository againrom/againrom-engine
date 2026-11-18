package ui

import (
	"image"
	"image/draw"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMapEditorPanelFrameLayout(t *testing.T) {
	screen := image.Rect(0, 0, 1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, saved := panelsEditor(t, screen.Dx(), screen.Dy(), lang)
			e.SetBodyFont(wordsFont(lang))
			rig := &undockRig{t: t, e: e}
			dock := func(id editorPopupID) popupLayout {
				s := e.panelSpecOf(id)
				l := layoutPopup(s, e.font, e.bodyFont())
				if l.Rect.Empty() || !l.Rect.In(screen) || !l.Close.In(l.Title) {
					t.Fatalf("panel %d has frame %v and title button %v", id, l.Rect, l.Close)
				}
				end := l.Close.Min.X
				if !l.TitleHelp.Empty() {
					end = l.TitleHelp.Min.X
				}
				if s.Title != "" && e.font.Advance(s.Title) > end-l.Title.Min.X-popupPad-2 {
					t.Fatalf("panel %d clips title %q", id, s.Title)
				}
				return l
			}
			for i, id := range []editorPopupID{popTree, popProps, popReport} {
				if i != panelBottom && e.sidePanelRect(i).Dx() == sidePanels[i].width {
					t.Fatalf("fresh dock %d uses the old fixed size %d", i, sidePanels[i].width)
				}
				strip, ok := e.stripSpec(i)
				if !ok {
					rig.click(center(dock(id).Close))
					strip, ok = e.stripSpec(i)
				}
				s := dock(id)
				if !ok || strip.CloseLabel != e.word(sidePanels[i].restoreWord) || !s.splitterRect().Empty() {
					t.Fatalf("panel %d has no restore arrow on its strip", i)
				}
				thickness := s.Rect.Dx()
				wantThickness := s.Title.Dy() + popupTitleInset
				if i == panelBottom {
					thickness, wantThickness = s.Rect.Dy(), s.Title.Dy()+2*popupPad
				}
				if thickness != wantThickness {
					t.Fatalf("strip thickness %d for title %v", thickness, s.Title)
				}
				mapWithStrip := e.mapArea()
				rig.click(center(s.Close))
				l := dock(id)
				if !e.panelOpen(i) || !e.mapArea().In(mapWithStrip) || e.mapArea() == mapWithStrip {
					t.Fatalf("restore arrow left map %v with strip map %v", e.mapArea(), mapWithStrip)
				}
				at := center(l.splitterRect())
				rig.at(at, true)
				if e.drag != id || e.dragEdge != panelInnerEdge(i) {
					t.Fatalf("splitter %v started drag %d on edge %d", l.splitterRect(), e.drag, e.dragEdge)
				}
				d := image.Pt(28, 0)
				if i == panelRight {
					d.X = -d.X
				}
				if i == panelBottom {
					d = image.Pt(0, -28)
				}
				rig.at(at.Add(d), true)
				rig.at(at.Add(d), false)
				m := e.mapArea()
				if !m.In(screen) || m.Dx() < mapLeastWidth || m.Dy() < mapLeastHeight || l.Rect == e.sidePanelRect(i) {
					t.Fatalf("resize left panel %v and map %v", e.sidePanelRect(i), m)
				}
			}
			before := e.panelsShown()
			if len(*saved) == 0 {
				t.Fatal("resizing did not write the panel profile")
			}
			e.SetPanelProfile((*saved)[len(*saved)-1], nil, nil)
			if e.panelsShown() != before {
				t.Fatalf("reopening profile changed panel sizes from %v to %v", before, e.panelsShown())
			}
			props, ok := e.propsSpec()
			l := layoutPopup(props, e.font, e.bodyFont())
			e.ChromeImage()
			tip, hit := e.tipAt(center(l.TitleHelp))
			if !ok || len(l.Notes) != 1 || len(l.Rows) != 0 || props.Notes[0] != e.word("editor.side.unselected") ||
				props.TitleHelp != e.word("editor.side.none") || !hit || !tip.Help || !tip.Rect.In(screen) {
				t.Fatalf("empty Properties has rows %v, notes %v and help %v", l.Rows, props.Notes, tip.Rect)
			}
		})
	}
}

func TestMapEditorDockHasOnlyAnInnerSplitter(t *testing.T) {
	e, _, _ := panelsEditor(t, 1920, 1080, "")
	rig := &undockRig{t: t, e: e}
	pic := e.ChromeImage()
	for _, id := range []editorPopupID{popTree, popProps} {
		l := layoutPopup(e.panelSpecOf(id), e.font, e.bodyFont())
		p := l.Rect.Max.Sub(image.Pt(popupGripSide/2+1, popupGripSide/2+1))
		rig.at(p, true)
		if e.dragEdge != 0 {
			t.Errorf("dock %d started a corner resize", id)
		}
		rig.at(p, false)
		for y := l.Rect.Max.Y - popupGripSide; y < l.Rect.Max.Y-1; y++ {
			for x := l.Rect.Max.X - popupGripSide; x < l.Rect.Max.X-1; x++ {
				if pic.RGBAAt(x, y) == editorPopupInk.Edge {
					t.Fatalf("dock %d paints the corner grip at (%d,%d)", id, x, y)
				}
			}
		}
	}
	e.undockPanel(panelRight)
	root := layoutPopup(e.panelSpecOf(popProps), e.font, e.bodyFont())
	p := root.Rect.Max.Sub(image.Pt(popupGripSide/2, popupGripSide/2))
	rig.at(p, true)
	if e.drag != popProps || e.dragEdge != edgeRight|edgeBottom {
		t.Fatalf("floating Properties grip at %v started drag %d on edge %d", p, e.drag, e.dragEdge)
	}
	rig.at(p, false)
}

func TestMapEditorResizeHandlesHighlightWithTheCursor(t *testing.T) {
	original := setEditorCursorShape
	setEditorCursorShape = func(ebiten.CursorShapeType) {}
	t.Cleanup(func() { setEditorCursorShape = original })
	e, _, _ := panelsEditor(t, 1920, 1080, "")
	e.SetBodyFont(wordsFont("en"))
	rig := &undockRig{t: t, e: e}
	check := func(l popupLayout, handle image.Rectangle, shape ebiten.CursorShapeType) {
		t.Helper()
		rig.at(center(e.mapArea()), false)
		idle := image.NewRGBA(handle)
		draw.Draw(idle, handle, e.ChromeImage(), handle.Min, draw.Src)
		rig.at(center(handle), false)
		e.windowCursor()
		hot := e.ChromeImage()
		if handle.Empty() || e.cursorShape != shape {
			t.Fatalf("handle %v in %v has cursor %v, want %v", handle, l.Rect, e.cursorShape, shape)
		}
		for y := handle.Min.Y; y < handle.Max.Y; y++ {
			for x := handle.Min.X; x < handle.Max.X; x++ {
				if idle.RGBAAt(x, y) == editorPopupInk.Edge && hot.RGBAAt(x, y) == editorPopupInk.Value {
					return
				}
			}
		}
		t.Errorf("handle %v has no hover accent", handle)
	}
	dock := layoutPopup(e.panelSpecOf(popProps), e.font, e.bodyFont())
	y := center(dock.Rect).Y
	check(dock, image.Rect(dock.Rect.Min.X, y-1, dock.Rect.Min.X+4, y+1), ebiten.CursorShapeEWResize)
	e.undockPanel(panelRight)
	float := layoutPopup(e.panelSpecOf(popProps), e.font, e.bodyFont())
	check(float, image.Rectangle{Min: float.Rect.Max.Sub(image.Pt(popupGripSide, popupGripSide)), Max: float.Rect.Max},
		ebiten.CursorShapeNWSEResize)
}
