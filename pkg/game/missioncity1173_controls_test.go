package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func nativeMissionControl1173() (mapload.PartyMember, sim.Entity, sav.CityUnitData) {
	m := mapload.PartyMember{Name: "Native controller", StartingHero: true, FigureDir: "fmage", FigureFace: 1, Mage: true,
		Hero:    data.NewHero(data.Spread{Body: 30, Reaction: 31, Mind: 32, Spirit: 33}, 1),
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
		Carry:   &mapload.Carry{SkillXP: [data.SkillSlots]int32{0, 1801, 411, 0, 0, 0}}}
	d := m.Hero.RecomputeWithSkillXP(m.Profile, data.Loadout{}, m.Carry.SkillXP)
	e := sim.Entity{ID: 1, HP: d.HealthMax - 2, MaxHP: d.HealthMax - 2, Mana: d.ManaMax + 3, MaxMana: d.ManaMax + 3,
		ToHit: d.Combat.ToHit, Defence: d.Combat.Defence, Absorption: d.Combat.Absorption, Skill: d.Skill, SkillXP: d.SkillXP,
		HealthRegenPeriod: 77, ManaRegenPeriod: 41, HealthHundredths: 37, ManaHundredths: 83,
		Reach: 5, AttackCharge: 19, AttackRelax: 23, SecondBase: 17, SecondSpread: 9}
	u := nativeCityUnitData(20, 10, m, m, nil, [sav.UnitStatWords]uint16{}, [sav.CharacterSkillSlots]uint16{})
	u.SpellbookFlag = 1
	return m, e, u
}

func TestNativeMission1173MaintainedPoolsAndTypedLoad(t *testing.T) {
	m, e, unit := nativeMissionControl1173()
	load := &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, OwnWeight: -9, ContainerPresent: true, InsertIndex: 37, Accumulator: 20},
		Load: -7, Capacity: 153, Speed: 21,
		Movement:         sim.HumanMovement{Present: true, RawSpeed: 11, NativeSpeed: 21, Load: -7, Capacity: 153},
		HealthHundredths: e.HealthHundredths, ManaHundredths: e.ManaHundredths}
	if err := load.Validate(); err != nil {
		t.Fatal(err)
	}
	m.Carry.LiveLoad = load
	e.ActorLoad, e.Load, e.Capacity, e.Speed, e.HumanMovement = load.Inventory, load.Load, load.Capacity, load.Speed, load.Movement
	beforeMember, beforeActor := mapload.CloneParty([]mapload.PartyMember{m})[0], e
	if _, err := nativeCityHumanState(m, nil, unit, false); err == nil {
		t.Fatal("legacy constructor erased current load continuity")
	}
	if err := nativeMissionHuman(&unit, m, nil, e); err != nil {
		t.Fatal(err)
	}
	word := func(n int) uint16 { return binary.LittleEndian.Uint16(unit.Scalar2[2*n:]) }
	for index, want := range map[int]uint16{4: 11, 5: 65527, 6: 65529, 7: 153, 8: uint16(e.MaxHP), 9: uint16(e.MaxHP), 10: 77, 11: uint16(e.MaxMana), 12: uint16(e.MaxMana), 13: 41} {
		if got := word(index); got != want {
			t.Fatalf("current Human word%d=%d want%d", index, got, want)
		}
	}
	if !bytes.Equal(unit.RawA6[17:19], []byte{17, 9}) || unit.Scalar2[28] != 37 || unit.Scalar2[29] != 83 || unit.Scalar2[34] != 5 || unit.Scalar2[39] != 19 || unit.Scalar2[40] != 23 {
		t.Fatal("current secondary damage, residues or equipment timing changed")
	}
	// The deliberate pool difference is maintained state, not a fabricated
	// permanent effect. Both future pool modifiers remain the genuine zero.
	if binary.LittleEndian.Uint16(unit.RawD4[8:10]) != 0 || binary.LittleEndian.Uint16(unit.RawD4[12:14]) != 0 {
		t.Fatal("pool difference became a modifier")
	}
	if !reflect.DeepEqual(m, beforeMember) || e != beforeActor {
		t.Fatal("native adapter mutated its inputs")
	}
	for _, change := range []func(*sim.Entity){func(e *sim.Entity) { e.HP-- }, func(e *sim.Entity) { e.ToHit++ }, func(e *sim.Entity) { e.ActorLoad.Source.Class = 2 }, func(e *sim.Entity) { e.HP, e.MaxHP = 65536, 65536 }} {
		bad := e
		change(&bad)
		_, _, target := nativeMissionControl1173()
		if err := nativeMissionHuman(&target, m, nil, bad); err == nil {
			t.Fatal("unqualified or lossy Human accepted")
		}
	}
}

