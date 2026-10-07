package sim

import "slices"

// PlayerParameterAutoHealing is a local selector. Historical native saves use
// selector3 for retreat; SAV-726 corrects the original meanings to retreat1 and
// autohealing3. Keep old native command meaning (DIV1302).
const PlayerParameterAutoHealing PlayerParameter = 4

type autoHealingPolicy struct {
	Present bool
	Percent uint32
}

// AutoHealing returns the explicitly carried player policy. Absence is the
// historical native quarter-pool rule, not No or an inferred default.
func (w *World) AutoHealing(player uint32) (uint32, bool) {
	if player >= relationSlots {
		return 0, false
	}
	p := w.autoHealing[player]
	return p.Percent, p.Present
}

// ImportAutoHealing transports Player+58 without deriving or normalizing an
// actor's saved floor. The game admits only an unambiguous source Player slot.
func (w *World) ImportAutoHealing(player, percent uint32) bool {
	if player >= relationSlots {
		return false
	}
	w.autoHealing[player] = autoHealingPolicy{Present: true, Percent: percent}
	return true
}

// SetAutoHealing is the mode command's producer. TOWN-AUTOHEAL-458/SAV-726:
// 0/1/2 select100/50/0;3..100 are literal percentages; invalid values retain the
// current percentage and still reach the actor derive walk. Publish atomically
// if a supported source derive can run for every actor on this side.
func (w *World) SetAutoHealing(player uint32, value int32) bool {
	if player >= relationSlots {
		return false
	}
	if w.savedGroups != nil && w.savedGroups.PlayersPresent && w.uniqueSavedPlayerSlot(player) == 0 {
		return false
	}
	percent, present := w.AutoHealing(player)
	if value >= 0 && value <= 2 {
		percent = uint32(2-value) * 50
	} else if value >= 3 && value <= 100 {
		percent = uint32(value)
	} else if !present {
		return false
	}
	next := *w
	next.entities = slices.Clone(w.entities)
	next.autoHealing[player] = autoHealingPolicy{Present: true, Percent: percent}
	for i := range next.entities {
		e := &next.entities[i]
		if e.Owner != player || e.ActorLoad.Source.Class == 0 || !e.ActorLoad.Source.HasOwner {
			continue
		}
		e.ActorLoad.Source.ManaReservePercent = percent
		if !next.deriveSource(i) {
			return false
		}
	}
	*w = next
	return true
}

func (w *World) globalHealAllowed(ci int) bool {
	e := w.entities[ci]
	percent, present := w.AutoHealing(e.Owner)
	if !present || e.KnownSpells == 0 || e.Mana == 0 {
		return false
	}
	floor := int32(int16(uint16(int32(int16(e.MaxMana)) * int32(percent) / 100)))
	if s := e.ActorLoad.Source; s.Class != 0 {
		if !s.HasSpellbook {
			return false
		}
		floor = int32(int16(s.ManaFloor))
	}
	return e.Mana > floor
}

func (w *World) unarmedHeal(ci int) (SpellRule, bool) {
	if _, present := w.AutoHealing(w.entities[ci].Owner); present {
		if !w.globalHealAllowed(ci) {
			return SpellRule{}, false
		}
		heal, ok := w.bookSpell(w.entities[ci], 6)
		return heal, ok && heal.Restorative
	}
	heal, ok := w.knownRestorative(ci)
	return heal, ok && w.affordsAutoHeal(ci, heal)
}

// Handover joins the current player's policy. Do this at the ownership writer,
// never as a LOAD repair or a reader-side mutation of the saved actor floor.
func (w *World) inheritAutoHealing(i int) {
	e := &w.entities[i]
	percent, present := w.AutoHealing(e.Owner)
	if !present || e.ActorLoad.Source.Class == 0 || !e.ActorLoad.Source.HasOwner {
		return
	}
	e.ActorLoad.Source.ManaReservePercent = percent
	e.ActorLoad.Source.ManaFloor = uint16(int32(int16(e.MaxMana)) * int32(percent) / 100)
	e.retireCurrentProfile()
}

// stafflessHealer reports whether entity i is a participant's mage that can do
// nothing in a fight but heal: no weapon spell, no armed damaging cast, and a
// restorative spell in its book.
func (w *World) stafflessHealer(i int) bool {
	e := w.entities[i]
	if e.Owner != SelfSlot || !isMage(e) || e.WeaponSpell != 0 {
		return false
	}
	if e.AutoSpell != 0 {
		if armed, ok := w.bookSpell(e, uint32(e.AutoSpell)); ok && armed.Damaging {
			return false
		}
	}
	_, heals := w.knownRestorative(i)
	return heals
}

// attackedTogether reports whether the command at index k is one of several
// unit attack orders in the same batch that name the same victim, which is what
// one click over a selected group produces.
func attackedTogether(cmds []Command, k int) bool {
	for j, c := range cmds {
		if j != k && c.Kind == KindAttack && c.X == cmds[k].X && c.Entity != cmds[k].Entity {
			return true
		}
	}
	return false
}
