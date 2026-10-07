package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func absentMemberFixture(t *testing.T) (mapload.PartyMember, *mapload.Table) {
	t.Helper()
	_, _, w := actorProjectionFixture(t, "Human")
	e := w.Entities()[0]
	load := e.CurrentActorLoad()
	load.Inventory.InsertIndex, load.Inventory.Accumulator = 19, -23
	load.Inventory.Source.HasOwner, load.Inventory.Source.ManaReservePercent = true, 37
	load.Inventory.Source.TypeID = 0x1234
	load.Speed, load.Movement.NativeSpeed = 70000, 70000
	h := mapload.SourceHumanState(load.Inventory.Source, load.Inventory.Accumulator)
	table := eqDefsTable(t)
	weapon := mapload.SourceConstructedItem(sim.PlainItem(eqSwordCode), table)
	weapon.ObjectID, weapon.Kind = 201, 2
	weapon.Effects = []sim.ItemEffect{{Kind: 41, Operand: 2 << 16}, {Kind: 41, Operand: 3 << 16}}
	other := sim.PlainItem(0xe01)
	other.ObjectID, other.Kind = 202, 3
	pack := sim.PlainItem(0xe01)
	pack.ObjectID, pack.Kind = 301, 3
	m := mapload.PartyMember{ID: string([]byte{'i', 0xff}), Name: string([]byte{0xc4, 0xe0, 0xed}),
		StartingHero: true, PlayerCharacter: true, Class: 73, FigureFace: 3, FigureDir: "fighter", Body: "body", Hero: h.Hero(),
		Profile: data.Profile{Fighter: true, HealthColumn: true, ManaColumn: true}, WeaponMaterialized: true,
		Carry: &mapload.Carry{LiveLoad: load, OrderedStacks: []sim.ItemStack{sim.StackItem(pack, 3)}},
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 71, Y: 83}, MapUnitID: 54321,
			HP: e.HP, MaxHP: e.MaxHP, Mana: e.Mana, MaxMana: e.MaxMana, HealthRegenPeriod: e.HealthRegenPeriod, ManaRegenPeriod: e.ManaRegenPeriod},
		KnownSpells: 2 | 8, Book: sim.Spellbook{State: sim.BookPresent}, SpellbookRestored: true, SpellbookPresent: true,
		PotionEffect: &sim.ActiveEffect{Target: 777, Kind: sim.EffectAbsorption, Magnitude: 7, Remaining: 19}}
	m.Book.Slots[0], m.Book.Slots[2] = sim.BookSpell{Range: 71, Defensive: 1, ManaCost: 531}, sim.BookSpell{Range: 9, ManaCost: 217}
	m.Carry.EquippedItems[0], m.Carry.EquippedItems[3] = weapon, other
	m.Carry.Equipped[0], m.Carry.Equipped[3] = weapon.Code, other.Code
	m.WornItems, m.Worn = m.Carry.EquippedItems, m.Carry.Equipped
	m.Carry.ItemInstances = []sim.ItemInstance{pack, pack, pack}
	m.Carry.Items = []uint16{pack.Code, pack.Code, pack.Code}
	m.CarriedItems, m.Carried = m.Carry.ItemInstances, m.Carry.Items
	for i, xp := range load.Inventory.Source.SkillXP {
		m.Carry.SkillXP[i] = int32(xp)
	}
	m.Weapon = mapload.CurrentItemWeapon(weapon, table)
	m.OriginalHuman = mapload.BindOriginalHuman(m, h)
	m.OriginalHuman.Retired = true
	return m, table
}

func emptyTemplateWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(17, sim.Bounds{Width: 128, Height: 128}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func assertAbsentMember(t *testing.T, want, got mapload.PartyMember, table *mapload.Table) {
	t.Helper()
	if want.ID != got.ID || want.Name != got.Name || want.StartingHero != got.StartingHero || want.Hero != got.Hero || want.Class != got.Class || want.FigureFace != got.FigureFace {
		t.Fatalf("absent identity/base changed: want %+v got %+v", want, got)
	}
	if !reflect.DeepEqual(want.Saved, got.Saved) || !reflect.DeepEqual(want.PotionEffect, got.PotionEffect) || want.KnownSpells != got.KnownSpells || want.Book != got.Book {
		t.Fatalf("absent saved/book/effect state changed: saved %+v/%+v book %+v/%+v", want.Saved, got.Saved, want.Book, got.Book)
	}
	if !reflect.DeepEqual(mapload.MemberItemEquipment(want, table), mapload.MemberItemEquipment(got, table)) || !reflect.DeepEqual(mapload.MemberCarriedItems(want, table), mapload.MemberCarriedItems(got, table)) {
		t.Fatalf("absent canonical holdings changed: want %+v got %+v", want.Carry, got.Carry)
	}
	if want.Carry != nil && (got.Carry == nil || want.Carry.SkillXP != got.Carry.SkillXP || !reflect.DeepEqual(want.Carry.LiveLoad, got.Carry.LiveLoad) || !reflect.DeepEqual(want.Carry.OrderedStacks, got.Carry.OrderedStacks)) {
		t.Fatalf("absent current load/XP/stacks changed: want %+v got %+v; load %+v/%+v", want.Carry, got.Carry, want.Carry.LiveLoad, got.Carry.LiveLoad)
	}
	if want.OriginalHuman != nil && (got.OriginalHuman == nil || want.OriginalHuman.Retired != got.OriginalHuman.Retired || want.OriginalHuman.State != got.OriginalHuman.State) {
		t.Fatalf("absent Human mode or values changed: want %+v got %+v", want.OriginalHuman, got.OriginalHuman)
	}
}

func TestAbsentPartyTemplateMissingBindingAndSecondSave(t *testing.T) {
	member, table := absentMemberFixture(t)
	w := emptyTemplateWorld(t)
	before, _ := w.MarshalBinary()
	for cycle := 0; cycle < 2; cycle++ {
		p := captureCurrentParty(777, member)
		a := currentActionData{Version: 1, Bindings: []currentActionBinding{{ID: 777, Missing: true}}, Party: []currentPartyMember{p}, Roster: []currentPartyMember{p}}
		if err := bindCurrentPartyRecords(&a, nil); err != nil {
			t.Fatal(err)
		}
		if err := stripCurrentPartyValues(&a, w, table); err != nil {
			t.Fatal(err)
		}
		for _, row := range append(a.Party[:len(a.Party):len(a.Party)], a.Roster...) {
			if row.Member != nil || row.Template == nil || row.Policy == nil || row.Name != nil {
				t.Fatal("absent member duplicates ordinary values")
			}
		}
		raw, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		var restored currentActionData
		if err := json.Unmarshal(raw, &restored); err != nil {
			t.Fatal(err)
		}
		if err := bindCurrentPartyRecords(&restored, nil); err != nil {
			t.Fatal(err)
		}
		ms := &Mission{World: w}
		if err := restoreCurrentPartyMembers(ms, &restored, table); err != nil {
			t.Fatal(err)
		}
		assertAbsentMember(t, member, ms.Party[0], table)
		assertAbsentMember(t, member, ms.Start.Roster[777], table)
		if !reflect.DeepEqual(ms.Start.IDs, []sim.EntityID{777}) {
			t.Fatal("removed entry party lost its identity")
		}
		member = ms.Party[0]
	}
	after, _ := w.MarshalBinary()
	if !bytes.Equal(before, after) || len(w.Entities()) != 0 || len(w.OriginalDeadActors()) != 0 || w.SavedObjects() != nil {
		t.Fatal("template restoration added runtime population")
	}
}

func TestAbsentPartyTemplateItemAbsenceYieldsToOrdinaryFields(t *testing.T) {
	for _, field := range []string{"weight", "equipment", "legacy"} {
		t.Run(field, func(t *testing.T) {
			member, table := absentMemberFixture(t)
			p, err := captureCurrentPartyTemplate(777, member, member, table)
			if err != nil {
				t.Fatal(err)
			}
			actor := &p.Template.Records.Objects[p.Template.Records.Actor-1]
			refs, _ := savedObjectRefs(actor, "HeldWeapon")
			if len(refs) != 1 || refs[0] == 0 {
				t.Fatal("fixture has no held ordinary Weapon")
			}
			weapon := &p.Template.Records.Objects[refs[0]-1]
			for i := range p.Template.Items {
				row := &p.Template.Items[i]
				if row.Object != refs[0] {
					continue
				}
				if row.WeightAnchor == nil || row.EquipmentAnchor == nil {
					t.Fatal("template omitted independent absence anchors")
				}
				bad := *row
				bad.WeightKnown = true
				*row = bad
				if _, err := p.restoreFromCurrent(emptyTemplateWorld(t), table); err == nil {
					t.Fatal("template accepted conflicting Item absence policy")
				}
				row.WeightKnown = false
				if field == "legacy" {
					row.WeightAnchor, row.EquipmentAnchor = nil, nil
				}
			}
			switch field {
			case "weight":
				mustSetValue(weapon, "F4A", 123)
			case "equipment":
				key, _ := savedStructureValue(weapon, "Identity")
				weight, _ := savedStructureValue(weapon, "F4A")
				effects, _ := savedObjectRefs(weapon, "Effects")
				*weapon = literalItemRecord1115("Weapon", eqSwordCode, 1, key, effects, 0)
				mustSetValue(weapon, "F4A", weight)
				mustSetValue(weapon, "T0C", uint32(eqSwordCode&0x1f))
				raw, err := savedObjectRaw(weapon, "W52", 24)
				if err != nil {
					t.Fatal(err)
				}
				raw[0] = 73
			}
			policy, _ := json.Marshal(p.Template.Items)
			got, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
			if err != nil {
				t.Fatal(err)
			}
			item := mapload.MemberItemEquipment(got, table)[0]
			if item.WeightPresent != (field != "equipment") || (item.SourceEquipment.Class != 0) != (field == "equipment") || field == "weight" && item.Weight != 123 || field == "equipment" && item.SourceEquipment.Attack[0] != 73 {
				t.Fatal("template ordinary edit lost its independent value authority", item)
			}
			unchanged, _ := json.Marshal(p.Template.Items)
			if !bytes.Equal(policy, unchanged) {
				t.Fatal("ordinary edit rewrote absent Item policy")
			}
			p, err = captureCurrentPartyTemplate(777, got, got, table)
			if err != nil {
				t.Fatal(err)
			}
			next, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
			if err != nil {
				t.Fatal(err)
			}
			assertAbsentMember(t, got, next, table)
		})
	}
}

func TestAbsentPartyTemplateOrdinaryFieldsWinAndBodyIsSeparate(t *testing.T) {
	member, table := absentMemberFixture(t)
	doc, body, _ := actorProjectionFixture(t, "Human")
	w := emptyTemplateWorld(t)
	p := captureCurrentParty(777, member)
	a := currentActionData{Bindings: []currentActionBinding{{ID: 777, Object: body.ObjectIndex}}, Party: []currentPartyMember{p}}
	doc.DeadActors = append(doc.DeadActors, body.ObjectIndex)
	mustSetValue(&doc.Objects[body.ObjectIndex-1], "Identity", 0x1234)
	if err := w.ImportOriginalDeadActors([]sim.OriginalDeadActor{{ID: 777, Source: sim.OriginalDeadSource{
		Identity: 0x1234, ArchiveIndex: 12, Class: 2, State: sim.DeadActorState{Cell: 0x0504, Stage: 5, HP: -10017, FineX: 128, FineY: 128},
	}}}); err != nil {
		t.Fatal(err)
	}
	bodyBefore, _ := w.MarshalBinary()
	before, _ := json.Marshal(doc)
	if err := bindCurrentPartyRecords(&a, &doc); err != nil {
		t.Fatal(err)
	}
	if err := stripCurrentPartyValues(&a, w, table); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(doc)
	if !bytes.Equal(before, after) {
		t.Fatal("absent template capture edited ordinary body or roots")
	}
	row := &a.Party[0]
	policy, _ := json.Marshal([]any{row.Policy, row.Base, row.Weapon, row.Template.Policy})
	r := &row.Template.Records.Objects[row.Template.Records.Actor-1]
	name := string([]byte{0xc4, 0xff, '2'})
	mustSetText(r, "Name", name)
	mustSetValue(r, "Body", 123)
	mustSetValue(r, "Health", 4)
	mustSetValue(r, "T08", 321)
	mustSetRaw(r, "Block12", constructedPositionBlock(11, 17, 0))
	xp, _ := savedActorRaw(r, "H1CC", 24)
	binary.LittleEndian.PutUint32(xp, 271)
	refs, _ := savedObjectRefs(r, "Inventory")
	mustSetValue(&row.Template.Records.Objects[refs[0]-1], "F40", 0xe03)
	mustSetValue(&doc.Objects[body.ObjectIndex-1], "Health", 9)
	mustSetText(&doc.Objects[body.ObjectIndex-1], "Name", "body only")
	if err := bindCurrentPartyRecords(&a, &doc); err != nil {
		t.Fatal(err)
	}
	ms := &Mission{World: w}
	if err := restoreCurrentPartyMembers(ms, &a, table); err != nil {
		t.Fatal(err)
	}
	got := ms.Party[0]
	if got.Name != name || got.Hero.Body != 123 || got.Saved.HP != 4 || got.Saved.MapUnitID != 321 || got.Saved.Cell != (mapload.Cell{X: 11, Y: 17}) || got.Carry.SkillXP[0] != 271 || got.Carried[0] != 0xe03 {
		t.Fatal("supplement or body overrode edited template fields", got)
	}
	unchanged, _ := json.Marshal([]any{row.Policy, row.Base, row.Weapon, row.Template.Policy})
	if !bytes.Equal(policy, unchanged) || len(doc.DeadActors) != 1 || len(w.Entities()) != 0 {
		t.Fatal("ordinary edit rewrote policy or population")
	}
	bodyAfter, _ := w.MarshalBinary()
	if !bytes.Equal(bodyBefore, bodyAfter) || len(w.OriginalDeadActors()) != 1 {
		t.Fatal("template changed held dead-manager body")
	}
}

func TestAbsentPartyTemplateRejectsMalformedContinuationAtomically(t *testing.T) {
	member, table := absentMemberFixture(t)
	valid, err := captureCurrentPartyTemplate(777, member, member, table)
	if err != nil {
		t.Fatal(err)
	}
	for name, corrupt := range map[string]func(*currentPartyMember){
		"legacy and fragment": func(p *currentPartyMember) { p.Member = &member },
		"no policy":           func(p *currentPartyMember) { p.Policy = nil },
		"no actor":            func(p *currentPartyMember) { p.Template.Records.Actor = 0 },
		"orphan": func(p *currentPartyMember) {
			p.Template.Records.Objects = append(p.Template.Records.Objects, p.Template.Records.Objects[1])
		},
		"repeated item":        func(p *currentPartyMember) { p.Template.Items = append(p.Template.Items, p.Template.Items[0]) },
		"repeated identity":    func(p *currentPartyMember) { p.Template.Items[1].ID = p.Template.Items[0].ID },
		"missing item":         func(p *currentPartyMember) { p.Template.Items = p.Template.Items[1:] },
		"cross namespace":      func(p *currentPartyMember) { p.Template.Records.Objects[0].RefSlots[0].Objects = []uint16{65000} },
		"bad book":             func(p *currentPartyMember) { p.Template.Policy.BookMode = 9 },
		"Human XP duplicate":   func(p *currentPartyMember) { p.Template.UnitSkillXP = &[6]int32{1} },
		"missing runtime mode": func(p *currentPartyMember) { p.Template.Policy.Load.EquipmentRuntimePresent = false },
		"invalid source active skill": func(p *currentPartyMember) {
			r := &p.Template.Records.Objects[p.Template.Records.Actor-1]
			attack, _ := savedActorRaw(r, "UA6", 24)
			attack[16] = 6
		},
	} {
		t.Run(name, func(t *testing.T) {
			raw, _ := json.Marshal(valid)
			var p currentPartyMember
			if err := json.Unmarshal(raw, &p); err != nil {
				t.Fatal(err)
			}
			corrupt(&p)
			w := emptyTemplateWorld(t)
			ms := &Mission{World: w, Party: []mapload.PartyMember{member}}
			ms.Start.Roster = map[sim.EntityID]mapload.PartyMember{5: member}
			before, _ := json.Marshal(ms.Party)
			world, _ := w.MarshalBinary()
			if err := restoreCurrentPartyMembers(ms, &currentActionData{Party: []currentPartyMember{valid, p}}, table); err == nil {
				t.Fatal("accepted malformed template")
			}
			after, _ := json.Marshal(ms.Party)
			worldAfter, _ := w.MarshalBinary()
			if !bytes.Equal(before, after) || !bytes.Equal(world, worldAfter) || len(ms.Start.Roster) != 1 || ms.Start.Roster[5].ID != member.ID {
				t.Fatal("failed template restoration partially changed session")
			}
		})
	}
	_, _, present := actorProjectionFixture(t, "Human")
	valid.Entity = present.Entities()[0].ID
	if _, err := valid.restoreFromCurrent(present, table); err == nil {
		t.Fatal("fragment attached to present Entity")
	}
}

func TestAbsentPartyTemplateNativeBooksFallbackAndUnitLocations(t *testing.T) {
	table := eqDefsTable(t)
	for _, mode := range []sim.BookState{sim.BookLegacy, sim.BookAbsent, sim.BookPresent} {
		member := mapload.PartyMember{ID: "native", Name: "unplaced", FigureFace: 3, FigureDir: "fighter", Hero: eqHero(),
			Profile: data.Profile{Fighter: true, HealthColumn: true}, Class: 41, StartingHero: true,
			Weapon: eqSword(t, table), Book: sim.Spellbook{State: mode}}
		if mode == sim.BookLegacy {
			member.KnownSpells = 1064962 | 1<<31
		}
		for cycle := 0; cycle < 2; cycle++ {
			p, err := captureCurrentPartyTemplate(90, member, member, table)
			if err != nil {
				t.Fatal(mode, err)
			}
			got, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
			if err != nil {
				t.Fatal(mode, err)
			}
			assertAbsentMember(t, member, got, table)
			if !reflect.DeepEqual(member.Weapon, got.Weapon) || got.Carry != nil || got.Saved != nil || got.OriginalHuman != nil {
				t.Fatal("empty starting weapon or absent modes changed", got)
			}
			member = got
		}
	}
	member, table := absentMemberFixture(t)
	member.Carry.LiveLoad.Inventory.Source.Class = 1
	member.OriginalHuman = nil
	member.Carry.LiveLoad.Inventory.ContainerPresent = false
	member.Carry.LiveLoad.Inventory.InsertIndex, member.Carry.LiveLoad.Inventory.Accumulator = 0, 0
	member.Carry.Items, member.Carry.ItemInstances, member.Carry.OrderedStacks = nil, nil, nil
	member.Carried, member.CarriedItems = nil, nil
	for cycle := 0; cycle < 2; cycle++ {
		p, err := captureCurrentPartyTemplate(91, member, member, table)
		if err != nil {
			t.Fatal(err)
		}
		if p.Template.Records.Objects[0].Class != "Unit" || len(p.Template.Equipment) != 1 || !p.Template.PackAbsent || p.Template.UnitSkillXP == nil {
			t.Fatal("Unit ordinary class or native absent-field/location policy lost")
		}
		bad, invalid := p, *p.Template
		bad.Template, invalid.UnitSkillXP = &invalid, nil
		if _, err := bad.restoreFromCurrent(emptyTemplateWorld(t), table); err == nil {
			t.Fatal("Unit with Carry accepted missing native skill experience")
		}
		if p.Name != nil {
			t.Fatal("Unit template name duplicated outside ordinary record")
		}
		member.Name = string([]byte{0xc4, 0xe0, byte('0' + cycle)})
		mustSetText(&p.Template.Records.Objects[0], "Name", member.Name)
		got, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
		if err != nil {
			t.Fatal(err)
		}
		assertAbsentMember(t, member, got, table)
		member = got
	}
}

func TestAbsentPartyTemplateSavedWidthOperandsYieldToOrdinaryEdits(t *testing.T) {
	member, table := absentMemberFixture(t)
	member.OriginalHuman = nil
	member.Saved.Cell = mapload.Cell{X: math.MinInt32, Y: math.MaxInt32}
	member.Saved.HP, member.Saved.MaxHP, member.Saved.HealthRegenPeriod = math.MinInt32, math.MaxInt32, -65537
	member.Saved.Mana, member.Saved.MaxMana, member.Saved.ManaRegenPeriod = 123456789, -123456789, math.MaxInt32
	for i, value := range []int32{member.Saved.HP, member.Saved.MaxHP, member.Saved.HealthRegenPeriod, member.Saved.Mana, member.Saved.MaxMana, member.Saved.ManaRegenPeriod} {
		member.Carry.LiveLoad.Inventory.Source.Stats[8+i] = uint16(value)
	}
	var p currentPartyMember
	for cycle := 0; cycle < 2; cycle++ {
		var err error
		p, err = captureCurrentPartyTemplate(777, member, member, table)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Template.SavedLifts) != 8 {
			t.Fatal("native Saved fields lost their width operands", p.Template.SavedLifts)
		}
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var decoded currentPartyMember
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		got, err := decoded.restoreFromCurrent(emptyTemplateWorld(t), table)
		if err != nil {
			t.Fatal(err)
		}
		assertAbsentMember(t, member, got, table)
		member = got
	}
	before, _ := json.Marshal(p.Template.SavedLifts)
	r := &p.Template.Records.Objects[p.Template.Records.Actor-1]
	mustSetValue(r, "Health", 42)
	mustSetRaw(r, "Block12", constructedPositionBlock(11, 17, 0))
	got, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
	if err != nil {
		t.Fatal(err)
	}
	if got.Saved.Cell != (mapload.Cell{X: 11, Y: 17}) || got.Saved.HP != 42 || got.Saved.MaxHP != member.Saved.MaxHP {
		t.Fatal("stale width operand overrode edited ordinary field", got.Saved)
	}
	after, _ := json.Marshal(p.Template.SavedLifts)
	if !bytes.Equal(before, after) {
		t.Fatal("ordinary edit changed native width policy")
	}
	for name, lifts := range map[string][]currentTemplateSavedLift{
		"repeated":           {p.Template.SavedLifts[0], p.Template.SavedLifts[0]},
		"outside field":      {{Field: 8, Lift: 1}},
		"outside cell width": {{Field: 0, Wire: 256, Lift: 1}},
		"overflow":           {{Field: 2, Wire: 42, Lift: math.MaxInt64}},
		"underflow":          {{Field: 2, Wire: 42, Lift: math.MinInt64}},
		"empty operand":      {{Field: 2, Wire: 42}},
	} {
		t.Run(name, func(t *testing.T) {
			bad, invalid := p, *p.Template
			bad.Template, invalid.SavedLifts = &invalid, lifts
			w := emptyTemplateWorld(t)
			ms := &Mission{World: w, Party: []mapload.PartyMember{member}}
			before, _ := json.Marshal(ms.Party)
			if err := restoreCurrentPartyMembers(ms, &currentActionData{Party: []currentPartyMember{bad}}, table); err == nil {
				t.Fatal("accepted invalid native width operand")
			}
			after, _ := json.Marshal(ms.Party)
			if !bytes.Equal(before, after) {
				t.Fatal("invalid width operand partially changed party")
			}
		})
	}
}

func TestAbsentPartyTemplateMissingIdentityStaysReservedAfterAdvance(t *testing.T) {
	member, table := absentMemberFixture(t)
	p, err := captureCurrentPartyTemplate(777, member, member, table)
	if err != nil {
		t.Fatal(err)
	}
	w := emptyTemplateWorld(t)
	ms := &Mission{World: w, savedDocument: &SnapshotSAVDocument{Document: &sav.DocumentData{}}}
	a := &currentActionData{Bindings: []currentActionBinding{{ID: 777, Missing: true}}, Party: []currentPartyMember{p}}
	if _, err := resolveCurrentActions(ms, a); err != nil {
		t.Fatal(err)
	}
	if err := restoreCurrentPartyMembers(ms, a, table); err != nil {
		t.Fatal(err)
	}
	sim.Step(w, nil)
	if id, ok := w.NextEntityID(); !ok || id <= 777 || len(w.Entities()) != 0 || len(w.OriginalDeadActors()) != 0 {
		t.Fatal("removed entry-party identity reused or resurrected", id, ok)
	}
}
