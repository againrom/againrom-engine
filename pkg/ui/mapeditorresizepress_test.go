package ui

import (
	"image"
	"testing"
	"time"
)

// pressDrawn presses the middle of the prompt's drawn button n.
func pressDrawn(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	r, ok := e.PromptButtonRect(n)
	if !ok {
		t.Fatalf("no drawn button %d", n)
	}
	p := r.Min.Add(r.Size().Div(2))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

// resizePrompt drags the open prompt's bottom edge or its corner outward.
func resizePrompt(t *testing.T, e *MapEditor, how string) {
	t.Helper()
	pick, d := bottomEdge, image.Pt(0, 40)
	if how == "corner" {
		pick, d = bottomRight, image.Pt(60, 40)
	}
	if before, after := edgeDrag(t, e, pick, d); after.Dy() <= before.Dy() {
		t.Fatalf("%s drag did not grow the window: %v -> %v", how, before, after)
	}
}

// The notice before test play and the single or network map question act on
// the buttons they draw after a resize by an edge or a corner: the press
// layout is the layout drawn.
func TestResizedNoticeAndNewMapQuestionButtonsAct(t *testing.T) {
	for _, how := range []string{"edge", "corner"} {
		for _, c := range []struct {
			name string
			n    int
			want int
		}{{"continue", noticeGo, 1}, {"cancel", noticeStay, 0}} {
			t.Run("notice/"+how+"/"+c.name, func(t *testing.T) {
				f, rows := endless()
				p := &mapPanelEdits{facts: f, rows: rows}
				e, _ := mapPanelEditor(t, p, nil)
				_ = e.Key("testplay")
				if e.prompt.kind != promptNotice {
					t.Fatal("no notice")
				}
				resizePrompt(t, e, how)
				pressDrawn(t, e, c.n)
				if e.prompt.kind == promptNotice || p.plays != c.want {
					t.Fatalf("button %d: notice open %v, %d test plays, want %d", c.n, e.prompt.kind == promptNotice, p.plays, c.want)
				}
			})
		}
		maps := []EditorMap{{Key: "new/32x32", Label: "New map 32 x 32"}, {Key: "scenario/10.alm", Label: "10"}}
		for _, c := range []struct {
			name string
			n    int
			want string
		}{{"single", newModeSingle, "new/32x32"}, {"multiplayer", newModeMulti, "new/32x32" + NewMapMultiplayer}, {"cancel", newModeStay, ""}} {
			t.Run("new map/"+how+"/"+c.name, func(t *testing.T) {
				e, opened := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, maps)
				*opened = nil
				_ = e.Key("new")
				_ = e.Key("clear")
				e.TextInput("32x32")
				_ = e.Key("enter")
				if e.prompt.kind != promptNewMode {
					t.Fatal("no question")
				}
				resizePrompt(t, e, how)
				pressDrawn(t, e, c.n)
				got := ""
				if len(*opened) > 0 {
					got = (*opened)[len(*opened)-1]
				}
				if got != c.want || e.prompt.kind == promptNewMode {
					t.Fatalf("button %d opened %q, want %q; prompt %+v", c.n, got, c.want, e.prompt)
				}
			})
		}
	}
}

// A widened Map panel follows a link where the row is drawn, to its last
// pixel.
func TestResizedMapPanelLinkRowsFollowTheDrawnRow(t *testing.T) {
	e, _ := mapPanelEditor(t, &mapPanelEdits{facts: playable()}, nil)
	_ = e.Key("panel")
	if !e.PanelOpen() {
		t.Fatal("panel is not open")
	}
	before, after := edgeDrag(t, e, rightEdge, image.Pt(80, 0))
	if after.Dx() <= before.Dx() {
		t.Fatalf("the panel did not widen: %v -> %v", before, after)
	}
	n := panelRow(t, e, "Name")
	r, ok := e.PanelRowRect(n)
	if !ok {
		t.Fatal("no name row")
	}
	p := image.Pt(r.Max.X-2, (r.Min.Y+r.Max.Y)/2)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	if !e.prompt.open() {
		t.Fatalf("a press at the last pixel %v of the drawn Name row %v opened nothing", p, r)
	}
	_ = e.Key("escape")
	title, _ := e.TitleRect()
	c := image.Pt(title.Max.X-12, (title.Min.Y+title.Max.Y)/2)
	e.Pointer(Input{CursorX: c.X, CursorY: c.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(Input{CursorX: c.X, CursorY: c.Y}, time.Unix(1, 0))
	if e.PanelOpen() {
		t.Fatalf("a press on the drawn close button at %v of the title %v left the panel open", c, title)
	}
}
