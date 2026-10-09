package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func partialCurrentGraph(t *testing.T, detached ...bool) (*FrontEnd, Snapshot, *sim.World) {
	t.Helper()
	f := cellStateFront(t)
	campaign := saveCampaign()
	campaign.Side = []int{11, 12}
	f.Campaign = resolved(campaign, nil)
	units := f.Table.Units.(dbCollection)
	f.Table.Units = dbCollection{units[1], units[1]}
	f.Town = NewTown(f.Campaign.Value())
	if err := f.App("partial current graph").OpenMission(f.MissionOpenerWith(10, nil)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	entities := []sim.Entity{{ID: 41, X: 15, Y: 15, HP: 29, MaxHP: 29, Owner: sim.SelfSlot, TypeID: 1, Speed: 1, TokenSize: 1, Capacity: data.UnitCapacity()}}
	w, err := sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	s.SavedDocument, err = f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal(err)
	}
	b := &s.SavedDocument.GroupBindings.Groups[0]
	b.ID = 71
	g := sim.SavedGroup{ID: 71, Selector: 7, Authored: true, Owner: sim.SavedGroupReference{Class: 1, Owner: sim.SelfSlot},
		Members: []sim.SavedGroupMember{{}, {Entity: 41, Bound: true}}, Words: []uint16{11, 19}, Path: []uint16{0x1010, 0x1212}}
	g.AI[0x12], g.AI[0x20], g.AI[0x38], g.AI[0x45] = 19, 3, 3, 1
	entities = append(entities, sim.Entity{ID: 0, X: 18, Y: 18, HP: 37, MaxHP: 37, Owner: sim.SelfSlot, TypeID: 1, Speed: 1, TokenSize: 1, Capacity: data.UnitCapacity()})
	if len(detached) != 0 && detached[0] {
		entities[1].X, entities[1].Y, entities[1].OffMap = 54, 100, true
	}
	w, err = sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.ImportSavedGroups([]sim.SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	var players []sim.SavedGroupPlayer
	for _, p := range s.SavedDocument.GroupBindings.Players {
		slot, err := savedStructureValue(&s.SavedDocument.Document.Objects[p.ObjectIndex-1], "Slot")
		if err != nil {
			t.Fatal(err)
		}
		players = append(players, sim.SavedGroupPlayer{ID: p.ID, Slot: slot})
	}
	if err := w.ImportSavedGroupPlayers(players, []sim.SavedGroupContainer{{GroupID: g.ID, PlayerID: b.ContainerID}}); err != nil {
		t.Fatal(err)
	}
	doc := s.SavedDocument.Document
	doc.Players = append(doc.Players, 0, doc.Players[0])
	s.Residue.VisualIdentities = []SnapshotVisualIdentity{{Entity: 0, Label: 0}, {Entity: 41, Label: 41}}
	s.Residue.VisualNext = 42
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return f, s, w
}

func TestCurrentPlayerRootsDoNotUseOwnerSlotAsIdentity(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("zero-slot owner").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	if !w.SetPurse(0, 0xf1234567) {
		t.Fatal("native slot zero is not a purse owner")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	state, err := f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal(err)
	}
	var last uint32
	foundZero := false
	for _, p := range state.GroupBindings.Players {
		if p.ID == 0 || p.ID <= last {
			t.Fatal("constructed Player identity is not positive and ordered", p)
		}
		last = p.ID
		r := &state.Document.Objects[p.ObjectIndex-1]
		slot, err := savedStructureValue(r, "Slot")
		if err != nil {
			t.Fatal(err)
		}
		if slot == 0 {
			foundZero = true
			money, err := savedStructureValue(r, "Money")
			if err != nil || money != w.Purse(0) || len(r.Groups) != 1 {
				t.Fatal("slot-zero root lost its current purse or actor group", money, err)
			}
		}
	}
	if !foundZero || before != w.Hash() {
		t.Fatal("construction omitted slot zero or changed current World")
	}
	if _, err := cloneSavedDocument(state); err != nil {
		t.Fatal("constructed exact Player bindings are invalid", err)
	}
}

func TestCurrentPartialGraphRootsOnlyMissingActors(t *testing.T) {
	f, s, w := partialCurrentGraph(t)
	s.SavedDocument.Document.World.TerrainIdentity = 0
	before, err := sav.EncodeDocumentData(*s.SavedDocument.Document)
	if err != nil {
		t.Fatal(err)
	}
	hash := w.Hash()
	groups, orders, _ := w.SavedGroups()
	state, err := f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal("missing live actor remained unreachable", err)
	}
	if state.Document.World.TerrainIdentity == 0 || s.SavedDocument.Document.World.TerrainIdentity != 0 {
		t.Fatal("absent current terrain identity was not constructed independently")
	}
	for _, r := range state.Document.Objects {
		for _, v := range r.Values {
			if (v.Name == "Identity" || v.Name == "This") && v.Value == state.Document.World.TerrainIdentity {
				t.Fatal("constructed terrain identity collides with an actor or Player")
			}
		}
	}
	var actual, root SnapshotSAVGroupBinding
	for _, b := range state.GroupBindings.Groups {
		if b.RootOnly {
			root = b
		} else {
			actual = b
		}
	}
	if len(state.Actors) != 2 || len(state.GroupBindings.Groups) != 2 || actual.ID != 71 || root.ID == 0 || root.ID == actual.ID || root.PlayerObject != actual.PlayerObject || root.InlineIndex != actual.InlineIndex+1 {
		t.Fatal("completion replaced existing Groups or omitted live actor", state.Actors, state.GroupBindings.Groups)
	}
	r := &state.Document.Objects[actual.PlayerObject-1].Groups[actual.InlineIndex]
	if !bytes.Equal(crossingRawField(t, r, "G3C")[:76], groups[0].AI[:]) || !bytes.Equal(crossingRawField(t, r, "G20"), []byte{11, 0, 19, 0}) {
		t.Fatal("completion replaced existing AI or ordered words")
	}
	refs, _ := savedObjectRefs(r, "Actors")
	if len(refs) != 2 || refs[0] != 0 {
		t.Fatal("completion changed current ordered null membership", refs)
	}
	rootRefs, _ := savedObjectRefs(&state.Document.Objects[root.PlayerObject-1].Groups[root.InlineIndex], "Actors")
	if len(rootRefs) != 1 || rootRefs[0] == refs[1] || state.Actors[0].EntityID != 0 || rootRefs[0] != state.Actors[0].ObjectIndex {
		t.Fatal("zero entity identity or distinct actor root lost", rootRefs, refs, state.Actors)
	}
	players := state.Document.Players
	// The Player list is built from the World's Players: a null or repeated
	// root that only the loaded document held is not carried forward.
	if len(players) != len(state.GroupBindings.Players) || slices.Contains(players, 0) {
		t.Fatal("Player roots are not the current Players", players)
	}
	if err := projectSavedGroups(state, w); err != nil || state.GroupBindings.Unavailable != "" {
		t.Fatal("repeated projection lost actor roots", err, state.GroupBindings.Unavailable)
	}
	if _, err := sav.EncodeDocumentData(*state.Document); err != nil {
		t.Fatal("completed graph is not encodable", err)
	}
	afterGroups, afterOrders, _ := w.SavedGroups()
	if w.Hash() != hash || !reflect.DeepEqual(groups, afterGroups) || !reflect.DeepEqual(orders, afterOrders) {
		t.Fatal("SAVE materialization changed native Group or World state")
	}
	unchanged, err := sav.EncodeDocumentData(*s.SavedDocument.Document)
	if err != nil || !bytes.Equal(before, unchanged) {
		t.Fatal("completion mutated the earlier snapshot", err)
	}
	state.Document.Objects = append(state.Document.Objects, mustNewDiaryRecord(0, 12345))
	if _, _, err := sav.ReindexDocumentData(*state.Document); err == nil || !strings.Contains(err.Error(), "unreachable") {
		t.Fatal("unrelated orphan was silently rooted or deleted", err)
	}
}

