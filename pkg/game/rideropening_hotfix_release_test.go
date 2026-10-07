package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Mission 71 Lancers move at 28 and the fallen pair's cells stay lit.
func TestReleaseMission71LancerSpeedAndFallenKnightSightThroughSAVAndColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("mission 71 riders").OpenMission(f.MissionOpenerWith(71, hasteParty())); err != nil {
		t.Fatal(err)
	}
	const riderSpeed = 28
	check := func(label string, mw *mapWorld) {
		t.Helper()
		var standing, fallen []sim.Entity
		for _, e := range mw.world.Entities() {
			if e.Owner != sim.SelfSlot || e.TypeID != 21 {
				continue
			}
			if e.Alive() {
				standing = append(standing, e)
			} else {
				fallen = append(fallen, e)
			}
		}
		if len(standing) != 1 || len(fallen) != 2 {
			t.Fatalf("%s: player Lancers standing %d fallen %d, want 1 and 2", label, len(standing), len(fallen))
		}
		for _, e := range append(standing, fallen...) {
			if e.Speed != riderSpeed {
				t.Errorf("%s: Lancer at (%d,%d) speed %d, want %d", label, e.X, e.Y, e.Speed, riderSpeed)
			}
		}
		width := int(mw.world.Bounds().Width)
		sight := mw.world.Sight(sim.SelfSlot)
		for _, e := range fallen {
			if e.Decay >= 3 {
				t.Fatalf("%s: fallen Lancer already at decay stage %d", label, e.Decay)
			}
			at := int(e.Y)*width + int(e.X)
			if sight[at] == 0 || mw.fog.visible[at] == 0 {
				t.Errorf("%s: cell (%d,%d) of a fallen Lancer is dark: sight %d fog %d", label, e.X, e.Y, sight[at], mw.fog.visible[at])
			}
		}
	}
	check("fresh", f.live)
	for i := 0; i < 2*fogPeriod+1; i++ {
		f.live.tick()
	}
	check("fresh after fog refreshes", f.live)

	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := loadLocalLegacySave(t, store, name)
	check("cold LOAD", cold.live)
	for i := 0; i < 2*fogPeriod+1; i++ {
		cold.live.tick()
	}
	check("cold LOAD after fog refreshes", cold.live)
}

// Hired riders (type ids 19 and 21) keep the rider speed term.
func TestReleaseHiredRidersKeepTheRiderSpeedThroughSAVAndColdLoad(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := hasteParty()
	want := map[int32]int32{}
	for _, name := range []string{"NPC09_1", "NPC08_1"} {
		row := data.FindHumanByName(f.Table.Humans, name)
		if row == data.NotFound {
			t.Fatalf("Humans row %s missing", name)
		}
		def, err := data.NewHumanDef(name, f.Table.Humans.EntryParams(row))
		if err != nil {
			t.Fatal(err)
		}
		dir, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
		want[def.Reaction] = def.Reaction/5 + 12 + 10
		party = append(party, mapload.PartyMember{ID: name, Name: name, Temporary: true, MercenaryType: 8, DefinitionRow: uint8(row),
			Profile: def.Profile(), Hero: def.Hero(), Class: def.TypeID, FigureDir: string(dir), FigureFace: face,
			HiredRotationSpeed: def.RotationSpeed,
			Saved:              &mapload.Saved{Cell: mapload.Cell{X: int32(55 + len(want)), Y: 54}, HP: 1000, MaxHP: 1000}})
	}
	if len(want) != 2 {
		t.Fatalf("rows share a Reaction: %v", want)
	}
	if err := f.App("hired riders").OpenMission(f.MissionOpenerWith(101, party)); err != nil {
		t.Fatal(err)
	}
	check := func(label string, mw *mapWorld) {
		t.Helper()
		found := 0
		for _, e := range mw.world.Entities() {
			if e.Owner == sim.SelfSlot && e.Humanoid && want[e.Reaction] != 0 {
				found++
				if e.Speed != want[e.Reaction] {
					t.Errorf("%s: hired rider with Reaction %d speed %d, want %d", label, e.Reaction, e.Speed, want[e.Reaction])
				}
			}
		}
		if found != 2 {
			t.Fatalf("%s: found %d hired riders, want 2", label, found)
		}
	}
	check("fresh", f.live)
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	check("after SAVE", f.live)
	check("cold LOAD", loadLocalLegacySave(t, store, name).live)
}
