package sim

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
)

func importSavedGroupPlayers1115(t *testing.T, w *World, players []SavedGroupPlayer, containers []SavedGroupContainer) {
	t.Helper()
	if err := w.ImportSavedGroupPlayers(players, containers); err != nil {
		t.Fatal(err)
	}
}

func roundTripSavedGroupPlayers1115(t *testing.T, w *World) *World {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if w.Hash() != fresh.Hash() {
		t.Fatal("Player/container registry changed across native SAVE")
	}
	players, present := w.SavedGroupPlayers()
	gotPlayers, gotPresent := fresh.SavedGroupPlayers()
	groups, orders, groupsPresent := w.SavedGroups()
	gotGroups, gotOrders, gotGroupsPresent := fresh.SavedGroups()
	groupsEqual := slices.EqualFunc(groups, gotGroups, func(a, b SavedGroup) bool {
		listsEqual := slices.Equal(a.Words, b.Words) && slices.Equal(a.Path, b.Path) && slices.Equal(a.Members, b.Members)
		a.Words, a.Path, a.Members, b.Words, b.Path, b.Members = nil, nil, nil, nil, nil, nil
		return listsEqual && reflect.DeepEqual(a, b)
	})
	ordersEqual := slices.EqualFunc(orders, gotOrders, func(a, b SavedActorOrder) bool {
		patrolEqual := slices.Equal(a.Patrol, b.Patrol)
		a.Patrol, b.Patrol = nil, nil
		return patrolEqual && reflect.DeepEqual(a, b)
	})
	if present != gotPresent || groupsPresent != gotGroupsPresent || !slices.Equal(players, gotPlayers) || !groupsEqual || !ordersEqual {
		t.Fatal("native SAVE lost Player/container values despite matching hash")
	}
	return &fresh
}

func savedGroupIDs1115(w *World) []uint32 {
	groups, _, _ := w.SavedGroups()
	ids := make([]uint32, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	return ids
}

func TestSavedGroupPlayers1115PresenceAndDetachedImport(t *testing.T) {
	w := savedGroupWorld(t)
	if players, present := w.SavedGroupPlayers(); present || len(players) != 0 {
		t.Fatal("saved Groups alone fabricated a Player registry", players, present)
	}
	w.savedGroups.Groups[0].Owner = SavedGroupReference{Key: 0x12345678, Archive: 18, Class: 2}
	w.savedGroups.Groups[0].Reference = SavedGroupReference{Key: 0x87654321, Archive: 19, Class: 1, Owner: 99}
	beforeGroup := w.savedGroups.Groups[0]
	players := []SavedGroupPlayer{{ID: 101, Slot: 1}, {ID: 102, Slot: 1}, {ID: 201, Slot: 2}}
	containers := []SavedGroupContainer{{GroupID: 71, PlayerID: 102}, {GroupID: 72, PlayerID: 201}}
	beforeHash := w.Hash()
	importSavedGroupPlayers1115(t, w, players, containers)
	if w.Hash() == beforeHash {
		t.Fatal("present Player/container registry is absent from hashed state")
	}
	groups, _, _ := w.SavedGroups()
	if groups[0].ContainerID != 102 || groups[1].ContainerID != 201 || groups[0].Owner != beforeGroup.Owner || groups[0].Reference != beforeGroup.Reference {
		t.Fatal("exact container was inferred from or overwrote Group references", groups)
	}
	wantPlayers := slices.Clone(players)
	beforeHash = w.Hash()
	players[0].Slot, containers[0].PlayerID = 88, 201
	exported, present := w.SavedGroupPlayers()
	if !present || !reflect.DeepEqual(exported, wantPlayers) {
		t.Fatal("duplicate semantic slots lost distinct Player identities", exported, present)
	}
	exported[1].ID = 999
	groups[0].ContainerID = 201
	got, _ := w.SavedGroupPlayers()
	if w.Hash() != beforeHash || !reflect.DeepEqual(got, wantPlayers) || w.savedGroups.Groups[0].ContainerID != 102 {
		t.Fatal("caller-owned import/export slices alias current Player/container state")
	}
	fresh := roundTripSavedGroupPlayers1115(t, w)
	got, present = fresh.SavedGroupPlayers()
	if !present || !reflect.DeepEqual(got, wantPlayers) {
		t.Fatal("native SAVE lost exact Player identities", got, present)
	}
	beforeHash = w.Hash()
	if err := w.ImportSavedGroupPlayers(wantPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}}); err == nil || w.Hash() != beforeHash {
		t.Fatal("second import replaced established container provenance", err)
	}
}

