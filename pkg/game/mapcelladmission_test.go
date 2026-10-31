package game

import (
	"bytes"
	"fmt"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// admissionAnswer is what the admission of kind on route answers for cell:
// "outside", "ring", "closed" by the ground arms, or "open".
func admissionAnswer(view *alm.Map, arms []mapload.GroundArm, kind string, route cellRoute, cell image.Point) string {
	a := admissionOf(kind, route)
	switch a.verdict(view, cell) {
	case cellOutside:
		return "outside"
	case cellInRing:
		return "ring"
	}
	if c := cell.Y*int(view.Width) + cell.X; c >= 0 && c < len(arms) && a.closedArms(arms[c], 0, sim.DomainGround) != 0 {
		return "closed"
	}
	return "open"
}

// TestCellAdmissionRowsAnswerEachCell pins every (record kind, route) row on
// a 32x32 map for an inside cell, the edge cell 7,7 of the border ring and a
// cell outside the map. Only the sack's add route changed: it refuses the
// edge cell by its border arm, as the sack's group route does.
func TestCellAdmissionRowsAnswerEachCell(t *testing.T) {
	_, e := openRecords(t)
	view, err := alm.Open(e.Document().SourceBytes)
	if err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	arms := mapload.GroundArms(view, s.inspector.table)
	inside, edge, outside := image.Pt(16, 16), image.Pt(7, 7), image.Pt(32, 32)
	if c := inside.Y*32 + inside.X; arms[c] != 0 {
		t.Fatalf("the inside cell is closed by %v; the fixture needs an open one", arms[c])
	}
	rows := []struct {
		kind  string
		route cellRoute
		want  [3]string
		feet  bool
	}{
		{"Unit", admitAdd, [3]string{"open", "ring", "outside"}, true},
		{"Unit", admitMove, [3]string{"open", "ring", "outside"}, true},
		{"Unit", admitField, [3]string{"open", "open", "open"}, true},
		{"Unit", admitGroup, [3]string{"open", "ring", "outside"}, true},
		{"Unit", admitCheck, [3]string{"open", "ring", "ring"}, false}, // the checker asks only for inside cells
		{"Structure", admitAdd, [3]string{"open", "ring", "outside"}, true},
		{"Structure", admitMove, [3]string{"open", "ring", "outside"}, true},
		{"Structure", admitField, [3]string{"open", "open", "open"}, true},
		{"Structure", admitGroup, [3]string{"open", "ring", "outside"}, true},
		{"Structure", admitCheck, [3]string{"open", "open", "open"}, false},
		{"Sack", admitAdd, [3]string{"open", "closed", "outside"}, false},
		{"Sack", admitMove, [3]string{"open", "open", "outside"}, false},
		{"Sack", admitField, [3]string{"open", "open", "open"}, false},
		{"Sack", admitGroup, [3]string{"open", "closed", "outside"}, false},
		{"Stock", admitMove, [3]string{"open", "open", "outside"}, false},
		{"Stock", admitField, [3]string{"open", "open", "open"}, false},
		{"Enchantment", admitAdd, [3]string{"open", "open", "outside"}, false},
		{"Enchantment", admitMove, [3]string{"open", "open", "outside"}, false},
		{"Enchantment", admitField, [3]string{"open", "open", "outside"}, false},
		{ringDrop, admitAdd, [3]string{"open", "ring", "outside"}, false},
		{"Trigger", admitField, [3]string{"open", "open", "open"}, false},
	}
	for _, r := range rows {
		var got [3]string
		for i, c := range []image.Point{inside, edge, outside} {
			got[i] = admissionAnswer(view, arms, r.kind, r.route, c)
		}
		if got != r.want || admissionOf(r.kind, r.route).footprint != r.feet {
			t.Errorf("%s on route %d answers %v, footprint %v; want %v, %v", r.kind, r.route, got, admissionOf(r.kind, r.route).footprint, r.want, r.feet)
		}
	}
}

// TestMapEditorAddsNoSackOnTheEdgeCell adds a sack on the edge cell 7,7 by
// each add route: each is refused with the words a group move of a sack onto
// that cell gives, and the map keeps its bytes. One cell inside, 8,8, takes it.
func TestMapEditorAddsNoSackOnTheEdgeCell(t *testing.T) {
	_, e := openRecords(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	w := words.For("")
	edge, inside := image.Pt(7, 7), image.Pt(8, 8)
	refusal := func(index int) string {
		return fmt.Sprintf(w.Text("editor.selection.closed"), fmt.Sprintf(w.Text("editor.selection.sack"), index), edge.X, edge.Y, w.Text("editor.pass.border"))
	}
	before := append([]byte(nil), s.model.Bytes()...)
	rec := ui.InspectionNewRecord{Kind: "Sack", Cell: edge, HasCell: true, Values: []ui.InspectionFieldValue{{Key: "item", Value: 0x0101}}}
	if err := s.CheckNewRecord(rec, "cell"); err == nil || err.Error() != refusal(1) {
		t.Fatalf("the Add window's cell step on %v: %v; want %q", edge, err, refusal(1))
	}
	if _, err := s.AddRecord(rec); err == nil || err.Error() != refusal(1) {
		t.Fatalf("AddRecord on %v: %v; want %q", edge, err, refusal(1))
	}
	if _, err := s.AddSack(edge, 0x0101); err == nil || err.Error() != refusal(1) {
		t.Fatalf("AddSack on %v: %v; want %q", edge, err, refusal(1))
	}
	if !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("a refused sack add wrote the map")
	}
	if _, err := s.AddSack(inside, 0x0101); err != nil {
		t.Fatalf("AddSack on %v: %v", inside, err)
	}
	moved, err := s.MoveRecords([]ui.RecordRef{{Kind: "Sack", Index: 1}}, edge.Sub(inside))
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Refusals) != 1 || moved.Refusals[0] != refusal(1) {
		t.Fatalf("the group move of sack 1 onto %v refuses %q; want %q", edge, moved.Refusals, refusal(1))
	}
}
