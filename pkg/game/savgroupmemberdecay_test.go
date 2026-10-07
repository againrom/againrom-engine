package game

import (
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A Group member that dies and fully decays between saves must still produce
// a savable, reloadable document: its object has no root left once the live
// world drops it from Group membership unless the dead-actor root producer
// runs early enough to replace that root before any reachability check.
func TestCurrentSaveSurvivesGroupMemberFullDecay(t *testing.T) {
	// partialCurrentGraph's own opening boilerplate reliably admits mission 10;
	// its own s/w carry an imported native SavedGroups span, which routes a
	// later save through projectSavedGroupRoster instead of the live-only
	// projectCurrentGroups rebuild this regression exercises, so only f is kept.
	f, _, _ := partialCurrentGraph(t)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	entities := []sim.Entity{
		{ID: 0, X: 10, Y: 10, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1},
		{ID: 41, X: 11, Y: 11, HP: 29, MaxHP: 29, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1},
	}
	w, err := sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	state, err := f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal(err)
	}
	s.SavedDocument = state
	if s.World, err = w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "before full decay")
	if err != nil {
		t.Fatal(err)
	}
	cold := cellStateFront(t)
	cold.Campaign, cold.Table = f.Campaign, f.Table
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("baseline grouped actor cannot LOAD", err, town)
	}
	if err := cold.App("group member decay").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	const member sim.EntityID = 41
	found := false
	for _, e := range cold.live.world.Entities() {
		found = found || e.ID == member
	}
	if !found {
		t.Fatal("baseline Group member missing right after LOAD")
	}
	cold.LiveKill(uint32(member))
	cold.LiveAdvance(24000)
	w, ok := cold.LiveWorld()
	if !ok {
		t.Fatal("no live world after decay")
	}
	terms := w.CurrentTerminalActors()
	if len(terms) != 1 || terms[0].ID != member {
		t.Fatalf("fixture did not fully decay the Group member as expected: %+v", terms)
	}
	dir := t.TempDir()
	seams := cold.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{})
	prepared, err := seams.Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "after full decay", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("save refused after a Group member's full decay:", err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal("commit refused after a Group member's full decay:", err)
	}
	raw2, err := ReadSaveFile(filepath.Join(dir, "after full decay.sav"))
	if err != nil {
		t.Fatal(err)
	}
	warm := cellStateFront(t)
	warm.Campaign, warm.Table = f.Campaign, f.Table
	open2, town2, err := warm.RestoreOriginal(raw2)
	if err != nil || town2 {
		t.Fatal("post-decay save cannot LOAD", err, town2)
	}
	if err := warm.App("reload after decay").OpenMission(open2); err != nil {
		t.Fatal(err)
	}
	survivorFound := false
	for _, e := range warm.live.world.Entities() {
		if e.ID == member {
			t.Fatal("fully decayed Group member returned as a live entity on reload")
		}
		if e.ID == 0 {
			survivorFound = e.X == 10 && e.Y == 10 && e.HP == 40
		}
	}
	if !survivorFound {
		t.Fatal("the rest of the Group's state did not survive the decayed member's SAVE/LOAD intact")
	}
}

func TestCurrentSavePreservesTornDownActorGroupCoordinate(t *testing.T) {
	f, _, _ := partialCurrentGraph(t)
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	const member sim.EntityID = 41
	w, err := sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 10, Y: 10, HP: 40, MaxHP: 40, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1},
		{ID: member, X: 11, Y: 11, HP: -10, MaxHP: 29, Decay: sim.DecayBones, Owner: sim.SelfSlot, Group: 5, TypeID: 1, Speed: 1, TokenSize: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.SavedDocument, err = f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal(err)
	}
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "torn-down Group member")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("current actor bindings", err)
	}
	var object uint16
	for _, binding := range actions.Bindings {
		if binding.ID == member && !binding.Structure && !binding.Missing {
			object = binding.Object
		}
	}
	if object == 0 || !slices.Contains(doc.DeadActors, object) {
		t.Fatal("torn-down body lacks a dead root", object)
	}
	for _, player := range doc.Players {
		for _, group := range doc.Objects[player-1].Groups {
			actors, _ := savedObjectRefs(&group, "Actors")
			if slices.Contains(actors, object) {
				t.Fatal("torn-down body remained a Group member")
			}
		}
	}
	cold := cellStateFront(t)
	cold.Campaign, cold.Table = f.Campaign, f.Table
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("torn-down body cannot LOAD", err, town)
	}
	if err := cold.App("torn-down Group coordinate").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	for _, e := range cold.live.world.Entities() {
		if e.ID == member {
			if e.Group != 5 || e.Decay != sim.DecayBones {
				t.Fatalf("torn-down actor Group/stage=%d/%d, want 5/%d", e.Group, e.Decay, sim.DecayBones)
			}
			return
		}
	}
	t.Fatal("torn-down actor disappeared on LOAD")
}
