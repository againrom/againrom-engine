package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func initialBasisTable() *mapload.Table {
	return &mapload.Table{
		Units: defCollection{{}, {
			name: "initial-unit",
			params: defRow(map[int]int32{
				0: 37, 1: 31, 2: 29, 3: 19, 4: 99,
				11: 9, 12: 21, 14: 17, 15: 4, 29: 200, 30: 4,
			}),
		}},
		Humans: defCollection{{}, {
			name: "initial-human",
			params: defRow(map[int]int32{
				0: 43, 1: 31, 2: 29, 3: 19, 4: 100, 5: 0,
				10: 61, 11: 11, 12: 22, 13: 33, 14: 44, 15: 55, 16: 7, 17: 1,
			}),
			strings: []string{"Melee{body=5,skillblade=7,skillaxe=9}"},
		}},
		Shapes: identityScale(), Materials: identityScale(), Weapons: equipWeapons(),
	}
}

func initialBasisMap(class int16, face uint16) *alm.Map {
	return &alm.Map{
		Width: 40, Height: 40,
		Units: []alm.Unit{{X: 0x1480, Y: 0x1480, ClassID: class, ClassSubID: face}},
	}
}

func assertInitialBase(t *testing.T, e sim.Entity, prefix [22]byte, body uint16, modifier [64]byte) {
	t.Helper()
	b := e.NativeBasis
	if !b.BasePresent {
		t.Error("constructor Base is absent")
	}
	if b.BaseKnown != 0x003fffff {
		t.Errorf("BaseKnown = %08x, want 003fffff", b.BaseKnown)
	}
	for i, want := range prefix {
		if b.Base[i] != want {
			t.Errorf("Base[%d] = %02x, want %02x", i, b.Base[i], want)
		}
		if !b.BaseByteKnown(i) {
			t.Errorf("Base[%d] is unknown", i)
		}
	}
	for _, i := range []int{22, 23} {
		if b.BaseByteKnown(i) {
			t.Errorf("unestablished constructor tail Base[%d] became known", i)
		}
	}
	if !b.BodyPresent || !b.BodyKnown || b.Body != body {
		t.Errorf("Body = present:%v known:%v value:%d, want true/true/%d", b.BodyPresent, b.BodyKnown, b.Body, body)
	}
	assertInitialModifier(t, e, modifier)
}

func TestResolvedNativeUnitInitialBaseUsesConstructorZeroPrefix(t *testing.T) {
	w, err := mapload.FromALMWith(initialBasisMap(200, 4), initialBasisTable(), mapload.DifficultyHard)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if e.Humanoid || e.ToHit == 0 || e.DamageBase == 0 || e.Skill != ([6]int32{17, 30, 30, 30, 30, 30}) {
		t.Fatalf("fixture did not resolve the nonzero Unit inputs: %+v", e)
	}
	// UNIT-CTOR-004 -> SAV-1032 -> SAV-HUMGAPS-449 establishes this
	// independent constructor prefix, including the nonzero live skill case.
	assertInitialBase(t, e, [22]byte{}, 37, [64]byte{})
}

