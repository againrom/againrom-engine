package game

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// modSkillCapFront is a front end on the lawful install running under a mod set
// that raises the skill cap to cap.
func modSkillCapFront(t *testing.T, cap int32) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	r, err := sim.NewRules(sim.RulesParams{SkillCap: cap})
	if err != nil {
		t.Fatal(err)
	}
	set := mod.Set{Base: BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), Mods: []mod.SetEntry{{ID: "skill-cap", Version: "1.0.0", Digest: "d",
		Settings: []mod.SettingValue{{Key: "skill_cap", Value: mod.Value{Kind: mod.KindInt, Int: int64(cap)}}}}}}
	if err := f.SetMods(r, set, false); err != nil {
		t.Fatal(err)
	}
	return f
}

// skillCapParty is a mage who has reached level 100 in two schools, beside a
// companion to cast at.
func skillCapParty() []mapload.PartyMember {
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	hero.Skill[1], hero.Skill[2] = 100, 100
	return []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 3,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}},
		{ID: "companion", Profile: data.Profile{HealthColumn: true}, Hero: data.Hero{Body: 60, Reaction: 60},
			Saved: &mapload.Saved{Cell: mapload.Cell{X: 54, Y: 55}, HP: 1000, MaxHP: 1000}}}
}

func skillCapMission(t *testing.T, f *FrontEnd) {
	t.Helper()
	app := f.App("skill cap mod")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(101, skillCapParty())); err != nil {
		t.Fatal(err)
	}
}

// skillCapHero is the party's mage in the live world of f.
func skillCapHero(t *testing.T, f *FrontEnd) sim.Entity {
	t.Helper()
	return releaseEntity(t, f.live, f.live.mission.ids[0])
}

