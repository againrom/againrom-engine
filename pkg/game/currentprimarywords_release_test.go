package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseCurrentHeroPrimaryWordsReplaceRetainedDocument(t *testing.T) {
	for _, field := range []string{"Reaction", "Mind", "Spirit"} {
		t.Run(field, func(t *testing.T) {
			f := releaseFront(t)
			openSkillMission(t, f, 90, skillCaseParty(f, skillCases()[0]))
			f, id := coldMission(t, skillExport(t, f, true, "mission SAVE"))
			before, _ := f.live.world.Entity(id)
			member := &f.live.mission.party[0]
			switch field {
			case "Reaction":
				member.Hero.Reaction += 2
			case "Mind":
				member.Hero.Mind += 2
			case "Spirit":
				member.Hero.Spirit += 2
			}
			rotation := mapload.RotationSpeedBase(member.Hired(), member.HiredRotationSpeed, member.Class, f.Table)
			if _, ok := Rearm(f.live.world, id, member.Hero, member.Profile, member.Weapon, member.WeaponMaterialized, f.Table, rotation); !ok {
				t.Fatal("current hero input did not reach production recomputation")
			}
			current, _ := f.live.world.Entity(id)
			word := func(e sim.Entity) uint16 {
				switch field {
				case "Reaction":
					return uint16(e.Reaction)
				case "Mind":
					return uint16(e.Mind)
				default:
					return uint16(e.Spirit)
				}
			}
			want := word(current)
			if want == word(before) || current.ActorLoad.Source.Class != 0 {
				t.Fatal("fixture has no changed class0 current word", field, want)
			}
			for _, b := range f.live.mission.state.savedDocument.Actors {
				if b.EntityID == id {
					mustSetValue(&f.live.mission.state.savedDocument.Document.Objects[b.ObjectIndex-1], field, uint32(word(before)))
				}
			}
			for cycle := range 2 {
				raw := skillExport(t, f, true, "mission SAVE")
				hero := savHero(t, raw)
				at := map[string]int{"Reaction": 1, "Mind": 2, "Spirit": 3}[field]
				if hero.stats[at] != want {
					t.Fatal("retained Document replaced current primary word", field, cycle, hero.stats[at], want)
				}
				f, id = coldMission(t, raw)
				f.LiveAdvance(2)
				got, _ := f.live.world.Entity(id)
				if word(got) != want {
					t.Fatal("cold LOAD or following ticks changed current primary word", field, cycle, word(got), want)
				}
			}
		})
	}
}

func TestCurrentManaFloorOverridesRetainedPlayerAndNextSave(t *testing.T) {
	f := participantFront(t)
	if !f.live.world.ImportAutoHealing(sim.SelfSlot, 95) || !f.live.world.SetAutoHealing(sim.SelfSlot, 1) {
		t.Fatal("legal mana-floor mode was not admitted")
	}
	if value, present := f.live.world.AutoHealing(sim.SelfSlot); !present || value != 50 {
		t.Fatal("current mana-floor producer did not write50", value, present)
	}
	var key uint32
	for _, b := range f.live.mission.state.savedDocument.GroupBindings.Players {
		r := &f.live.mission.state.savedDocument.Document.Objects[b.ObjectIndex-1]
		if slot, _ := savedStructureValue(r, "Slot"); slot == 1 {
			key, _ = savedStructureValue(r, "This")
			mustSetValue(r, "F58", 95)
		}
	}
	if key == 0 {
		t.Fatal("fixture lacks an exact self Player root")
	}
	for cycle := range 2 {
		s, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, "current mana-floor")
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, r := range doc.Objects {
			if r.Class == "Player" && actorProjectionValue(t, r, "This") == key {
				found = true
				if actorProjectionValue(t, r, "F58") != 50 {
					t.Fatal("retained Player F58 replaced current mana-floor", cycle)
				}
			}
		}
		if !found {
			t.Fatal("exact Player root disappeared")
		}
		cold := participantFront(t)
		open, town, err := cold.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("mana-floor cold LOAD failed", town, err)
		}
		if err := cold.App("mana-floor continuation").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		cold.live.tick()
		if value, present := cold.live.world.AutoHealing(sim.SelfSlot); !present || value != 50 {
			t.Fatal("cold LOAD or next tick lost current mana-floor", value, present)
		}
		f = cold
	}
}
