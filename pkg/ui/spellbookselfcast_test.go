package ui

import "testing"

// sbDouble is a press that the double-click detector marked as the second of
// a pair, then its release.
func sbDouble(v *Viewer, x, y int) ([]order, bool) {
	v.command(appInput{PrimaryPressed: true, CursorX: x, CursorY: y})
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
	ords, ok := v.command(appInput{PrimaryPressed: true, PrimaryDouble: true, CursorX: x, CursorY: y})
	v.command(appInput{PrimaryReleased: true, CursorX: x, CursorY: y})
	return ords, ok
}

func sbSelfBook() []SpellEntry {
	return []SpellEntry{{ID: 1, Name: "Fire Arrow"}, {ID: 6, Name: "Heal", SelfCastable: true}}
}

func sbHolders() []MapEntity {
	const heal = uint32(1) << 6
	return []MapEntity{
		{ID: sbUnitID, Life: LifeAlive, SpellStateKnown: true, KnownSpells: heal},
		{ID: sbOtherID, Life: LifeAlive, SpellStateKnown: true, KnownSpells: heal},
		{ID: 7, Life: LifeAlive, SpellStateKnown: true},
	}
}

func TestADoubleClickOnASelfCastableRowCastsOnEachHolder(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbHolders())
	book := sbSelfBook()
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, book)
	x, y := sbEntryPoint(t, v, book, 1)
	ords, ok := sbDouble(v, x, y)
	if !ok || len(ords) != 1 || ords[0] != (order{kind: orderKindCast, entity: sbUnitID, victim: sbUnitID, spell: 6}) {
		t.Fatalf("one mage: ok=%v orders=%+v", ok, ords)
	}
	if v.selectedSpell != 0 || v.spellArmed {
		t.Fatalf("row not clear: selected=%d armed=%v", v.selectedSpell, v.spellArmed)
	}

	v.sel = selection{sbUnitID, sbOtherID, 7}
	v.SetSpellbook(sbUnitID, book)
	ords, ok = sbDouble(v, x, y)
	if !ok || len(ords) != 2 || ords[0].entity != sbUnitID || ords[0].victim != sbUnitID ||
		ords[1].entity != sbOtherID || ords[1].victim != sbOtherID {
		t.Fatalf("two mages and a warrior: ok=%v orders=%+v", ok, ords)
	}
}

func TestADoubleClickOnADamageRowKeepsSelectThenClear(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbHolders())
	book := sbSelfBook()
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, book)
	x, y := sbEntryPoint(t, v, book, 0)
	ords, ok := sbDouble(v, x, y)
	if ok || len(ords) != 0 || v.selectedSpell != 0 {
		t.Fatalf("damage row: ok=%v orders=%+v selected=%d", ok, ords, v.selectedSpell)
	}
}

func TestASingleClickOnASelfCastableRowStillSelectsAndArms(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbHolders())
	book := sbSelfBook()
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, book)
	x, y := sbEntryPoint(t, v, book, 1)
	sbClick(v, x, y)
	if v.selectedSpell != 6 || !v.spellArmed {
		t.Fatalf("selected=%d armed=%v", v.selectedSpell, v.spellArmed)
	}
}
