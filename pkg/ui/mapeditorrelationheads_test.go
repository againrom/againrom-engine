package ui

import (
	"image"
	"slices"
	"testing"

	"againrom/pkg/words"
)

// relationHeadsEditor is the modes editor on the Players page of a map
// whose players are names, in book's words (nil for English): player 1
// holds Hostile toward 2, Ally toward 3 and Neutral toward 4, player 2
// holds 7 toward 1, and every other relation is Hostile.
func relationHeadsEditor(t *testing.T, book *words.Book, names []string) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	if book != nil {
		e.SetWords(*book)
	}
	host := e.doc.Edits.(*modesFull)
	host.players = names
	host.relations = map[[2]int]int64{{0, 1}: 1, {0, 2}: 2, {0, 3}: 0, {1, 0}: 7}
	var records []InspectionRecord
	for _, r := range e.doc.Records {
		if r.Kind != "Player" {
			records = append(records, r)
		}
	}
	for i, name := range names {
		records = append(records, InspectionRecord{Kind: "Player", Index: i, Section: 5, Label: name})
	}
	e.doc.Records, e.relations = records, relationMatrix{}
	if err := e.Key("players"); err != nil {
		t.Fatal(err)
	}
	return e
}

var fourPlayers = []string{"1: Self", "2: Enemy", "3: Friend", "4: Nobody"}

// Each column of the relation matrix is headed with its row's head, written
// upright from bottom to top in a head row as tall as the longest head and
// no taller than relationHeadMax; a head too long for that row is cut as
// editorClip cuts it, the way a cut row head ends.
func TestRelationMatrixHeadsEachColumnWithItsRowHead(t *testing.T) {
	long := "3: The brigands of the northern marshes and their kin"
	for _, names := range [][]string{fourPlayers, {"1: Self", "2: Enemy", long, "4: Nobody"}} {
		e := relationHeadsEditor(t, nil, names)
		s, ok := e.listSpec()
		g, body := s.Grid, e.bodyFont()
		if !ok || g == nil || !slices.Equal(g.ColHeads, g.RowHeads) || !slices.Equal(g.RowHeads, names) {
			t.Fatalf("the matrix's column heads %q are not its row heads %q", g.ColHeads, g.RowHeads)
		}
		longest := 0
		for _, n := range names {
			longest = max(longest, body.Advance(n))
		}
		want := min(relationHeadMax, max(relationSide, longest+8))
		if g.HeadH != want || g.height() != 4*relationSide+want+popupPad {
			t.Fatalf("head row %d high, grid %d high; want a head row of %d", g.HeadH, g.height(), want)
		}
		if longest+8 > relationHeadMax && want != relationHeadMax || want > relationHeadMax {
			t.Fatalf("head row %d high for a %d-pixel head, the cap %d", want, longest, relationHeadMax)
		}
		l := layoutPopup(s, e.font, body)
		if len(l.GridColHeads) != 4 {
			t.Fatalf("%d column heads shown", len(l.GridColHeads))
		}
		pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
		drawPopup(pic, e.font, body, nil, s, l, editorPopupInk, image.Pt(-1, -1), false)
		for c, r := range l.GridColHeads {
			if r.Dy() != want || r.Dx() != relationSide || r.Max.Y != l.Grid.Min.Y || r.Min.X != l.GridCells[c].Min.X-1 {
				t.Fatalf("column head %d at %v, cells from %v, want %d high", c, r, l.Grid, want)
			}
			// The head as the body font writes it flat, turned a quarter turn
			// counter-clockwise: its first glyph at the bottom.
			head := editorClip(body, names[c], r.Dy()-4)
			if names[c] == long && (head == long || body.Advance(head) > relationHeadMax-4) {
				t.Fatalf("the long head is drawn as %q", head)
			}
			w, h := body.Advance(head), fontHeight(body, 15)
			flat := image.NewRGBA(image.Rect(0, 0, w, h))
			body.Draw(flat, head, 0, 0, editorPopupInk.Label)
			painted := 0
			for y := range h {
				for x := range w {
					if ink := flat.RGBAAt(x, y); ink.A != 0 {
						painted++
						at := image.Pt(r.Min.X+(r.Dx()-h)/2+y, r.Max.Y-3-x)
						if !at.In(r) || pic.RGBAAt(at.X, at.Y) != ink {
							t.Fatalf("column head %d: flat pixel %d,%d is not at %v turned", c, x, y, at)
						}
					}
				}
			}
			if painted == 0 {
				t.Fatalf("column head %d paints nothing", c)
			}
		}
	}
}

