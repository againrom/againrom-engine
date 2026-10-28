package ui

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"sync"
)

type InspectionTextFields interface {
	SetTextField(kind string, index int, key, value string) (*InspectionDocument, error)
}

const editorPlayersFilter = 8

func (e *MapEditor) keyPlayers(string) error {
	if e.doc == nil {
		return nil
	}
	wasPlayers := e.listView() && !e.brushActive() && e.filter == editorPlayersFilter
	e.leave(leaveDrag | leavePick | leaveReport | leaveCatalog | leaveBrush)
	e.message = ""
	e.filter, e.listShown = 0, !wasPlayers
	if !wasPlayers {
		e.filter = editorPlayersFilter
	}
	e.resetList(0)
	rows := e.rows()
	if len(rows) == 0 {
		e.selected = -1
		return nil
	}
	for _, row := range rows {
		if row == e.selected {
			e.Select(row, false)
			return nil
		}
	}
	e.Select(rows[0], false)
	return nil
}

// keyAdd opens the Add window on the kind the shown list holds.
func (e *MapEditor) keyAdd(string) error {
	if !e.listView() || e.brushActive() {
		return nil
	}
	if row, ok := addKindFor(e.filter); ok {
		e.openAdd(row.Kind)
	}
	return nil
}

func (e *MapEditor) stepPlayer(delta int) {
	rows := e.rows()
	if len(rows) == 0 {
		return
	}
	at := 0
	for i, row := range rows {
		if row == e.selected {
			at = min(len(rows)-1, max(0, i+delta))
			break
		}
	}
	e.Select(rows[at], false)
}

// relationMatrix is the Players page's relation matrix as one document's
// host gives it: the players in index order, their list labels, and the
// relation field each row holds toward each column.
type relationMatrix struct {
	doc     *InspectionDocument
	players []int
	labels  []string
	fields  [][]InspectionField
}

// relationSide is a matrix cell's side; relationHeadMax caps the row
// headings' column.
const (
	relationSide    = 26
	relationHeadMax = 140
)

// relationWords name the values the claims give a meaning: bit 0 is
// hostile, an attack sets it while the low two bits are clear, and bit 1
// blocks that (AI-DIPLO-004); the shipped maps hold 0, 1 and 2
// (AI-DIPLO-005).
var relationWords = []string{"editor.relation.neutral", "editor.relation.hostile", "editor.relation.ally"}

// relationMatrixOf reads the matrix of the open document once.
func (e *MapEditor) relationMatrixOf() *relationMatrix {
	m := &e.relations
	if e.doc == nil || m.doc == e.doc {
		return m
	}
	*m = relationMatrix{doc: e.doc}
	for _, r := range e.doc.Records {
		if r.Kind == "Player" {
			m.players, m.labels = append(m.players, r.Index), append(m.labels, r.Label)
		}
	}
	n := len(m.players)
	for _, p := range m.players {
		row := make([]InspectionField, n)
		for _, f := range e.doc.Edits.Fields("Player", p) {
			if f.Toward >= 1 && f.Toward <= n && !f.Text {
				row[f.Toward-1] = f
			}
		}
		m.fields = append(m.fields, row)
	}
	return m
}

// relationGridSpec is the Players page's matrix as popup data: row i,
// column j is player i's relation toward player j, a picture for a value
// the claims name and its number for any other; the diagonal and a cell
// the host offers no field for are inert. The rows are the matrix's
// heading and its legend.
func (e *MapEditor) relationGridSpec() (*popupGrid, []popupRow, bool) {
	if e.listKind() != "records" || e.filter != editorPlayersFilter {
		return nil, nil, false
	}
	m := e.relationMatrixOf()
	n, offered := len(m.players), false
	g := &popupGrid{Side: relationSide, Rows: n, Cols: n, HeadW: relationSide}
	for i := range n {
		g.RowHeads = append(g.RowHeads, m.labels[i])
		g.ColHeads = append(g.ColHeads, strconv.Itoa(i+1))
		if body := e.bodyFont(); body != nil {
			g.HeadW = min(relationHeadMax, max(g.HeadW, body.Advance(m.labels[i])+8))
		}
		for j := range n {
			f := m.fields[i][j]
			cell := popupGridCell{Inert: i == j || f.Key == ""}
			if !cell.Inert {
				offered = true
				if cell.Icon = relationIcon(f.Value); cell.Icon == nil {
					cell.Label = strconv.FormatInt(f.Value, 10)
				}
			}
			g.Cells = append(g.Cells, cell)
		}
	}
	if !offered {
		return nil, nil, false
	}
	rows := []popupRow{{Kind: popupHeading, Label: e.word("editor.relation.title")}}
	for v, w := range relationWords {
		rows = append(rows, popupRow{Kind: popupText, Label: e.word(w), Value: strconv.Itoa(v), Icons: []*image.RGBA{relationIcon(int64(v))}})
	}
	return g, rows, true
}

// relationGridHeight is the room the matrix and its rows take in the list
// popup, 0 with none.
func (e *MapEditor) relationGridHeight() int {
	g, rows, ok := e.relationGridSpec()
	if !ok {
		return 0
	}
	return g.height() + len(rows)*(fontHeight(e.bodyFont(), 15)+popupRowExtra)
}

// relationPress is a press on matrix cell n: it sets the next relation
// value of that row toward that column as one edit.
func (e *MapEditor) relationPress(n int) {
	m := e.relationMatrixOf()
	k := len(m.players)
	if k == 0 || n < 0 || n >= k*k || n/k == n%k {
		return
	}
	f := m.fields[n/k][n%k]
	if f.Key == "" {
		return
	}
	_ = e.adoptEdit(e.doc.Edits.SetField("Player", m.players[n/k], f.Key, relationNext(f.Value)))
}

// relationNext is the value a press on a cell holding v sets: the named
// values in turn, and the first after any other.
func relationNext(v int64) int64 {
	if v >= 0 && v < int64(len(relationWords)-1) {
		return v + 1
	}
	return 0
}

// relationIcons are the matrix's pictures, drawn here: a grey ring for
// neutral, a red cross for hostile and a green disc for ally.
var relationIcons = sync.OnceValue(func() []*image.RGBA {
	inks := []color.RGBA{{176, 176, 176, 255}, {214, 48, 40, 255}, {64, 184, 72, 255}}
	var out []*image.RGBA
	for v, ink := range inks {
		img := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y := range 16 {
			for x := range 16 {
				dx, dy := float64(x)-7.5, float64(y)-7.5
				d := math.Hypot(dx, dy)
				on := []bool{d >= 4.5 && d <= 6.5, d <= 7 && (math.Abs(dx-dy) <= 1.5 || math.Abs(dx+dy) <= 1.5), d <= 6.5}[v]
				if on {
					img.SetRGBA(x, y, ink)
				}
			}
		}
		out = append(out, img)
	}
	return out
})

// relationIcon is value v's picture, nil for a value the claims do not name.
func relationIcon(v int64) *image.RGBA {
	if pics := relationIcons(); v >= 0 && v < int64(len(pics)) {
		return pics[v]
	}
	return nil
}
