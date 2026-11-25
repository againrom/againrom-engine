package scriptprofile

import "testing"

// TestGroupRowCarriesItsFallback pins the group command row of each game: it
// is found by its shape and states the command a node falls back to.
func TestGroupRowCarriesItsFallback(t *testing.T) {
	for _, ops := range []Ops{OpsFirst, OpsSecond} {
		row, ok := GroupRow(ops)
		if !ok || row.ID != 6 || row.Fallback != 3 {
			t.Fatalf("ops %d: group row %d fallback %d found %t", ops, row.ID, row.Fallback, ok)
		}
		if v, ok := row.Variant(row.Fallback); !ok || len(v.Signature) != 0 {
			t.Errorf("ops %d: fallback %d is not a variant without parameters", ops, row.Fallback)
		}
		for _, o := range OpRows(ops, false) {
			if o.Fallback != 0 && !o.Grouped {
				t.Errorf("ops %d: row %d has a fallback and no group", ops, o.ID)
			}
		}
	}
}
