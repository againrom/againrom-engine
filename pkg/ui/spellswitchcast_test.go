package ui

import "testing"

// offBook is castKindBook with the unit-target spell's cell blank, as a book
// shows a spell the mods turned off.
func offBook(off bool) []SpellEntry {
	book := castKindBook()
	if off {
		book[0] = SpellEntry{}
	}
	return book
}

func castOrdersOnUnit(t *testing.T, v *Viewer) []order {
	t.Helper()
	x, y := cellPoint(v, castCells.other[0], castCells.other[1])
	ords, _ := tapAt(v, x, y)
	var casts []order
	for _, o := range ords {
		if o.kind == orderKindCast {
			casts = append(casts, o)
		}
	}
	return casts
}

// A spell bound to a quick slot before the mod turned it off is neither
// selected nor armed by the slot key and C, and orders no cast; the same input
// with the spell on orders it.
func TestAnOffSpellInAQuickSlotIsNotCast(t *testing.T) {
	for _, off := range []bool{false, true} {
		_, v := panelCasterApp(t)
		v.SetSpellbook(panelCasterID, offBook(off))
		var slots ShortcutSlots
		v.SetQuickSpells(&slots)
		slots[0] = Shortcut{Spell: castTarget}
		v.quickSpell(0, false, -1, -1)
		v.castKey()
		casts := castOrdersOnUnit(t, v)
		if off && (len(casts) != 0 || v.selectedSpell != 0 || v.spellArmed) {
			t.Errorf("off: selected %d armed %v orders %+v", v.selectedSpell, v.spellArmed, casts)
		}
		if !off && (len(casts) != 1 || casts[0].spell != castTarget) {
			t.Errorf("on: orders %+v, want one cast of the spell", casts)
		}
	}
}

// A current spell saved before the mod turned it off restores without a cast.
func TestAnOffSavedSpellIsNotCast(t *testing.T) {
	for _, off := range []bool{false, true} {
		_, v := panelCasterApp(t)
		v.SetSpellbook(panelCasterID, offBook(off))
		s := v.SaveApplication()
		s.PressedSpell = castTarget
		if err := v.RestoreSaveApplication(s); err != nil {
			t.Fatal(err)
		}
		v.castKey()
		casts := castOrdersOnUnit(t, v)
		if off && (len(casts) != 0 || v.spellArmed) {
			t.Errorf("off: armed %v orders %+v", v.spellArmed, casts)
		}
		if !off && (len(casts) != 1 || casts[0].spell != castTarget) {
			t.Errorf("on: orders %+v, want one cast of the spell", casts)
		}
	}
}
