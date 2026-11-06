package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// castRecoveryActor is a Human in a book cast's recovery: action 0xd, phase
// 7, order progress 2, the cast Spell in U64 and the remaining wait in U6C.
func castRecoveryActor(identity, target, spell, wait uint32, spells ...uint16) sav.DocumentRecordData {
	r := loadedLinksActor("Human", identity, target, 0, 0, spell, 0xd, 0xc, 0, spells...)
	mustSetRaw(&r, "U58", binary.LittleEndian.AppendUint32(nil, 7))
	savedObjectSetValue(&r, "U6C", wait)
	for i := range r.Raw {
		if r.Raw[i].Name == "U158" {
			r.Raw[i].Bytes[9] = 2
		}
	}
	return r
}

// A SAVE before the World's first tick after a LOAD writes a caster in a book
// cast's recovery as the recovery it was read from: action 0xd, phase 7, the
// cast wait as countdown and its Spell under the new key, so the cast-source
// seal keeps the action a reload reads as the same current motion.
func TestSaveAfterLoadWritesBookCastRecovery(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 4, Y: 4, HP: 40, MaxHP: 40}, {ID: 2, X: 6, Y: 4, HP: 40, MaxHP: 40}})
	if err != nil {
		t.Fatal(err)
	}
	motion := sim.SavedActorMotion{Entity: 1, Position: sim.SavedActorPosition{Cell: 0x404, PackedCell: 0x404, FineX: 128, FineY: 128}, ActorAction: 0xd}
	if err := w.ImportOriginalActorMotions([]sim.SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	actors := []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}}
	loaded := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			castRecoveryActor(0x100, 0x200, 0x701, 6, 3, 4),
			loadedLinksActor("Human", 0x200, 0, 0, 0, 0, 0, 0xc, 0),
			loadedLinksSpell(0x700, 1), loadedLinksSpell(0x701, 2),
		}},
		Actors: append([]SnapshotSAVActor(nil), actors...),
	}
	if err := importSavedActorActions(&Mission{World: w}, loaded); err != nil {
		t.Fatal(err)
	}
	if e, _ := w.Entity(1); e.CastWait != 6 || e.HasAttackTarget || e.PendingOrder.Kind != sim.PendingNone {
		t.Fatalf("control: LOAD did not read the recovery as cast wait 6 alone: %+v", e)
	}
	if motions, _, _, _ := w.SavedActorMotions(); len(motions) != 1 || !motions[0].Current || motions[0].ActorAction != 0xd {
		t.Fatalf("control: the cast action is not the current motion: %+v", motions)
	}
	// The motion producer leaves a current motion's action word as written;
	// the document holds the loaded action and no order words yet.
	state := &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			castRecoveryActor(0x5100, 0, 0, 0, 3, 4),
			loadedLinksActor("Human", 0x5200, 0, 0, 0, 0, 0, 0xc, 0),
			loadedLinksSpell(0x6100, 1), loadedLinksSpell(0x6101, 2),
		}},
		Actors: append([]SnapshotSAVActor(nil), actors...),
	}
	mustSetRaw(&state.Document.Objects[0], "U58", make([]byte, 4))
	if err := projectSavedActorActions(state, w, loaded); err != nil {
		t.Fatal(err)
	}
	sealCastSources(state.Document)
	r := &state.Document.Objects[0]
	word := func(name string) uint32 {
		raw, err := savedMotionRaw(r, name, 4)
		if err != nil {
			t.Fatal(err)
		}
		return binary.LittleEndian.Uint32(raw)
	}
	value := func(name string) uint32 {
		v, err := savedStructureValue(r, name)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	order, err := savedMotionRaw(r, "U158", 148)
	if err != nil {
		t.Fatal(err)
	}
	type words struct{ action, phase, countdown, spell, target uint32 }
	if got, want := (words{word("U54"), word("U58"), value("U6C"), value("U64"), value("U5C")}), (words{0xd, 7, 6, 0x6101, 0x5200}); got != want || order[9] != 2 {
		t.Fatalf("recovery written as %+v progress %d, want %+v progress 2", got, order[9], want)
	}
}
