package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

const (
	newGroupLeft1115  = uint32(0x10100101)
	newGroupRight1115 = uint32(0x20200202)
	newGroupA         = uint32(1004)
	newGroupB1115     = uint32(1006)
	newGroupC1115     = uint32(1008)
)

// The ALM operands below are wire action/condition/trigger records, not a
// Script assembled from the current world's native IDs. GiveUnit targets the
// uniquely joined MapUnitID 91 and binds through the production original door.
func newGroupGivePayload1115(destination uint32) []byte {
	b := make([]byte, 12+2*796+184)
	put := func(off int, value uint32) { binary.LittleEndian.PutUint32(b[off:], value) }
	put(0, 1)
	a := 4
	put(a+0x40, 19)
	put(a+0x44, 10)
	put(a+0x4c, 91)
	put(a+0x50, destination)
	put(a+0x74, 4) // Target_Unit
	put(a+0x78, 3) // Target_Player
	c := 4 + 796
	put(c, 1)
	c += 4
	put(c+0x40, 0x10002)
	put(c+0x44, 1)
	put(c+0x74, 7) // constant zero
	tr := 4 + 796 + 4 + 796
	put(tr, 1)
	tr += 4
	put(tr+0x80, 1)
	put(tr+0x84, 1)
	put(tr+0x98, 10)
	put(tr+0xb4, 1) // once, with zero == zero
	return b
}

func newGroupFront(t *testing.T, destination int32) *FrontEnd {
	t.Helper()
	var payload []byte
	if destination >= 0 {
		payload = newGroupGivePayload1115(uint32(destination))
	}
	b := synth.ALM(synth.ALMOptions{Width: 40, Height: 40,
		Units: []synth.ALMUnit{{X: 5<<8 | 128, Y: 6<<8 | 128}}, Type7Payload: payload})
	binary.LittleEndian.PutUint16(rawSections1092(t, b)[6][0x40:], 91)
	f := poolFixtureFrontMap(t, b)
	f.Table, f.Campaign = actorRegistryTable(), resolved(saveCampaign(), nil)
	f.Units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{}}
	for _, class := range []int32{3, 33, 35} {
		f.Units.Classes[class] = worldFixtureArt(16, 16, 8, 16, 3, 3, 1)
	}
	return f
}

func newGroupSetValue1115(t *testing.T, record *sav.DocumentRecordData, name string, value uint32) {
	t.Helper()
	for i := range record.Values {
		if record.Values[i].Name == name {
			record.Values[i].Value = value
			return
		}
	}
	t.Fatalf("literal fixture lacks %s", name)
}

