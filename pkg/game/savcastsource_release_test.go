package game

import (
	"encoding/binary"
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// castsWithoutSource names every actor a document writes with a cast in
// progress and no Spell for it: order progress 2 with neither U64 nor U68,
// or a Weapon borrow in U68 without U64.
func castsWithoutSource(doc sav.DocumentData) []string {
	var found []string
	for i := range doc.Objects {
		r := &doc.Objects[i]
		state, errState := savedMotionRaw(r, "U54", 4)
		phase, errPhase := savedMotionRaw(r, "U58", 4)
		order, errOrder := savedMotionRaw(r, "U158", 148)
		if r.Class != "Human" && r.Class != "Unit" || errState != nil || errPhase != nil || errOrder != nil {
			continue
		}
		spell, _ := savedStructureValue(r, "U64")
		borrow := uint16(0)
		if refs, _ := savedObjectRefs(r, "U68"); len(refs) == 1 {
			borrow = refs[0]
		}
		switch {
		case borrow != 0 && doc.Objects[borrow-1].Class == "Weapon" && spell == 0:
			found = append(found, fmt.Sprintf("object %d borrows a Weapon without U64", i+1))
		case spell != 0 || borrow != 0:
		case order[9] == 2:
			found = append(found, fmt.Sprintf("object %d state %#x phase %d order progress 2 without U64", i+1, binary.LittleEndian.Uint32(state), binary.LittleEndian.Uint32(phase)))
		}
	}
	return found
}

// TestReleaseEverySaveOfABookCastWritesItsSpell: on mission 101 the hero
// learns Teleport from the shipped Astral book in the mission and casts it. A
// menu SAVE on every tick from the order through the recovery writes no actor
// whose cast lacks its Spell (DIV-3356), and while the hero's state names the
// cast, U64 is the completed key of the Spell whose S08 is 26 (DIV-3330).
func TestReleaseEverySaveOfABookCastWritesItsSpell(t *testing.T) {
	f, _ := firstGameTeleportFront(t)
	mw := f.live
	casterID := mw.mission.ids[0]
	caster := releaseEntity(t, mw, casterID)
	m := mw.mission.state.Map
	planes := mapload.Planes(m, f.Table)
	x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y, 5)
	if !found {
		t.Fatal("installed mission has no free Teleport destination five cells away")
	}
	mw.attackOrCast(uint32(casterID), 0, 26, int(x), int(y), true)
	casting := 0
	for k := 0; k < 30; k++ {
		mw.tick()
		raw, stage, err := writerCensusSave(f, true, t.TempDir())
		if err != nil {
			t.Fatal(stage, err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if bad := castsWithoutSource(doc); len(bad) != 0 {
			t.Fatalf("SAVE on tick %d: %v", mw.world.Tick(), bad)
		}
		ms, _, err := loadOriginalMission(f, raw)
		if err != nil {
			t.Fatalf("LOAD of the tick %d SAVE: %v", mw.world.Tick(), err)
		}
		for _, b := range ms.savedDocument.Actors {
			if b.EntityID != casterID {
				continue
			}
			r := &ms.savedDocument.Document.Objects[b.ObjectIndex-1]
			state, _ := savedMotionRaw(r, "U54", 4)
			if code := binary.LittleEndian.Uint32(state); code != 0xd && code != 0xe {
				continue
			}
			casting++
			key, _ := savedStructureValue(r, "U64")
			spell := uint32(0)
			for i := range ms.savedDocument.Document.Objects {
				s := &ms.savedDocument.Document.Objects[i]
				if this, err := savedStructureValue(s, "This"); s.Class == "Spell" && err == nil && this == key {
					spell, _ = savedStructureValue(s, "S08")
				}
			}
			if key == 0 || spell != 26 {
				t.Fatalf("SAVE on tick %d: caster state %#x, U64 %#x names Spell %d, want Teleport 26", mw.world.Tick(), state, key, spell)
			}
		}
	}
	if casting == 0 {
		t.Fatal("no SAVE wrote the hero in a cast state")
	}
	t.Logf("%d of 30 SAVEs wrote the hero casting", casting)
}

// TestReleaseOriginalBookRecoveryHoldsTheCaster: an original document whose
// caster stands in book recovery (state 0xe, phase 7, countdown 6 in U6C)
// loads with that countdown, so the caster starts no cast before it ends
// (MAGIC-CADENCE-126, DIV-3357). The document is the hero's Teleport written
// by SAVE in its charge and turned to recovery; the original action reader
// reads it.
func TestReleaseOriginalBookRecoveryHoldsTheCaster(t *testing.T) {
	f, _ := firstGameTeleportFront(t)
	mw := f.live
	casterID := mw.mission.ids[0]
	caster := releaseEntity(t, mw, casterID)
	m := mw.mission.state.Map
	planes := mapload.Planes(m, f.Table)
	x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y, 5)
	if !found {
		t.Fatal("installed mission has no free Teleport destination five cells away")
	}
	mw.attackOrCast(uint32(casterID), 0, 26, int(x), int(y), true)
	for k := 0; k < 30; k++ {
		mw.tick()
		raw, stage, err := writerCensusSave(f, true, t.TempDir())
		if err != nil {
			t.Fatal(stage, err)
		}
		ms, _, err := loadOriginalMission(f, raw)
		if err != nil {
			t.Fatal(err)
		}
		recovering := false
		for _, b := range ms.savedDocument.Actors {
			r := &ms.savedDocument.Document.Objects[b.ObjectIndex-1]
			state, errState := savedMotionRaw(r, "U54", 4)
			phase, errPhase := savedMotionRaw(r, "U58", 4)
			if b.EntityID == casterID && errState == nil && errPhase == nil && binary.LittleEndian.Uint32(state) == 0xe {
				binary.LittleEndian.PutUint32(phase, 7)
				savedObjectSetValue(r, "U6C", 6)
				recovering = true
			}
		}
		if !recovering {
			continue
		}
		if err := importSavedActorActions(ms, ms.savedDocument); err != nil {
			t.Fatal(err)
		}
		if e, _ := ms.World.Entity(casterID); e.CastWait != 6 {
			t.Fatalf("recovery countdown 6 read as CastWait %d", e.CastWait)
		}
		return
	}
	t.Fatal("no SAVE wrote the hero casting")
}
