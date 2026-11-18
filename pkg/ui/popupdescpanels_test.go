package ui

import (
	"bytes"
	"fmt"
	"image"
	"testing"
)

func TestPanelDescriptionUsesTheEditorFrame(t *testing.T) {
	window := image.Pt(1920, 1080)
	room := image.Rectangle{Max: window}
	cases := []struct {
		left, hover, frame string
	}{{"docked", "", "dock"}, {"docked", "splitter", "dock"},
		{"minimized", "", "strip"}, {"floating", "grip", "window"}}
	for _, lang := range []string{"en", "ru"} {
		font := wordsFont(lang)
		encode := mockupHostEncoder(font)
		for _, sample := range cases {
			t.Run(lang+"/"+sample.left+"/"+sample.hover, func(t *testing.T) {
				d := popupDesc{Window: "side-panels", Panels: &popupDescPanels{Left: sample.left, Hover: sample.hover},
					Sample: &popupScope{Values: map[string]string{"structure": "Watchtower"}}}
				m, err := NewEditorMockup(d.encode(), lang, window, font, font, encode)
				if err != nil {
					t.Fatal(err)
				}
				p := m.panels
				if len(p.panels) != 2 || !p.canvas.In(room) || p.canvas.Dx() < mapLeastWidth || p.canvas.Dy() < mapLeastHeight {
					t.Fatalf("panels %d, canvas %v", len(p.panels), p.canvas)
				}
				bottom, left := p.panels[0], p.panels[1]
				if left.kind != sample.frame || bottom.kind != "strip" {
					t.Fatalf("frame kinds: %s/%s", left.kind, bottom.kind)
				}
				if left.layout.Rect.Overlaps(p.canvas) != (sample.frame == "window") {
					t.Fatalf("left frame %v, canvas %v", left.layout.Rect, p.canvas)
				}
				for _, part := range p.panels {
					l := part.layout
					if !l.Rect.In(room) || !l.Close.In(l.Rect) || l.Close.Empty() {
						t.Errorf("frame %v, close %v", l.Rect, l.Close)
					}
					if hit, _ := l.hit(center(l.Close)); hit != popupClose {
						t.Errorf("close returns %v", hit)
					}
					titleEnd := l.Close.Min.X
					if !l.TitleHelp.Empty() {
						titleEnd = l.TitleHelp.Min.X
					}
					if part.spec.Title != "" && font.Advance(part.spec.Title) > titleEnd-l.Title.Min.X-popupPad-2 {
						t.Errorf("title does not fit frame %v", l.Rect)
					}
					if l.gripRect().Empty() != (part.kind != "window") {
						t.Errorf("unexpected grip on %s", part.kind)
					}
				}
				if sample.left == "minimized" && (left.spec.Title != "" || left.spec.CloseLabel != ">" ||
					len(left.layout.Rows) > 0 || !left.layout.splitterRect().Empty()) {
					t.Error("the side strip must contain only the restore button")
				}
				if left.kind == "dock" && left.layout.edgeAt(center(left.layout.splitterRect())) != edgeRight {
					t.Error("splitter mark missed the layout's inner resize edge")
				}
				page := m.Page()
				wantBox := func(part string, r image.Rectangle) {
					t.Helper()
					mark := fmt.Sprintf(`data-part="%s" data-box="%d %d %d %d"`, part, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
					if !bytes.Contains(page, []byte(mark)) {
						t.Errorf("missing HTML box %s %v", part, r)
					}
				}
				wantBox("map", p.canvas)
				for _, part := range p.panels {
					wantBox(part.kind, part.layout.Rect)
					wantBox("close", part.layout.Close)
				}
				switch sample.hover {
				case "splitter":
					wantBox("splitter", left.layout.splitterRect())
					if !p.pointer.In(left.layout.splitterRect()) || m.Picture().RGBAAt(p.pointer.X, p.pointer.Y) != editorPopupInk.Value {
						t.Error("the PNG does not paint the hovered splitter")
					}
				case "grip":
					wantBox("grip", left.layout.gripRect())
					if left.layout.edgeAt(p.pointer) != edgeRight|edgeBottom {
						t.Error("the grip does not use the floating window's diagonal edge")
					}
					lit := false
					pic := m.Picture()
					for y := left.layout.gripRect().Min.Y; y < left.layout.gripRect().Max.Y; y++ {
						for x := left.layout.gripRect().Min.X; x < left.layout.gripRect().Max.X; x++ {
							lit = lit || pic.RGBAAt(x, y) == editorPopupInk.Value
						}
					}
					if !lit {
						t.Error("the PNG does not paint the hovered grip")
					}

				}
				got, err := PageDescription(page)
				if err != nil || !bytes.Equal(got, d.encode()) {
					t.Errorf("page description round trip: %v", err)
				}
			})
		}
	}
}