// Two different Player objects have repeated root references. Empty Groups
// and actor identities are spelled by the independent archive writer. Group 5
// deliberately resolves its owner to LEFT while its enclosing Player is RIGHT.
func newGroupLiteral(t *testing.T, f *FrontEnd, duplicateSlot bool) []byte {
	t.Helper()
	load, runtime := &[4]int16{100, 0, 0, 300}, &[3]byte{1, 8, 4}
	a := &poolFixtureActor{mapID: 91, cell: 0x100f, hp: 7, maxHP: 31, name: "New group A", loadWords: load, equipmentRuntime: runtime}
	b := &poolFixtureActor{mapID: 900, cell: 0x120f, hp: 9, maxHP: 41, human: true, name: "New group B", loadWords: load, equipmentRuntime: runtime}
	c := &poolFixtureActor{mapID: 900, cell: 0x140f, hp: 11, maxHP: 51, human: true, name: "New group C", loadWords: load, equipmentRuntime: runtime}
	left := &poolFixturePlayer{groups: [][]*poolFixtureActor{{}, {}, {a, b}}}
	right := &poolFixturePlayer{groups: [][]*poolFixtureActor{{}, {c}}}
	body := poolFixtureBody([]*poolFixturePlayer{left, nil, left, right, nil, right}, nil)
	for i, actor := range []*poolFixtureActor{a, b, c} {
		binary.LittleEndian.PutUint32(body[actor.off+12:], uint32(501+i))
		body[actor.off+16] = 1
		binary.LittleEndian.PutUint16(body[actor.off+17:], []uint16{35, 3, 33}[i])
		body[actor.off+509], body[actor.off+510], body[actor.off+511] = 1, 1, 3
		body[actor.off+179], body[actor.off+189] = 64, 100
	}
	doc, err := sav.DecodeDocumentData(completeDocumentTail1115(t, f, savedContainer(body)))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(doc.Players, []uint16{1, 0, 1, 4, 0, 4}) || len(doc.Objects) != 5 {
		t.Fatal("independent fixture object/alias layout changed", doc.Players, len(doc.Objects))
	}
	selector := uint32(10)
	for i, index := range []uint16{1, 4} {
		p := &doc.Objects[index-1]
		slot, key := uint32(i+1), []uint32{newGroupLeft1115, newGroupRight1115}[i]
		if duplicateSlot {
			slot = 1
		}
		newGroupSetValue1115(t, p, "Slot", slot)
		newGroupSetValue1115(t, p, "SlotAgain", slot)
		newGroupSetValue1115(t, p, "This", key)
		for j := range p.Groups {
			newGroupSetValue1115(t, &p.Groups[j], "G1C", selector)
			owner := key
			if selector == 50 {
				owner = newGroupLeft1115
			}
			newGroupSetValue1115(t, &p.Groups[j], "G44", owner)
			selector += 10
		}
	}
	for _, row := range []struct {
		index uint16
		key   uint32
	}{{2, newGroupLeft1115}, {3, newGroupLeft1115}, {5, newGroupRight1115}} {
		newGroupSetValue1115(t, &doc.Objects[row.index-1], "Reference", row.key)
	}
	out, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func newGroupOpen1115(t *testing.T, destination int32, duplicateSlot bool) *FrontEnd {
	t.Helper()
	f := newGroupFront(t, destination)
	raw := newGroupLiteral(t, f, duplicateSlot)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("synthetic exact Player original LOAD", town, err)
	}
	if err := f.App("new native Group integration").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	clear(raw)
	actors := newGroupActors1115(t, f.live.world)
	rightSlot := uint32(2)
	if duplicateSlot {
		rightSlot = 1
	}
	if actors[newGroupA].Owner != 1 || actors[newGroupB1115].Owner != 1 || actors[newGroupC1115].Owner != rightSlot {
		t.Fatal("Player suffix did not independently set actor ownership", actors)
	}
	return f
}

func newGroupActors1115(t *testing.T, w *sim.World) map[uint32]sim.Entity {
	t.Helper()
	return newGroupActors(t, w)
}

func newGroupActors(t *testing.T, w *sim.World) map[uint32]sim.Entity {
	t.Helper()
	out := make(map[uint32]sim.Entity)
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			out[e.SourceBinding.Identity] = e
		}
	}
	if len(out) != 3 || out[newGroupA].SourceBinding.TypeID != 35 || out[newGroupB1115].SourceBinding.TypeID != 3 || out[newGroupC1115].SourceBinding.TypeID != 33 {
		t.Fatal("literal source identities/classes changed", out)
	}
	return out
}

type newGroupWant1115 struct {
	id, player uint32
	members    []uint32
}

func newGroupInitial1115() []newGroupWant1115 {
	return []newGroupWant1115{{1, 1, nil}, {2, 1, nil}, {3, 1, []uint32{newGroupA, newGroupB1115}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}}
}

func newGroupMemberKeys1115(t *testing.T, w *sim.World, group sim.SavedGroup) []uint32 {
	t.Helper()
	keys := make(map[sim.EntityID]uint32)
	for key, e := range newGroupActors1115(t, w) {
		keys[e.ID] = key
	}
	var out []uint32
	for _, m := range group.Members {
		if !m.Bound || keys[m.Entity] == 0 {
			t.Fatal("literal actor was not exactly materialized", group.ID, m)
		}
		out = append(out, keys[m.Entity])
	}
	return out
}

