package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func freshRecords(t *testing.T) *mapedit.Editor {
	t.Helper()
	e, err := mapedit.NewMap(alm.DefaultNewMap())
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func addAuthoredStructure(t *testing.T, e *mapedit.Editor) int {
	t.Helper()
	rec, err := e.NewStructureRecord(8*256+128, 9*256+128, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	i, err := e.Place(mapedit.KindStructure, rec)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestNewStructureAndCloneReservePositiveScriptIdentities(t *testing.T) {
	e := freshRecords(t)
	b := e.Bytes()
	if i := addAuthoredStructure(t, e); i != 0 {
		t.Fatal(i)
	}
	rec, err := e.Record(mapedit.KindStructure, 0)
	want := make([]byte, 20)
	binary.LittleEndian.PutUint32(want, 8*256+128)
	binary.LittleEndian.PutUint32(want[4:], 9*256+128)
	binary.LittleEndian.PutUint32(want[8:], 5)
	binary.LittleEndian.PutUint16(want[12:], 1)
	binary.LittleEndian.PutUint16(want[14:], 1)
	binary.LittleEndian.PutUint16(want[18:], 1)
	if err != nil || !bytes.Equal(rec, want) {
		t.Fatalf("record %x, want %x (%v)", rec, want, err)
	}
	undoRedoExact(t, e, 1, b, e.Bytes())
	if err := e.AddArm(mapedit.ArmWinStructure, 90); err != nil {
		t.Fatal(err)
	}
	before := e.Bytes()
	i, err := e.CloneRecord(mapedit.KindStructure, 0, 12*256+128, 13*256+128)
	if err != nil || i != 1 {
		t.Fatal(i, err)
	}
	m, _ := e.Map()
	if m.Objects[1].Field12 != 91 || m.Objects[0].Field12 != 1 || m.Objects[1].Field0C != 1 {
		t.Fatal(m.Objects)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) || !e.Redo() {
		t.Fatal("clone history is not atomic")
	}
	if err := e.SetField(mapedit.KindStructure, 1, "identity", 32767); err != nil {
		t.Fatal(err)
	}
	before = e.Bytes()
	if _, err := e.CloneRecord(mapedit.KindStructure, 0, 0, 0); err == nil || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("identity exhaustion changed the map")
	}
}

func TestStructureIdentityRetargetIsAtomicAndRefusesAmbiguity(t *testing.T) {
	e := freshRecords(t)
	addAuthoredStructure(t, e)
	addAuthoredStructure(t, e)
	if err := e.AddArm(mapedit.ArmWinStructure, 1); err != nil {
		t.Fatal(err)
	}
	if err := e.AddArm(mapedit.ArmWinStructure, 50); err != nil {
		t.Fatal(err)
	}
	b := e.Bytes()
	for _, bad := range []int64{0, -1, 2, 50, 32768, 65535} {
		if err := e.SetField(mapedit.KindStructure, 0, "identity", bad); err == nil || !bytes.Equal(e.Bytes(), b) {
			t.Fatalf("identity %d was accepted or changed bytes", bad)
		}
	}
	if err := e.SetField(mapedit.KindStructure, 0, "identity", 17); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	m, _ := e.Map()
	sc, _ := m.Script()
	if m.Objects[0].Field12 != 17 || sc.Conditions[0].Value[0] != 17 || sc.Conditions[2].Value[0] != 50 {
		t.Fatal(m.Objects, sc.Conditions)
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) || !e.Redo() || !bytes.Equal(e.Bytes(), after) {
		t.Fatal("retarget history is not byte exact")
	}
	rec, _ := e.Record(mapedit.KindStructure, 0)
	if _, err := e.Place(mapedit.KindStructure, rec); err != nil {
		t.Fatal(err)
	}
	b = e.Bytes()
	if err := e.SetField(mapedit.KindStructure, 0, "identity", 18); err == nil || !bytes.Equal(b, e.Bytes()) {
		t.Fatal("renaming an ambiguous source identity was accepted")
	}
}

func TestInitialStockAndIndependentRecipesKeepLinksAndUndoTogether(t *testing.T) {
	e := freshRecords(t)
	u, err := e.NewUnitRecord(4*256+128, 5*256+128, 40, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlaceUnit(u); err != nil {
		t.Fatal(err)
	}
	if err := e.SetField(mapedit.KindUnit, 0, "group", 0xfedcba98); err != nil {
		t.Fatal(err)
	}
	before := e.Bytes()
	i, err := e.AddUnitStock(0)
	if err != nil || i != 0 {
		t.Fatal(i, err)
	}
	after := e.Bytes()
	m, _ := e.Map()
	l, _ := m.Loot()
	unit, _ := e.UnitRecord(0)
	if binary.LittleEndian.Uint32(unit[0x18:]) != 1 || m.Units[0].GroupID != 0xfedcba98 || l.Records[0].Owner != uint32(m.Units[0].UnitID) || len(l.Records[0].Elements) != 0 {
		t.Fatal(m.Units, l)
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), before) || !e.Redo() || !bytes.Equal(e.Bytes(), after) {
		t.Fatal("stock and unit link are not one edit")
	}
	if err := e.SetField(mapedit.KindLoot, i, "additem", 0x0101); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(7*256+128, 8*256+128, 0x0202)); err != nil {
		t.Fatal(err)
	}
	for loot := 0; loot < 2; loot++ {
		before = e.Bytes()
		if recipe, err := e.AddItemRecipe(loot, 1); err != nil || recipe != loot {
			t.Fatal(recipe, err)
		}
		after = e.Bytes()
		if !e.Undo() || !bytes.Equal(e.Bytes(), before) || !e.Redo() || !bytes.Equal(e.Bytes(), after) {
			t.Fatal("recipe and item link are not one edit")
		}
	}
	for _, set := range []mapedit.FieldSet{{Key: "spell", Value: 5}, {Key: "power", Value: -7}, {Key: "addeffect", Value: 12}, {Key: "low1", Value: 9}} {
		if err := e.SetField(mapedit.KindEnchantment, 0, set.Key, set.Value); err != nil {
			t.Fatal(err)
		}
	}
	saved, err := e.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	m, err = alm.Open(saved)
	if err != nil {
		t.Fatal(err)
	}
	l, _ = m.Loot()
	if l.Records[0].Elements[0].TileMarkerIndex != 1 || l.Records[1].Elements[0].TileMarkerIndex != 2 ||
		len(m.Enchantments) != 2 || m.Enchantments[0].SpellRaw != uint32(5)|uint32(uint16(65529))<<16 ||
		!reflect.DeepEqual(m.Enchantments[0].Elements, []alm.EnchantmentElement{{Kind: 12, Low: 9}}) || m.Enchantments[1].SpellRaw != 0 || len(m.Enchantments[1].Elements) != 0 {
		t.Fatal(l, m.Enchantments)
	}
	if err := e.SetField(mapedit.KindEnchantment, 0, "effect1", 0); err != nil {
		t.Fatal(err)
	}
	m, _ = e.Map()
	if len(m.Enchantments[0].Elements) != 0 || len(m.Enchantments[1].Elements) != 0 {
		t.Fatal(m.Enchantments)
	}
}

func TestRecordAuthoringRefusalsPreserveBytesAndHistory(t *testing.T) {
	e := freshRecords(t)
	u, _ := e.NewUnitRecord(128, 128, 40, 2, 2)
	_, _ = e.PlaceUnit(u)
	raw := e.Bytes()
	e, _ = mapedit.New(raw)
	for _, bad := range []int{-1, 1} {
		if _, err := e.AddUnitStock(bad); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
			t.Fatal("bad unit index changed bytes or history")
		}
	}
	_, _ = e.AddUnitStock(0)
	_ = e.SetField(mapedit.KindLoot, 0, "additem", 0x0101)
	raw = e.Bytes()
	e, _ = mapedit.New(raw)
	if _, err := e.AddUnitStock(0); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
		t.Fatal("duplicate stock changed bytes or history")
	}
	if err := e.DeleteUnit(0); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
		t.Fatal("deleting a stocked unit orphaned its stock")
	}
	for _, bad := range []int{-1, 0, 2, int(^uint(0) >> 1)} {
		if _, err := e.AddItemRecipe(0, bad); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
			t.Fatal("bad element index changed bytes or history")
		}
	}
	_, _ = e.AddItemRecipe(0, 1)
	raw = e.Bytes()
	e, _ = mapedit.New(raw)
	if _, err := e.AddItemRecipe(0, 1); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
		t.Fatal("duplicate recipe changed bytes or history")
	}
	if err := e.SetField(mapedit.KindEnchantment, 0, "addeffect", 256); err == nil || !bytes.Equal(e.Bytes(), raw) || e.CanUndo() {
		t.Fatal("overflow effect changed bytes or history")
	}
}

