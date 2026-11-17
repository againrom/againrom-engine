package game

import "testing"

// TestEditorUnitFieldValueWidthFollowsTheTableBound pins each card number's
// widest value in characters, sign included, from the field table's bound:
// hp and mana reach -32768, hpmax reaches 32767.
func TestEditorUnitFieldValueWidthFollowsTheTableBound(t *testing.T) {
	_, e := unitWindowFixture(t, "en", false)
	fields := e.ObjectFields()
	want := map[string]int{"hp": 6, "hpmax": 5, "mana": 6}
	for key, w := range want {
		if got := fields[fieldIndex(t, fields, key)].ValueWidth; got != w {
			t.Errorf("%s: ValueWidth %d, want %d", key, got, w)
		}
	}
}
