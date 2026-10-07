package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// This comparator is deliberately separate from the source-time oracle.
// After a real command, the authoritative expectation is the current World.
// Only the replaced transport pointer tails still come from the source.
// An Unavailable marker means the retained Document is NOT a current projection.
func groups1155CurrentDifferences(source groups1155Source, state *SnapshotSAVDocument, world *sim.World) []string {
	var c group1155Comparison
	if state == nil || state.Document == nil || state.GroupBindings == nil || world == nil {
		return []string{"current Group Document/bindings absent"}
	}
	b := state.GroupBindings
	if b.Unavailable != "" {
		return []string{"current Group Document unavailable: " + b.Unavailable}
	}
	if _, err := cloneSavedGroupBindings(b, state.Document, state.Actors); err != nil {
		return []string{"current Group bindings invalid: " + err.Error()}
	}
	groups, orders, present := world.SavedGroups()
	players, playersPresent := world.SavedGroupPlayers()
	type site struct {
		player uint16
		inline uint32
	}
	actualIDs, actualMembers := map[uint32]bool{}, map[sim.EntityID]bool{}
	for _, g := range groups {
		actualIDs[g.ID] = true
		for _, m := range g.Members {
			if m.Bound {
				actualMembers[m.Entity] = true
			}
		}
	}
	actorsByObject := map[uint16]sim.EntityID{}
	for _, a := range state.Actors {
		if !a.Retired {
			actorsByObject[a.ObjectIndex] = a.EntityID
		}
	}
	entities := map[sim.EntityID]sim.Entity{}
	for _, e := range world.Entities() {
		entities[e.ID] = e
	}
	roots := map[site]bool{}
	for _, row := range b.Groups {
		if !row.RootOnly {
			continue
		}
		if actualIDs[row.ID] {
			c.fail("current Group %d is falsely marked as an absent root", row.ID)
			continue
		}
		r := &state.Document.Objects[row.PlayerObject-1].Groups[row.InlineIndex]
		selector := player1154Values(*r)["G1C"]
		refs, _ := savedObjectRefs(r, "Actors")
		valid := true
		for _, object := range refs {
			id, bound := actorsByObject[object]
			e, live := entities[id]
			if !bound || !live || actualMembers[id] || e.Group != selector {
				c.fail("current absent root %d member %d lacks an exact ungrouped actor/selector", row.ID, object)
				valid = false
			}
		}
		if valid {
			roots[site{row.PlayerObject, row.InlineIndex}] = true
		}
	}
	if !present || !playersPresent || !b.PlayersPresent || len(groups)+len(roots) != len(b.Groups) || len(players) != len(nativeSavedPlayerBindings(b)) {
		c.fail("current Group/Player population differs")
	}
	playerObjects := map[uint32]uint16{}
	used := map[uint16]bool{}
	for _, row := range b.Players {
		if row.ID == 0 || playerObjects[row.ID] != 0 || row.ObjectIndex == 0 || used[row.ObjectIndex] {
			c.fail("current Player identities are not injective")
		}
		playerObjects[row.ID], used[row.ObjectIndex] = row.ObjectIndex, true
	}
	for _, p := range players {
		index := playerObjects[p.ID]
		if index == 0 || int(index) > len(state.Document.Objects) || player1154Values(state.Document.Objects[index-1])["Slot"] != p.Slot {
			c.fail("current Player %d exact binding/Slot differs", p.ID)
		}
	}
	byGroup := map[uint32]SnapshotSAVGroupBinding{}
	for _, row := range b.Groups {
		if _, duplicate := byGroup[row.ID]; duplicate {
			c.fail("current Group ID %d repeated", row.ID)
		}
		byGroup[row.ID] = row
	}
	actorObjects := map[sim.EntityID]uint16{}
	for _, a := range state.Actors {
		if !a.Retired {
			actorObjects[a.EntityID] = a.ObjectIndex
		}
	}
	bound, unbound := map[sim.EntityID]uint16{}, map[uint16]uint16{}
	used = map[uint16]bool{}
	for _, m := range b.Members {
		if m.ObjectIndex == 0 || used[m.ObjectIndex] {
			c.fail("current Group member DTO identities are not injective")
		}
		used[m.ObjectIndex] = true
		if m.Bound {
			if _, duplicate := bound[m.EntityID]; duplicate {
				c.fail("current Group member native identity repeated")
			}
			bound[m.EntityID] = m.ObjectIndex
			if actorObjects[m.EntityID] != 0 && actorObjects[m.EntityID] != m.ObjectIndex {
				c.fail("current Group member and actor exact bindings differ")
			}
		} else {
			if unbound[m.UnresolvedHandle] != 0 || m.UnresolvedHandle == 0 {
				c.fail("current Group unresolved identity repeated/zero")
			}
			unbound[m.UnresolvedHandle] = m.ObjectIndex
		}
	}
	key := func(ref sim.SavedGroupReference, row SnapshotSAVGroupReferenceBinding) uint32 {
		if ref.Class == 0 {
			if row.ObjectIndex != 0 {
				c.fail("unresolved current Group reference has a DTO target")
			}
			return 0
		}
		if row.ObjectIndex == 0 || int(row.ObjectIndex) > len(state.Document.Objects) {
			c.fail("current Group reference lacks exact target")
			return 0
		}
		record := state.Document.Objects[row.ObjectIndex-1]
		field := "Identity"
		if record.Class == "Player" || record.Class == "Spell" {
			field = "This"
		}
		v, ok := player1154Values(record)[field]
		if !ok || v == 0 {
			c.fail("current Group reference target lacks its wire key")
		}
		return v
	}
	sites := map[site]bool{}
	for _, g := range groups {
		row, exists := byGroup[g.ID]
		where := site{row.PlayerObject, row.InlineIndex}
		if !exists || sites[where] || row.ContainerID != g.ContainerID || row.Authored != g.Authored || row.PlayerObject != playerObjects[g.ContainerID] {
			c.fail("current Group %d lost exact container/inline identity", g.ID)
		}
		sites[where] = true
		if row.PlayerObject == 0 || int(row.PlayerObject) > len(state.Document.Objects) || uint64(row.InlineIndex) >= uint64(len(state.Document.Objects[row.PlayerObject-1].Groups)) {
			c.fail("current Group %d has no inline record", g.ID)
			continue
		}
		want := group1155Raw{selector: g.Selector, ai: make([]byte, 80), words: g.Words, path: g.Path}
		copy(want.ai, g.AI[:])
		if !g.Authored && g.ID > 0 && int(g.ID) <= len(source.groups) {
			copy(want.ai[76:], source.groups[g.ID-1].ai[76:])
		}
		want.reference.Key, want.owner.Key = key(g.Reference, row.Reference), key(g.Owner, row.Owner)
		members := make([]uint16, len(g.Members))
		for i, m := range g.Members {
			members[i] = unbound[m.Archive]
			if m.Bound {
				members[i] = bound[m.Entity]
			}
			if members[i] == 0 && (m.Bound || m.Archive != 0) {
				c.fail("current Group %d member %d lost exact binding", g.ID, i)
			}
		}
		c.group(state.Document.Objects[row.PlayerObject-1].Groups[row.InlineIndex], want, members, fmt.Sprintf("current Group %d", g.ID))
	}
	count, total := 0, 0
	for i, object := range state.Document.Objects {
		total += len(object.Groups)
		for j := range object.Groups {
			if roots[site{uint16(i + 1), uint32(j)}] {
				continue
			}
			count++
			if !sites[site{uint16(i + 1), uint32(j)}] {
				c.fail("current Document contains extra Group at %d/%d", i+1, j)
			}
		}
	}
	if count != len(groups) {
		c.fail("current Document Group count %d != World %d", count, len(groups))
	}
	if total != len(groups)+len(roots) {
		c.fail("current Document total Group count %d != gameplay %d plus absent roots %d", total, len(groups), len(roots))
	}
	// The registered producer is Move. Imported active patrol remains a
	// separate list; no Group Path copy/setter is invoked to manufacture it.
	archives := map[sim.EntityID]uint16{}
	for _, e := range world.Entities() {
		archives[e.ID] = e.SourceBinding.ArchiveIndex
	}
	for _, d := range world.OriginalDeadActors() {
		archives[d.ID] = d.Source.ArchiveIndex
	}
	for _, order := range orders {
		index := actorObjects[order.Entity]
		if index == 0 {
			index = bound[order.Entity]
		}
		if index == 0 || int(index) > len(state.Document.Objects) {
			c.fail("current order actor %d lacks an exact object", order.Entity)
			continue
		}
		want := group1155ActorRaw{state: order.State, patrol: order.Patrol, order: make([]byte, 148)}
		copy(want.order, order.Raw[:])
		if raw, found := source.actors[archives[order.Entity]]; found {
			copy(want.order[144:], raw.order[144:])
		}
		// An escort's authored pointer operands need a different producer
		// oracle; this helper must name that boundary rather than compare stale
		// raw source pointers with typed current Entity targets.
		if order.Authored && (order.State == 8 || order.State == 0x11) {
			c.fail("current authored escort actor %d is outside Move projection witness", order.Entity)
			continue
		}
		c.order(state.Document.Objects[index-1], want, fmt.Sprintf("current actor %d", order.Entity))
	}
	slices.Sort(c)
	return c
}