func TestRecipeAndOwnerFieldsPreserveUntargetedRawBytes(t *testing.T) {
	e := freshRecords(t)
	i := addAuthoredStructure(t, e)
	rec, _ := e.Record(mapedit.KindStructure, i)
	binary.LittleEndian.PutUint32(rec[14:], 0xabcd0001)
	_, _ = e.Place(mapedit.KindStructure, rec)
	before := e.Bytes()
	if err := e.SetField(mapedit.KindStructure, 1, "owner", 2); err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(before)
	_, p := recordAt(t, before, 4)
	binary.LittleEndian.PutUint16(want[p+20+14:], 2)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("owner edit changed the unused high word or other bytes")
	}
	recipe := make([]byte, 38)
	le := binary.LittleEndian
	le.PutUint32(recipe, 0xdecafbad)
	le.PutUint16(recipe[12:], 213)
	le.PutUint16(recipe[14:], 0x1234)
	le.PutUint16(recipe[16:], 0xabcd)
	le.PutUint32(recipe[18:], 5|99<<16)
	le.PutUint32(recipe[22:], 2)
	for j, v := range []uint16{12, 7, 8, 21, 9, 10} {
		le.PutUint16(recipe[26+2*j:], v)
	}
	if _, err := e.Place(mapedit.KindEnchantment, recipe); err != nil {
		t.Fatal(err)
	}
	before = e.Bytes()
	if err := e.SetField(mapedit.KindEnchantment, 0, "low2", 500); err != nil {
		t.Fatal(err)
	}
	want = bytes.Clone(before)
	_, p = recordAt(t, before, 9)
	le.PutUint16(want[p+4+26+6+2:], 500)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("recipe low-word edit changed a tag, head, spell, effect order or other word")
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), before) || !e.Redo() || !bytes.Equal(e.Bytes(), want) {
		t.Fatal("raw recipe field history is not byte exact")
	}
}

