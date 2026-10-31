package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type originalBookCounts struct {
	Restored, Absent, Empty, Spells int
	Party, Excluded, Unmatched      int
	Corpses                         int // of Restored, corpse entities
}

// applyOriginalActorSpellbooks imports no new actors and has no heuristic
// fallback. Both source and target identities must be unique among living,
// on-map actors. Persistent party restoration remains the party path's job.
// A Corpse record gives its book to the corpse entity the dead join made of
// it; LOAD runs this step again with those records after that join.
func applyOriginalActorSpellbooks(ms *Mission, source []sav.ActorSpellbook, r *OriginalSaveResume) error {
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original actor spellbooks: mission has no map/world")
	}
	party := make(map[sim.EntityID]bool, len(ms.Start.IDs))
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	byMapID := make(map[uint16][]sim.Entity)
	for _, e := range ms.World.Entities() {
		if e.MapUnitID != 0 && e.Alive() && !e.OffMap {
			byMapID[e.MapUnitID] = append(byMapID[e.MapUnitID], e)
		}
	}
	corpses := make(map[uint32]sim.EntityID)
	for _, d := range ms.World.OriginalDeadActors() {
		if _, held := ms.World.Entity(d.ID); held && d.Current.Stage >= 2 && d.Current.Stage <= 4 {
			corpses[d.Source.Identity] = d.ID
		}
	}
	counts := r.Books
	var batch []sim.OriginalActorSpellbook
	seen := make(map[uint16]bool)
	for _, saved := range source {
		if saved.Corpse {
			if id, held := corpses[saved.Identity]; held {
				batch = append(batch, counts.book(id, saved))
				counts.Corpses++
			}
			continue
		}
		var target sim.Entity
		if ms.actorRegistry != nil {
			bound, err := registryTarget(ms, saved.Off)
			if err != nil {
				return err
			}
			if bound == nil {
				counts.Excluded++
				continue
			}
			target = *bound
		} else {
			if saved.MapUnitID != 0 && originalPartyCarriesMapUnit(ms.Party, saved.MapUnitID) {
				counts.Party++
				continue
			}
			if saved.RuntimeID == 0 || saved.Stage != 0 || sim.LifeDecaying.Holds(int64(saved.HP), 0, false) || saved.MapUnitID == 0 ||
				int(saved.Cell&255) >= int(ms.Map.Width) || int(saved.Cell>>8) >= int(ms.Map.Height) {
				counts.Excluded++
				continue
			}
			if seen[saved.MapUnitID] {
				return fmt.Errorf("original actor spellbooks: ambiguous source MapUnitID %d at saved offset %d", saved.MapUnitID, saved.Off)
			}
			seen[saved.MapUnitID] = true
			targets := byMapID[saved.MapUnitID]
			if len(targets) == 0 {
				counts.Unmatched++
				continue
			}
			if len(targets) != 1 {
				return fmt.Errorf("original actor spellbooks: ambiguous target MapUnitID %d at saved offset %d", saved.MapUnitID, saved.Off)
			}
			if party[targets[0].ID] {
				counts.Party++
				continue
			}
			target = targets[0]
		}
		batch = append(batch, counts.book(target.ID, saved))
	}
	if err := ms.World.ImportOriginalActorSpellbooks(batch); err != nil {
		return err
	}
	r.Books = counts
	return nil
}

// book is the imported value of one saved book for entity id, counted.
func (c *originalBookCounts) book(id sim.EntityID, saved sav.ActorSpellbook) sim.OriginalActorSpellbook {
	imported := sim.OriginalActorSpellbook{ID: id,
		KnownSpells: saved.KnownSpells(), Book: importedSpellbook(saved.HasSpellbook, saved.Spells)}
	imported.CreatureSpells, imported.HasCreatureSpells = importedCreatureSpells(saved)
	c.Restored++
	c.Spells += len(saved.Spells)
	if !saved.HasSpellbook {
		c.Absent++
	} else if len(saved.Spells) == 0 {
		c.Empty++
	}
	return imported
}

// importedCreatureSpells is the class slot import: an actor whose source has
// an order block takes the block's slots as they are. A block that is all zero
// leaves a creature the class built with slots holding them, and an actor with
// no order block keeps the slots the class built.
func importedCreatureSpells(saved sav.ActorSpellbook) ([sim.CreatureSpellSlots]sim.CreatureSpell, bool) {
	var slots [sim.CreatureSpellSlots]sim.CreatureSpell
	if !saved.HasCreatureSpells {
		return slots, false
	}
	for i, s := range saved.CreatureSpells {
		slots[i] = sim.CreatureSpell{ID: s.ID, Threshold: s.Threshold}
	}
	return slots, true
}
