package game

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func skillRankLoadFront(t *testing.T, levels [data.SkillSlots]int32) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	spells := make(dbCollection, 21)
	for i := 1; i < len(spells); i++ {
		spells[i] = dbEntry{name: "synthetic skill spell", params: make([]int32, 19)}
	}
	p := spells[20].params
	p[1], p[2], p[4], p[6], p[8], p[14], p[18] = 1, 4, 1, 8, 1, 2, 1
	spells[20].strings = []string{"Absorbtion=5:duration 64"}
	f.Table.Spells = spells
	party := []mapload.PartyMember{{ID: "hero", Name: "Skill hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile:     data.Profile{HealthColumn: true, ManaColumn: true},
		Hero:        data.Hero{Body: 40, Reaction: 40, Mind: 40, Spirit: 40, Skill: levels},
		KnownSpells: 1 << 20, Carry: &mapload.Carry{},
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 3, Y: 4}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}}}
	if err := f.App("skill rank LOAD").OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	return f
}

func skillRankLegacySave(t *testing.T, f *FrontEnd, levels, effective, xp [data.SkillSlots]int32, legacy bool) []byte {
	t.Helper()
	raw, _, _ := saveCurrentEffect(t, f)
	doc, actor := trainingActor(t, raw, 0)
	base, attack, experience := modBlock(actor, "U114"), modBlock(actor, "UA6"), modBlock(actor, "H1CC")
	if len(base) != 24 || len(attack) != 24 || len(experience) != 24 {
		t.Fatal("synthetic hero lacks ordinary skill blocks")
	}
	for i := range levels {
		binary.LittleEndian.PutUint16(base[2+2*i:], uint16(levels[i]))
		binary.LittleEndian.PutUint16(attack[2+2*i:], uint16(effective[i]))
		binary.LittleEndian.PutUint32(experience[4*i:], uint32(xp[i]))
	}
	if legacy {
		a, err := readCurrentActions(doc)
		if err != nil || a == nil || len(a.Party) != 1 || a.Party[0].Base == nil {
			t.Fatal("synthetic legacy training policy", err)
		}
		a.Party[0].Base.LegacyTraining = true
		leaf, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := sav.EncodeDocumentData(*doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func skillRankLoadedHero(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	if len(f.live.mission.ids) != 1 {
		t.Fatal("synthetic LOAD changed party size")
	}
	e, ok := f.live.entity(f.live.mission.ids[0])
	if !ok {
		t.Fatal("synthetic LOAD lost hero")
	}
	return e
}

func skillRankBookCheck(t *testing.T, f *FrontEnd, e sim.Entity) {
	t.Helper()
	book := e
	sim.RefreshBook(mapload.TableRules(f.Table), &book, mapload.SpellRules(f.Table))
	if book.Book != e.Book {
		t.Fatalf("repaired book = %+v, want %+v", e.Book, book.Book)
	}
}
func TestSkillRankLoadRepairsLegacyTrainingAndNextAward(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "trained", true: "legacy"}[legacy], func(t *testing.T) {
			skillRankNextAwardCase(t, legacy)
		})
	}
}

func skillRankNextAwardCase(t *testing.T, legacy bool) {
	t.Helper()
	levels := [data.SkillSlots]int32{17, 28, 70, 22, 17, 1}
	xp := [data.SkillSlots]int32{0, 12860, 244972, 7140, 355426, 0}
	want := levels
	want[4] = 62
	f := skillRankLoadFront(t, want)
	raw := skillRankLegacySave(t, f, levels, levels, xp, legacy)
	cold := openCurrentEffectSave(t, f, raw)
	hero := skillRankLoadedHero(t, cold)
	if !hero.NativeTraining.Present || hero.NativeTraining.Levels != want || hero.Skill != want || hero.SkillXP != xp {
		t.Fatalf("LOAD trained/effective/XP = %+v/%v/%v, want %v/%v/%v", hero.NativeTraining, hero.Skill, hero.SkillXP, want, want, xp)
	}
	skillRankBookCheck(t, cold, hero)
	if rows := cold.live.view.MessageLines(); len(rows) != 0 {
		t.Fatal("LOAD posted skill repair messages", rows)
	}
	cold.live.attackOrCast(uint32(hero.ID), uint32(hero.ID), 20, int(hero.X), int(hero.Y), false)
	awarded := false
	for tick := 0; tick < 128; tick++ {
		cold.live.tick()
		after := skillRankLoadedHero(t, cold)
		if after.Skill != want || after.NativeTraining.Levels != want {
			t.Fatalf("next award changed repaired skill: %+v/%v", after.NativeTraining, after.Skill)
		}
		if rows := cold.live.view.MessageLines(); len(rows) != 0 {
			t.Fatal("next award posted skill raise messages", rows)
		}
		if after.SkillXP[4] > xp[4] {
			awarded = true
			hero = after
			break
		}
	}
	if !awarded {
		t.Fatal("accepted Earth spell did not award XP", cold.live.world.BookSpellRefusal(hero.ID, hero.ID, 20))
	}
	resaved, _, _ := saveCurrentEffect(t, cold)
	_, actor := trainingActor(t, resaved, 0)
	if got := int16(binary.LittleEndian.Uint16(modBlock(actor, "U114")[10:])); got != 62 {
		t.Fatalf("SAVE trained Earth = %d, want 62", got)
	}
	again := openCurrentEffectSave(t, cold, resaved)
	loaded := skillRankLoadedHero(t, again)
	if !loaded.NativeTraining.Present || loaded.NativeTraining.Levels != want || loaded.Skill != want || loaded.SkillXP != hero.SkillXP {
		t.Fatalf("second cold LOAD changed repaired sheet: %+v/%v/%v", loaded.NativeTraining, loaded.Skill, loaded.SkillXP)
	}
}

