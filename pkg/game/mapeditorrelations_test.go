package game

import (
	"bytes"
	"image"
	"strconv"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// relationOf is player i's stored relation toward player j in raw.
func relationOf(t *testing.T, raw []byte, i, j int) int64 {
	t.Helper()
	m, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	v, err := m.FieldValue(mapedit.KindPlayer, i, "relation"+strconv.Itoa(j))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// On a map of three players the Players page's matrix cell of player 1
// toward player 2 is the record's relation word: a click writes the next
// value into that word alone, as one edit one undo takes back, and a click
// on the diagonal writes nothing.
func TestMapEditorRelationMatrixWritesOneRelationWord(t *testing.T) {
	m := built(t, synth.ALMOptions{Width: 32, Height: 32}, func(m *mapedit.Editor) {
		for range 2 {
			rec, err := m.NewPlayerRecord()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.Place(mapedit.KindPlayer, rec); err != nil {
				t.Fatal(err)
			}
		}
	}, nil)
	_, e, _ := openTestMapEditor(t, m.Bytes())
	toward := 0
	for _, f := range e.Document().Edits.Fields("Player", 0) {
		if f.Toward > 0 {
			toward++
		}
	}
	if toward != 3 {
		t.Fatalf("player 1 offers %d relation fields, want 3", toward)
	}
	_ = e.Key("players")
	click := func(n int) {
		r, ok := e.PlaceGridCellRect(n)
		if !ok {
			t.Fatalf("matrix cell %d is not shown", n)
		}
		p := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
	}
	start := bytes.Clone(e.Document().SourceBytes)
	click(0)
	if !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatal("a click on the diagonal changed the map")
	}
	was := relationOf(t, start, 0, 1)
	want := (was + 1) % 3
	click(1)
	now := e.Document().SourceBytes
	changed := 0
	for i := range min(len(now), len(start)) {
		if now[i] != start[i] {
			changed++
		}
	}
	if got := relationOf(t, now, 0, 1); got != want || len(now) != len(start) || changed == 0 || changed > 2 {
		t.Fatalf("the click set %d (want %d from %d) and changed %d bytes", got, want, was, changed)
	}
	if relationOf(t, now, 1, 0) != relationOf(t, start, 1, 0) {
		t.Fatal("the click changed the mirror cell")
	}
	if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, start) {
		t.Fatal("one undo did not take the click back", err)
	}
}
