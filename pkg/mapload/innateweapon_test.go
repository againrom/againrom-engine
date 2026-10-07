package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const iwKey = 0x53
const iwCarrySlot = 0xf

func iwWeaponRow(carry int32) []int32 {
	p := make([]int32, 17)
	for i := range p {
		p[i] = -1
	}
	p[slotWeaponType] = data.SkillBlade
	p[slotWeaponMin], p[slotWeaponMax] = 5, 9
	p[slotWeaponToHit], p[slotWeaponDefence] = 7, 2
	p[slotWeaponCharge], p[slotWeaponRelax] = 15, 9
	p[slotWeaponRange], p[iwCarrySlot] = 6, carry
	return p
}

func iwWeapons() defCollection {
	silent := weaponRow(data.SkillBlade, 5, 9, 7, 2, 15, 9)
	silent[slotWeaponRange] = 6
	return defCollection{
		{},
		{name: "Carried", params: iwWeaponRow(1)},
		{name: "Innate", params: iwWeaponRow(0)},
		{name: "Empty", params: iwWeaponRow(-1)},
		{name: "Silent", params: silent},
	}
}

func iwLoad(t *testing.T, weapon string) (*sim.World, *mapload.Table) {
	t.Helper()
	row := equipUnitRow(iwKey, 1, 3, 9, 10, 4, 99, 77, []string{weapon})
	tbl := equipTable(row, iwWeapons(), identityScale(), identityScale())
	w, err := mapload.FromALMWith(srcMap(iwKey), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	return w, tbl
}

func iwCode(t *testing.T, name string, tbl *mapload.Table) uint16 {
	t.Helper()
	w, err := data.ResolveWeapon(name, tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	return uint16(w.Code)
}

func iwTerminalDeath(w *sim.World) {
	sim.Step(w, []sim.Command{{Kind: sim.KindTerminalKill, Entity: 0}})
	for {
		entities := w.Entities()
		if len(entities) == 0 || entities[0].Dwell == 0 {
			return
		}
		sim.Step(w, nil)
	}
}

func TestAnInnateWeaponRemainsHeldWhileAlive(t *testing.T) {
	w, tbl := iwLoad(t, "Innate")
	want := iwCode(t, "Innate", tbl)
	items, ok := w.EquippedItems(0)
	if !ok || items[0].Code != want || !items[0].SourceEquipment.Definition.Present || items[0].SourceEquipment.Definition.Suitable != 0 {
		t.Fatalf("live innate weapon = %+v, want code %#x and Suitable=0", items[0], want)
	}
	stock := w.Stock()
	if len(stock) != 1 || stock[0].Equipped[0] != want {
		t.Fatalf("live stock = %+v, want held weapon %#x", stock, want)
	}
}

func TestACarriedWeaponStillReachesTheEquipmentSet(t *testing.T) {
	w, tbl := iwLoad(t, "Carried")
	item, _ := w.EquippedItems(0)
	if item[0].Code != iwCode(t, "Carried", tbl) || item[0].SourceEquipment.Definition.Suitable != 1 {
		t.Fatal("ordinary weapon lost its held slot or suitability", item[0])
	}
}

func TestAWeaponRowTooShortToNameTheColumnIsStillCarried(t *testing.T) {
	w, tbl := iwLoad(t, "Silent")
	item, _ := w.EquippedItems(0)
	if item[0].Code != iwCode(t, "Silent", tbl) || !item[0].SourceEquipment.Definition.Present || item[0].SourceEquipment.Definition.Suitable != 1 {
		t.Fatal("silent suitability row changed its ordinary held item", item[0])
	}
}

func TestACustomCreatureRowConstructsItsAttackAndBook(t *testing.T) {
	row := equipUnitRow(iwKey, 4, 3, 9, 10, 4, 99, 77, []string{"Innate"})
	for len(row.params) < 54 {
		row.params = append(row.params, -1)
	}
	row.params[slotDamageArm] = armAlwaysHits
	row.params[48], row.params[49] = 1, 25
	tbl := equipTable(row, iwWeapons(), identityScale(), identityScale())
	tbl.Spells = defCollection{{}, {name: "Custom Spell", params: spellRow(5, 1, 1, 6, 4, 8, 0)}}
	m := srcMap(iwKey)
	m.Units[0].ClassSubID = 4
	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if !e.AlwaysHits {
		t.Fatal("custom AlwaysHits arm was lost")
	}
	bound, _, err := mapload.ConstructActorBasis(e, mapload.PartyMember{}, &m.Units[0], tbl, 1, 1, nil, 0)
	if err != nil || bound.SourceBinding.ClassFlags&0x12 != 0x12 {
		t.Fatalf("custom class flags=%#x, error=%v", bound.SourceBinding.ClassFlags, err)
	}
	if e.Skill[data.SkillGeneral] != 10 {
		t.Fatalf("custom General skill=%d, want authored ToHit 10", e.Skill[data.SkillGeneral])
	}
	for i := 1; i < data.SkillSlots; i++ {
		if e.Skill[i] != 30 {
			t.Fatalf("custom top-tier skill %d=%d, want 30", i, e.Skill[i])
		}
	}
	if e.KnownSpells != 2 || !e.Book.WirePresent(e.KnownSpells) || e.Book.Slots[0].Range != 6 || e.Book.Slots[0].ManaCost != 5 {
		t.Fatalf("custom spellbook: mask %#x, book %+v", e.KnownSpells, e.Book)
	}
	item, _ := w.EquippedItems(0)
	if item[0].SourceEquipment.Definition.Suitable != 0 || item[0].Code != iwCode(t, "Innate", tbl) {
		t.Fatalf("custom innate weapon: %+v", item[0])
	}

	ordinary, _ := iwLoad(t, "Carried")
	control := ordinary.Entities()[0]
	if control.AlwaysHits || control.KnownSpells != 0 || control.Book.WirePresent(0) || control.SourceBinding.ClassFlags&0x12 != 0 {
		t.Fatalf("ordinary row gained attack or book state: %+v", control)
	}
}

func TestAnUnrepresentedCustomCreatureSpellDoesNotRefuseTheWorld(t *testing.T) {
	row := equipUnitRow(iwKey, 1, 3, 9, 10, 4, 99, 77, []string{"Innate"})
	for len(row.params) < 50 {
		row.params = append(row.params, -1)
	}
	row.params[48] = 29
	tbl := equipTable(row, iwWeapons(), identityScale(), identityScale())
	w, err := mapload.FromALMWith(srcMap(iwKey), tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if !e.Book.WirePresent(e.KnownSpells) || e.KnownSpells != 0 {
		t.Fatalf("unrepresented custom book: mask %#x, book %+v", e.KnownSpells, e.Book)
	}
}

func TestAnEmptyCarryCellIsNotAZero(t *testing.T) {
	w, tbl := iwLoad(t, "Empty")
	item, _ := w.EquippedItems(0)
	if item[0].Code != iwCode(t, "Empty", tbl) || item[0].SourceEquipment.Definition.Suitable != -1 {
		t.Fatal("negative suitability is a nonzero, carried weapon", item[0])
	}
}

func TestAnInnateWeaponStillArmsTheCreatureItBelongsTo(t *testing.T) {
	innate, _ := iwLoad(t, "Innate")
	carried, _ := iwLoad(t, "Carried")
	got, want := innate.Entities()[0], carried.Entities()[0]
	if got != want || got.Reach != 6 || got.DamageBase != 8 || got.DamageSpread != 10 || got.AttackCharge != 15 || got.AttackRelax != 9 {
		t.Fatalf("suitability altered live combat: innate %+v, carried %+v", got, want)
	}
}

func TestACreatureArmedOnlyWithAnInnateWeaponLeavesNoSack(t *testing.T) {
	innate, tbl := iwLoad(t, "Innate")
	want := iwCode(t, "Innate", tbl)
	iwTerminalDeath(innate)
	if got := innate.Sacks(); len(got) != 0 {
		t.Fatalf("innate weapon became loot: %+v", got)
	}
	if held, _ := innate.Equipped(0); held[0] != want {
		t.Fatalf("innate weapon left dead actor: %#x, want %#x", held[0], want)
	}

	carried, tbl := iwLoad(t, "Carried")
	iwTerminalDeath(carried)
	got := carried.Sacks()
	if len(got) != 1 || len(got[0].Items) != 1 || got[0].Items[0] != iwCode(t, "Carried", tbl) {
		t.Fatalf("ordinary held weapon did not drop: %+v", got)
	}

	silent, tbl := iwLoad(t, "Silent")
	iwTerminalDeath(silent)
	got = silent.Sacks()
	if len(got) != 1 || len(got[0].Items) != 1 || got[0].Items[0] != iwCode(t, "Silent", tbl) {
		t.Fatalf("silent suitability suppressed a drop: %+v", got)
	}
}
