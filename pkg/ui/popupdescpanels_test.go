package ui

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/pkg/words"
)

func TestPanelDescriptionUsesTheEditorFrame(t *testing.T) {
	window := image.Pt(1920, 1080)
	room := image.Rectangle{Max: window}
	cases := []struct {
		right, hover, frame string
	}{{"docked", "", "dock"}, {"docked", "help", "dock"}, {"hosted", "splitter", "dock"},
		{"minimized", "", "strip"}, {"floating", "grip", "window"}}
	for _, lang := range []string{"en", "ru"} {
		font := wordsFont(lang)
		book := words.For(lang)
		encode := mockupHostEncoder(font)
		for _, sample := range cases {
			t.Run(lang+"/"+sample.right+"/"+sample.hover, func(t *testing.T) {
				d := popupDesc{Window: "side-panels", Panels: &popupDescPanels{Right: sample.right, Hover: sample.hover},
					Sample: &popupScope{Values: map[string]string{"structure": "Watchtower"}}}
				m, err := NewEditorMockup(d.encode(), lang, window, font, font, encode)
				if err != nil {
					t.Fatal(err)
				}
				p := m.panels
				if len(p.panels) != 3 || !p.canvas.In(room) || p.canvas.Dx() < mapLeastWidth || p.canvas.Dy() < mapLeastHeight {
					t.Fatalf("panels %d, canvas %v", len(p.panels), p.canvas)
				}
				left, bottom, right := p.panels[0], p.panels[1], p.panels[2]
				if left.kind != "dock" || bottom.kind != "strip" || right.kind != sample.frame {
					t.Fatalf("frame kinds: %s/%s/%s", left.kind, bottom.kind, right.kind)
				}
				if right.layout.Rect.Overlaps(p.canvas) != (sample.frame == "window") {
					t.Fatalf("right frame %v, canvas %v", right.layout.Rect, p.canvas)
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
				if sample.right == "docked" {
					if right.spec.TitleHelp != encode(book.Text("editor.side.none")) || len(right.spec.Rows) != 0 ||
						len(right.spec.Notes) != 1 || right.spec.Notes[0] != encode(book.Text("editor.side.unselected")) {
						t.Error("the Properties instructions belong to title help and its status belongs to notes")
					}
				}
				if sample.right == "hosted" && (!strings.Contains(right.spec.Title, "Watchtower") || len(right.layout.Rows) == 0) {
					t.Error("the selected structure has no hosted title or rows")
				}
				if sample.right == "minimized" && (right.spec.Title != "" || right.spec.CloseLabel != "<" ||
					len(right.layout.Rows) > 0 || !right.layout.splitterRect().Empty()) {
					t.Error("the side strip must contain only the restore button")
				}
				for _, dock := range []struct {
					panel mockupPanel
					edge  popupEdge
				}{{left, edgeRight}, {right, edgeLeft}} {
					if dock.panel.kind == "dock" && dock.panel.layout.edgeAt(center(dock.panel.layout.splitterRect())) != dock.edge {
						t.Error("splitter mark missed the layout's inner resize edge")
					}
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
					wantBox("splitter", right.layout.splitterRect())
					if !p.pointer.In(right.layout.splitterRect()) || m.Picture().RGBAAt(p.pointer.X, p.pointer.Y) != editorPopupInk.Value {
						t.Error("the PNG does not paint the hovered splitter")
					}
				case "grip":
					wantBox("grip", right.layout.gripRect())
					if right.layout.edgeAt(p.pointer) != edgeRight|edgeBottom {
						t.Error("the grip does not use the floating window's diagonal edge")
					}
					lit := false
					pic := m.Picture()
					for y := right.layout.gripRect().Min.Y; y < right.layout.gripRect().Max.Y; y++ {
						for x := right.layout.gripRect().Min.X; x < right.layout.gripRect().Max.X; x++ {
							lit = lit || pic.RGBAAt(x, y) == editorPopupInk.Value
						}
					}
					if !lit {
						t.Error("the PNG does not paint the hovered grip")
					}
				case "help":
					wantBox("tip", p.tip.Rect)
					if !p.tip.Help || p.tip.Rect.Empty() || !p.tip.Rect.In(room) {
						t.Errorf("invalid help tip %v", p.tip)
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
