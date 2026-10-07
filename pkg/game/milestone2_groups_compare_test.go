package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type group1155Comparison []string

func (c *group1155Comparison) fail(format string, args ...any) {
	*c = append(*c, fmt.Sprintf(format, args...))
}

func (c *group1155Comparison) raw(record sav.DocumentRecordData, name string, want []byte, label string) {
	found := 0
	for _, v := range record.Raw {
		if v.Name == name {
			found++
			if !bytes.Equal(v.Bytes, want) {
				c.fail("%s %s bytes differ", label, name)
			}
		}
	}
	if found != 1 {
		c.fail("%s %s raw presence %d != 1", label, name, found)
	}
}

func (c *group1155Comparison) count(record sav.DocumentRecordData, name string, want int, label string) {
	found := 0
	for _, v := range record.Counts {
		if v.Name == name {
			found++
			if uint64(v.Count) != uint64(want) {
				c.fail("%s %s count %d != %d", label, name, v.Count, want)
			}
		}
	}
	if found != 1 {
		c.fail("%s %s count presence %d != 1", label, name, found)
	}
}

func (c *group1155Comparison) group(record sav.DocumentRecordData, want group1155Raw, members []uint16, label string) {
	if record.Class != "Group" || len(record.Values) != 3 || len(record.Raw) != 3 || len(record.Counts) != 3 || len(record.RefSlots) != 1 {
		c.fail("%s Group field population differs", label)
	}
	values := player1154Values(record)
	for name, w := range map[string]uint32{"G1C": want.selector, "G40": want.reference.Key, "G44": want.owner.Key} {
		if got, present := values[name]; !present || got != w {
			c.fail("%s %s=%#x present=%t != %#x", label, name, got, present, w)
		}
	}
	c.raw(record, "G3C", want.ai, label)
	c.raw(record, "G20", group1155Bytes(want.words), label)
	c.raw(record, "G4C", group1155Bytes(want.path), label)
	c.count(record, "G20", len(want.words), label)
	c.count(record, "G4C", len(want.path), label)
	c.count(record, "Actors", len(members), label)
	if len(record.RefSlots) != 1 || record.RefSlots[0].Name != "Actors" || !slices.Equal(record.RefSlots[0].Objects, members) {
		c.fail("%s ordered member identities/null slots differ from %v", label, members)
	}
}

func (c *group1155Comparison) order(record sav.DocumentRecordData, want group1155ActorRaw, label string) {
	c.raw(record, "U158", want.order, label)
	c.raw(record, "U158_90", group1155Bytes(want.patrol), label)
	c.count(record, "U158_90", len(want.patrol), label)
	c.raw(record, "U50", binary.LittleEndian.AppendUint32(nil, want.state), label)
}

func groups1155DocumentDifferences(want groups1155Source, join map[uint16]uint16, doc *sav.DocumentData) []string {
	var c group1155Comparison
	if doc == nil {
		return []string{"initial complete Group Document absent"}
	}
	used := map[uint16]uint16{}
	for archive, local := range join {
		if archive == 0 || local == 0 || int(local) > len(doc.Objects) {
			c.fail("invalid raw archive %d -> Document %d", archive, local)
		}
		if prior := used[local]; prior != 0 && prior != archive {
			c.fail("distinct raw identities %d/%d collapse into Document %d", prior, archive, local)
		}
		used[local] = archive
	}
	get := func(index uint16, class string) *sav.DocumentRecordData {
		local := join[index]
		if local == 0 || int(local) > len(doc.Objects) || doc.Objects[local-1].Class != class {
			c.fail("archive %d lacks exact %s Document binding", index, class)
			return nil
		}
		return &doc.Objects[local-1]
	}
	var roots []uint16
	for _, index := range want.players.roots {
		roots = append(roots, join[index])
	}
	if !slices.Equal(doc.Players, roots) {
		c.fail("Group Player roots/aliases differ")
	}
	groupCount := 0
	for _, record := range doc.Objects {
		groupCount += len(record.Groups)
	}
	if groupCount != len(want.groups) {
		c.fail("Document Group population %d != raw %d", groupCount, len(want.groups))
	}
	for index, p := range want.players.players {
		if record := get(index, "Player"); record != nil {
			if uint64(len(record.Groups)) != uint64(p.groupCount) {
				c.fail("Player archive %d Group population differs", index)
			}
			c.count(*record, "Groups", int(p.groupCount), fmt.Sprintf("Player archive %d", index))
		}
	}
	for _, g := range want.groups {
		player := get(g.player, "Player")
		if player == nil || g.inline >= len(player.Groups) {
			c.fail("raw Group Player %d inline %d omitted", g.player, g.inline)
			continue
		}
		members := make([]uint16, len(g.members))
		for i, archive := range g.members {
			members[i] = join[archive]
			if archive != 0 && members[i] == 0 {
				c.fail("raw Group member archive %d lacks origin", archive)
			}
		}
		c.group(player.Groups[g.inline], g, members, fmt.Sprintf("raw Group Player %d inline %d", g.player, g.inline))
	}
	for index, a := range want.actors {
		if record := get(index, a.class); record != nil {
			if got, exists := player1154Values(*record)["Identity"]; !exists || got != a.identity {
				c.fail("Group actor archive %d identity key differs", index)
			}
			c.order(*record, a, fmt.Sprintf("raw Group actor %d", index))
		}
	}
	return c
}