func TestCurrentActorRootKeepsExactEqualSlotOwners(t *testing.T) {
	fixture := func(t *testing.T) (*SnapshotSAVDocument, *sim.World) {
		t.Helper()
		left, right, actor := mustNewRecord("Player"), mustNewRecord("Player"), mustNewRecord("Unit")
		for _, p := range []*sav.DocumentRecordData{&left, &right} {
			mustSetValue(p, "Slot", 1)
		}
		mustSetValue(&left, "This", 100)
		mustSetValue(&right, "This", 200)
		mustSetValue(&actor, "Reference", 200)
		state := &SnapshotSAVDocument{Document: &sav.DocumentData{Players: []uint16{1, 0, 1, 2}, Objects: []sav.DocumentRecordData{left, right, actor}},
			Actors: []SnapshotSAVActor{{EntityID: 0, ObjectIndex: 3}}, GroupBindings: &SnapshotSAVGroupBindings{Version: 1, PlayersPresent: true,
				Players: []SnapshotSAVGroupPlayerBinding{{ID: 7, ObjectIndex: 1}, {ID: 9, ObjectIndex: 2}}}}
		w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{{ID: 0, X: 2, Y: 2, Owner: 1, HP: 10, MaxHP: 10}})
		if err != nil {
			t.Fatal(err)
		}
		return state, w
	}
	s, w := fixture(t)
	if err := appendCurrentActorRoots(s, w); err != nil {
		t.Fatal(err)
	}
	if len(s.Document.Objects[0].Groups) != 0 || len(s.Document.Objects[1].Groups) != 1 || len(s.GroupBindings.Groups) != 1 || s.GroupBindings.Groups[0].ContainerID != 9 || !slices.Equal(s.Document.Players, []uint16{1, 0, 1, 2}) {
		t.Fatal("equal-slot Players were merged or the wrong owner was chosen", s.GroupBindings)
	}
	for _, tc := range []struct {
		name string
		edit func(*SnapshotSAVDocument)
	}{
		{"ambiguous owner", func(s *SnapshotSAVDocument) { mustSetValue(&s.Document.Objects[2], "Reference", 300) }},
		{"colliding owner identity", func(s *SnapshotSAVDocument) { mustSetValue(&s.Document.Objects[0], "This", 200) }},
		{"invalid owner slot", func(s *SnapshotSAVDocument) { mustSetValue(&s.Document.Objects[1], "Slot", 65536) }},
		{"duplicate owner binding", func(s *SnapshotSAVDocument) { s.GroupBindings.Players[1].ObjectIndex = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, w := fixture(t)
			tc.edit(s)
			if err := appendCurrentActorRoots(s, w); err == nil {
				t.Fatal("invalid owner metadata created actor roots")
			}
			if len(s.GroupBindings.Groups) != 0 || len(s.Document.Objects[0].Groups)+len(s.Document.Objects[1].Groups) != 0 {
				t.Fatal("invalid owner metadata partly changed the graph")
			}
		})
	}
}

