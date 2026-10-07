package game

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// bindingOrderSave writes the live mission through the player's SAV dialog
// under name in a new directory and returns the file's bytes.
func bindingOrderSave(t *testing.T, f *FrontEnd, name string) []byte {
	t.Helper()
	dir := t.TempDir()
	seams := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	prepared, err := seams.Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: name, Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("SAVE refused:", err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal("SAVE commit refused:", err)
	}
	raw, err := ReadSaveFile(filepath.Join(dir, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// bindingOrderFront builds the fixture mission: one living party member and
// bodies further Group members, each lying at the first stage of decay. A
// cold LOAD of its SAV then runs the ordinary ticks until every body has left
// play as a terminal actor, the only actors whose bindings the action
// supplement reaches through the current values alone.
func bindingOrderFront(t *testing.T, bodies int) (*FrontEnd, *sim.World) {
	t.Helper()
	f, _, _ := partialCurrentGraph(t)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	entities := []sim.Entity{{ID: 0, X: 10, Y: 10, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1}}
	for i := 0; i < bodies; i++ {
		entities = append(entities, sim.Entity{ID: sim.EntityID(41 + i), X: int32(11 + i%8), Y: int32(12 + i/8), HP: -10, MaxHP: 29,
			Decay: sim.DecayBones, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1})
	}
	w, err := sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	if s.SavedDocument, err = f.materializeCurrentWorld(s, w); err != nil {
		t.Fatal(err)
	}
	if s.World, err = w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "bodies at the first stage")
	if err != nil {
		t.Fatal(err)
	}
	cold := cellStateFront(t)
	cold.Campaign, cold.Table = f.Campaign, f.Table
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("baseline cannot LOAD", err, town)
	}
	if err := cold.App("binding order").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	cold.LiveAdvance(24000)
	live, ok := cold.LiveWorld()
	if !ok {
		t.Fatal("no live world after decay")
	}
	if got := len(live.CurrentTerminalActors()); got != bodies {
		t.Fatalf("%d terminal actors after the decay ticks, want %d", got, bodies)
	}
	return cold, live
}

// bindingOrderIDs lists the actor bindings of a SAV in file order.
func bindingOrderIDs(t *testing.T, raw []byte) []sim.EntityID {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("action supplement", err)
	}
	var ids []sim.EntityID
	for _, b := range a.Bindings {
		if !b.Structure {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

// TestSaveOfOneStateWritesOneByteSequence saves a mission whose terminal
// actors are reached only through the current values. Each SAVE of that state
// holds the same bytes, and the bindings after the ones the action rows reach
// run in ascending entity ID.
func TestSaveOfOneStateWritesOneByteSequence(t *testing.T) {
	const bodies, saves = 12, 12
	f, w := bindingOrderFront(t, bodies)
	before := w.Hash()
	first := bindingOrderSave(t, f, "one state 0")
	for i := 1; i < saves; i++ {
		next := bindingOrderSave(t, f, "one state 0")
		if !bytes.Equal(first, next) {
			t.Fatalf("SAVE %d of one state differs from SAVE 1; bindings %v then %v", i+1, bindingOrderIDs(t, first), bindingOrderIDs(t, next))
		}
	}
	if w.Hash() != before {
		t.Fatal("SAVE changed the World")
	}
	ids := bindingOrderIDs(t, first)
	var terminal []sim.EntityID
	for _, terminalActor := range w.CurrentTerminalActors() {
		terminal = append(terminal, terminalActor.ID)
	}
	slices.Sort(terminal)
	if len(ids) < bodies || !slices.Equal(ids[len(ids)-bodies:], terminal) {
		t.Fatalf("bindings %v do not end with the terminal actors %v in ascending ID", ids, terminal)
	}
}

// TestLoadIgnoresTheOrderOfActionBindings loads one SAV in the order it was
// written and again with its bindings reversed: both restore one World and
// both write the same SAV again.
func TestLoadIgnoresTheOrderOfActionBindings(t *testing.T) {
	f, w := bindingOrderFront(t, 12)
	written := bindingOrderSave(t, f, "written")
	doc, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || len(a.Bindings) < 5 {
		t.Fatal("action supplement", err)
	}
	slices.Reverse(a.Bindings)
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, raw); err != nil {
		t.Fatal(err)
	}
	reversed, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(written, reversed) {
		t.Fatal("the reversed file equals the written one")
	}
	var rewritten [2][]byte
	var hashes [2]uint64
	for i, file := range [][]byte{written, reversed} {
		cold := cellStateFront(t)
		cold.Campaign, cold.Table = f.Campaign, f.Table
		open, town, err := cold.RestoreOriginal(file)
		if err != nil || town {
			t.Fatal("cold LOAD refused", i, err, town)
		}
		if err := cold.App("binding order LOAD").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		loaded, _ := cold.LiveWorld()
		if !slices.EqualFunc(loaded.CurrentTerminalActors(), w.CurrentTerminalActors(), func(x, y sim.CurrentTerminalActor) bool { return x.ID == y.ID }) {
			t.Fatalf("LOAD %d restored other terminal actors", i)
		}
		hashes[i] = loaded.Hash()
		rewritten[i] = bindingOrderSave(t, cold, "rewritten")
	}
	if hashes[0] != hashes[1] {
		t.Fatal("the reversed file restores another World")
	}
	if !bytes.Equal(rewritten[0], rewritten[1]) {
		t.Fatal("the reversed file writes another SAV after LOAD")
	}
	if got, want := bindingOrderIDs(t, rewritten[0]), bindingOrderIDs(t, written); !slices.Equal(got, want) {
		t.Fatalf("bindings after LOAD %v, written %v", got, want)
	}
}
