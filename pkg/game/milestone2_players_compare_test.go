package game

import (
	"bytes"
	"fmt"
	"slices"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func player1154Values(r sav.DocumentRecordData) map[string]uint32 {
	out := map[string]uint32{}
	for _, v := range r.Values {
		out[v.Name] = v.Value
	}
	return out
}

func players1154DocumentDifferences(want players1154Source, join map[uint16]uint16, doc *sav.DocumentData) []string {
	var differences []string
	fail := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if doc == nil {
		return []string{"retained complete Document absent"}
	}
	// An origin correspondence cannot make two source identities one DTO
	// object, even when all values agree. Genuine aliases reuse one source ID.
	used := map[uint16]uint16{}
	for archive, local := range join {
		if archive == 0 || local == 0 || int(local) > len(doc.Objects) {
			fail("invalid origin %d -> %d", archive, local)
			continue
		}
		if prior := used[local]; prior != 0 && prior != archive {
			fail("distinct archive identities %d/%d collapse into Document object %d", prior, archive, local)
		}
		used[local] = archive
	}
	roots := make([]uint16, len(want.roots))
	for i, index := range want.roots {
		roots[i] = join[index]
		if index != 0 && roots[i] == 0 {
			fail("raw Player root %d archive %d has no origin", i, index)
		}
	}
	if !slices.Equal(roots, doc.Players) {
		fail("Player root slots/aliases %v != raw %v", doc.Players, roots)
	}
	players := 0
	for _, record := range doc.Objects {
		if record.Class == "Player" {
			players++
		}
	}
	if players != len(want.players) {
		fail("distinct Document Players %d != raw %d", players, len(want.players))
	}
	get := func(index uint16, class string) *sav.DocumentRecordData {
		local := join[index]
		if local == 0 || int(local) > len(doc.Objects) {
			fail("archive %d has no retained %s", index, class)
			return nil
		}
		record := &doc.Objects[local-1]
		if record.Class != class {
			fail("archive %d class %s != %s", index, record.Class, class)
			return nil
		}
		return record
	}
	raw := func(record *sav.DocumentRecordData, name string, w []byte, label string) {
		found := 0
		for _, field := range record.Raw {
			if field.Name == name {
				found++
				if !bytes.Equal(field.Bytes, w) {
					fail("%s %s differs from raw bytes", label, name)
				}
			}
		}
		if found != 1 {
			fail("%s %s presence %d != 1", label, name, found)
		}
	}
	for index, w := range want.players {
		record := get(index, "Player")
		if record == nil {
			continue
		}
		label := fmt.Sprintf("Player archive %d", index)
		values := player1154Values(*record)
		if len(record.Values) != len(w.values) {
			fail("%s scalar field count %d != %d", label, len(record.Values), len(w.values))
		}
		for name, value := range w.values {
			if got, ok := values[name]; !ok || got != value {
				fail("%s %s=%#x present=%t != raw %#x", label, name, got, ok, value)
			}
		}
		if len(record.Texts) != 1 || record.Texts[0].Name != "Name" || record.Texts[0].Value != w.name {
			fail("%s Name differs or absent", label)
		}
		raw(record, "Raw10", w.raw10, label)
		raw(record, "PRaw32", w.tail, label)
	}
	for _, w := range want.diaries {
		loc := w.location
		owner := get(loc.OwnerArchiveIndex, loc.OwnerClass)
		if owner == nil {
			continue
		}
		label := fmt.Sprintf("%s archive %d Diary", loc.OwnerClass, loc.OwnerArchiveIndex)
		var record *sav.DocumentRecordData
		if loc.OwnerClass == "Player" {
			found := 0
			for i := range owner.Inline {
				if owner.Inline[i].Name == "Diary" {
					found++
					record = &owner.Inline[i].Record
				}
			}
			if found != 1 {
				fail("%s inline presence %d != 1", label, found)
				continue
			}
		} else {
			found := 0
			for _, slot := range owner.RefSlots {
				if slot.Name == "Diary" {
					found++
					if !slices.Equal(slot.Objects, []uint16{join[loc.ArchiveIndex]}) {
						fail("%s reference/alias differs", label)
					}
				}
			}
			if found != 1 {
				fail("%s reference presence %d != 1", label, found)
			}
			if loc.Off < 0 {
				continue
			}
			record = get(loc.ArchiveIndex, "Diary")
		}
		if record == nil {
			continue
		}
		if record.Class != "Diary" {
			fail("%s class %s != Diary", label, record.Class)
		}
		values := player1154Values(*record)
		if got, ok := values["D2C"]; !ok || got != w.self || len(record.Values) != 1 {
			fail("%s D2C differs or absent", label)
		}
		counts := map[string]uint32{}
		for _, c := range record.Counts {
			counts[c.Name] = c.Count
		}
		for name, count := range map[string]uint32{"Journal": uint32(len(w.dwords) / 4), "JournalWords": uint32(len(w.words) / 2)} {
			if got, ok := counts[name]; !ok || got != count {
				fail("%s %s count=%d present=%t != raw %d", label, name, got, ok, count)
			}
		}
		if len(record.Counts) != 2 {
			fail("%s count field population differs", label)
		}
		raw(record, "Journal", w.dwords, label)
		raw(record, "JournalWords", w.words, label)
	}
	return differences
}

type players1154LivePopulation struct {
	players, purses, unavailable, playerDiaries, actorDiaries, entries int
	manaOwners                                                         int
	excluded                                                           []string
}

func players1154LiveDifferences(want players1154Source, join map[uint16]uint16, state *SnapshotSAVDocument, world *sim.World) ([]string, players1154LivePopulation) {
	var differences []string
	var population players1154LivePopulation
	fail := func(format string, args ...any) { differences = append(differences, fmt.Sprintf(format, args...)) }
	if state == nil || state.GroupBindings == nil || state.PlayerPurses == nil || world == nil {
		return []string{"live Player/Purse binding absent"}, population
	}
	players, present := world.SavedGroupPlayers()
	if !present || !state.GroupBindings.PlayersPresent || len(players) != len(want.players) || len(state.GroupBindings.Players) != len(want.players) || len(state.PlayerPurses.Players) != len(want.players) {
		fail("live Player identity/binding population differs from %d raw identities", len(want.players))
	}
	byObject := map[uint16]uint32{}
	for _, b := range state.GroupBindings.Players {
		if b.ID == 0 || byObject[b.ObjectIndex] != 0 {
			fail("Player binding identity repeated/zero")
		}
		byObject[b.ObjectIndex] = b.ID
	}
	byID := map[uint32]sim.SavedGroupPlayer{}
	for _, p := range players {
		if _, exists := byID[p.ID]; exists || p.ID == 0 {
			fail("live Player ID %d repeated/zero", p.ID)
		}
		byID[p.ID] = p
	}
	purseByID := map[uint32]SnapshotSAVPlayerPurse{}
	for _, p := range state.PlayerPurses.Players {
		if _, exists := purseByID[p.PlayerID]; exists {
			fail("purse Player ID %d repeated", p.PlayerID)
		}
		purseByID[p.PlayerID] = p
	}
	slots := map[uint32]int{}
	for _, p := range want.players {
		slots[p.values["Slot"]]++
	}
	seenID := map[uint32]bool{}
	participants := 0
	for index, w := range want.players {
		id := byObject[join[index]]
		p, ok := byID[id]
		if !ok || id == 0 || seenID[id] || p.Slot != w.values["Slot"] {
			fail("Player archive %d lost exact live identity/Slot", index)
		}
		seenID[id] = true
		population.players++
		row, ok := purseByID[id]
		if !ok || row.Slot != w.values["Slot"] {
			fail("Player archive %d purse binding differs", index)
		}
		// The native purse domain is the 50-cell owner array (sim/carry.go,
		// relations.go). It is implementation capacity, not a ROM1 assertion.
		unavailable := ""
		if w.values["Slot"] >= 50 {
			unavailable = savedPurseSlotUnavailable
		} else if slots[w.values["Slot"]] != 1 {
			unavailable = savedPurseSlotAmbiguous
		}
		if row.Unavailable != unavailable {
			fail("Player archive %d purse coverage %q != %q", index, row.Unavailable, unavailable)
		}
		if unavailable == "" {
			population.purses++
			if money := world.Purse(w.values["Slot"]); money != w.values["Money"] {
				fail("Player archive %d live Money=%#x != raw %#x", index, money, w.values["Money"])
			}
		} else {
			population.unavailable++
		}
		if w.values["Participant"] == 0 {
			participants++
			if uint32(world.Outcome()) != w.values["Outcome"] {
				fail("participant archive %d live Outcome differs from raw", index)
			}
		}
	}
	if participants != 1 {
		fail("raw participant selector count %d != 1", participants)
	}
	// PartyWalk calls Walk: the first non-null raw Player root, independently
	// of Slot and Participant. Its inline Diary is the single Player carrier.
	firstOff := -1
	for _, root := range want.roots {
		if root != 0 {
			firstOff = want.players[root].off
			break
		}
	}
	// EntityID 0 is a genuine live identity (originalActorRegistry.go numbers
	// the first retained map unit 0, before the party's own offset range), not
	// a "missing" sentinel: every membership test below on this map must use
	// the comma-ok form, never a bare ==0/!=0 read of the stored EntityID.
	entities := map[uint16]sim.EntityID{}
	actorBasis := map[uint16]sim.SourceActor{}
	for _, e := range world.Entities() {
		if e.SourceBinding.ArchiveIndex != 0 {
			if _, bound := entities[e.SourceBinding.ArchiveIndex]; bound {
				fail("repeated live actor archive binding %d", e.SourceBinding.ArchiveIndex)
			}
			entities[e.SourceBinding.ArchiveIndex] = e.ID
			actorBasis[e.SourceBinding.ArchiveIndex] = e.ActorLoad.Source
		}
	}
	byOff := map[int]*player1154Raw{}
	for _, player := range want.players {
		byOff[player.off] = player
	}
	// SAV-GRPOWNER-561: the Player suffix stamps its current group members.
	// One direct enclosing Player makes that source unambiguous; a member in
	// several Player containers is excluded by name, never resolved by Slot.
	for _, member := range want.members {
		containers := want.memberContainers[member.ArchiveIndex]
		if len(containers) != 1 {
			population.excluded = append(population.excluded, fmt.Sprintf("F58 actor archive %d: %d direct Player containers", member.ArchiveIndex, len(containers)))
			continue
		}
		if _, bound := entities[member.ArchiveIndex]; !bound {
			population.excluded = append(population.excluded, fmt.Sprintf("F58 actor archive %d: no live actor binding", member.ArchiveIndex))
			continue
		}
		owner := byOff[containers[0]]
		if owner == nil {
			fail("F58 actor archive %d: enclosing Player start absent", member.ArchiveIndex)
			continue
		}
		basis := actorBasis[member.ArchiveIndex]
		population.manaOwners++
		if basis.Class == 0 || !basis.HasOwner || basis.ManaReservePercent != owner.values["F58"] {
			fail("F58 actor archive %d from Player archive %d: live owner/mana reserve %#x != raw %#x", member.ArchiveIndex, owner.index, basis.ManaReservePercent, owner.values["F58"])
		}
	}
	wanted := map[sim.SavedDiaryOwner]diary1154Raw{}
	for _, d := range want.diaries {
		loc := d.location
		if loc.Off < 0 {
			continue
		}
		owner := sim.SavedDiaryOwner{}
		if loc.OwnerClass == "Player" {
			if loc.OwnerOff != firstOff {
				continue
			}
			owner.Player = true
			population.playerDiaries++
		} else {
			// The complete-Document importer (savdiaries1171.go
			// importSavedDiaries, run from importSavedDocument after
			// applyOriginalDiaries) carries a Diary for every live
			// archive-bound actor, independent of which Player Group (if
			// any) directly contains it — direct Player containment is the
			// F58/mana-reserve owner's own scope above, not this one.
			id, bound := entities[loc.OwnerArchiveIndex]
			if !bound {
				continue
			}
			owner.Actor = id
			population.actorDiaries++
		}
		wanted[owner] = d
		population.entries += len(d.entries())
	}
	live := world.SavedDiaries()
	if len(live) != len(wanted) {
		fail("live Diary subjects %d != independently scoped %d", len(live), len(wanted))
	}
	seen := map[sim.SavedDiaryOwner]bool{}
	for _, d := range live {
		w, ok := wanted[d.Owner]
		if !ok || seen[d.Owner] {
			fail("unexpected/repeated live Diary owner %+v", d.Owner)
			continue
		}
		seen[d.Owner] = true
		if d.Length != len(w.dwords)/4 || d.Length != len(w.words)/2 || !slices.Equal(d.Entries, w.entries()) {
			fail("live Diary owner %+v counts/entries differ from raw", d.Owner)
		}
	}
	for owner := range wanted {
		if !seen[owner] {
			fail("live Diary owner %+v omitted", owner)
		}
	}
	return differences, population
}