func TestCurrentGroupRootObserverRejectsChangedTopology(t *testing.T) {
	raw := groups1155DyingFixture(t)
	want, _, _ := groups1155Inputs(t, raw)
	f := newGroupFront(t, -1)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal(town, err)
	}
	if err := f.App("current Group observer").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	s := groupDocumentSnapshot1115(t, f)
	state, err := f.materializeCurrentWorld(s, f.live.world)
	if err != nil {
		t.Fatal(err)
	}
	if differences := groups1155CurrentDifferences(want, state, f.live.world); len(differences) != 0 {
		t.Fatal("valid current root rejected", differences)
	}
	root := -1
	for i, row := range state.GroupBindings.Groups {
		if row.RootOnly {
			root = i
		}
	}
	if root < 0 {
		t.Fatal("fixture has no explicit actor root")
	}
	for _, name := range []string{"unmarked root", "changed AI", "changed selector", "actual Group marked absent", "grouped member", "missing root binding", "duplicate root binding", "missing actual Group"} {
		t.Run(name, func(t *testing.T) {
			changed, err := cloneSavedDocument(state)
			if err != nil {
				t.Fatal(err)
			}
			b := changed.GroupBindings
			r := b.Groups[root]
			record := &changed.Document.Objects[r.PlayerObject-1].Groups[r.InlineIndex]
			switch name {
			case "unmarked root":
				b.Groups[root].RootOnly = false
			case "changed AI":
				crossingRawField(t, record, "G3C")[0x12] = 37
			case "changed selector":
				mustSetValue(record, "G1C", 91)
			case "actual Group marked absent":
				b.Groups[0].RootOnly = true
			case "grouped member":
				groups, _, _ := f.live.world.SavedGroups()
				var object uint16
				for _, g := range groups {
					for _, m := range g.Members {
						for _, actor := range changed.Actors {
							if m.Bound && actor.EntityID == m.Entity {
								object = actor.ObjectIndex
							}
						}
					}
				}
				if object == 0 {
					t.Fatal("fixture has no actual Group member")
				}
				mustSetRefs(record, "Actors", []uint16{object})
			case "missing root binding":
				b.Groups = slices.Delete(b.Groups, root, root+1)
			case "duplicate root binding":
				b.Groups = append(b.Groups, r)
			case "missing actual Group":
				g := b.Groups[0]
				player := &changed.Document.Objects[g.PlayerObject-1]
				player.Groups = slices.Delete(player.Groups, int(g.InlineIndex), int(g.InlineIndex)+1)
				b.Groups = b.Groups[1:]
				for i := range b.Groups {
					if b.Groups[i].PlayerObject == g.PlayerObject && b.Groups[i].InlineIndex > g.InlineIndex {
						b.Groups[i].InlineIndex--
					}
				}
			}
			if differences := groups1155CurrentDifferences(want, changed, f.live.world); len(differences) == 0 {
				t.Fatal("observer concealed changed Group topology", name)
			}
		})
	}
}
