package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// loadedLinksActor is an actor record with the order words and action links
// the loaded-link carry reads.
func loadedLinksActor(class string, identity, u5c, u60, u61, u64, action, state uint32, inner byte, spells ...uint16) sav.DocumentRecordData {
	order := make([]byte, 148)
	order[8] = inner
	r := sav.DocumentRecordData{Class: class,
		Values: []sav.DocumentValueData{{Name: "Identity", Value: identity}, {Name: "Stage", Value: 0},
			{Name: "U5C", Value: u5c}, {Name: "U60", Value: u60}, {Name: "U61", Value: u61}, {Name: "U64", Value: u64},
			{Name: "U6C", Value: 0}, {Name: "U40", Value: 0}, {Name: "U48", Value: 0}, {Name: "U136", Value: 0}},
		Raw: []sav.DocumentRawData{
			{Name: "U54", Bytes: binary.LittleEndian.AppendUint32(nil, action)},
			{Name: "U58", Bytes: []byte{0, 0, 0, 0}},
			{Name: "U50", Bytes: binary.LittleEndian.AppendUint32(nil, state)},
			{Name: "U158", Bytes: order},
		},
	}
	if len(spells) != 0 {
		r.RefSlots = []sav.DocumentRefsData{{Name: "Spells", Objects: spells}}
	}
	return r
}

func loadedLinksSpell(this, id uint32) sav.DocumentRecordData {
	return sav.DocumentRecordData{Class: "Spell", Values: []sav.DocumentValueData{{Name: "This", Value: this}, {Name: "S08", Value: id}}}
}

// loadedLinksFixture is a loaded document and the document a SAVE writes
// after its LOAD: every actor and Spell has a new key, and the writer has
// not yet written the action links. Human 1 names Human 2, Human 2 names
// actor 4, which has died and has no record in the new document, and Unit 3
// names itself and the second Spell of its book.
func loadedLinksFixture(t *testing.T) (loaded, state *SnapshotSAVDocument, w *sim.World) {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 4, Y: 4, HP: 40, MaxHP: 40}, {ID: 2, X: 6, Y: 4, HP: 40, MaxHP: 40},
		{ID: 3, X: 8, Y: 4, HP: 40, MaxHP: 40}, {ID: 4, X: 10, Y: 4, HP: -57, MaxHP: 40, Decay: sim.DecayFallen, Dwell: 3}})
	if err != nil {
		t.Fatal(err)
	}
	loaded = &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			loadedLinksActor("Human", 0x100, 0x200, 59, 48, 0, 4, 0xc, 0),
			loadedLinksActor("Human", 0x200, 0x400, 0, 0, 0, 4, 0xc, 0),
			loadedLinksActor("Unit", 0x300, 0x300, 0, 0, 0x701, 0x1a, 0xb, 11, 5, 6),
			loadedLinksActor("Unit", 0x400, 0, 0, 0, 0, 0, 0, 0),
			loadedLinksSpell(0x700, 1), loadedLinksSpell(0x701, 2),
		}},
		Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}, {EntityID: 3, ObjectIndex: 3}, {EntityID: 4, ObjectIndex: 4}},
	}
	if err := importSavedActorActions(&Mission{World: w}, loaded); err != nil {
		t.Fatal(err)
	}
	state = &SnapshotSAVDocument{
		Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
			loadedLinksActor("Human", 0x5100, 0, 0, 0, 0, 4, 0xc, 0),
			loadedLinksActor("Human", 0x5200, 0, 0, 0, 0, 4, 0xc, 0),
			loadedLinksActor("Unit", 0x5300, 0, 0, 0, 0, 0x1a, 0xb, 11, 4, 5),
			loadedLinksSpell(0x6100, 1), loadedLinksSpell(0x6101, 2),
		}},
		Actors: []SnapshotSAVActor{{EntityID: 1, ObjectIndex: 1}, {EntityID: 2, ObjectIndex: 2}, {EntityID: 3, ObjectIndex: 3}, {EntityID: 4, Retired: true}},
	}
	return loaded, state, w
}

func loadedLinksOf(t *testing.T, state *SnapshotSAVDocument, index int) [4]uint32 {
	t.Helper()
	var out [4]uint32
	for i, name := range []string{"U5C", "U60", "U61", "U64"} {
		v, err := savedStructureValue(&state.Document.Objects[index], name)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = v
	}
	return out
}

// After a LOAD, a SAVE writes the action links of an actor the World runs no
// order for as the loaded record holds them, each address renamed to the key
// its target has in the new document, and 0 for a target the document lacks.
func TestSaveAfterLoadWritesLoadedActionLinksUnderNewKeys(t *testing.T) {
	loaded, state, w := loadedLinksFixture(t)
	for _, e := range w.Entities() {
		if e.HasAttackTarget || e.PendingOrder.Kind != sim.PendingNone {
			t.Fatalf("the World holds an order the fixture does not give: %+v", e)
		}
	}
	if err := projectSavedActorActions(state, w, loaded); err != nil {
		t.Fatal(err)
	}
	if got := loadedLinksOf(t, state, 0); got != [4]uint32{0x5200, 59, 48, 0} {
		t.Errorf("Human naming another actor wrote %#x, want its target's new key and the loaded cell", got)
	}
	if got := loadedLinksOf(t, state, 1); got != [4]uint32{} {
		t.Errorf("Human naming an actor the document lacks wrote %#x, want 0", got)
	}
	if got := loadedLinksOf(t, state, 2); got != [4]uint32{0x5300, 0, 0, 0x6101} {
		t.Errorf("Unit naming itself and its book's Spell wrote %#x, want its own new key and the new Spell key", got)
	}
}

// An actor whose order words are no longer the loaded ones has run an order
// since the LOAD; its loaded links do not stand for that order.
func TestSaveAfterAnOrderKeepsNoLoadedActionLinks(t *testing.T) {
	loaded, state, w := loadedLinksFixture(t)
	mustSetRaw(&state.Document.Objects[0], "U54", binary.LittleEndian.AppendUint32(nil, 1))
	if err := projectSavedActorActions(state, w, loaded); err != nil {
		t.Fatal(err)
	}
	if got := loadedLinksOf(t, state, 0); got != [4]uint32{} {
		t.Errorf("actor with changed order words wrote loaded links %#x", got)
	}
	if got := loadedLinksOf(t, state, 2); got[0] != 0x5300 {
		t.Errorf("another actor's carry changed: %#x", got)
	}
}