func newGroupRoster1115(t *testing.T, w *sim.World, want []newGroupWant1115, duplicateSlot bool) []sim.SavedGroup {
	t.Helper()
	groups, _, present := w.SavedGroups()
	players, hasPlayers := w.SavedGroupPlayers()
	rightSlot := uint32(2)
	if duplicateSlot {
		rightSlot = 1
	}
	if !present || !hasPlayers || !reflect.DeepEqual(players, []sim.SavedGroupPlayer{{ID: 1, Slot: 1}, {ID: 2, Slot: rightSlot}}) || len(groups) != len(want) {
		t.Fatal("distinct native Player/Group population", players, groups, want)
	}
	formations, haveFormations := w.SavedPlayerFormations()
	if !haveFormations || !reflect.DeepEqual(formations, []sim.SavedPlayerFormation{{PlayerID: 1, CommandID: 1, TriggerID: 1}, {PlayerID: 2, CommandID: int16(rightSlot), TriggerID: rightSlot}}) {
		t.Fatal("exact formation identities changed through Group generation/GiveUnit/reindex", formations)
	}
	for i, expected := range want {
		g := groups[i]
		if g.ID != expected.id || g.ContainerID != expected.player || g.Authored != (g.ID > 5) || !slices.Equal(newGroupMemberKeys1115(t, w, g), expected.members) {
			t.Fatalf("native Group[%d]=%+v members=%v, want %+v", i, g, newGroupMemberKeys1115(t, w, g), expected)
		}
		owner := uint32(1)
		if expected.player == 2 && g.ID != 5 {
			owner = rightSlot
		}
		if g.Owner.Class != 1 || g.Owner.Owner != owner {
			t.Fatal("owner was inferred from enclosing container", g)
		}
		exactOwner := expected.player
		if g.ID == 5 {
			exactOwner = 1 // imported explicit owner differs from right container
		}
		if g.OwnerID != exactOwner {
			t.Fatal("Group lost exact formation owner", g.ID, g.OwnerID, exactOwner)
		}
	}
	return groups
}

func newGroupCount1115(t *testing.T, record sav.DocumentRecordData, name string) uint32 {
	t.Helper()
	for _, count := range record.Counts {
		if count.Name == name {
			return count.Count
		}
	}
	t.Fatalf("missing count %s", name)
	return 0
}

