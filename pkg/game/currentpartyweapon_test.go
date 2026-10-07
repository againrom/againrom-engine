package game

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentPartyWeaponUsesHeldObjectAndDefinition(t *testing.T) {
	table := eqDefsTable(t)
	spellName := string([]byte{0xc8, 0xf1, 0xea, 0xf0, 0xe0, ' ', '2'})
	table.Spells = dbCollection{{}, {name: spellName}, {name: "Other"}}
	stale := *eqSword(t, table)
	stale.ToHit, stale.DamageBase, stale.SpellName, stale.SpellPower = 999, 999, "Old", 999
	item := sim.ItemInstance{Code: uint16(stale.Code), Kind: 2, WeightPresent: true, Weight: -3,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 37, OwnKind: 3,
			Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 12, Charge: -1, Relax: 11},
			Spell:      sim.SourceItemSpell{Present: true, ID: 2}},
		Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(0xfff9)<<16 | 1}, {Kind: 41, Operand: 2}}}
	binary.LittleEndian.PutUint16(item.SourceEquipment.Attack[:], 54321)
	item.SourceEquipment.Attack[14], item.SourceEquipment.Attack[15] = 91, 17
	binary.LittleEndian.PutUint16(item.SourceEquipment.Defence[:], 12345)
	before := item.Clone()
	policy := capturePartyWeapon(mapload.PartyMember{Weapon: &stale}, item)
	if policy.Fallback != nil || policy.Name != nil || policy.SpellName != nil {
		t.Fatal("held weapon duplicated in supplement")
	}
	got := policy.restore(item, false, table)
	if got == nil || got.Code != stale.Code || got.Row != 37 || got.ToHit != 54321 || got.Defence != 12345 ||
		got.DamageBase != 91 || got.DamageSpread != 17 || got.Range != 3 || got.AttackType != 12 ||
		got.ChargeTime != -1 || got.RelaxTime != 11 || got.Weight != -3 || got.SpellPower != -7 ||
		got.SpellName != string([]byte{0xc8, 0xf1, 0xea, 0xf0, 0xe0, '_', '2'}) {
		t.Fatalf("current saved operands lost to appearance or stale fallback: %+v", got)
	}
	if !reflect.DeepEqual(item, before) {
		t.Fatal("derived view changed the owned item")
	}
	member := mapload.PartyMember{Weapon: &stale, Hero: eqHero()}
	member.WornItems[0] = item
	if live := mapload.PartyLoadout(member, table).Weapon; live == nil || *live != *got {
		t.Fatal("town or mission start reused a stale weapon cache", live, got)
	}
	if view := headlessMemberFromParty(member, table).Weapon; view == nil || view.Code != uint16(got.Code) || view.Name != got.Name || view.Defense != got.Defence {
		t.Fatal("headless member reads the old weapon", view, got)
	}
	if name := partyPanelSubject(member, table).Char.Weapon; name != got.Name {
		t.Fatal("character panel reads the old weapon", name, got.Name)
	}
	loadout, ok := mapload.ResolveItemLoadout(member.WornItems, &stale, true, table)
	if !ok || loadout.Weapon == nil || *loadout.Weapon != *got {
		t.Fatal("live derivation reads the same-code cache", loadout.Weapon, got)
	}
	item.Effects = nil
	bare := policy.restore(item, false, table)
	if bare.SpellName != "" || bare.SpellPower != 0 {
		t.Fatal("removed kind41 returned from owned Spell or stale fallback", bare)
	}
	item.SourceEquipment.Definition.Present = false
	if got := policy.restore(item, false, table); got != nil {
		t.Fatal("missing concrete definition replaced by appearance row", got)
	}
	plain := sim.PlainItem(uint16(stale.Code))
	native := policy.restore(plain, false, table)
	if native == nil || native.ToHit == stale.ToHit || native.SpellName != "" || native.Row == 37 {
		t.Fatal("native held item used same-code stale fallback", native)
	}
}

