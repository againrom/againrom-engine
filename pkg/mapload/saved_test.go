package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestASavedMemberStandsWhereTheFilePutHim. The drop cell is where a FRESH
// party begins, and a resumed character does not begin: the map's own
// authorised start says nothing about where he was standing when the save
// was taken.
//
// THE FIXTURE'S SAVED CELL IS NOT THE DROP CELL AND NOT ADJACENT TO IT, so a
// member placed by the drop walk — which puts the hero on the drop cell exactly
// and everyone else within a few steps of it — cannot land on this one by
// accident.
func TestASavedMemberStandsWhereTheFilePutHim(t *testing.T) {
	drop := mapload.Cell{X: 17, Y: 20}
	m := startMap(t, 60, 60, drop)
	saved := mapload.Cell{X: 44, Y: 51}
	w, st := mustStart(t, m, []mapload.PartyMember{{
		Class: 100,
		Saved: &mapload.Saved{Cell: saved, HP: 107, MaxHP: 131, Mana: 30, MaxMana: 90},
	}})
	if st.Drop != drop {
		t.Fatalf("the start's drop cell is %v, want the map's own %v", st.Drop, drop)
	}
	if st.Cells[0] != saved {
		t.Errorf("the member was placed at %v, want the save's own %v", st.Cells[0], saved)
	}
	got := lastEntity(t, w)
	if got.X != saved.X || got.Y != saved.Y {
		t.Errorf("the entity stands at (%d,%d), want (%d,%d)", got.X, got.Y, saved.X, saved.Y)
	}
}

func TestASavedMemberWithNoStatedCellTakesTheDrop(t *testing.T) {
	drop := mapload.Cell{X: 17, Y: 20}
	m := startMap(t, 60, 60, drop)
	pools := func() *mapload.Saved {
		return &mapload.Saved{HP: 107, MaxHP: 131, Mana: 30, MaxMana: 90}
	}
	w, st := mustStart(t, m, []mapload.PartyMember{
		{Class: 100, Saved: pools()},
		{Class: 100, Saved: pools()},
	})
	if st.Drop != drop {
		t.Fatalf("the start's drop cell is %v, want the map's own %v", st.Drop, drop)
	}
	if st.Cells[0] != drop {
		t.Errorf("member 0 was placed at %v, want the map's own drop %v", st.Cells[0], drop)
	}
	for i, c := range st.Cells {
		if c == (mapload.Cell{}) {
			t.Errorf("member %d was placed at the zero cell; the cleared position was read as a stated one", i)
		}
		if dx, dy := c.X-drop.X, c.Y-drop.Y; dx*dx+dy*dy > 25 {
			t.Errorf("member %d was placed at %v, want a cell within the drop walk of %v", i, c, drop)
		}
	}
	// The pools half still arrived: the member is the file's wounded character
	// and not the fold's full-health one.
	got := lastEntity(t, w)
	if got.HP != 107 || got.MaxHP != 131 {
		t.Errorf("the member arrived at %d/%d health, want the file's own 107/131", got.HP, got.MaxHP)
	}
}

// TestASavedMembersPoolsAreTheFilesAndNotTheFolds. PartySpawn folds one
// health number out of the profile and the mint writes it into both HP and
// MaxHP, so every party member this tree has ever built arrived at full
// health. A resumed character wounded to a third of his health has to arrive
// wounded, and that is a state the fold cannot express at all.
func TestASavedMembersPoolsAreTheFilesAndNotTheFolds(t *testing.T) {
	m := startMap(t, 60, 60, mapload.Cell{X: 17, Y: 20})
	s := &mapload.Saved{
		Cell: mapload.Cell{X: 30, Y: 31},
		HP:   107, MaxHP: 131, Mana: 30, MaxMana: 90,
		HealthRegenPeriod: 77, ManaRegenPeriod: 33,
	}
	// A member carrying a REAL profile and a hero, so the fold produces numbers
	// of its own for the file's to be visibly different from.
	w, _ := mustStart(t, m, []mapload.PartyMember{{
		Class:   100,
		Hero:    data.NewCampaignHero(data.SkillBlade),
		Profile: data.Profile{HealthColumn: true, ManaColumn: true},
		Saved:   s,
	}})
	got := lastEntity(t, w)
	if got.HP != s.HP || got.MaxHP != s.MaxHP {
		t.Errorf("health %d/%d, want the file's %d/%d", got.HP, got.MaxHP, s.HP, s.MaxHP)
	}
	if got.HP == got.MaxHP {
		t.Error("the restored member arrived at full health, which is the fold's answer " +
			"and not the file's")
	}
	if got.Mana != s.Mana || got.MaxMana != s.MaxMana {
		t.Errorf("mana %d/%d, want the file's %d/%d", got.Mana, got.MaxMana, s.Mana, s.MaxMana)
	}
	if got.HealthRegenPeriod != s.HealthRegenPeriod || got.ManaRegenPeriod != s.ManaRegenPeriod {
		t.Errorf("periods %d/%d, want the file's %d/%d", got.HealthRegenPeriod,
			got.ManaRegenPeriod, s.HealthRegenPeriod, s.ManaRegenPeriod)
	}
	def := data.UnitDefaults()
	if got.HealthRegenPeriod == def.HealthRegenPeriod || got.ManaRegenPeriod == def.ManaRegenPeriod {
		t.Error("a period is still the base constructor's, so the file's is not being read")
	}
}