func TestCurrentActorRootOrdinaryEditsOverrideAbsence(t *testing.T) {
	f, s, w := partialCurrentGraph(t)
	raw, err := f.ExportCurrentSave(s, "partial current graph")
	if err != nil {
		t.Fatal(err)
	}
	before, _, _ := w.SavedGroups()
	for _, change := range []string{"none", "AI", "words", "selector", "owner"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal(err)
			}
			var root currentGroupContinuation
			for _, row := range a.Groups {
				if row.RootOnly {
					root = row
				}
			}
			if root.Object == 0 || len(root.RootMembers) != 1 {
				t.Fatal("missing explicit absent topology binding", a.Groups)
			}
			g := &doc.Objects[root.Object-1].Groups[root.Inline]
			switch change {
			case "AI":
				crossingRawField(t, g, "G3C")[0x12] = 47
			case "words":
				crossingWordList(t, g, "G20", []uint16{17, 23})
			case "selector":
				mustSetValue(g, "G1C", 93)
			case "owner":
				mustSetValue(g, "G44", 0)
			}
			refs, _ := savedObjectRefs(g, "Actors")
			mustSetValue(&doc.Objects[refs[0]-1], "Health", 23)
			edited, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := cellStateFront(t)
			cold.Campaign, cold.Table = f.Campaign, f.Table
			open, town, err := cold.RestoreOriginal(edited)
			if err != nil || town {
				t.Fatal("ordinary actor root cannot LOAD", err, town)
			}
			if err := cold.App("current roots").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			groups, _, _ := cold.live.world.SavedGroups()
			want := len(before)
			if change != "none" {
				want++
			}
			if len(groups) != want || len(cold.live.world.Entities()) != 2 {
				t.Fatal("absence swallowed ordinary edits or lost actor population", len(groups), want, cold.live.world.Entities())
			}
			var added sim.SavedGroup
			for _, g := range groups {
				if g.ID != before[0].ID {
					added = g
				}
			}
			if change != "none" && (added.ID <= w.GroupHighWater() || cold.live.world.GroupHighWater() < added.ID) {
				t.Fatal("ordinary edit did not acquire a fresh native identity", added.ID, cold.live.world.GroupHighWater())
			}
			if change == "AI" && added.AI[0x12] != 47 || change == "words" && !slices.Equal(added.Words, []uint16{17, 23}) || change == "selector" && added.Selector != 93 || change == "owner" && added.Owner.Class != 0 {
				t.Fatal("unchanged absence leaf replaced ordinary Group values", added)
			}
			found := false
			for _, e := range cold.live.world.Entities() {
				if e.X == 18 && e.Y == 18 {
					found = e.HP == 23
					if change == "selector" && e.Group != 93 {
						t.Fatal("unchanged native selector replaced ordinary Group selector", e.Group)
					}
				}
			}
			if !found {
				t.Fatal("unchanged absence leaf replaced ordinary actor health")
			}
			next, _, err := cold.Snapshot(true)
			if err != nil {
				t.Fatal("restored root metadata is inconsistent", err)
			}
			if _, err := cold.ExportCurrentSave(next, "second root cycle"); err != nil {
				t.Fatal("second cycle lost current Group authority", err)
			}
		})
	}
}