func TestSkillRankLoadRepairsBaseBelowAlreadyAlignedEffective(t *testing.T) {
	levels := [data.SkillSlots]int32{1, 22, 1, 1, 1, 1}
	effective := levels
	effective[1] = 55
	xp := [data.SkillSlots]int32{0, 171859}
	f := skillRankLoadFront(t, levels)
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "trained", true: "legacy"}[legacy], func(t *testing.T) {
			raw := skillRankLegacySave(t, f, levels, effective, xp, legacy)
			cold := openCurrentEffectSave(t, f, raw)
			hero := skillRankLoadedHero(t, cold)
			if !hero.NativeTraining.Present || hero.NativeTraining.Levels[1] != 55 || hero.Skill[1] != 55 || hero.SkillXP != xp {
				t.Fatalf("Fire trained/effective/XP = %+v/%v/%v, want 55/55/%v", hero.NativeTraining, hero.Skill, hero.SkillXP, xp)
			}
			if cold.live.world.NativeTrainingNeedsProducer(hero.ID) {
				t.Fatal("repaired Fire still needs an unavailable producer")
			}
		})
	}
}

func TestSkillRankLoadPreservesUnrepairedEffectiveLevel(t *testing.T) {
	levels := [data.SkillSlots]int32{1, 22, 70, 1, 1, 1}
	effective := levels
	effective[1], effective[2] = 55, 90
	xp := [data.SkillSlots]int32{0, 171859, 244972}
	f := skillRankLoadFront(t, levels)
	raw := skillRankLegacySave(t, f, levels, effective, xp, false)
	cold := openCurrentEffectSave(t, f, raw)
	hero := skillRankLoadedHero(t, cold)
	want := levels
	want[1] = 55
	if !hero.NativeTraining.Present || hero.NativeTraining.Levels != want || hero.Skill != effective || hero.SkillXP != xp {
		t.Fatalf("mixed repaired sheet = %+v/%v/%v, want %v/%v/%v", hero.NativeTraining, hero.Skill, hero.SkillXP, want, effective, xp)
	}
	resaved, _, _ := saveCurrentEffect(t, cold)
	again := openCurrentEffectSave(t, cold, resaved)
	hero = skillRankLoadedHero(t, again)
	if hero.NativeTraining.Levels != want || hero.Skill != effective || hero.SkillXP != xp {
		t.Fatal("SAVE and second LOAD changed unrepaired effective Water", hero.NativeTraining, hero.Skill, hero.SkillXP)
	}
}

func TestSkillRankLoadKeepsSettledSlotsAndCapsRepair(t *testing.T) {
	for _, tc := range []struct {
		name            string
		level, xp, want int32
	}{{"exact threshold", 61, 333929, 61}, {"school first purchase", 1, 101, 1}, {"school second purchase", 2, 211, 2}, {"beyond school stamp", 1, 102, 2}, {"zero XP", 17, 0, 17}, {"above XP", 70, 244972, 70}, {"cap", 17, 13779612, 100}, {"already capped", 100, 355426, 100}} {
		t.Run(tc.name, func(t *testing.T) {
			levels := [data.SkillSlots]int32{1, 1, 1, 1, tc.level, 1}
			xp := [data.SkillSlots]int32{0, 0, 0, 0, tc.xp, 0}
			seed := levels
			seed[4] = tc.want
			f := skillRankLoadFront(t, seed)
			raw := skillRankLegacySave(t, f, levels, levels, xp, false)
			cold := openCurrentEffectSave(t, f, raw)
			hero := skillRankLoadedHero(t, cold)
			if hero.NativeTraining.Levels[4] != tc.want || hero.Skill[4] != tc.want || hero.SkillXP != xp {
				t.Fatalf("trained/effective/XP = %+v/%v/%v, want %d/%d/%v", hero.NativeTraining, hero.Skill, hero.SkillXP, tc.want, tc.want, xp)
			}
		})
	}
}