func TestResolvedNativeHumanInitialBaseKeepsPreEquipmentSkillBytes(t *testing.T) {
	w, err := mapload.FromALMWith(initialBasisMap(7, 1), initialBasisTable(), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e := w.Entities()[0]
	if !e.Humanoid || e.Skill[0] != 61 || e.Skill[1] != 18 || e.Skill[2] != 31 {
		t.Fatalf("fixture did not apply the authored Human equipment bonuses: %+v", e)
	}
	// Bytes 2/3 stay zero even though live General is 61. The five
	// maintained skill words use row inputs 11/22/33/44/55 before +7/+9.
	assertInitialBase(t, e, [22]byte{
		0, 0, 0, 0, 11, 0, 22, 0, 33, 0, 44, 0, 55, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}, 48, [64]byte{0: 5, 18: 7, 22: 7, 24: 9, 32: 5, 33: 4, 42: 2})
}

func TestStartedNativePartyInitialBaseKeepsPreEquipmentSkillBytes(t *testing.T) {
	tbl := initialBasisTable()
	weapon, err := data.ResolveWeapon("Melee", tbl.Shapes, tbl.Materials, tbl.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	p := mapload.PartyMember{
		Class:   3,
		Hero:    data.Hero{Body: 39, Reaction: 31, Mind: 29, Spirit: 19, Skill: [6]int32{79, 13, 23, 31, 41, 53}},
		Profile: data.Profile{Fighter: true, HealthColumn: true},
		Weapon:  &weapon,
	}
	p.WornItems[0] = sim.ItemInstance{
		Code: 0x0101, Kind: 2,
		Effects: []sim.ItemEffect{{Kind: 2, Operand: 5}, {Kind: 27, Operand: 7}, {Kind: 28, Operand: 9}},
	}
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	for _, scripted := range []bool{false, true} {
		t.Run(map[bool]string{false: "start", true: "scripted-rebuild"}[scripted], func(t *testing.T) {
			var w *sim.World
			var st mapload.Start
			var err error
			if scripted {
				w, st, err = mapload.StartMissionScripted(m, tbl, mapload.DifficultyNormal, []mapload.PartyMember{p}, nil)
			} else {
				w, st, err = mapload.StartMission(m, tbl, mapload.DifficultyNormal, []mapload.PartyMember{p})
			}
			if err != nil {
				t.Fatal(err)
			}
			e, ok := w.Entity(st.IDs[0])
			if !ok || e.Skill[0] != 79 || e.Skill[1] != 20 || e.Skill[2] != 32 {
				t.Fatalf("party inputs did not reach the equipped live actor: %+v", e)
			}
			assertInitialBase(t, e, [22]byte{
				0, 0, 0, 0, 13, 0, 23, 0, 31, 0, 41, 0, 53, 0,
				0, 0, 0, 0, 0, 0, 0, 0,
			}, 44, [64]byte{0: 5, 18: 7, 22: 7, 24: 9, 32: 5, 33: 4, 42: 2})
		})
	}
}

func TestNativeInitialBaseDoesNotInitializeUnresolvedPlacement(t *testing.T) {
	w, err := mapload.FromALMWith(initialBasisMap(201, 4), initialBasisTable(), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	if b := w.Entities()[0].NativeBasis; b.HasValues() {
		t.Fatalf("unresolved placement received native constructor history: %+v", b)
	}
}

func TestNativeInitialBaseRebuildAndCodecKeepUnknownTailHistory(t *testing.T) {
	w, err := mapload.FromALMWith(initialBasisMap(200, 4), initialBasisTable(), mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	// This is an independently admitted history value, not a constructor
	// expected value. Rebuild must retain its known byte and unknown tail.
	b := sim.NativeActorBasis{BasePresent: true, BaseKnown: 0x003fffff}
	b.Base[0], b.Base[22], b.Base[23] = 0xc1, 0xa5, 0x5a
	if err := w.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 0, Basis: b}}); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := sim.NewStructuredWorld(mapload.Seed, w.Bounds(), sim.ModeCanonical,
		mapload.Planes(initialBasisMap(200, 4), initialBasisTable()), w.Entities(), nil,
		w.Relations(), w.Sacks(), w.Stock(), w.Spells(), w.Ghost(), w.Structures())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rebuilt.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var restored sim.World
	if err := restored.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if got := restored.Entities()[0].NativeBasis; got != b {
		t.Fatalf("rebuild/codec changed independent Base history: %+v, want %+v", got, b)
	}
	if got := restored.Entities()[0].NativeBasis; got.BaseByteKnown(22) || got.BaseByteKnown(23) {
		t.Fatal("rebuild/codec promoted unknown tail bytes")
	}
	ents := restored.Entities()
	ents[0].NativeBasis = sim.NativeActorBasis{}
	arbitrary, err := sim.NewWorld(mapload.Seed, restored.Bounds(), sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatal(err)
	}
	if got := arbitrary.Entities()[0].NativeBasis; got.HasValues() {
		t.Fatalf("generic World constructor invented native history: %+v", got)
	}
}

func TestNativeInitialBaseSourceActorKeepsSeparateAuthority(t *testing.T) {
	tbl := initialBasisTable()
	m := initialBasisMap(200, 4)
	w, err := mapload.FromALMWith(m, tbl, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	e, _, err := mapload.ConstructActorBasis(w.Entities()[0], mapload.PartyMember{}, &m.Units[0], tbl, 1, 1, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if e.ActorLoad.Source.Class == 0 || e.NativeBasis.HasValues() {
		t.Fatalf("source-backed constructor retained competing native history: %+v", e)
	}
}
