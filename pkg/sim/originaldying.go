package sim

import "fmt"

type OriginalDyingActor struct {
	ID    EntityID
	HP    int16
	Timer int8
}

// A restored dying owner-graph actor still carries its profile and equipment.
// Late corpses, unbound ALM bodies and ordinary living imports do not acquire
// this admission by health alone.
func originalDyingEntity(e Entity) bool {
	return e.SourceBinding.Class != 0 && e.SourceBinding.RuntimeID != 0 && e.HP <= 0 && e.MaxHP > 0 && e.Decay == DecayFallen
}

// ImportOriginalDyingActors publishes the source tuple after a detached actor
// construction. Action cleanup does not run clearFelled's death transitions:
// no second defence halving, fresh dwell, Group detach, rewards or loot drop.
// Timer is carried in the existing native dwell slot; later progression uses
// native dying rules, not a claim of complete ROM1 callback equivalence.
func (w *World) ImportOriginalDyingActors(batch []OriginalDyingActor) error {
	if w == nil {
		return fmt.Errorf("original dying actors require a world")
	}
	seen := map[EntityID]bool{}
	for _, d := range batch {
		i := indexOfEntity(w.entities, d.ID)
		if seen[d.ID] || i < 0 || d.HP > 0 || d.Timer < 0 {
			return fmt.Errorf("unsupported original dying actor %d HP %d timer %d", d.ID, d.HP, d.Timer)
		}
		seen[d.ID] = true
		e := w.entities[i]
		if !originalDyingEntity(e) || int16(e.HP) != d.HP || e.Dwell != uint16(d.Timer) {
			return fmt.Errorf("original dying actor %d has no restored construction binding", d.ID)
		}
	}
	for _, d := range batch {
		i := indexOfEntity(w.entities, d.ID)
		e := &w.entities[i]
		// A current actor's full value stays while its low word is the saved
		// word; the pool and width operands own anything beyond that word.
		if int16(e.HP) != d.HP {
			e.HP = int32(d.HP)
		}
		e.Decay, e.Dwell = DecayFallen, uint16(d.Timer)
		w.clearFelledActions(i)
		w.entities[i].clearKillCredit()
		w.clearInvalidTargetReferences(d.ID)
	}
	return nil
}