func TestSkillRankLoadPreservesSavedSheet(t *testing.T) {
	settled := [data.SkillSlots]int32{1, 1, 1, 1, 62, 1}
	deficient := settled
	deficient[4] = 17
	xp := [data.SkillSlots]int32{0, 0, 0, 0, 355426}
	f := skillRankLoadFront(t, settled)
	control := openCurrentEffectSave(t, f, skillRankLegacySave(t, f, settled, settled, xp, false))
	want := skillRankLoadedHero(t, control)
	loaded := openCurrentEffectSave(t, f, skillRankLegacySave(t, f, deficient, deficient, xp, false))
	for cycle := range 2 {
		got := skillRankLoadedHero(t, loaded)
		if !reflect.DeepEqual(got, want) {
			currentValueDiagnostics(t, "repaired/control sheet", reflect.ValueOf(got), reflect.ValueOf(want))
			t.Fatalf("cycle %d repaired LOAD changed the saved sheet: got %+v want %+v", cycle, got, want)
		}
		raw, _, _ := saveCurrentEffect(t, loaded)
		loaded = openCurrentEffectSave(t, loaded, raw)
	}
}

func TestSkillRankSharedMemberRestorationPreservesWornBonusAndRules(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		cap, bonus, xp, want int32
	}{{"worn Fire bonus", 100, 33, 171859, 55}, {"lower cap", 50, 0, 171859, 50}, {"extended cap", 150, 0, 17000000, 103}} {
		t.Run(tc.name, func(t *testing.T) {
			levels := [data.SkillSlots]int32{1, 22, 1, 1, 1, 1}
			effective := levels
			effective[1] += tc.bonus
			xp := [data.SkillSlots]int32{0, tc.xp}
			f := skillRankLoadFront(t, levels)
			raw := skillRankLegacySave(t, f, levels, effective, xp, false)
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal(err)
			}
			if err := bindCurrentPartyRecords(a, &doc); err != nil {
				t.Fatal(err)
			}
			rules, err := sim.NewRules(sim.RulesParams{SkillCap: tc.cap})
			if err != nil {
				t.Fatal(err)
			}
			f.Table.Rules = rules
			state := &currentPartyState{Native: true, Class: 1, SkillXP: xp, Book: sim.Spellbook{State: sim.BookAbsent},
				Saved: &mapload.Saved{HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100}}
			if tc.bonus != 0 {
				item := sim.PlainItem(eqShieldCode)
				item.Effects = []sim.ItemEffect{{Kind: 33, Operand: uint32(tc.bonus)}}
				state.Equipment[1] = item
			}
			member, err := a.Party[0].restoreFromState(state, f.Table)
			if err != nil {
				t.Fatal(err)
			}
			if member.Hero.Skill[1] != tc.want || member.Carry.SkillXP != xp {
				t.Fatalf("shared member repaired base/XP = %d/%v, want %d/%v", member.Hero.Skill[1], member.Carry.SkillXP, tc.want, xp)
			}
			loadout := mapload.PartyLoadout(member, f.Table)
			mapload.ApplyItemEffects(&loadout, mapload.MemberItemEquipment(member, f.Table), member.Profile.Fighter)
			d := member.Hero.RecomputeWithSkillXP(member.Profile, loadout, xp)
			if d.Skill[1] != tc.want+tc.bonus {
				t.Fatalf("shared member effective Fire = %d, want %d", d.Skill[1], tc.want+tc.bonus)
			}
		})
	}
}