func TestCurrentDetachedActorIsAdmittedBeforeContinuation(t *testing.T) {
	f, s, _ := partialCurrentGraph(t, true)
	raw, err := f.ExportCurrentSave(s, "off-map current actor")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"none", "ordinary health", "current outside position", "missing presence", "duplicate presence", "missing binding", "unknown binding"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil {
				t.Fatal(err)
			}
			actorAt := -1
			for i, row := range a.Actions.Actors {
				if row.Entity == 0 {
					actorAt = i
				}
			}
			if actorAt < 0 || !a.Actions.Actors[actorAt].OffMap {
				t.Fatal("fixture lost explicit native presence")
			}
			switch change {
			case "ordinary health":
				for _, b := range a.Bindings {
					if b.ID == 0 && !b.Structure && !b.Missing {
						mustSetValue(&doc.Objects[b.Object-1], "Health", 19)
					}
				}
			case "missing presence":
				a.Actions.Actors = slices.Delete(a.Actions.Actors, actorAt, actorAt+1)
			case "current outside position":
				a.Actions.Actors[actorAt].OffMap = false
			case "duplicate presence":
				a.Actions.Actors = append(a.Actions.Actors, a.Actions.Actors[actorAt])
			case "missing binding", "unknown binding":
				for i := range a.Bindings {
					if a.Bindings[i].ID == 0 && !a.Bindings[i].Structure {
						if change == "missing binding" {
							a.Bindings[i].Object, a.Bindings[i].Missing = 0, true
						} else {
							a.Bindings[i].ID = 999
						}
					}
				}
			}
			if change != "none" && change != "ordinary health" {
				payload, err := json.Marshal(a)
				if err != nil {
					t.Fatal(err)
				}
				if err := sav.SetNativeActions(&doc.State, payload); err != nil {
					t.Fatal(err)
				}
			}
			edited, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			cold := cellStateFront(t)
			cold.Campaign, cold.Table = f.Campaign, f.Table
			app := cold.App("detached current actor")
			if err := app.OpenMission(cold.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			before, _, err := cold.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			hash := cold.live.world.Hash()
			open, town, err := cold.RestoreOriginal(edited)
			if err == nil && !town {
				err = app.OpenMission(open)
			}
			if change != "none" && change != "ordinary health" && change != "current outside position" {
				if err == nil {
					t.Fatal("invalid presence admitted an outside-map actor")
				}
				after, _, snapshotErr := cold.Snapshot(true)
				if snapshotErr != nil || cold.live.world.Hash() != hash || !reflect.DeepEqual(before, after) {
					t.Fatal("invalid exact presence changed the active session", snapshotErr)
				}
				return
			}
			if err != nil || town {
				t.Fatal("valid off-map current actor cannot LOAD", err, town)
			}
			wantHP := int32(37)
			if change == "ordinary health" {
				wantHP = 19
			}
			found := false
			for _, e := range cold.live.world.Entities() {
				if e.ID == 0 {
					found = e.X == 54 && e.Y == 100 && e.HP == wantHP && e.OffMap == (change != "current outside position")
				}
			}
			if !found {
				t.Fatal("off-map coordinate or ordinary pool changed", cold.live.world.Entities())
			}
		})
	}
}