func TestCurrentPartyWeaponKeepsOnlyAbsentFallback(t *testing.T) {
	name := string([]byte{0xce, 0xf0, 0xf3, 0xe6, 0xe8, 0xe5})
	weapon := data.Weapon{Name: name, SpellName: name, Code: 0x101, DamageBase: 7, SpellPower: -2}
	p := mapload.PartyMember{Weapon: &weapon}
	policy := capturePartyWeapon(p, sim.ItemInstance{})
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	var decoded currentPartyWeapon
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	got := decoded.restore(sim.ItemInstance{}, false, nil)
	if got == nil || *got != weapon {
		t.Fatal("absent starting weapon or byte text lost", got)
	}
	got.DamageBase++
	if decoded.Fallback.DamageBase != 7 {
		t.Fatal("fallback restore aliases supplement")
	}
	if decoded.restore(sim.ItemInstance{}, true, nil) != nil {
		t.Fatal("sold materialized weapon returned")
	}
	p.WeaponMaterialized = true
	if capturePartyWeapon(p, sim.ItemInstance{}).Fallback != nil {
		t.Fatal("sold weapon retained in new supplement")
	}
	if mapload.MemberWeapon(p, nil) != nil || headlessMemberFromParty(p, nil).Weapon != nil || partyPanelSubject(p, nil).Char.Weapon != "" {
		t.Fatal("sold materialized weapon returned in a runtime view")
	}
	p.WeaponMaterialized = false
	if live := mapload.MemberWeapon(p, nil); live == nil || *live != weapon {
		t.Fatal("absent starting weapon was not preserved in runtime", live)
	} else {
		live.DamageBase++
		if p.Weapon.DamageBase != 7 {
			t.Fatal("runtime view aliases the starting-weapon value")
		}
	}
	if got := decoded.restore(sim.PlainItem(0x102), false, eqDefsTable(t)); got == nil || got.Code != 0x102 || got.Name == name {
		t.Fatal("fallback replaced a newly held item", got)
	}
}

func TestCurrentWeaponNameFollowsCurrentCodeAndLiteral(t *testing.T) {
	table := eqDefsTable(t)
	table.Names = data.ItemNames{data.ItemCode(eqSwordCode): "Catalog Sword", data.ItemCode(eqMaceCode): "Catalog Mace"}
	weapon := *eqSword(t, table)
	weapon.Code = data.ItemCode(eqSwordCode)
	weapon.Name = string([]byte{0xce, 0xf0, 0xf3, 0xe6, 0xe8, 0xe5})
	member := mapload.PartyMember{Weapon: &weapon}
	held := mapload.ItemInstanceFromCode(eqSwordCode, table)
	member.WornItems[0] = held
	policy := captureNamedPartyWeapon(member, held)
	if policy.NameCode != eqSwordCode || string(policy.Name) != weapon.Name {
		t.Fatalf("current city name was not captured: policy=%+v weaponCode=%#x heldCode=%#x empty=%v", policy, weapon.Code, held.Code, held.Empty())
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	var decoded currentPartyWeapon
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.NameCode != policy.NameCode || string(decoded.Name) != weapon.Name {
		t.Fatalf("current city name changed in JSON: %+v", decoded)
	}
	want := mapload.CurrentItemWeapon(held, table)
	if want == nil {
		t.Fatal("source weapon has no current definition")
	}
	want.Name = weapon.Name
	if got := decoded.restore(held, false, table); got == nil || *got != *want {
		t.Fatalf("same-code current literal changed stats or name: got %+v want %+v", got, want)
	}
	if got := mapload.MemberWeapon(member, table); got == nil || *got != *want {
		t.Fatalf("live town weapon does not use current literal: got %+v want %+v", got, want)
	}
	changed := mapload.ItemInstanceFromCode(eqMaceCode, table)
	member.WornItems[0] = changed
	for _, got := range []*data.Weapon{decoded.restore(changed, false, table), mapload.MemberWeapon(member, table)} {
		if got == nil || got.Code != data.ItemCode(eqMaceCode) || got.Name != "Catalog Mace" {
			t.Fatalf("changed code reused the old literal or missed the stored catalog: %+v", got)
		}
	}
	member.Weapon.Name = ""
	if got := captureNamedPartyWeapon(member, held).restore(held, false, table); got == nil || got.Name != "Catalog Sword" {
		t.Fatalf("absent current literal missed the stored catalog: %+v", got)
	}
}