func TestSavedGroupPlayers1115PresentEmptyIsNotAbsent(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{8, 8}, nil)
	before := w.Hash()
	if err := w.ImportSavedGroupPlayers(nil, nil); err == nil || w.Hash() != before {
		t.Fatal("Player registry accepted without saved Groups", err)
	}
	if err := w.ImportSavedGroups(nil, nil); err != nil {
		t.Fatal(err)
	}
	before = w.Hash()
	importSavedGroupPlayers1115(t, w, nil, nil)
	if players, present := w.SavedGroupPlayers(); !present || len(players) != 0 || w.Hash() == before {
		t.Fatal("present-empty Player registry collapsed into absence", players, present)
	}
	fresh := roundTripSavedGroupPlayers1115(t, w)
	if players, present := fresh.SavedGroupPlayers(); !present || len(players) != 0 {
		t.Fatal("native SAVE dropped present-empty Player registry", players, present)
	}
}

func TestSavedGroupPlayers1115MalformedImportIsAtomic(t *testing.T) {
	goodPlayers := []SavedGroupPlayer{{ID: 101, Slot: 1}, {ID: 201, Slot: 2}}
	goodContainers := []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}}
	for _, tc := range []struct {
		name       string
		players    []SavedGroupPlayer
		containers []SavedGroupContainer
	}{
		{"zero-player", []SavedGroupPlayer{{ID: 0, Slot: 1}, {ID: 201, Slot: 2}}, goodContainers},
		{"duplicate-player", []SavedGroupPlayer{{ID: 101, Slot: 1}, {ID: 101, Slot: 2}}, goodContainers},
		{"unordered-players", []SavedGroupPlayer{{ID: 201, Slot: 2}, {ID: 101, Slot: 1}}, goodContainers},
		{"zero-group", goodPlayers, []SavedGroupContainer{{GroupID: 0, PlayerID: 101}, {GroupID: 72, PlayerID: 201}}},
		{"duplicate-group", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 71, PlayerID: 201}}},
		{"unordered-groups", goodPlayers, []SavedGroupContainer{{GroupID: 72, PlayerID: 201}, {GroupID: 71, PlayerID: 101}}},
		{"missing-group", goodPlayers, goodContainers[:1]},
		{"extra-group", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}, {GroupID: 73, PlayerID: 101}}},
		{"unknown-group", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 73, PlayerID: 201}}},
		{"late-unknown-player", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 999}}},
		{"late-zero-player", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 0}}},
		{"nonmonotonic-container-blocks", goodPlayers, []SavedGroupContainer{{GroupID: 71, PlayerID: 201}, {GroupID: 72, PlayerID: 101}}},
		{"no-players", nil, goodContainers},
		{"no-mapping", goodPlayers, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := savedGroupWorld(t)
			before, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			hash := w.Hash()
			if err := w.ImportSavedGroupPlayers(tc.players, tc.containers); err == nil {
				t.Fatal("malformed Player/container import accepted")
			}
			after, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if players, present := w.SavedGroupPlayers(); present || len(players) != 0 || w.Hash() != hash || !bytes.Equal(before, after) {
				t.Fatal("rejected import partially changed current state", players, present)
			}
		})
	}
}

func TestSavedGroupPlayers1115SlotsAreNotIdentityOrFixedPlayerCount(t *testing.T) {
	w := savedGroupWorld(t)
	players := []SavedGroupPlayer{{ID: 1, Slot: 0}, {ID: 4000, Slot: ^uint32(0)}}
	importSavedGroupPlayers1115(t, w, players, []SavedGroupContainer{{GroupID: 71, PlayerID: 1}, {GroupID: 72, PlayerID: 4000}})
	fresh := roundTripSavedGroupPlayers1115(t, w)
	if got, present := fresh.SavedGroupPlayers(); !present || !reflect.DeepEqual(got, players) {
		t.Fatal("semantic slot was range-clamped or treated as file-local identity", got, present)
	}
}