func TestCurrentGroupsKeepAbsentPlayerAndFormationCarriers(t *testing.T) {
	f, s, base := partialCurrentGraph(t)
	w, err := sim.NewWorld(11, base.Bounds(), sim.ModeCanonical, nil, base.Entities())
	if err != nil {
		t.Fatal(err)
	}
	groups, _, _ := base.SavedGroups()
	g := groups[0]
	g.Authored, g.ContainerID, g.OwnerID, g.Owner = false, 0, 0, sim.SavedGroupReference{}
	g.Members = []sim.SavedGroupMember{{Entity: 41, Bound: true}, {}, {Entity: 0, Bound: true}}
	if err := w.ImportSavedGroups([]sim.SavedGroup{g}, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ConstructSavedStructures(nil, nil); err != nil {
		t.Fatal(err)
	}
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	s.SavedDocument = nil
	raw, err := f.ExportCurrentSave(s, "current Groups without Player provenance")
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"unchanged", "ordinary AI", "ordinary cell baseline"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a.GroupPlayers == nil || *a.GroupPlayers || a.GroupFormations == nil || *a.GroupFormations || len(a.Groups) != 1 {
				t.Fatal("absent provenance not explicit", a, err)
			}
			row := a.Groups[0]
			if change == "ordinary AI" {
				crossingRawField(t, &doc.Objects[row.Object-1].Groups[row.Inline], "G3C")[0x12] = 67
			}
			if len(a.AbsentStructureCells) == 0 {
				t.Fatal("fixture lacks actor-cell transport")
			}
			if change == "ordinary cell baseline" {
				doc.World.Cells[0].Cost = 43
			}
			input, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for range 2 {
				cold := cellStateFront(t)
				cold.Campaign, cold.Table = f.Campaign, f.Table
				open, town, err := cold.RestoreOriginal(input)
				if err != nil || town {
					t.Fatal("ordinary Group cannot LOAD", town, err)
				}
				if err := cold.App("current Group provenance").OpenMission(open); err != nil {
					t.Fatal(err)
				}
				got, _, present := cold.live.world.SavedGroups()
				_, players := cold.live.world.SavedGroupPlayers()
				_, formations := cold.live.world.SavedPlayerFormations()
				if !present || players || formations || len(got) != 1 || got[0].ID != g.ID || got[0].ContainerID != 0 || got[0].OwnerID != 0 || got[0].Reference != g.Reference || got[0].Owner != g.Owner || !slices.Equal(got[0].Words, g.Words) || !slices.Equal(got[0].Path, g.Path) {
					t.Fatal("ordinary transport invented native container history", got, players, formations)
				}
				wantAI := g.AI
				if change == "ordinary AI" {
					wantAI[0x12] = 67
				}
				if got[0].AI != wantAI || len(got[0].Members) != 3 || got[0].Members[1].Bound || got[0].Members[1].Archive != 0 || len(cold.live.world.Entities()) != 2 {
					t.Fatal("current AI, null member or actor population changed", got)
				}
				sources, cells, present := cold.live.world.SavedStructures()
				wantCells := 0
				if change == "ordinary cell baseline" {
					wantCells = 1
				}
				if !present || len(sources) != 0 || len(cells) != wantCells || wantCells == 1 && cells[0].BaselineCost != 43 {
					t.Fatal("ordinary actor cells changed native structure presence", sources, cells, present)
				}
				next, _, err := cold.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				input, err = cold.ExportCurrentSave(next, "second absent Player cycle")
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