// Independently allocate the five DTO object indices by the expected
// Player/member encounter order. Never use a producer permutation or its
// bindings as the expected answer. Inline Groups allocate no object index.
func newGroupDocument1115(t *testing.T, f *FrontEnd, want []newGroupWant1115, duplicateSlot bool) Snapshot {
	t.Helper()
	groups := newGroupRoster1115(t, f.live.world, want, duplicateSlot)
	s := groupDocumentSnapshot(t, f)
	doc, bindings := s.SavedDocument.Document, s.SavedDocument.GroupBindings
	if bindings.Unavailable != "" {
		t.Fatal("bounded native Group producer refused", bindings.Unavailable)
	}
	objects, players := map[uint32]uint16{}, map[uint32]uint16{}
	sites := make(map[uint32]SnapshotSAVGroupBinding)
	next := uint16(1)
	for _, player := range []uint32{1, 2} {
		players[player], next = next, next+1
		var inline uint32
		for _, g := range want {
			if g.player != player {
				continue
			}
			sites[g.id] = SnapshotSAVGroupBinding{ID: g.id, ContainerID: player, Authored: g.id > 5, PlayerObject: players[player], InlineIndex: inline}
			inline++
			for _, key := range g.members {
				if objects[key] == 0 {
					objects[key], next = next, next+1
				}
			}
		}
	}
	if next != 6 || len(doc.Objects) != 5 || !slices.Equal(doc.Players, []uint16{players[1], 0, players[1], players[2], 0, players[2]}) {
		t.Fatal("whole graph encounter reindex/aliases changed", doc.Players, players, objects, len(doc.Objects))
	}
	if !bindings.PlayersPresent || !reflect.DeepEqual(bindings.Players, []SnapshotSAVGroupPlayerBinding{{ID: 1, ObjectIndex: players[1]}, {ID: 2, ObjectIndex: players[2]}}) || len(bindings.Groups) != len(want) || len(bindings.Members) != 3 || len(s.SavedDocument.Actors) != 3 {
		t.Fatal("incomplete exact Player/Group/member bindings", bindings)
	}
	actors := newGroupActors1115(t, f.live.world)
	byNative := make(map[sim.EntityID]uint32)
	for key, e := range actors {
		byNative[e.ID] = key
		if actorProjectionValue(t, doc.Objects[objects[key]-1], "Identity") != key {
			t.Fatal("object permutation changed source identity", key, objects[key])
		}
	}
	for _, a := range s.SavedDocument.Actors {
		if a.Retired || a.ObjectIndex != objects[byNative[a.EntityID]] {
			t.Fatal("actor binding missed current encounter reindex", a, objects)
		}
	}
	for _, m := range bindings.Members {
		if !m.Bound || m.ObjectIndex != objects[byNative[m.EntityID]] {
			t.Fatal("member binding missed current encounter reindex", m, objects)
		}
	}
	var previousID uint32
	for _, b := range bindings.Groups {
		site, ok := sites[b.ID]
		if !ok || b.ID <= previousID || b.PlayerObject != site.PlayerObject || b.InlineIndex != site.InlineIndex || b.ContainerID != site.ContainerID || b.Authored != site.Authored {
			t.Fatal("Group binding did not follow exact inline record", b, site)
		}
		previousID = b.ID
		ownerPlayer := b.ContainerID
		if b.ID == 5 {
			ownerPlayer = 1
		}
		if b.Owner.ObjectIndex != players[ownerPlayer] || b.Owner.Key != []uint32{0, newGroupLeft1115, newGroupRight1115}[ownerPlayer] || b.Owner.Class != 1 {
			t.Fatal("Group owner binding confused container/slot identity", b)
		}
	}
	for player, object := range players {
		record := doc.Objects[object-1]
		var count, members uint32
		for i, expected := range want {
			if expected.player != player {
				continue
			}
			g := record.Groups[count]
			ownerKey := newGroupLeft1115
			if expected.player == 2 && expected.id != 5 {
				ownerKey = newGroupRight1115
			}
			if newGroupCount1115(t, g, "Actors") != uint32(len(expected.members)) || actorProjectionValue(t, g, "G1C") != groups[i].Selector || actorProjectionValue(t, g, "G44") != ownerKey || !bytes.Equal(actorProjection1115Raw(t, g, "G3C")[:76], groups[i].AI[:]) {
				t.Fatal("inline Group count/selector/current AI differs", expected.id)
			}
			var refs []uint16
			for _, key := range expected.members {
				refs = append(refs, objects[key])
			}
			for _, slots := range g.RefSlots {
				if slots.Name == "Actors" && !slices.Equal(slots.Objects, refs) {
					t.Fatal("inline member references differ", slots.Objects, refs)
				}
			}
			count++
			members += uint32(len(expected.members))
		}
		if record.Class != "Player" || uint32(len(record.Groups)) != count || newGroupCount1115(t, record, "Groups") != count || newGroupCount1115(t, record, "Actors") != members {
			t.Fatal("Player counts aggregate aliases or stale Groups", player, count, members)
		}
	}
	return s
}