// The hover line over a matrix cell names both players and the value in the
// editor's words, a value the claims do not name by its number; over a
// column head it is the whole head. The diagonal, a row head and a row the
// popup shows whole keep no hover line, and a cut row keeps its own.
func TestRelationMatrixHoverNamesBothPlayersAndTheValue(t *testing.T) {
	ru, err := words.Load(words.Engine(), "ru")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		book *words.Book
		want map[int]string
	}{
		{"en", nil, map[int]string{1: "1: Self -> 2: Enemy: Hostile", 2: "1: Self -> 3: Friend: Ally", 3: "1: Self -> 4: Nobody: Neutral", 4: "2: Enemy -> 1: Self: 7", 14: "4: Nobody -> 3: Friend: Hostile"}},
		{"ru", &ru, map[int]string{1: "1: Self -> 2: Enemy: Вражда", 2: "1: Self -> 3: Friend: Союз", 3: "1: Self -> 4: Nobody: Нейтралитет", 4: "2: Enemy -> 1: Self: 7", 14: "4: Nobody -> 3: Friend: Вражда"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := relationHeadsEditor(t, tc.book, fourPlayers)
			s, _ := e.listSpec()
			body, bounds := e.bodyFont(), image.Rect(0, 0, 1024, 768)
			l := layoutPopup(s, e.font, body)
			tipAt := func(p image.Point) (popupTip, bool) { return popupTipAt(bounds, e.font, body, s, l, p, true) }
			for n, want := range tc.want {
				cell := l.GridCells[n]
				tip, ok := tipAt(center(cell))
				if !ok || !slices.Equal(tip.Lines, []string{want}) {
					t.Fatalf("cell %d hover line %q, want %q", n, tip.Lines, want)
				}
				if tip.Rect.Min != image.Pt(cell.Min.X, cell.Max.Y) || tip.Rect.Dx() != body.Advance(want)+8 || !tip.Rect.In(bounds) {
					t.Fatalf("cell %d hover box %v under the cell %v", n, tip.Rect, cell)
				}
			}
			for _, n := range []int{0, 5, 10, 15} {
				if tip, ok := tipAt(center(l.GridCells[n])); ok {
					t.Fatalf("diagonal cell %d has a hover line %q", n, tip.Lines)
				}
			}
			for c, r := range l.GridColHeads {
				if tip, ok := tipAt(center(r)); !ok || !slices.Equal(tip.Lines, []string{fourPlayers[c]}) || tip.Rect.Min.Y != r.Max.Y {
					t.Fatalf("column head %d hover line %q at %v", c, tip.Lines, tip.Rect)
				}
			}
			if tip, ok := tipAt(center(l.GridRowHeads[1])); ok {
				t.Fatalf("a row head has a hover line %q", tip.Lines)
			}
			for _, rr := range l.Rows {
				full, cut := rowFull(e.font, body, s.Rows[rr.Index], rr)
				tip, ok := tipAt(center(rr.Rect))
				if ok != cut || ok && (!slices.Equal(tip.Lines, editorWrap(body, full, rr.Rect.Dx()-8)) || tip.Rect.Min != image.Pt(rr.Rect.Min.X, rr.Rect.Max.Y)) {
					t.Fatalf("row %q hover line %v %q, cut %v", full, ok, tip.Lines, cut)
				}
			}
			// The popup's painter draws the cell's hover line in its one box.
			frames := func(s popupSpec) int {
				n := 0
				widgetRecord = func(c widgetCall) {
					if c.kind == widgetFrame {
						n++
					}
				}
				defer func() { widgetRecord = nil }()
				drawPopup(image.NewRGBA(bounds), e.font, body, nil, s, l, editorPopupInk, center(l.GridCells[1]), true)
				return n
			}
			bare, grid := s, *s.Grid
			grid.Cells = slices.Clone(grid.Cells)
			grid.Cells[1].Tip, bare.Grid = "", &grid
			if over, off := frames(s), frames(bare); over != off+1 {
				t.Fatalf("frames with the cell's hover line %d, without %d", over, off)
			}
		})
	}
}