func TestReleaseModSkillCapRaisesAHeroPastTheOriginalCapAndTheSaveCarriesIt(t *testing.T) {
	f := modSkillCapFront(t, 150)
	skillCapMission(t, f)
	hero := skillCapHero(t, f)
	school := -1
	for slot := 1; slot <= 2; slot++ {
		if hero.Skill[slot] == 100 {
			school = slot
		}
	}
	if school < 0 {
		t.Fatalf("the hero does not start at level 100: %+v", hero.Skill)
	}
	rule, ok := f.live.world.Spell(3)
	if !ok {
		t.Fatal("installed spell missing")
	}
	school = int(rule.School)
	victim := releaseEntity(t, f.live, f.live.mission.ids[1])
	for n := 0; n < 3 && skillCapHero(t, f).Skill[school] <= 100; n++ {
		for tick := 0; skillCapHero(t, f).CastWait != 0; tick++ {
			if tick > 256 {
				t.Fatal("recovery stuck")
			}
			f.live.tick()
		}
		f.live.attackOrCast(uint32(f.live.mission.ids[0]), 0, 3, int(victim.X), int(victim.Y), true)
		for tick := 0; tick < 64; tick++ {
			f.live.tick()
		}
	}
	raised := skillCapHero(t, f)
	if raised.Skill[school] <= 100 || raised.SkillXP[school] <= (sim.Rules{}).SkillXP(100) {
		t.Fatalf("the award path did not take the skill past 100: level %d xp %d", raised.Skill[school], raised.SkillXP[school])
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || mark.Format != modMarkFormat || len(mark.Mods) != 1 || mark.Mods[0].ID != "skill-cap" || len(mark.Domain) == 0 {
		t.Fatalf("the save does not carry the mod mark: %+v %v %v", mark, present, err)
	}
	trueLevel := int32(0)
	for _, d := range mark.Domain {
		trueLevel = max(trueLevel, d.Levels[school])
		o, _, _ := readModObject(&doc.Objects[d.Object])
		if o.Levels[school] != 100 || o.BaseLevels[school] > 100 || o.XP[school] > uint32((sim.Rules{}).SkillXP(100)) {
			t.Fatalf("the ordinary fields hold %+v, not the original domain", o)
		}
	}
	if trueLevel != raised.Skill[school] {
		t.Fatalf("the mark holds level %d, the hero has %d", trueLevel, raised.Skill[school])
	}

	// A cold LOAD under the same mod restores the true level and experience.
	cold := modSkillCapFront(t, 150)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(err)
	}
	if err := cold.App("cold load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var restored *sim.Entity
	for _, e := range cold.live.world.Entities() {
		if e.Skill[school] > 100 {
			e := e
			restored = &e
		}
	}
	if restored == nil || restored.Skill[school] != raised.Skill[school] || restored.SkillXP[school] != raised.SkillXP[school] {
		t.Fatalf("the cold load did not restore level %d xp %d: %+v", raised.Skill[school], raised.SkillXP[school], restored)
	}

	// The restored world runs under the mod's rules, so the hero still gains
	// experience past the shipped table after the LOAD.
	if got := cold.live.world.Rules().SkillCap(); got != 150 {
		t.Fatalf("the loaded world's skill cap is %d, want 150", got)
	}
	var target sim.Entity
	for _, e := range cold.live.world.Entities() {
		if e.ID != restored.ID && e.Owner == restored.Owner && e.Alive() && e.MaxHP > 0 {
			target = e
		}
	}
	if target.MaxHP == 0 {
		t.Fatal("no companion to cast at in the loaded world")
	}
	before := releaseEntity(t, cold.live, restored.ID).SkillXP[school]
	for n := 0; n < 3 && releaseEntity(t, cold.live, restored.ID).SkillXP[school] == before; n++ {
		for tick := 0; releaseEntity(t, cold.live, restored.ID).CastWait != 0; tick++ {
			if tick > 256 {
				t.Fatal("recovery stuck after the load")
			}
			cold.live.tick()
		}
		cold.live.attackOrCast(uint32(restored.ID), 0, 3, int(target.X), int(target.Y), true)
		for tick := 0; tick < 64; tick++ {
			cold.live.tick()
		}
	}
	if after := releaseEntity(t, cold.live, restored.ID).SkillXP[school]; after <= before {
		t.Fatalf("no experience was awarded after the load: %d then %d", before, after)
	}

	// Saving the restored game again carries the same true state.
	snapshot2, label2, err := cold.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := cold.ExportCurrentSave(snapshot2, label2)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := sav.DecodeDocumentData(raw2)
	if err != nil {
		t.Fatal(err)
	}
	mark2, present, err := readModMark(doc2)
	if err != nil || !present {
		t.Fatal(err)
	}
	again := int32(0)
	for _, d := range mark2.Domain {
		again = max(again, d.Levels[school])
	}
	if again != trueLevel {
		t.Fatalf("the second save carries level %d, the first carried %d", again, trueLevel)
	}

	// A LOAD without the mod refuses and names it.
	plain := releaseFront(t)
	if _, _, err := plain.RestoreOriginal(raw); err == nil || !strings.Contains(err.Error(), "skill-cap") {
		t.Fatalf("a load without the mod: %v", err)
	}
	// A LOAD under a different setting refuses and names the setting.
	if _, _, err := modSkillCapFront(t, 140).RestoreOriginal(raw); err == nil || !strings.Contains(err.Error(), "skill_cap") {
		t.Fatalf("a load under another setting: %v", err)
	}
}

func TestReleaseModSkillCapLeavesAnUnmoddedSaveUntouched(t *testing.T) {
	export := func(f *FrontEnd) []byte {
		t.Helper()
		skillCapMission(t, f)
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := export(f)
	if bytes.Contains(a, modMarkName) {
		t.Fatal("an unmodded save carries a mod mark")
	}
	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	if err := g.SetMods(sim.Rules{}, mod.Set{}, false); err != nil {
		t.Fatal(err)
	}
	if b := export(g); !bytes.Equal(a, b) {
		t.Fatal("an empty mod set changed the bytes of the save")
	}
	// The unmodded save loads without a mod set, as before.
	if _, _, err := releaseFront(t).RestoreOriginal(a); err != nil {
		t.Fatal(err)
	}
}