func newGroupNative1115(t *testing.T, f *FrontEnd, snapshot Snapshot, destination int32) *FrontEnd {
	t.Helper()
	snapshot.ghost, snapshot.terrainBase = nil, nil
	native, err := EncodeSave(snapshot, "generated exact Groups")
	if err != nil {
		t.Fatal(err)
	}
	loaded, label, err := DecodeSave(native)
	if err != nil || label != "generated exact Groups" || !reflect.DeepEqual(loaded, snapshot) {
		t.Fatal("AGS lost current graph/native state", label, err)
	}
	// Live ticks populate the existing Residue.Swing/Phase maps. Gob does not
	// promise map entry order; require every decoded field, including those
	// maps, to agree instead of mistaking envelope byte order for state drift.
	again, err := EncodeSave(loaded, label)
	if err != nil {
		t.Fatal(err)
	}
	second, secondLabel, err := DecodeSave(again)
	if err != nil || secondLabel != label || !reflect.DeepEqual(second, snapshot) {
		t.Fatal("AGS second complete decoded Snapshot is not stable", err)
	}
	fresh := newGroupFront(t, destination)
	open, town, err := fresh.Restore(loaded)
	if err != nil || town {
		t.Fatal("generated Group AGS prepare", town, err)
	}
	if err := fresh.App("generated Group AGS resume").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	restored := groupDocumentSnapshot(t, fresh)
	if !reflect.DeepEqual(restored.SavedDocument, snapshot.SavedDocument) || f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("AGS Restore/OpenMission/Snapshot changed exact state")
	}
	beforeSAV, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	afterSAV, err := sav.EncodeDocumentData(*restored.SavedDocument.Document)
	if err != nil || !bytes.Equal(beforeSAV, afterSAV) {
		t.Fatal("SAV document bytes changed across AGS Restore", err)
	}
	for tick := range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("generated Group AGS continuation diverged at", tick)
		}
	}
	return fresh
}

