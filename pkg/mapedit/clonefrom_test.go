package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/mapedit"
)

// A unit cloned from another map brings its stock and the stock's recipe, a
// ground sack brings its item's recipe, the source map is left as it was,
// and each clone is one undo step of the target.
func TestCloneFromAnotherMapCopiesTheRecordsItsItemsLink(t *testing.T) {
	src := freshRecords(t)
	u, _ := src.NewUnitRecord(10*256+128, 11*256+128, 40, 2, 2)
	_, _ = src.PlaceUnit(u)
	_, _ = src.AddUnitStock(0)
	_ = src.SetField(mapedit.KindLoot, 0, "additem", 0x0101)
	_, _ = src.AddItemRecipe(0, 1)
	_ = src.SetField(mapedit.KindEnchantment, 0, "addeffect", 12)
	sack, err := src.Place(mapedit.KindLoot, src.GroundSack(14*256+128, 15*256+128, 0x0202))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.AddItemRecipe(sack, 1); err != nil {
		t.Fatal(err)
	}
	_ = src.SetField(mapedit.KindEnchantment, 1, "addeffect", 7)
	source := src.Bytes()
	sm, _ := src.Map()

	dst := freshRecords(t)
	empty := dst.Bytes()
	i, err := dst.CloneFrom(src, mapedit.KindUnit, 0, 20*256+128, 21*256+128)
	if err != nil || i != 0 {
		t.Fatal(i, err)
	}
	afterUnit := dst.Bytes()
	j, err := dst.CloneFrom(src, mapedit.KindLoot, sack, 22*256+128, 23*256+128)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(src.Bytes(), source) {
		t.Fatal("a clone into another map wrote the source")
	}
	m, _ := dst.Map()
	l, _ := m.Loot()
	if len(m.Units) != 1 || m.Units[0].X != 20*256+128 || m.Units[0].ClassID != sm.Units[0].ClassID {
		t.Fatalf("units %+v", m.Units)
	}
	if len(l.Records) != 2 || l.Records[0].Owner != uint32(m.Units[0].UnitID) || l.Records[0].Elements[0].ItemCode() != 0x0101 ||
		!l.Records[j].Ground() || l.Records[j].Elements[0].ItemCode() != 0x0202 {
		t.Fatalf("loot %+v", l.Records)
	}
	if len(m.Enchantments) != 2 || m.Enchantments[0].Elements[0].Kind != sm.Enchantments[0].Elements[0].Kind ||
		m.Enchantments[1].Elements[0].Kind != sm.Enchantments[1].Elements[0].Kind ||
		l.Records[0].Elements[0].TileMarkerIndex != 1 || l.Records[j].Elements[0].TileMarkerIndex != 2 {
		t.Fatalf("recipes %+v, links %d and %d", m.Enchantments, l.Records[0].Elements[0].TileMarkerIndex, l.Records[j].Elements[0].TileMarkerIndex)
	}
	if !dst.Undo() || !bytes.Equal(dst.Bytes(), afterUnit) || !dst.Undo() || !bytes.Equal(dst.Bytes(), empty) {
		t.Fatal("each clone from another map is not one byte-exact undo step")
	}
	// Control: within one map a sack's clone shares the recipe, so the recipe
	// count stays.
	if _, err := src.CloneFrom(src, mapedit.KindLoot, sack, 24*256+128, 25*256+128); err != nil {
		t.Fatal(err)
	}
	if m, _ := src.Map(); len(m.Enchantments) != 2 {
		t.Fatalf("a clone within one map copied the recipe: %d recipes", len(m.Enchantments))
	}
}

// A clone from a copy of the map shares a ground sack's recipe the map still
// holds, byte for byte what a clone within the map writes; a recipe the map
// has changed since the copy is copied.
func TestCloneFromCopySharesTheRecipeTheMapStillHolds(t *testing.T) {
	e := freshRecords(t)
	sack, err := e.Place(mapedit.KindLoot, e.GroundSack(14*256+128, 15*256+128, 0x0202))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddItemRecipe(sack, 1); err != nil {
		t.Fatal(err)
	}
	_ = e.SetField(mapedit.KindEnchantment, 0, "addeffect", 7)
	before := e.Bytes()
	snapshot, _ := mapedit.New(before)
	clone, _ := mapedit.New(before)
	if _, err := clone.CloneRecord(mapedit.KindLoot, sack, 24*256+128, 25*256+128); err != nil {
		t.Fatal(err)
	}
	if _, err := e.CloneFromCopy(snapshot, mapedit.KindLoot, sack, 24*256+128, 25*256+128); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Bytes(), clone.Bytes()) {
		t.Fatal("a clone from a copy of the map differs from a clone within it")
	}
	// Control: CloneFrom the same copy copies the recipe.
	other, _ := mapedit.New(before)
	if _, err := other.CloneFrom(snapshot, mapedit.KindLoot, sack, 24*256+128, 25*256+128); err != nil {
		t.Fatal(err)
	}
	if m, _ := other.Map(); len(m.Enchantments) != 2 {
		t.Fatalf("CloneFrom a copy left %d recipes, want 2", len(m.Enchantments))
	}
	// A recipe the map changed after the copy is not the copy's: it is copied.
	changed, _ := mapedit.New(before)
	_ = changed.SetField(mapedit.KindEnchantment, 0, "addeffect", 12)
	if _, err := changed.CloneFromCopy(snapshot, mapedit.KindLoot, sack, 24*256+128, 25*256+128); err != nil {
		t.Fatal(err)
	}
	if m, _ := changed.Map(); len(m.Enchantments) != 2 {
		t.Fatalf("a clone over a changed recipe left %d recipes, want 2", len(m.Enchantments))
	}
}
