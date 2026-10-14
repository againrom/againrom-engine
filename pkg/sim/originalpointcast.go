package sim

import "slices"

// OriginalPointCast is a point cast an original SAV stores in flight: actor
// state 14 in sub-phase 5, its countdown, its destination bytes and its Spell
// (SAV-1254, MAGIC-TELEPORT-370). The countdown counts the actor ticks left
// before the application.
type OriginalPointCast struct {
	Entity    EntityID
	Spell     uint16
	X, Y      int32
	Countdown uint8
}

// ImportOriginalPointCasts admits each stored first-game Teleport in flight as
// the charging book cast an engine admission leaves: paid, aimed at its cell,
// with the stored countdown, so it relocates its caster on the tick the
// original does. A row this build cannot run is left out and not counted:
// another spell, a caster without a living body or that Spell in its book,
// a cell off the map, or a caster already holding a book cast (DIV-3328). It
// answers how many it admitted.
func (w *World) ImportOriginalPointCasts(casts []OriginalPointCast) int {
	if w.SecondGame() {
		return 0
	}
	staged := slices.Clone(w.bookCasts)
	admitted := 0
	for _, p := range casts {
		i := indexOfEntity(w.entities, p.Entity)
		if i < 0 || p.Countdown == 0 {
			continue
		}
		e := w.entities[i]
		rule, ok := w.bookSpell(e, uint32(p.Spell))
		if !ok || !e.Alive() || !knowsSpell(e, uint32(p.Spell)) || rule.arm() != teleportSpellID {
			continue
		}
		if _, inside := cellIndexIn(w.bounds, p.X, p.Y); !inside {
			continue
		}
		k := 0
		for k < len(staged) && staged[k].Caster < e.ID {
			k++
		}
		if k < len(staged) && staged[k].Caster == e.ID {
			continue
		}
		c := bookCast{Caster: e.ID, Spell: p.Spell, X: p.X, Y: p.Y, AtCell: true,
			Phase: bookCharging, Remaining: p.Countdown, Paid: true}
		if bookCastFault(c) != "" {
			continue
		}
		staged = slices.Insert(staged, k, c)
		w.entities[i].AdmittedBookSpell = p.Spell
		admitted++
	}
	w.bookCasts = staged
	return admitted
}