func TestSkillRankLegacyMemberWithoutBaseRepairsNativeTraining(t *testing.T) {
	levels := [data.SkillSlots]int32{1, 22, 1, 1, 1, 1}
	xp := [data.SkillSlots]int32{0, 171859}
	f := skillRankLoadFront(t, levels)
	effective := levels
	effective[1] = 55
	raw := skillRankLegacySave(t, f, levels, effective, xp, false)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal(err)
	}
	if err := bindCurrentPartyRecords(a, &doc); err != nil {
		t.Fatal(err)
	}
	p := a.Party[0]
	member := f.live.mission.state.Party[0]
	p.Member, p.Policy, p.Base = &member, nil, nil
	state := &currentPartyState{Native: true, Class: 1, SkillXP: xp, Book: sim.Spellbook{State: sim.BookAbsent},
		Saved: &mapload.Saved{HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100}}
	got, err := p.restoreFromState(state, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hero.Skill[1] != 55 || got.Carry == nil || got.Carry.SkillXP != xp {
		t.Fatalf("legacy native member base/XP = %d/%+v, want 55/%v", got.Hero.Skill[1], got.Carry, xp)
	}
	a.Party[0] = p
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentEffectSave(t, f, raw)
	hero := skillRankLoadedHero(t, cold)
	if !hero.NativeTraining.Present || hero.NativeTraining.Levels[1] != 55 || hero.Skill[1] != 55 || hero.SkillXP != xp {
		t.Fatalf("legacy member World trained/effective/XP = %+v/%v/%v, want 55/55/%v", hero.NativeTraining, hero.Skill, hero.SkillXP, xp)
	}
}

func skillRankCityDocument(t *testing.T, party []mapload.PartyMember, table *mapload.Table) (sav.DocumentData, *currentActionData) {
	t.Helper()
	city, err := nativeCityData(party, nil, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	value, err := sav.CityFromData(city)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := value.DocumentData()
	if err != nil {
		t.Fatal(err)
	}
	a := &currentActionData{Version: 1}
	actors := city.Objects[city.Players[0]-1].Player.Groups[0].Actors
	if err := projectCurrentCityParty(&doc, party, actors, a, table); err != nil {
		t.Fatal(err)
	}
	return doc, a
}

func TestSkillRankCityLoadRepairsNativeBase(t *testing.T) {
	levels := [data.SkillSlots]int32{1, 22, 1, 1, 1, 1}
	f := skillRankLoadFront(t, levels)
	member := f.live.mission.state.Party[0]
	member.KnownSpells, member.Book = 0, sim.Spellbook{State: sim.BookAbsent}
	member.SpellbookPresent, member.SpellbookRestored = false, true
	doc, a := skillRankCityDocument(t, []mapload.PartyMember{member}, f.Table)
	if len(a.Bindings) != 1 {
		t.Fatal("synthetic city has no unique hero")
	}
	actor := &doc.Objects[a.Bindings[0].Object-1]
	binary.LittleEndian.PutUint16(modBlock(actor, "UA6")[4:], 55)
	binary.LittleEndian.PutUint16(modBlock(actor, "U114")[4:], 22)
	binary.LittleEndian.PutUint32(modBlock(actor, "H1CC")[4:], 171859)
	for cycle := range 2 {
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		doc, err = sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err = readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		party, _, current, err := restoreCurrentCityParty(&doc, a, f.Table)
		if err != nil || !current || len(party) != 1 {
			t.Fatal("synthetic city LOAD", err)
		}
		if party[0].Hero.Skill[1] != 55 || party[0].Carry == nil || party[0].Carry.SkillXP[1] != 171859 {
			t.Fatalf("cycle %d city trained Fire/XP = %d/%+v, want 55/171859", cycle, party[0].Hero.Skill[1], party[0].Carry)
		}
		d, _, _ := mapload.PartyDisplayWithTable(party[0], f.Table)
		if d.Skill[1] != 55 {
			t.Fatalf("cycle %d city effective Fire = %d, want 55", cycle, d.Skill[1])
		}
		actor = &doc.Objects[a.Bindings[0].Object-1]
		binary.LittleEndian.PutUint16(modBlock(actor, "U114")[4:], uint16(party[0].Hero.Skill[1]))
	}
}

func TestSkillRankDetachedNativeTemplateRepairsWithoutWorldActor(t *testing.T) {
	levels := [data.SkillSlots]int32{1, 1, 1, 1, 17, 1}
	f := skillRankLoadFront(t, levels)
	member := f.live.mission.state.Party[0]
	member.Carry = &mapload.Carry{SkillXP: [data.SkillSlots]int32{0, 0, 0, 0, 355426, 0}}
	p, err := captureCurrentPartyTemplate(777, member, member, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	world := emptyTemplateWorld(t)
	before := world.Hash()
	got, err := p.restoreFromCurrent(world, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	if got.Hero.Skill[4] != 62 || got.Carry == nil || got.Carry.SkillXP[4] != 355426 {
		t.Fatalf("detached template base/XP = %d/%+v, want 62/355426", got.Hero.Skill[4], got.Carry)
	}
	if len(world.Entities()) != 0 || world.Hash() != before {
		t.Fatal("detached member repair changed World")
	}
}