func newGroupSAV1115(t *testing.T, f *FrontEnd, snapshot Snapshot, want []newGroupWant1115, destination int32) {
	t.Helper()
	wantGroups, wantOrders, _ := f.live.world.SavedGroups()
	actors := newGroupActors1115(t, f.live.world)
	orders := make(map[uint32]sim.SavedActorOrder)
	for key, actor := range actors {
		for _, order := range wantOrders {
			if order.Entity == actor.ID {
				orders[key] = order
			}
		}
	}
	wire, err := sav.EncodeDocumentData(*snapshot.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(wire)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil || len(graph.Groups) != len(want) || len(graph.Actors) != 3 {
		t.Fatal("produced SAV graph population", err, graph)
	}
	keys := make(map[uint16]uint32)
	for _, actor := range graph.Actors {
		keys[actor.ArchiveIndex] = actor.Identity
		order, exists := orders[actor.Identity]
		if !exists || actor.OwnerSlot != uint16(actors[actor.Identity].Owner) || actor.ActorState != order.State || !bytes.Equal(actor.Order[:144], order.Raw[:]) || !slices.Equal(actor.Patrol, order.Patrol) {
			t.Fatal("produced SAV actor owner/current order", actor.Identity, actor.OwnerSlot, actor.ActorState, order.State)
		}
	}
	for i, group := range graph.Groups {
		var members []uint32
		for _, index := range group.Members {
			members = append(members, keys[index])
		}
		if !slices.Equal(members, want[i].members) || group.Selector != wantGroups[i].Selector || !bytes.Equal(group.AI[:76], wantGroups[i].AI[:]) || !slices.Equal(group.Words, wantGroups[i].Words) || !slices.Equal(group.AIWords, wantGroups[i].Path) || !group.Owner.Resolved || uint32(group.Owner.PlayerSlot) != wantGroups[i].Owner.Owner {
			t.Fatal("SAV readback lost current Group roster/owner/AI", i, group, members, want[i])
		}
	}
	fromSAV := newGroupFront(t, destination)
	open, town, err := fromSAV.RestoreOriginal(wire)
	if err != nil || town {
		t.Fatal("produced SAV production reload", town, err)
	}
	if err := fromSAV.App("generated Group SAV reload").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	loadedGroups, loadedOrders, present := fromSAV.live.world.SavedGroups()
	if !present || len(loadedGroups) != len(want) {
		t.Fatal("SAV reload lost Group population", loadedGroups)
	}
	for i, group := range loadedGroups {
		if group.Authored || group.ID != uint32(i+1) || group.ContainerID != want[i].player || group.Owner.Owner != wantGroups[i].Owner.Owner || group.Selector != wantGroups[i].Selector || group.AI != wantGroups[i].AI || !slices.Equal(group.Words, wantGroups[i].Words) || !slices.Equal(group.Path, wantGroups[i].Path) || !slices.Equal(newGroupMemberKeys1115(t, fromSAV.live.world, group), want[i].members) {
			t.Fatal("production SAV reload changed bounded Group semantics", i, group, want[i])
		}
	}
	for key, actor := range newGroupActors1115(t, fromSAV.live.world) {
		if actor.Owner != actors[key].Owner {
			t.Fatal("SAV reload changed actor owner", key, actor.Owner)
		}
		found := false
		for _, order := range loadedOrders {
			if order.Entity == actor.ID {
				found = true
				prior := orders[key]
				if order.Authored || order.State != prior.State || order.Raw != prior.Raw || !slices.Equal(order.Patrol, prior.Patrol) {
					t.Fatal("production SAV reload changed current order", key, order, prior)
				}
			}
		}
		if !found {
			t.Fatal("SAV reload lost actor order", key)
		}
	}
	// SAV movement-value production remains open. This deliberately compares
	// bounded Groups/owners/current orders, not a false whole-World equality.
}

func TestSavedGroupNew1115OrdinaryGenerationsAndExactReindex(t *testing.T) {
	f := newGroupOpen1115(t, -1, false)
	first := newGroupDocument1115(t, f, newGroupInitial1115(), false)
	before, err := sav.EncodeDocumentData(*first.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		actor uint32
		want  []newGroupWant1115
	}{
		{newGroupA, []newGroupWant1115{{2, 1, nil}, {3, 1, []uint32{newGroupB1115}}, {6, 1, []uint32{newGroupA}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}}},
		{newGroupB1115, []newGroupWant1115{{3, 1, nil}, {6, 1, []uint32{newGroupA}}, {7, 1, []uint32{newGroupB1115}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}}},
		{newGroupA, []newGroupWant1115{{6, 1, nil}, {7, 1, []uint32{newGroupB1115}}, {8, 1, []uint32{newGroupA}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}}},
	}
	for generation, tc := range cases {
		actor := newGroupActors1115(t, f.live.world)[tc.actor]
		f.live.enqueue(uint32(actor.ID), 24+generation, 16+generation)
		f.live.tick()
		current := newGroupDocument1115(t, f, tc.want, false)
		newGroupSAV1115(t, f, current, tc.want, -1)
		f = newGroupNative1115(t, f, current, -1)
		t.Log("ordinary generation", generation+1, "exact cleanup, Player block, graph reindex, SAV reload and native next20 PASS")
	}
	after, err := sav.EncodeDocumentData(*first.SavedDocument.Document)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("later generations mutated earlier owned Snapshot", err)
	}
}

