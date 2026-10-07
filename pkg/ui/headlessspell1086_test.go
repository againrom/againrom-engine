package ui

import "testing"

func TestHeadlessSpellPointUsesVisibleBookAndPointerDispatch(t *testing.T) {
	a := spellbookTestApp(t)
	mapAtScaleOne(a)
	v := a.flow.viewer
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, sbBook())
	x, y, err := a.HeadlessSpellPoint(6)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	if v.selectedSpell != 6 {
		t.Fatalf("selected spell=%d", v.selectedSpell)
	}
	if _, _, err := a.HeadlessSpellPoint(31); err == nil {
		t.Fatal("absent spell has a hit point")
	}
	if err := a.HeadlessKey("book"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.HeadlessSpellPoint(6); err == nil {
		t.Fatal("hidden spell has a hit point")
	}
}
