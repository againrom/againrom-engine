package ui

import "slices"

// spellAim is the kind of target a spell asks for.
type spellAim uint8

const (
	// aimTarget is the zero value, so a custom entry that sets no flag stays a
	// unit-target spell.
	aimTarget spellAim = iota
	aimArea
	aimSelf
)

// spellAim reads the kind off the book entry; a spell the book lacks is a
// target spell.
func (v *Viewer) spellAim(id uint32) spellAim {
	for _, entry := range v.spellbook {
		if entry.ID != id {
			continue
		}
		if entry.SelfOnly {
			return aimSelf
		}
		if entry.PointTarget {
			return aimArea
		}
		return aimTarget
	}
	return aimTarget
}

// castCursorAt is the cursor an armed spell answers over the game area, and
// false where the ordinary cursor stands. An area spell answers everywhere,
// seen or not; a target spell over a unit the party sees; a self spell over a
// selected unit that holds it. The click is made under the answer, so
// elsewhere it is the ordinary click.
func (v *Viewer) castCursorAt(x, y int) (string, bool) {
	if v.itemCast != nil {
		return "cast", true
	}
	if v.selectedSpell == 0 {
		return "", false
	}
	switch v.spellAim(v.selectedSpell) {
	case aimArea:
		return "cast", true
	case aimSelf:
		if id, hit := v.castVictimAt(x, y); hit && v.spellCasterHolds(id) {
			return "cast", true
		}
	default:
		if _, hit := v.castVictimAt(x, y); hit {
			return "cast", true
		}
	}
	return "", false
}

// castVictimAt is the unit decide would name for a cast click, less a unit the
// party may not see.
func (v *Viewer) castVictimAt(x, y int) (uint32, bool) {
	id, hit := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), v.selectedSpell == controlSpiritSpellID)
	if !hit {
		return 0, false
	}
	for _, e := range v.entities {
		if e.ID == id {
			return id, v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y)
		}
	}
	return 0, false
}

func (v *Viewer) spellCasterHolds(id uint32) bool {
	casters := bookCasters(presentSelected(v.sel, v.entities), v.selectedSpell)
	return slices.ContainsFunc(casters, func(e MapEntity) bool { return e.ID == id })
}