func TestSavedGroupPlayers1115PlayerSlotAndExactContainerReachHash(t *testing.T) {
	build := func(slot, container uint32) *World {
		w := savedGroupWorld(t)
		importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, slot}, {102, 1}, {201, 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: container}, {GroupID: 72, PlayerID: 201}})
		return w
	}
	base := build(1, 101)
	if changedSlot := build(7, 101); changedSlot.Hash() == base.Hash() {
		t.Fatal("Player semantic slot is absent from hashed state")
	}
	if changedContainer := build(1, 102); changedContainer.Hash() == base.Hash() {
		t.Fatal("exact Group container is absent from hashed state")
	}
}

func TestSavedGroupPlayers1115OversizedRegistryImportIsAtomic(t *testing.T) {
	w := savedGroupWorld(t)
	players := make([]SavedGroupPlayer, 65536)
	for i := range players {
		players[i] = SavedGroupPlayer{ID: uint32(i + 1), Slot: 1}
	}
	before := w.Hash()
	if err := w.ImportSavedGroupPlayers(players, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}}); err == nil || w.Hash() != before {
		t.Fatal("oversized Player registry accepted or partially published", err)
	}
	if got, present := w.SavedGroupPlayers(); present || len(got) != 0 {
		t.Fatal("oversized registry changed provenance presence")
	}
}

func TestSavedGroupPlayers1115MalformedGroupReplacementIsAtomic(t *testing.T) {
	for _, failure := range []string{"unknown-player", "descending-player-blocks", "unknown-before-known", "new-id-below-highwater"} {
		t.Run(failure, func(t *testing.T) {
			w := savedGroupWorld(t)
			importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, 1}, {201, 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}})
			groups, orders, _ := w.SavedGroups()
			switch failure {
			case "unknown-player":
				groups[1].ContainerID = 999
			case "descending-player-blocks":
				groups[0].ContainerID, groups[1].ContainerID = 201, 101
			case "unknown-before-known":
				groups[0].ContainerID = 0
			case "new-id-below-highwater":
				groups = append(groups, SavedGroup{ID: 70, ContainerID: 201})
			}
			before := w.Hash()
			if err := w.ImportSavedGroups(groups, orders); err == nil || w.Hash() != before {
				t.Fatal("malformed Group replacement accepted or partially published", err)
			}
			roundTripSavedGroupPlayers1115(t, w)
		})
	}
}

func TestSavedGroupPlayers1115OrdinaryCommandContainerResolution(t *testing.T) {
	for _, tc := range []struct {
		name       string
		players    []SavedGroupPlayer
		owners     [2]uint32
		containers [2]uint32
		want       uint32
	}{
		{"same-exact-container-with-duplicate-slots", []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{102, 102}, 102},
		{"unique-slot-with-no-known-container", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{0, 0}, 101},
		{"unique-slot-with-one-known-container", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{101, 0}, 101},
		{"owner-changed-from-old-container", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{2, 2}, [2]uint32{101, 101}, 201},
		{"duplicate-slot-with-no-exact-container", []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{0, 0}, 0},
		{"duplicate-slot-with-only-one-known-container", []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{101, 0}, 0},
		{"mixed-owners-despite-common-container", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{1, 2}, [2]uint32{101, 101}, 0},
		{"mixed-exact-containers-despite-unique-owner-slot", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{101, 201}, 0},
		{"mixed-exact-identities-with-equal-slots", []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, [2]uint32{1, 1}, [2]uint32{101, 102}, 0},
		{"owner-has-no-player", []SavedGroupPlayer{{101, 1}, {201, 2}}, [2]uint32{9, 9}, [2]uint32{101, 101}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := savedGroupWorld(t)
			importSavedGroupPlayers1115(t, w, tc.players, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}})
			groups, orders, _ := w.SavedGroups()
			groups[0].Members = []SavedGroupMember{{Archive: 1, Entity: 10, Bound: true}}
			groups[1].Members = []SavedGroupMember{{Archive: 2, Entity: 20, Bound: true}}
			for i := range groups {
				groups[i].ContainerID = tc.containers[i]
				groups[i].Owner = SavedGroupReference{Key: 0xbadcafe, Archive: 19, Class: 2}
				w.entities[i].Owner = tc.owners[i]
			}
			if err := w.ImportSavedGroups(groups, orders); err != nil {
				t.Fatal(err)
			}
			w.commandSavedGroup([]int{1, 0}, orderGuard, cell{12, 13})
			g := w.savedGroupFor(10)
			if g == nil || g.ID != 73 || g.ContainerID != tc.want || !g.Authored {
				t.Fatal("ordinary command chose wrong exact container", g)
			}
			if got := g.Members; len(got) != 2 || got[0].Entity != 20 || got[1].Entity != 10 {
				t.Fatal("container derivation changed supplied member order", got)
			}
			if got := savedGroupIDs1115(w); len(got) != 3 || w.savedGroupByID(71) == nil || w.savedGroupByID(72) == nil {
				t.Fatal("command removed a Group emptied by that same command", got)
			}
			roundTripSavedGroupPlayers1115(t, w)
		})
	}
}