func TestNativeMission1173CountedPackAndCurrentBook(t *testing.T) {
	m := mapload.PartyMember{Name: "Book owner", Carry: &mapload.Carry{}, KnownSpells: 1<<1 | 1<<3, Book: sim.Spellbook{State: sim.BookPresent}}
	m.Book.Slots[0] = sim.BookSpell{Range: 57, Defensive: 0x80, ManaCost: 0xfffd}
	m.Book.Slots[2] = sim.BookSpell{Range: 91, Defensive: 3, ManaCost: 1234}
	m.Carry.LiveLoad = &sim.ActorLoadSnapshot{Inventory: sim.ActorLoad{Present: true, OwnWeight: -9, ContainerPresent: true, InsertIndex: 37, Accumulator: 20}, Load: 1, Capacity: 123}
	item := sim.ItemInstance{Code: 0x0e01, Kind: 3, Price: 71, Weight: 2, WeightPresent: true}
	ordered := []sim.ItemStack{sim.StackItem(item, 3), sim.StackItem(item, 7)}
	var unit sav.CityUnitData
	seq := 0
	objects, err := nativeMissionItems(nil, &unit, m, nativeItemTestTable(), 123, &seq, ordered)
	if err != nil {
		t.Fatal(err)
	}
	if unit.ContainerFlag != 1 || unit.ContainerTails != [2]uint32{37, 20} || len(unit.Container) != 2 || unit.Container[0] == unit.Container[1] {
		t.Fatal("pack boundaries or stored insertion/weight bookkeeping lost", unit.Container, unit.ContainerTails)
	}
	for i, want := range []uint16{3, 7} {
		p := objects[unit.Container[i]-1].Item
		if binary.LittleEndian.Uint16(p.Fields[2:4]) != want || binary.LittleEndian.Uint16(p.Fields[:2]) != 0x0e01 || binary.LittleEndian.Uint32(p.Token[25:29]) != 71 {
			t.Fatal("ordered stack operands changed", i)
		}
	}
	if unit.SpellbookFlag != 1 || unit.SpellbookCount != 4 || len(unit.Spells) != 3 || unit.Spells[1] != 0 {
		t.Fatal("positional book collapsed", unit.Spells)
	}
	for i, want := range map[int][]byte{0: {1, 57, 0x80, 0xfd, 0xff}, 2: {3, 91, 3, 0xd2, 4}} {
		if got := objects[unit.Spells[i]-1].Spell.Fields[:5]; !bytes.Equal(got, want) {
			t.Fatalf("book slot%d=%x want%x", i, got, want)
		}
	}
	for name, change := range map[string]func(*mapload.PartyMember, []sim.ItemStack){
		"count_overflow":   func(_ *mapload.PartyMember, p []sim.ItemStack) { p[0].Count = 65536 },
		"zero_count":       func(_ *mapload.PartyMember, p []sim.ItemStack) { p[0].Count = 0 },
		"changed_weight":   func(m *mapload.PartyMember, _ []sim.ItemStack) { m.Carry.LiveLoad.Inventory.Accumulator++ },
		"absent_container": func(m *mapload.PartyMember, _ []sim.ItemStack) { m.Carry.LiveLoad.Inventory.ContainerPresent = false },
		"source_human":     func(m *mapload.PartyMember, _ []sim.ItemStack) { m.Carry.LiveLoad.Inventory.Source.Class = 2 },
		"invalid_book":     func(m *mapload.PartyMember, _ []sim.ItemStack) { m.Book.Slots[1].ManaCost = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			bad := mapload.CloneParty([]mapload.PartyMember{m})[0]
			pack := append([]sim.ItemStack(nil), ordered...)
			change(&bad, pack)
			var target sav.CityUnitData
			n := 0
			if _, err := nativeMissionItems(nil, &target, bad, nativeItemTestTable(), 123, &n, pack); err == nil {
				t.Fatal("unrepresentable item/source state accepted")
			}
		})
	}
	// A current ObjectID names the in-memory graph, not the next SAV graph.
	// The single producer accepts it and mints a fresh file-local identity.
	retained := append([]sim.ItemStack(nil), ordered...)
	retained[0].ObjectID = 0x60001234
	var retainedUnit sav.CityUnitData
	retainedSeq := 0
	retainedObjects, err := nativeMissionItems(nil, &retainedUnit, m, nativeItemTestTable(), 123, &retainedSeq, retained)
	if err != nil {
		t.Fatalf("current item identity refused: %v", err)
	}
	gotIdentity := binary.LittleEndian.Uint32(retainedObjects[retainedUnit.Container[0]-1].Item.Token[29:33])
	if gotIdentity == 0 || gotIdentity == uint32(retained[0].ObjectID) {
		t.Fatalf("constructed SAV identity = %#x, want a fresh nonzero file-local identity", gotIdentity)
	}
}