// TestANilSavedChangesNothing. The whole party this tree builds carries no Saved,
// and the point of the pointer is that such a member takes exactly the arm he
// always took — the drop cell, the fold's own health on both fields, and the base
// constructor's two periods.
func TestANilSavedChangesNothing(t *testing.T) {
	drop := mapload.Cell{X: 17, Y: 20}
	m := startMap(t, 60, 60, drop)
	p := mapload.PartyMember{Class: 100, Hero: data.NewCampaignHero(data.SkillBlade)}
	w, st := mustStart(t, m, []mapload.PartyMember{p})
	if st.Cells[0] != drop {
		t.Errorf("a member with no Saved was placed at %v, want the drop cell %v", st.Cells[0], drop)
	}
	got := lastEntity(t, w)
	if got.HP != mapload.SpawnHP || got.MaxHP != mapload.SpawnHP {
		t.Errorf("health %d/%d, want the provisional %d on both",
			got.HP, got.MaxHP, mapload.SpawnHP)
	}
	def := data.UnitDefaults()
	if got.HealthRegenPeriod != def.HealthRegenPeriod || got.ManaRegenPeriod != def.ManaRegenPeriod {
		t.Errorf("periods %d/%d, want the constructor's %d/%d", got.HealthRegenPeriod,
			got.ManaRegenPeriod, def.HealthRegenPeriod, def.ManaRegenPeriod)
	}
}

// TestASavedCellIsTakenSoTheRestOfThePartyWalksAwayFromIt. A member with no Saved
// beside members that have one still walks, and the walk must treat their cells as
// occupied: two entities on one cell is a state the start's own crowding counter
// exists to avoid.
func TestASavedCellIsTakenSoTheRestOfThePartyWalksAwayFromIt(t *testing.T) {
	drop := mapload.Cell{X: 17, Y: 20}
	m := startMap(t, 60, 60, drop)
	held := mapload.Cell{X: 18, Y: 20}
	_, st := mustStart(t, m, []mapload.PartyMember{
		{Class: 100, Saved: &mapload.Saved{Cell: held}},
		{Class: 100},
		{Class: 100},
	})
	seen := map[mapload.Cell]int{}
	for i, c := range st.Cells {
		seen[c]++
		if seen[c] > 1 {
			t.Errorf("member %d shares cell %v with an earlier member", i, c)
		}
	}
	if st.Cells[0] != held {
		t.Errorf("the saved member moved to %v", st.Cells[0])
	}
}

// TestASavedMemberStillWearsAndCarriesWhatHeWasHanded. A Saved decides the cell
// and the pools and NOTHING ELSE: the worn set and the pack are PartyMember's own
// fields and reach the world through the same Stock entry a generated member's
// do, so a restored character's equipment must not take a different path.
func TestASavedMemberStillWearsAndCarriesWhatHeWasHanded(t *testing.T) {
	m := startMap(t, 60, 60, mapload.Cell{X: 17, Y: 20})
	var worn [sim.EquipSlots]uint16
	worn[0] = 0x1106
	worn[6] = 0xb70f
	w, st := mustStart(t, m, []mapload.PartyMember{{
		Class:   100,
		Worn:    worn,
		Carried: []uint16{0x0e1c},
		Saved:   &mapload.Saved{Cell: mapload.Cell{X: 30, Y: 31}, HP: 5, MaxHP: 9},
	}})
	id := st.IDs[0]
	eq, ok := w.Equipped(id)
	if !ok || eq != worn {
		t.Errorf("equipment reads %v (present %v), want %v", eq, ok, worn)
	}
	items, ok := w.Carried(id)
	if !ok || len(items) != 1 || items[0] != 0x0e1c {
		t.Errorf("the pack reads %v (present %v), want one item 0x0e1c", items, ok)
	}
}
