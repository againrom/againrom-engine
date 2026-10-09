package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func groupDocumentFront1115(t *testing.T) *FrontEnd {
	t.Helper()
	f := actorRegistryFront1111(t)
	f.Campaign = resolved(saveCampaign(), nil)
	return f
}

// Independent wire construction: repeated Player roots do not create another
// Player/Group; A,null,B,A,C loads as null,B,A,C through ordinary append's
// detach-before-append semantics. These are source archive references, not
// handcrafted Snapshot group bindings.
func groupDocumentLiteral1115(t *testing.T, front *FrontEnd) []byte {
	t.Helper()
	load, runtime := &[4]int16{100, 0, 0, 300}, &[3]byte{1, 8, 4}
	a := &poolFixtureActor{cell: 0x100f, hp: 7, maxHP: 31, name: "Group actor A", loadWords: load, equipmentRuntime: runtime}
	b := &poolFixtureActor{mapID: 900, cell: 0x120f, hp: 9, maxHP: 41, human: true, name: "Group actor B", loadWords: load, equipmentRuntime: runtime}
	c := &poolFixtureActor{mapID: 900, cell: 0x140f, hp: 11, maxHP: 51, human: true, name: "Group actor C", loadWords: load, equipmentRuntime: runtime}
	player := &poolFixturePlayer{groups: [][]*poolFixtureActor{{a, nil, b, a, c}}}
	body := poolFixtureBody([]*poolFixturePlayer{player, nil, player}, nil)
	for i, actor := range []*poolFixtureActor{a, b, c} {
		binary.LittleEndian.PutUint32(body[actor.off+12:], uint32(501+i))
		body[actor.off+16] = 1
		binary.LittleEndian.PutUint16(body[actor.off+17:], []uint16{35, 3, 33}[i])
		body[actor.off+509], body[actor.off+510], body[actor.off+511] = 1, 1, 3
		body[actor.off+179], body[actor.off+189] = 64, 100
	}
	return completeDocumentTail1115(t, front, savedContainer(body))
}

func groupDocumentSnapshot(t *testing.T, front *FrontEnd) Snapshot {
	t.Helper()
	snapshot, _, err := front.Snapshot(true)
	if err != nil || snapshot.SavedDocument == nil || snapshot.SavedDocument.Document == nil || snapshot.SavedDocument.GroupBindings == nil || snapshot.SavedDocument.GroupBindings.Version != 1 {
		t.Fatal("complete group document/bindings absent", err)
	}
	return snapshot
}

func groupDocumentBindings1115(t *testing.T, state *SnapshotSAVDocument, world *sim.World) map[sim.EntityID]uint16 {
	t.Helper()
	if state == nil || state.Document == nil || state.GroupBindings == nil {
		t.Fatal("missing group projection state")
	}
	byNative := map[sim.EntityID]uint32{}
	for _, actor := range world.Entities() {
		if actor.SourceBinding.Class != 0 {
			byNative[actor.ID] = actor.SourceBinding.Identity
		}
	}
	out := map[sim.EntityID]uint16{}
	for _, binding := range state.Actors {
		if binding.Retired {
			continue
		}
		if binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) {
			t.Fatal("invalid current actor object binding", binding)
		}
		key := actorProjectionValue(t, state.Document.Objects[binding.ObjectIndex-1], "Identity")
		if key != byNative[binding.EntityID] {
			t.Fatal("actor reindex bound a different source identity", binding, key, byNative)
		}
		out[binding.EntityID] = binding.ObjectIndex
	}
	boundMembers := 0
	for _, binding := range state.GroupBindings.Members {
		if !binding.Bound {
			continue
		}
		boundMembers++
		if out[binding.EntityID] == 0 || binding.ObjectIndex != out[binding.EntityID] {
			t.Fatal("Group member reindex disagrees with actor reindex", binding, out)
		}
	}
	if len(out) != 3 || boundMembers != 3 || len(state.GroupBindings.Groups) != 1 {
		t.Fatal("incomplete original Group bindings", out, boundMembers, state.GroupBindings.Groups)
	}
	for _, binding := range state.GroupBindings.Groups {
		if binding.ID != 1 || binding.PlayerObject == 0 || int(binding.PlayerObject) > len(state.Document.Objects) {
			t.Fatal("wrong original Group identity/Player binding", binding)
		}
		player := state.Document.Objects[binding.PlayerObject-1]
		if player.Class != "Player" || binding.InlineIndex != 0 || len(player.Groups) != 1 || player.Groups[0].Class != "Group" {
			t.Fatal("Group reindex lost enclosing Player/inline location", binding, player.Class)
		}
	}
	return out
}

