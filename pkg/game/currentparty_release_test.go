package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseCurrentPartyOrdinaryWireWins(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "ordinary party", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	party[0].Carry = &mapload.Carry{Equipped: party[0].Worn, EquippedItems: party[0].WornItems,
		Items: []uint16{0xe01}, ItemInstances: []sim.ItemInstance{sim.PlainItem(0xe01)}}
	if err := f.App("ordinary party fields").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || len(a.Party) != 1 {
		t.Fatal("missing current party binding", err)
	}
	var index uint16
	for _, b := range a.Bindings {
		if b.ID == a.Party[0].Entity && !b.Structure && !b.Missing {
			index = b.Object
		}
	}
	if index == 0 {
		t.Fatal("party lacks ordinary actor")
	}
	leaf, _, err := sav.NativeActions(doc.State)
	if err != nil {
		t.Fatal(err)
	}
	name := string([]byte{0xc4, 0xe0, 0xed, 0xe0, 0xf1, ' ', '2'})
	r := &doc.Objects[index-1]
	flags, err := savedStructureValue(r, "U4C")
	if err != nil {
		t.Fatal(err)
	}
	flags ^= 4
	savedObjectSetValue(r, "U4C", flags)
	savedObjectSetValue(r, "U4B", 142)
	typeID, err := savedStructureValue(r, "T0E")
	if err != nil {
		t.Fatal(err)
	}
	face := 142
	if typeID < 0x1a {
		face &= 0x7f
	}
	for _, player := range doc.Players {
		savedObjectSetValue(&doc.Objects[player-1], "Hero", 0)
	}
	changed := 0
	for i := range r.Texts {
		if r.Texts[i].Name == "Name" {
			r.Texts[i].Value = name
			changed++
		}
	}
	for i := range r.Values {
		if r.Values[i].Name == "Health" {
			r.Values[i].Value = 7
			changed++
		}
	}
	for i := range r.Raw {
		if r.Raw[i].Name == "H1CC" {
			binary.LittleEndian.PutUint32(r.Raw[i].Bytes, 1777)
			changed++
		}
	}
	if changed != 3 {
		t.Fatal("wire controls were not applied", changed)
	}
	unchanged, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(leaf, unchanged) {
		t.Fatal("wire controls also edited native continuation")
	}
	encoded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	mutated := filepath.Join(t.TempDir(), "ordinary.sav")
	if err := os.WriteFile(mutated, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		cold := loadAreaContinuation(t, mutated)
		ms := cold.live.mission.state
		id := ms.Start.IDs[0]
		e, ok := cold.live.entity(id)
		if !ok || e.HP != 7 || e.SkillXP[0] != 1777 {
			t.Fatalf("cycle %d: ordinary World values lost: %+v", cycle, e)
		}
		p := ms.Party[0]
		if p.Name != name || p.Carry == nil || p.Carry.SkillXP[0] != 1777 {
			t.Fatalf("cycle %d: retained member overwrote ordinary values: %+v", cycle, p)
		}
		if p.StartingHero || p.Profile.Fighter != (flags&4 == 0) {
			t.Fatalf("cycle %d: ordinary primary/profile overwritten: hero=%v fighter=%v flags=%d", cycle, p.StartingHero, p.Profile.Fighter, flags)
		}
		if p.FigureFace != face {
			t.Fatalf("cycle %d: ordinary face replaced by stale presentation: got %d want %d", cycle, p.FigureFace, face)
		}
		if p.ID != party[0].ID {
			t.Fatal("ordinary value edit changed stable party identity", p.ID, party[0].ID)
		}
		if cycle == 0 {
			mutated = saveCorpseMission(t, cold, t.TempDir())
		}
	}
}

func TestReleaseCurrentPartyWeaponWireWins(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "ordinary weapon", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	item := mapload.SourceConstructedItem(sim.PlainItem(uint16(party[0].Weapon.Code)), f.Table)
	if item.SourceEquipment.Class != sim.SourceWeapon || !item.SourceEquipment.Definition.Present || !item.WeightPresent {
		t.Fatal("fixture has no concrete current weapon")
	}
	party[0].WornItems[0], party[0].Worn[0] = item, item.Code
	if err := f.App("ordinary weapon fields").OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || len(a.Party) != 1 || a.Party[0].Member != nil || a.Party[0].Policy == nil || a.Party[0].Weapon == nil || a.Party[0].Weapon.Fallback != nil {
		t.Fatal("current held weapon retained in supplement", err)
	}
	var actor uint16
	for _, b := range a.Bindings {
		if b.ID == a.Party[0].Entity && !b.Structure && !b.Missing {
			actor = b.Object
		}
	}
	if actor == 0 {
		t.Fatal("missing actor binding")
	}
	refs, _ := savedObjectRefs(&doc.Objects[actor-1], "HeldWeapon")
	if len(refs) != 1 || refs[0] == 0 {
		t.Fatal("missing ordinary held weapon")
	}
	leaf, _, err := sav.NativeActions(doc.State)
	if err != nil {
		t.Fatal(err)
	}
	r := &doc.Objects[refs[0]-1]
	row := uint8(2)
	if item.SourceEquipment.DefinitionRow == row {
		row = 3
	}
	savedObjectSetValue(r, "T0C", uint32(row))
	savedObjectSetValue(r, "W50", 7)
	savedObjectSetValue(r, "F4A", 65527)
	changed := 0
	for i := range r.Raw {
		switch r.Raw[i].Name {
		case "W52":
			binary.LittleEndian.PutUint16(r.Raw[i].Bytes, 54321)
			r.Raw[i].Bytes[14], r.Raw[i].Bytes[15] = 91, 17
			changed++
		case "W6A":
			binary.LittleEndian.PutUint16(r.Raw[i].Bytes, 12345)
			changed++
		}
	}
	if changed != 2 {
		t.Fatal("ordinary weapon blocks absent")
	}
	unchanged, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(leaf, unchanged) {
		t.Fatal("weapon control changed supplement")
	}
	encoded, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ordinary-weapon.sav")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	definition := mapload.BindSourceItemDefinition(sim.ItemInstance{SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: row}}, f.Table).SourceEquipment.Definition
	for cycle := 0; cycle < 2; cycle++ {
		cold := loadAreaContinuation(t, path)
		ms := cold.live.mission.state
		eq, ok := ms.World.EquippedItems(ms.Start.IDs[0])
		if !ok || eq[0].SourceEquipment.DefinitionRow != row || eq[0].SourceEquipment.Definition != definition ||
			eq[0].Weight != -9 || binary.LittleEndian.Uint16(eq[0].SourceEquipment.Attack[:]) != 54321 {
			t.Fatalf("cycle %d: ordinary item lost to supplementary operands: %+v", cycle, eq[0])
		}
		p := ms.Party[0].Weapon
		if p == nil || p.Code != party[0].Weapon.Code || p.Row != int32(row) || p.ToHit != 54321 || p.Defence != 12345 ||
			p.DamageBase != 91 || p.DamageSpread != 17 || p.Range != 7 || p.Weight != -9 ||
			p.AttackType != definition.AttackType || p.ChargeTime != definition.Charge || p.RelaxTime != definition.Relax {
			t.Fatalf("cycle %d: member weapon disagrees with current ordinary item: %+v", cycle, p)
		}
		if cycle == 0 {
			path = saveCorpseMission(t, cold, t.TempDir())
		}
	}
}
