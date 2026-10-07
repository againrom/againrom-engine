package sim

import "testing"

func TestCurrentCorpseLoadClearsRemovedWeaponCache(t *testing.T) {
	w := deadWorld(t)
	w.equipment[0][0] = ItemInstance{Code: 0x1234, Effects: []ItemEffect{{Kind: 41, Operand: 7 | 15<<16}}}
	syncWeaponItem(&w.entities[0], w.equipment[0][0])
	if w.entities[0].WeaponSpellSource != WeaponSpellItem {
		t.Fatal("fixture has no item spell")
	}
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 3, -40)}); err != nil {
		t.Fatal(err)
	}
	if w.entities[0].WeaponSpellSource != WeaponSpellNone || !w.equipment[0][0].Empty() {
		t.Fatal("removed weapon retained its derived spell")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal("cold corpse cannot SAVE", err)
	}
}