func TestSavedGroupDocument1115CurrentStateReindexesWholeGraph(t *testing.T) {
	if path := os.Getenv("AGAINROM_GROUP_DOCUMENT1115_NATIVE"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		incoming, _, err := DecodeSave(raw)
		if err != nil {
			t.Fatal(err)
		}
		front := groupDocumentFront1115(t)
		open, town, err := front.Restore(incoming)
		if err != nil || town {
			t.Fatal("fresh reordered native Restore", town, err)
		}
		if err := front.App("fresh reordered Group").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		current := groupDocumentSnapshot(t, front)
		if current.SavedDocument.GroupBindings.Unavailable != "" || !reflect.DeepEqual(current.SavedDocument, incoming.SavedDocument) || fmt.Sprintf("%x", front.live.world.Hash()) != os.Getenv("AGAINROM_GROUP_DOCUMENT1115_WORLD") {
			t.Fatal("fresh reordered native LOAD/Snapshot lost graph, bindings or World")
		}
		groupDocumentBindings1115(t, current.SavedDocument, front.live.world)
		doc, err := sav.EncodeDocumentData(*current.SavedDocument.Document)
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(doc)) != os.Getenv("AGAINROM_GROUP_DOCUMENT1115_SAV") {
			t.Fatal("fresh reordered graph does not export the same SAV", err)
		}
		again, err := EncodeSave(current, "fresh reordered Group")
		if err != nil {
			t.Fatal(err)
		}
		reloaded, _, err := DecodeSave(again)
		if err != nil || !reflect.DeepEqual(reloaded.SavedDocument, incoming.SavedDocument) {
			t.Fatal("fresh LOAD to second native SAVE changed graph/bindings", err)
		}
		for range 20 {
			front.live.tick()
		}
		if fmt.Sprintf("%x", front.live.world.Hash()) != os.Getenv("AGAINROM_GROUP_DOCUMENT1115_NEXT") {
			t.Fatal("fresh reordered Group next20 ticks differ")
		}
		t.Log("reordered Group bindings: fresh-process native LOAD, Snapshot, SAVE and next20 ticks PASS")
		return
	}
	front := groupDocumentFront1115(t)
	raw := groupDocumentLiteral1115(t, front)
	mission, report, err := loadOriginalMission(front, raw)
	if err != nil || report.GroupsRestored != 1 {
		t.Fatal("production diagnostic import", err, report)
	}
	groupDocumentBindings1115(t, mission.savedDocument, mission.World)
	open, town, err := front.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("production frontend import", town, err)
	}
	if err := front.App("Group document projection").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	clear(raw)
	before := groupDocumentSnapshot(t, front)
	beforeBytes, err := sav.EncodeDocumentData(*before.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	oldBindings := groupDocumentBindings1115(t, before.SavedDocument, front.live.world)
	groups, orders, present := front.live.world.SavedGroups()
	if !present || len(groups) != 1 || len(groups[0].Members) != 4 || len(orders) != 3 || groups[0].Members[0].Bound || groups[0].Members[0].Archive != 0 {
		t.Fatal("literal null,B,A,C current registry differs", groups, orders)
	}
	actors := registryActors1111(t, front.live.world)
	if groups[0].Members[1].Entity != actors[3].ID || groups[0].Members[2].Entity != actors[35].ID || groups[0].Members[3].Entity != actors[33].ID {
		t.Fatal("literal loaded Group order differs", groups[0].Members)
	}
	oldMembers := slices.Clone(groups[0].Members)
	groups[0].Members = []sim.SavedGroupMember{oldMembers[3], oldMembers[0], oldMembers[2], oldMembers[1]}
	groups[0].Selector = 0xfedcba98
	groups[0].Words = []uint16{0xff00, 0x1201, 0xff00}
	groups[0].Path = []uint16{0x1002, 0x3040}
	groups[0].AI[0x20], groups[0].AI[0x44], groups[0].AI[0x45] = 0xff, 29, 0
	for i := range orders {
		orders[i].State = 0xb
		orders[i].Raw[0x10], orders[i].Raw[0x21] = byte(71+i), byte(91+i)
		orders[i].Patrol = []uint16{0x100f, 0x1210, 0x100f, 0x1411}
	}
	if err := front.live.world.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal("typed current-state update", err)
	}
	worldBefore := front.live.world.Hash()
	current := groupDocumentSnapshot(t, front)
	if front.live.world.Hash() != worldBefore {
		t.Fatal("Snapshot mutated live Group/order state")
	}
	if current.SavedDocument.GroupBindings.Unavailable != "" {
		t.Fatal("existing original Group state was not projected", current.SavedDocument.GroupBindings.Unavailable)
	}
	currentBindings := groupDocumentBindings1115(t, current.SavedDocument, front.live.world)
	if reflect.DeepEqual(oldBindings, currentBindings) {
		t.Fatal("noncanonical first-encounter reorder did not remap bindings")
	}
	if roots := current.SavedDocument.Document.Players; len(roots) != 3 || roots[0] == 0 || roots[1] != 0 || roots[2] != roots[0] {
		t.Fatal("Player alias/null roots changed", roots)
	}
	encoded, err := sav.EncodeDocumentData(*current.SavedDocument.Document)
	if err != nil {
		t.Fatal("reindexed complete SAV", err)
	}
	parsed, err := sav.Open(encoded)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := parsed.ActorGraph()
	if err != nil || len(graph.Groups) != 1 || len(graph.Actors) != 3 {
		t.Fatal("current SAV actor graph", err)
	}
	group := graph.Groups[0]
	byArchive := map[uint16]uint32{}
	for _, actor := range graph.Actors {
		byArchive[actor.ArchiveIndex] = actor.Identity
		n, exists := map[uint32]byte{1004: 0, 1006: 1, 1007: 2}[actor.Identity]
		if !exists || actor.ActorState != 0xb || actor.Order[0x10] != 71+n || actor.Order[0x21] != 91+n || !slices.Equal(actor.Patrol, []uint16{0x100f, 0x1210, 0x100f, 0x1411}) {
			t.Fatal("current order fields not exported", actor.Identity, actor.ActorState, actor.Patrol)
		}
	}
	var keys []uint32
	for _, archive := range group.Members {
		keys = append(keys, byArchive[archive])
	}
	if !slices.Equal(keys, []uint32{1007, 0, 1004, 1006}) || group.Selector != 0xfedcba98 || !slices.Equal(group.Words, []uint16{0xff00, 0x1201, 0xff00}) || !slices.Equal(group.AIWords, []uint16{0x1002, 0x3040}) || group.AI[0x20] != 0xff || group.AI[0x44] != 29 || group.AI[0x45] != 0 {
		t.Fatal("current ordered Group fields/null member differ", keys, group)
	}
	stillBefore, err := sav.EncodeDocumentData(*before.SavedDocument.Document)
	if err != nil || !bytes.Equal(beforeBytes, stillBefore) || !reflect.DeepEqual(oldBindings, groupDocumentBindings1115(t, before.SavedDocument, front.live.world)) {
		t.Fatal("new snapshot mutated a previously owned document/binding set", err)
	}
	// A second projection must be stable; native encode/decode carries the
	// exact reindexed binding metadata beside the complete document.
	again := groupDocumentSnapshot(t, front)
	if !reflect.DeepEqual(current.SavedDocument, again.SavedDocument) {
		t.Fatal("projection is not deterministic")
	}
	native, err := EncodeSave(current, "reindexed Group state")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(native)
	if err != nil || !reflect.DeepEqual(current.SavedDocument, decoded.SavedDocument) {
		t.Fatal("native format lost reindexed Group metadata", err)
	}
	path := filepath.Join(t.TempDir(), "reindexed-group.ags")
	if err := os.WriteFile(path, native, 0600); err != nil {
		t.Fatal(err)
	}
	worldHash := fmt.Sprintf("%x", front.live.world.Hash())
	for range 20 {
		front.live.tick()
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestSavedGroupDocument1115CurrentStateReindexesWholeGraph$", "-test.v")
	child.Env = append(os.Environ(), "AGAINROM_GROUP_DOCUMENT1115_NATIVE="+path,
		"AGAINROM_GROUP_DOCUMENT1115_WORLD="+worldHash,
		fmt.Sprintf("AGAINROM_GROUP_DOCUMENT1115_SAV=%x", sha256.Sum256(encoded)),
		fmt.Sprintf("AGAINROM_GROUP_DOCUMENT1115_NEXT=%x", front.live.world.Hash()))
	output, err := child.CombinedOutput()
	if err != nil || !bytes.Contains(output, []byte("fresh-process native LOAD, Snapshot, SAVE and next20 ticks PASS")) {
		t.Fatalf("fresh reordered Group process: %v\n%s", err, output)
	}
	t.Log(string(output))
}