func TestSavedGroupNew1115OriginalGiveUnitDestinationWithoutCleanup(t *testing.T) {
	f := newGroupOpen1115(t, 2, false)
	newGroupDocument1115(t, f, newGroupInitial1115(), false)
	a := newGroupActors1115(t, f.live.world)[newGroupA]
	instants := f.live.world.Script().Instants()
	if len(instants) != 1 || instants[0].Op != sim.ScriptInstantGiveUnit || !instants[0].HasUnit || instants[0].Unit != a.ID || !instants[0].HasPlayer || instants[0].Player != 2 || f.live.world.ScriptLatched(0) {
		t.Fatal("original ALM GiveUnit did not bind the exact source actor", instants)
	}
	for i := 0; i < 32 && !f.live.world.ScriptLatched(0); i++ {
		f.live.tick()
	}
	if !f.live.world.ScriptLatched(0) || newGroupActors1115(t, f.live.world)[newGroupA].Owner != 2 {
		t.Fatal("production original GiveUnit did not fire")
	}
	want := []newGroupWant1115{{1, 1, nil}, {2, 1, nil}, {3, 1, []uint32{newGroupB1115}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}, {6, 2, []uint32{newGroupA}}}
	current := newGroupDocument1115(t, f, want, false)
	for _, binding := range current.SavedDocument.Actors {
		if binding.EntityID == a.ID && actorProjectionValue(t, current.SavedDocument.Document.Objects[binding.ObjectIndex-1], "Reference") != newGroupRight1115 {
			t.Fatal("handover retained actor Token.Reference for the old Player")
		}
	}
	newGroupSAV1115(t, f, current, want, 2)
	newGroupNative1115(t, f, current, 2)
	for range 32 {
		f.live.tick()
	}
	newGroupDocument1115(t, f, want, false)
	if !f.live.world.ScriptLatched(0) {
		t.Fatal("native continuation forgot original once latch")
	}
}

func TestSavedGroupNew1115DuplicateSlotKeepsExactIdentityAndHandoverGap(t *testing.T) {
	f := newGroupOpen1115(t, 1, true)
	initial := newGroupDocument1115(t, f, newGroupInitial1115(), true)
	a := newGroupActors1115(t, f.live.world)[newGroupA]
	f.live.enqueue(uint32(a.ID), 24, 16)
	f.live.tick()
	known := []newGroupWant1115{{2, 1, nil}, {3, 1, []uint32{newGroupB1115}}, {6, 1, []uint32{newGroupA}}, {4, 2, nil}, {5, 2, []uint32{newGroupC1115}}}
	newGroupDocument1115(t, f, known, true)
	for i := 0; i < 32 && !f.live.world.ScriptLatched(0); i++ {
		f.live.tick()
	}
	if !f.live.world.ScriptLatched(0) {
		t.Fatal("ambiguous-slot GiveUnit did not execute gameplay")
	}
	groups, _, _ := f.live.world.SavedGroups()
	if len(groups) != 6 || groups[0].ID != 2 || groups[1].ID != 3 || groups[2].ID != 6 || len(groups[2].Members) != 0 || groups[3].ID != 4 || groups[4].ID != 5 || groups[5].ID != 7 || groups[5].ContainerID != 0 || !slices.Equal(newGroupMemberKeys1115(t, f.live.world, groups[5]), []uint32{newGroupA}) {
		t.Fatal("ambiguous handover inferred Player identity or ran ordinary cleanup", groups)
	}
	gap := groupDocumentSnapshot(t, f)
	if !strings.Contains(gap.SavedDocument.GroupBindings.Unavailable, "exact native Player container") || !reflect.DeepEqual(gap.SavedDocument.GroupBindings.Players, initial.SavedDocument.GroupBindings.Players) || !reflect.DeepEqual(gap.SavedDocument.GroupBindings.Groups, initial.SavedDocument.GroupBindings.Groups) {
		t.Fatal("ambiguous slot did not remain an atomic export gap", gap.SavedDocument.GroupBindings)
	}
	for _, object := range initial.SavedDocument.Document.Players {
		if object != 0 && !reflect.DeepEqual(gap.SavedDocument.Document.Objects[object-1].Groups, initial.SavedDocument.Document.Objects[object-1].Groups) {
			t.Fatal("failed Group transaction changed retained inline records", object)
		}
	}
	newGroupNative1115(t, f, gap, 1)
}