type groups1155Population struct {
	bound, unbound, nulls, orders int
	unavailable, issues           []string
}

func groups1155LiveDifferences(want groups1155Source, join map[uint16]uint16, state *SnapshotSAVDocument, world *sim.World) ([]string, groups1155Population) {
	var c group1155Comparison
	var p groups1155Population
	if state == nil || state.GroupBindings == nil || world == nil {
		return []string{"Group live state/bindings absent"}, p
	}
	b := state.GroupBindings
	if b.Unavailable != "" {
		p.unavailable = append(p.unavailable, b.Unavailable)
	}
	groups, orders, present := world.SavedGroups()
	players, playersPresent := world.SavedGroupPlayers()
	if !present || !playersPresent || !b.PlayersPresent || len(groups) != len(want.groups) || len(b.Groups) != len(want.groups) || len(players) != len(want.players.players) || len(b.Players) != len(players) {
		c.fail("raw/live Group or Player identity population differs")
	}
	// The import allocates distinct Player IDs in first-root encounter order.
	// That explicit namespace gives the comparator an independent anchor even
	// if a wrong same-Slot container and its DTO binding agree with one another.
	playerIDs := map[uint16]uint32{}
	for _, root := range want.players.roots {
		if root != 0 && playerIDs[root] == 0 {
			playerIDs[root] = uint32(len(playerIDs) + 1)
		}
	}
	playerObjects := map[uint32]uint16{}
	for _, row := range b.Players {
		if _, duplicate := playerObjects[row.ID]; row.ID == 0 || duplicate {
			c.fail("native Player binding identity %d repeated/zero", row.ID)
		}
		playerObjects[row.ID] = row.ObjectIndex
	}
	for archive, id := range playerIDs {
		if playerObjects[id] == 0 || playerObjects[id] != join[archive] {
			c.fail("raw Player %d lost exact native container %d", archive, id)
		}
		found := 0
		for _, v := range players {
			if v.ID == id {
				found++
				if v.Slot != want.players.players[archive].values["Slot"] {
					c.fail("native Player %d Slot differs", id)
				}
			}
		}
		if found != 1 {
			c.fail("native Player %d presence %d != 1", id, found)
		}
	}
	entities := map[sim.EntityID]sim.Entity{}
	actorIDs := map[uint16]sim.EntityID{}
	bindActor := func(archive uint16, id sim.EntityID) {
		if prior, duplicate := actorIDs[archive]; duplicate && prior != id {
			c.fail("raw actor %d binds distinct native entities", archive)
		}
		actorIDs[archive] = id
	}
	for _, e := range world.Entities() {
		entities[e.ID] = e
		if e.SourceBinding.Class != 0 {
			bindActor(e.SourceBinding.ArchiveIndex, e.ID)
		}
	}
	for _, d := range world.OriginalDeadActors() {
		if _, exists := entities[d.ID]; exists {
			bindActor(d.Source.ArchiveIndex, d.ID)
		}
	}
	seenNative := map[sim.EntityID]uint16{}
	byEntity := map[sim.EntityID]SnapshotSAVGroupMemberBinding{}
	byHandle := map[uint16]SnapshotSAVGroupMemberBinding{}
	seenObject := map[uint16]bool{}
	for _, row := range b.Members {
		if row.ObjectIndex == 0 || seenObject[row.ObjectIndex] {
			c.fail("Group member Document identity %d repeated/zero", row.ObjectIndex)
		}
		seenObject[row.ObjectIndex] = true
		if row.Bound {
			if _, repeated := byEntity[row.EntityID]; repeated {
				c.fail("Group native member identity %d repeated", row.EntityID)
			}
			byEntity[row.EntityID] = row
		} else {
			if _, repeated := byHandle[row.UnresolvedHandle]; repeated || row.UnresolvedHandle == 0 {
				c.fail("Group unresolved member handle repeated/zero")
			}
			byHandle[row.UnresolvedHandle] = row
		}
	}
	if len(b.Members) != len(want.actors) {
		c.fail("Group exact member binding population %d != raw identities %d", len(b.Members), len(want.actors))
	}
	byGroup := map[uint32]sim.SavedGroup{}
	bindings := map[uint32]SnapshotSAVGroupBinding{}
	for _, g := range groups {
		if _, repeated := byGroup[g.ID]; repeated || g.ID == 0 {
			c.fail("native Group ID %d repeated/zero", g.ID)
		}
		byGroup[g.ID] = g
	}
	for _, row := range b.Groups {
		if _, repeated := bindings[row.ID]; repeated {
			c.fail("Group binding ID %d repeated", row.ID)
		}
		bindings[row.ID] = row
	}
	for i, w := range want.groups {
		id := uint32(i + 1)
		g, exists := byGroup[id]
		row, bound := bindings[id]
		container := playerIDs[w.player]
		if !exists || !bound || row.PlayerObject != join[w.player] || row.InlineIndex != uint32(w.inline) || row.ContainerID != container || g.ContainerID != container || row.Authored || g.Authored {
			c.fail("raw Group %d lost exact Player/inline/ContainerID binding", id)
		}
		if g.Selector != w.selector || !bytes.Equal(g.AI[:], w.ai[:76]) || !slices.Equal(g.Words, w.words) || !slices.Equal(g.Path, w.path) {
			c.fail("raw Group %d live selector/AI76/Words/Path differs", id)
		}
		for _, ref := range []struct {
			name      string
			got, want sim.SavedGroupReference
			binding   SnapshotSAVGroupReferenceBinding
		}{{"G40", g.Reference, w.reference, row.Reference}, {"G44", g.Owner, w.owner, row.Owner}} {
			if ref.got != ref.want || ref.binding != (SnapshotSAVGroupReferenceBinding{ObjectIndex: join[ref.want.Archive], Key: ref.want.Key, Class: ref.want.Class, Owner: ref.want.Owner}) {
				c.fail("raw Group %d independent %s resolution/binding differs", id, ref.name)
			}
		}
		if len(g.Members) != len(w.current) {
			c.fail("raw Group %d normalized membership count %d != %d", id, len(g.Members), len(w.current))
		}
		for j, archive := range w.current {
			if archive == 0 {
				p.nulls++
				if j >= len(g.Members) || g.Members[j] != (sim.SavedGroupMember{}) {
					c.fail("raw Group %d null slot %d lost", id, j)
				}
				continue
			}
			entity, live := actorIDs[archive]
			expect := sim.SavedGroupMember{Archive: archive, Entity: entity, Bound: live}
			if j >= len(g.Members) || g.Members[j] != expect {
				c.fail("raw Group %d normalized member %d archive %d differs", id, j, archive)
			}
			binding := byHandle[archive]
			if live {
				p.bound++
				binding = byEntity[entity]
				if prior, duplicate := seenNative[entity]; duplicate && prior != archive {
					c.fail("distinct raw actors %d/%d collapse into native entity %d", prior, archive, entity)
				}
				seenNative[entity] = archive
				if entities[entity].Owner != want.players.players[w.player].values["Slot"] {
					c.fail("raw actor %d Player suffix owner differs", archive)
				}
			} else {
				p.unbound++
				p.unavailable = append(p.unavailable, fmt.Sprintf("Group %d actor archive %d has no live entity; raw Document and unresolved member retained", id, archive))
			}
			if binding.ObjectIndex != join[archive] || binding.Bound != live || live && binding.EntityID != entity || !live && binding.UnresolvedHandle != archive {
				c.fail("raw actor %d Group member binding differs", archive)
			}
		}
	}
	byOrder := map[sim.EntityID]sim.SavedActorOrder{}
	for _, o := range orders {
		if _, repeated := byOrder[o.Entity]; repeated {
			c.fail("native order actor %d repeated", o.Entity)
		}
		byOrder[o.Entity] = o
	}
	for archive, w := range want.actors {
		entity, exists := actorIDs[archive]
		if !exists {
			continue // Named unmaterialized member above, not a reduced raw count.
		}
		p.orders++
		o, present := byOrder[entity]
		if !present || o.Authored || o.State != w.state || !bytes.Equal(o.Raw[:], w.order[:144]) || !slices.Equal(o.Patrol, w.patrol) {
			c.fail("raw Group actor %d live order144/state/active Patrol differs", archive)
		}
	}
	if len(orders) != p.orders {
		c.fail("live Group orders %d != independently bound %d", len(orders), p.orders)
	}
	p.issues = world.SavedGroupIssues()
	return c, p
}