func TestSavedGroupDocument1115LateCoverageGapPublishesNoPartialGraph(t *testing.T) {
	front := groupDocumentFront1115(t)
	mission, _, err := loadOriginalMission(front, groupDocumentLiteral1115(t, front))
	if err != nil {
		t.Fatal(err)
	}
	state, err := cloneSavedDocument(mission.savedDocument)
	if err != nil {
		t.Fatal(err)
	}
	before, err := cloneSavedDocument(state)
	if err != nil {
		t.Fatal(err)
	}
	groups, orders, _ := mission.World.SavedGroups()
	slices.Reverse(groups[0].Members)
	groups[0].Words = []uint16{0x4321, 0xabcd}
	// A native-authored order is supported and must publish the whole graph.
	orders[len(orders)-1].Authored = true
	if err := mission.World.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	worldBefore := mission.World.Hash()
	if err := projectSavedGroups(state, mission.World); err != nil {
		t.Fatal("current authored order must be saveable", err)
	}
	if state.GroupBindings.Unavailable != "" || reflect.DeepEqual(state, before) || mission.World.Hash() != worldBefore {
		t.Fatal("supported order failed to publish current Group graph/bindings")
	}
	if _, err := cloneSavedDocument(state); err != nil {
		t.Fatal("published bindings disagree with the graph", err)
	}
	state, err = cloneSavedDocument(before)
	if err != nil {
		t.Fatal(err)
	}
	// Exceed the bounded patrol representation in the final order, after the
	// candidate has prepared the changed Group and earlier orders.
	orders[len(orders)-1].Patrol = make([]uint16, savedGroupFieldListLimit+1)
	if err := mission.World.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	worldBefore = mission.World.Hash()
	if err := projectSavedGroups(state, mission.World); err == nil || !strings.Contains(err.Error(), "patrol exceeds list bound") {
		t.Fatal("malformed final order did not fail at its patrol bound", err)
	}
	if !reflect.DeepEqual(state, before) || mission.World.Hash() != worldBefore {
		t.Fatal("invalid final order published a partial Group graph or changed World")
	}
}