func TestCoordinateSpellEntryIsCarriedWithoutRecipeRemoval(t *testing.T) {
	e := freshRecords(t)
	rec := make([]byte, 26)
	binary.LittleEndian.PutUint32(rec[4:], 128)
	_, _ = e.Place(mapedit.KindEnchantment, rec)
	before := e.Bytes()
	e = newEditor(t, before)
	if err := e.Delete(mapedit.KindEnchantment, 0); err == nil || !bytes.Equal(before, e.Bytes()) || e.CanUndo() {
		t.Fatal("coordinate-bearing caster removal wrote bytes or history")
	}
}

func TestFirstAbsentSectionsKeepLoaderPrecedence(t *testing.T) {
	e := freshRecords(t)
	raw := e.Bytes()
	for _, tid := range []uint32{4, 8} {
		h, _ := recordAt(t, raw, tid)
		n := int(binary.LittleEndian.Uint32(raw[h+8:])) + 20
		raw = splice(raw, h, n, nil)
		addWord(raw, 12, -1)
	}
	e = newEditor(t, raw)
	addAuthoredStructure(t, e)
	u, _ := e.NewUnitRecord(10*256+128, 10*256+128, 40, 2, 2)
	_, _ = e.PlaceUnit(u)
	if _, err := e.AddUnitStock(0); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	h5, _ := recordAt(t, after, 5)
	h4, _ := recordAt(t, after, 4)
	h9, _ := recordAt(t, after, 9)
	h8, _ := recordAt(t, after, 8)
	h6, _ := recordAt(t, after, 6)
	if h5 >= h4 || h9 >= h8 || h8 >= h6 {
		t.Fatalf("loader precedence differs: 5@%d 4@%d 9@%d 8@%d 6@%d", h5, h4, h9, h8, h6)
	}
	undoRedoExact(t, e, 3, raw, after)
}

func TestRecipeDeletionKeepsEveryRemainingBinding(t *testing.T) {
	e := freshRecords(t)
	for i := 0; i < 3; i++ {
		_, _ = e.Place(mapedit.KindLoot, e.GroundSack(uint32(10+i)*256+128, 10*256+128, 0x0101))
		_, _ = e.AddItemRecipe(i, 1)
		_ = e.SetField(mapedit.KindEnchantment, i, "base", int64(i+11))
	}
	before := e.Bytes()
	if err := e.Delete(mapedit.KindEnchantment, 1); err == nil || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("referenced recipe was removed")
	}
	_ = e.SetField(mapedit.KindLoot, 1, "recipe1", 0)
	before = e.Bytes()
	if err := e.Delete(mapedit.KindEnchantment, 1); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	m, _ := e.Map()
	l, _ := m.Loot()
	if len(m.Enchantments) != 2 || m.Enchantments[0].B != 11 || m.Enchantments[1].B != 13 ||
		l.Records[0].Elements[0].TileMarkerIndex != 1 || l.Records[1].Elements[0].TileMarkerIndex != 0 || l.Records[2].Elements[0].TileMarkerIndex != 2 {
		t.Fatal("recipe removal retargeted a different recipe", m.Enchantments, l)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) || !e.Redo() || !bytes.Equal(after, e.Bytes()) {
		t.Fatal("recipe deletion and link renumbering are not one edit")
	}
}