func TestSavedGroupPlayers1115CommandRemovesOnlyFirstPreexistingExactEmpty(t *testing.T) {
	w := savedGroupWorld(t)
	groups, orders, _ := w.SavedGroups()
	live := groups[0]
	live.ID = 600
	groups = []SavedGroup{
		{ID: 900, Selector: 90, Owner: SavedGroupReference{Class: 1, Owner: 1}},
		{ID: 800, Selector: 80, Owner: SavedGroupReference{Class: 1, Owner: 99}},
		{ID: 700, Selector: 70},
		live,
	}
	for i := range w.entities {
		w.entities[i].Owner = 1
	}
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, 2}, {201, 1}}, []SavedGroupContainer{
		{GroupID: 600, PlayerID: 201}, {GroupID: 700, PlayerID: 201}, {GroupID: 800, PlayerID: 201}, {GroupID: 900, PlayerID: 101},
	})
	w.commandSavedGroup([]int{1, 0, 2}, orderGuard, cell{12, 13})
	if got := savedGroupIDs1115(w); !slices.Equal(got, []uint32{900, 700, 600, 901}) {
		t.Fatal("cleanup removed foreign/second/newly-empty Group or changed registry order", got)
	}
	if w.savedGroupByID(900).Owner != groups[0].Owner || w.savedGroupByID(700).Owner != groups[2].Owner || len(w.savedGroupByID(600).Members) != 0 {
		t.Fatal("cleanup rewrote retained Group references or kept detached members")
	}
	if g := w.savedGroupByID(901); g.ContainerID != 201 || len(g.Members) != 3 || g.Members[0].Entity != 20 || g.Members[1].Entity != 10 || g.Members[2].Entity != 30 {
		t.Fatal("new Group lost exact container or current member order", g)
	}
	roundTripSavedGroupPlayers1115(t, w)
}

func TestSavedGroupPlayers1115UnknownCommandContainerDoesNotClean(t *testing.T) {
	w := savedGroupWorld(t)
	importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 102}})
	groups, orders, _ := w.SavedGroups()
	groups[0].ContainerID, groups[1].ContainerID = 0, 0
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	w.commandSavedGroup([]int{0, 1}, orderGuard, cell{12, 13})
	if got := savedGroupIDs1115(w); !slices.Equal(got, []uint32{71, 72, 73}) || w.savedGroupFor(10).ContainerID != 0 {
		t.Fatal("unknown container cleaned another unknown Group", got)
	}
}

func TestSavedGroupPlayers1115NewGroupsEnterEmptyPlayerBlocks(t *testing.T) {
	for _, handover := range []bool{false, true} {
		name := "ordinary-command"
		if handover {
			name = "handover"
		}
		t.Run(name, func(t *testing.T) {
			w := savedGroupWorld(t)
			if !handover {
				// Ungrouped actors exercise unique-slot fallback without a
				// contradictory old exact container deciding admission.
				w.savedGroups.Groups[0].Members = []SavedGroupMember{{Archive: 3, Entity: 30, Bound: true}}
			}
			importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, 1}, {151, 3}, {201, 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: 201}, {GroupID: 72, PlayerID: 201}})
			if handover {
				w.handOver(0, 1)
			} else {
				w.commandSavedGroup([]int{0}, orderGuard, cell{12, 13})
			}
			if got := savedGroupIDs1115(w); !slices.Equal(got, []uint32{73, 71, 72}) || w.savedGroupFor(10).ContainerID != 101 {
				t.Fatal("new Group did not enter initially empty first Player block", got)
			}
			if handover {
				w.handOver(0, 3)
			} else {
				w.entities[1].Owner = 3
				w.commandSavedGroup([]int{1}, orderGuard, cell{14, 15})
			}
			if got := savedGroupIDs1115(w); !slices.Equal(got, []uint32{73, 74, 71, 72}) || w.savedGroupByID(74).ContainerID != 151 {
				t.Fatal("new Group did not enter initially empty middle Player block", got)
			}
			roundTripSavedGroupPlayers1115(t, w)
		})
	}
}