func TestSavedGroupDocument1115AuthoredMoveProjectsCommandGroup(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		t.Run(fmt.Sprintf("from-map-%t", fromMap), func(t *testing.T) {
			front := groupDocumentFront1115(t)
			raw := groupDocumentLiteral1115(t, front)
			app := front.App("Group export boundary")
			if fromMap {
				if err := app.OpenMission(front.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			dir, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
			path := filepath.Join(dir, "game1115.sav")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load := agsSaveSeams(front, store, OriginalStore{Dir: dir}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1115.sav")
			clear(raw)
			// Only the test-created synthetic source is removed.
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			before := groupDocumentSnapshot(t, front)
			if before.SavedDocument.GroupBindings.Unavailable != "" {
				t.Fatal("original Group fixture starts outside export coverage", before.SavedDocument.GroupBindings.Unavailable)
			}
			initialGroups, _, present := front.live.world.SavedGroups()
			if !present || len(initialGroups) != 1 || len(initialGroups[0].Members) != 4 || initialGroups[0].Members[0].Bound || initialGroups[0].Members[0].Archive != 0 {
				t.Fatal("original LOAD door lost null Group member", initialGroups)
			}
			groupDocumentBindings1115(t, before.SavedDocument, front.live.world)
			binding := before.SavedDocument.GroupBindings.Groups[0]
			storedGroup := before.SavedDocument.Document.Objects[binding.PlayerObject-1].Groups[binding.InlineIndex]
			if len(storedGroup.RefSlots) != 1 || len(storedGroup.RefSlots[0].Objects) != 4 || storedGroup.RefSlots[0].Objects[0] != 0 {
				t.Fatal("ordinary Snapshot lost loaded null Group member", storedGroup.RefSlots)
			}
			actor := registryActors1111(t, front.live.world)[35]
			front.live.enqueue(uint32(actor.ID), 20, 16)
			front.live.tick()
			groups, orders, _ := front.live.world.SavedGroups()
			if len(groups) != 2 || !groups[1].Authored || len(groups[1].Members) != 1 || groups[1].Members[0].Entity != actor.ID {
				t.Fatal("ordinary Move did not create current authored Group", groups)
			}
			current := groupDocumentSnapshot(t, front)
			if current.SavedDocument.GroupBindings.Unavailable != "" {
				t.Fatal("current native command Group was not projected", current.SavedDocument.GroupBindings.Unavailable)
			}
			bindings := current.SavedDocument.GroupBindings
			if !bindings.PlayersPresent || len(bindings.Players) != 1 || len(bindings.Groups) != 2 ||
				bindings.Groups[0].Authored || !bindings.Groups[1].Authored ||
				bindings.Groups[1].ContainerID != bindings.Players[0].ID ||
				bindings.Groups[1].PlayerObject != bindings.Players[0].ObjectIndex ||
				groups[1].Owner.Key != 0 || groups[1].Reference != (sim.SavedGroupReference{}) {
				t.Fatal("native-generated Group lost exact construction/container authority", bindings)
			}
			// Read the independent ordinary Group shape before exercising full
			// current SAVE, cold World equality and the next real movement.
			wire, err := sav.EncodeDocumentData(*current.SavedDocument.Document)
			if err != nil {
				t.Fatal("generated Group SAV encode", err)
			}
			parsed, err := sav.Open(wire)
			if err != nil {
				t.Fatal(err)
			}
			graph, err := parsed.ActorGraph()
			if err != nil || len(graph.Groups) != 2 || len(graph.Groups[0].Members) != 3 ||
				graph.Groups[0].Members[0] != 0 || len(graph.Groups[1].Members) != 1 {
				t.Fatal("generated Group or source null member lost in SAV", graph.Groups, err)
			}
			var movedIdentity uint32
			for _, a := range graph.Actors {
				if a.ArchiveIndex == graph.Groups[1].Members[0] {
					movedIdentity = a.Identity
				}
			}
			if movedIdentity != 1004 {
				t.Fatal("generated Group SAV bound a different actor", movedIdentity)
			}
			beforeSave := front.live.world.Hash()
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal("ordinary native SAVE failed for authored Group", entries, err, app.HeadlessMessage())
			}
			if beforeSave != front.live.world.Hash() {
				t.Fatal("Group SAVE mutated current state")
			}
			writtenRaw, err := store.Read(entries[0].Name)
			if err != nil {
				t.Fatal(err)
			}
			written, err := sav.DecodeDocumentData(writtenRaw)
			if err != nil {
				t.Fatal(err)
			}
			fresh := groupDocumentFront1115(t)
			freshApp := fresh.App("fresh current native Group")
			fs, fl, fn := agsSaveSeams(fresh, store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, fn)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			gg, oo, has := fresh.live.world.SavedGroups()
			if !has || !reflect.DeepEqual(groups, gg) || !reflect.DeepEqual(orders, oo) || front.live.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("fresh native LOAD changed authored Group/current World")
			}
			loaded := groupDocumentSnapshot(t, fresh)
			if loaded.SavedDocument.GroupBindings.Unavailable != "" {
				t.Fatal("current SAVE/LOAD lost exact Group authority")
			}
			retained := fresh.live.mission.state.savedDocument.Document
			if retained == nil {
				t.Fatal("cold LOAD discarded the ordinary graph")
			}
			// Owner zero uses the free ordinary Player slot two in this fixture.
			// LOAD returns only those two named fields to native zero; every
			// other ordinary field and unknown byte must remain unchanged.
			actions, err := readCurrentActions(&written)
			if err != nil || actions == nil || len(actions.PlayerSlots) != 1 {
				t.Fatal("mixed Group lacks its single zero-owner slot bridge", err)
			}
			slot := actions.PlayerSlots[0]
			if slot.Object == 0 || int(slot.Object) > len(written.Objects) || slot.Wire != 2 || slot.Native != 0 || slot.Trigger != nil || slot.Shared {
				t.Fatal("mixed Group changed its zero-owner slot bridge", slot)
			}
			player := &written.Objects[slot.Object-1]
			if player.Class != "Player" || savedRecordValueForTest(t, *player, "Slot") != 2 || savedRecordValueForTest(t, *player, "SlotAgain") != 2 {
				t.Fatal("zero-owner transport bridge selected another Player")
			}
			mustSetValue(player, "Slot", 0)
			mustSetValue(player, "SlotAgain", 0)
			wantGraph, err := canonicalDocument1115(t, written)
			if err != nil {
				t.Fatal(err)
			}
			gotGraph, err := canonicalDocument1115(t, *retained)
			if err != nil || !bytes.Equal(wantGraph, gotGraph) {
				t.Fatal("cold LOAD changed the complete graph beyond its native Player slot", err)
			}
			for range 20 {
				front.live.tick()
				fresh.live.tick()
				if front.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("next native action changed across ordinary SAVE/LOAD")
				}
			}
		})
	}
}
