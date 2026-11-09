package game

import "testing"

// TestCurrentWorldNamesAnUnplacedCreaturesFaceOneRow pins DIV-4748 at the SAV producer.
func TestCurrentWorldNamesAnUnplacedCreaturesFaceOneRow(t *testing.T) {
	f, s, w := partialCurrentGraph(t)
	rows := f.Table.Units.(dbCollection)
	other, wanted := rows[1], rows[1]
	other.params, wanted.params = append([]int32(nil), rows[1].params...), append([]int32(nil), rows[1].params...)
	other.params[29], other.params[30] = 1, 2
	wanted.params[29], wanted.params[30] = 1, 1
	f.Table.Units = dbCollection{rows[0], other, wanted}

	s.SavedDocument = nil
	s.CurrentPartyIDs, s.CurrentRoster = nil, nil
	s.Residue.VisualIdentities, s.Residue.VisualNext = nil, 0
	current, err := f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal(err)
	}
	units := 0
	for i := range current.Document.Objects {
		r := &current.Document.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		units++
		row, err := savedStructureValue(r, "T0C")
		if err != nil || row != 2 {
			t.Fatalf("Unit record %d names row %d (%v), want the face-1 row 2", i+1, row, err)
		}
	}
	if units == 0 {
		t.Fatal("the fixture wrote no Unit record")
	}
}
