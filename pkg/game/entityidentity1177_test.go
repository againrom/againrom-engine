package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestGuardedCharacterLostDoesNotMisreadAReusedIdAsTheDepartedMembersDeath(t *testing.T) {
	world := func(occupant *sim.Entity) *mapWorld {
		t.Helper()
		entities := []sim.Entity{{ID: 1, Owner: sim.SelfSlot, HP: 10, MaxHP: 10}}
		if occupant != nil {
			entities = append(entities, *occupant)
		}
		w := heroWorld(t, nil, nil, entities)
		return partyWorld(t, w, townCompanionParty(), []sim.EntityID{1, 2})
	}

	// A Ghost — never a persist-band TypeID (raisedGhost sources it from the
	// world's declared GhostTemplate, not from the encoded byte form; a
	// creature class row sits well outside sim.PersistLow..PersistHigh) —
	// reoccupies id 2 after the companion it belonged to already departed.
	reused := world(&sim.Entity{ID: 2, Owner: sim.SelfSlot, HP: -10001, MaxHP: 50, TypeID: 5})
	reused.mission.departed = map[sim.EntityID]bool{2: true}
	if reused.guardedCharacterLost() {
		t.Fatal("an unrelated summon reusing the departed companion's id read as that companion's death")
	}
	if !reused.mission.departed[2] {
		t.Fatal("the exemption for the departed companion's own id was cleared by a reused-id occupant")
	}

	// The departed companion herself genuinely returning — a persist-band
	// TypeID, exactly as a minted party member or Hero-flagged NPC carries —
	// must still clear the exemption and count as a loss when dead.
	returned := world(&sim.Entity{ID: 2, Owner: sim.SelfSlot, HP: -10001, MaxHP: 10, TypeID: sim.HumanTypeID, Humanoid: true})
	returned.mission.departed = map[sim.EntityID]bool{2: true}
	if !returned.guardedCharacterLost() {
		t.Fatal("a currently owned dead character inherited an exemption meant for a reused id")
	}
	if returned.mission.departed[2] {
		t.Fatal("the exemption survived a genuine returned party actor's own death")
	}
}

// TestReserveResidueEntityIDsCoversWhatWorldObservationCannot is the review
// pass's own D-1: a repair rewrite that widens a save's world half
// through sim.UpgradeSaveForm alone, which reserves only what a decoded
// World itself observes (World.reserveObservableEntityIDs: live entities,
// originalDead records, script references). This is the residue half in
// isolation: an id present only in residue is reserved, and a floor already
// above every residue id is left alone (reservation is a monotonic max,
// sim.World.ReserveEntityIDs).
func TestReserveResidueEntityIDsCoversWhatWorldObservationCannot(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64),
		[]sim.Entity{{ID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if next, ok := w.NextEntityID(); !ok || next != 2 {
		t.Fatalf("NextEntityID before reservation = %d, %v, want 2, true", next, ok)
	}

	residue := SnapshotResidue{DepartedCharacters: []uint32{5}, Commanded: []uint32{3}}
	ReserveResidueEntityIDs(w, residue)

	next, ok := w.NextEntityID()
	if !ok {
		t.Fatal("NextEntityID reports the namespace exhausted")
	}
	if next != 6 {
		t.Fatalf("NextEntityID after reserving residue ids {3, 5} = %d, want 6: a later summon "+
			"could reoccupy a departed companion's own id", next)
	}

	before := next
	ReserveResidueEntityIDs(w, SnapshotResidue{DepartedCharacters: []uint32{2}})
	if after, ok := w.NextEntityID(); !ok || after != before {
		t.Fatalf("NextEntityID after reserving an already-covered id = %d, %v, want %d, true", after, ok, before)
	}
}

func TestReleaseInstalledGhostRowStaysOutsidePersistBand(t *testing.T) {
	f := releaseFront(t)
	idx := data.NotFound
	for i := 1; i < f.Table.Units.Len(); i++ {
		if f.Table.Units.EntryName(i) == "Ghost" {
			idx = i
			break
		}
	}
	if idx == data.NotFound {
		t.Fatal("installed Units table has no Ghost row")
	}
	def, err := data.NewUnitDef("Ghost", f.Table.Units.EntryParams(idx))
	if err != nil {
		t.Fatalf("Ghost row: %v", err)
	}
	if sim.InPersistBand(def.TypeID) {
		t.Fatalf("installed Ghost row's TypeID = %d, inside the persist band [%d, %d): "+
			"guardedCharacterLost's reused-id guard (pkg/game/world.go) assumes a raised Ghost is "+
			"never in this band, and this install no longer holds that",
			def.TypeID, sim.PersistLow, sim.PersistHigh)
	}
}