func TestSavedGroupPlayers1115HandoverUsesDestinationWithoutCleanup(t *testing.T) {
	for _, tc := range []struct {
		name    string
		players []SavedGroupPlayer
		dest    uint32
		want    uint32
	}{
		{"unique-destination", []SavedGroupPlayer{{101, 1}, {201, 2}}, 2, 201},
		{"duplicate-destination", []SavedGroupPlayer{{101, 1}, {201, 2}, {202, 2}}, 2, 0},
		{"absent-destination", []SavedGroupPlayer{{101, 1}, {201, 2}}, 9, 0},
		{"same-slot-cannot-reuse-exact-source", []SavedGroupPlayer{{101, 1}, {102, 1}, {201, 2}}, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := savedGroupWorld(t)
			w.savedGroups.Groups[0].Members = []SavedGroupMember{{Archive: 1, Entity: 10, Bound: true}}
			importSavedGroupPlayers1115(t, w, tc.players, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 72, PlayerID: 201}})
			o := w.savedOrder(10)
			o.State, o.Patrol = 0xa, []uint16{0x1111, 0x1212}
			o.Raw[8], o.Raw[10], o.Raw[11], o.Raw[0x70] = 7, 25, 26, 13
			beforeOrder := *o
			beforeOrder.Patrol = slices.Clone(o.Patrol)
			beforeGroup := *w.savedGroupByID(72)
			w.handOver(0, tc.dest)
			g := w.savedGroupFor(10)
			if g == nil || g.ID != 73 || g.ContainerID != tc.want || w.entities[0].Owner != tc.dest || g.Owner.Class != 1 || g.Owner.Owner != tc.dest {
				t.Fatal("handover reused source container or ignored destination owner", g, w.entities[0])
			}
			if !reflect.DeepEqual(*w.savedOrder(10), beforeOrder) || !reflect.DeepEqual(*w.savedGroupByID(72), beforeGroup) {
				t.Fatal("handover changed actor-order policy or cleaned destination empty Group")
			}
			w.handOver(0, tc.dest)
			if got := savedGroupIDs1115(w); !slices.Equal(got, []uint32{71, 72, 73, 74}) || w.savedGroupFor(10).ContainerID != tc.want {
				t.Fatal("repeated handover cleaned empty Groups or reused identity", got)
			}
			if !reflect.DeepEqual(*w.savedOrder(10), beforeOrder) {
				t.Fatal("repeated handover reset retained actor order")
			}
			roundTripSavedGroupPlayers1115(t, w)
		})
	}
}

func TestSavedGroupPlayers1115IdentityHighwaterSurvivesRemovalAndNativeSave(t *testing.T) {
	w := savedGroupWorld(t)
	groups, orders, _ := w.SavedGroups()
	groups[1].ID = 900
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	importSavedGroupPlayers1115(t, w, []SavedGroupPlayer{{101, 1}, {201, 2}}, []SavedGroupContainer{{GroupID: 71, PlayerID: 101}, {GroupID: 900, PlayerID: 101}})
	w.commandSavedGroup([]int{0, 1}, orderGuard, cell{12, 13})
	if g := w.savedGroupFor(10); g.ID != 901 || w.savedGroupByID(900) != nil {
		t.Fatal("removing highest preexisting empty Group recycled its identity", g)
	}
	groups, orders, _ = w.SavedGroups()
	removed := *w.savedGroupByID(901)
	groups = slices.DeleteFunc(groups, func(g SavedGroup) bool { return g.ID == 901 })
	if err := w.ImportSavedGroups(groups, orders); err != nil {
		t.Fatal(err)
	}
	fresh := roundTripSavedGroupPlayers1115(t, w)
	for _, current := range []*World{w, fresh} {
		before := current.Hash()
		if err := current.ImportSavedGroups(append(slices.Clone(groups), removed), orders); err == nil || current.Hash() != before {
			t.Fatal("removed Group identity was reintroduced below highwater", err)
		}
		current.commandSavedGroup([]int{0, 1}, orderGuard, cell{14, 15})
		if g := current.savedGroupFor(10); g == nil || g.ID != 902 || g.ContainerID != 101 {
			t.Fatal("removed Group highwater was lost by replacement or native SAVE", g)
		}
	}
	if w.Hash() != fresh.Hash() {
		t.Fatal("next command differs after native SAVE of removed highwater identity")
	}
}
