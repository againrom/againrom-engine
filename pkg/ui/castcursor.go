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
// false where the ordinary cursor stands. Where a cast needs sight and the
// cell fails castSightAt, every cast arm answers move (MENU-CAST-153).
// Otherwise an item cast and an area spell answer everywhere; a target spell
// over a unit the party sees; a self spell over a selected unit that holds it.
// The click is made under the answer, so elsewhere it is the ordinary click.
func (v *Viewer) castCursorAt(x, y int) (string, bool) {
	if v.itemCast == nil && v.selectedSpell == 0 {
		return "", false
	}
	if !v.castSightAt(x, y) {
		return "move", true
	}
	if v.itemCast != nil {
		return "cast", true
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

// SetCastNeedsSight turns the current-sight gate of castSightAt on or off.
func (v *Viewer) SetCastNeedsSight(on bool) { v.castNeedsSight = on }

// castSightAt reports whether the ground cell under a frame point takes a
// cast. A point that names no cell is not gated.
func (v *Viewer) castSightAt(x, y int) bool {
	col, row, inside := v.groundCellAt(float64(x), float64(y))
	return !inside || v.castSightCell(col, row)
}

// castSightCell is TERR-CLICK-262's hover predicate on the client fog plane.
// The plane holds one state per lattice vertex, as the shroud reads it: cell
// (col,row) has the corners (col,row), (col+1,row), (col+1,row+1) and
// (col,row+1). FogVisible is the pair 11, FogExplored 10, FogUnseen 00. The
// plane has no raw pair 01, so the corners' OR is 11 exactly when one corner
// is in sight now. DIV-3324.
func (v *Viewer) castSightCell(col, row int) bool {
	if !v.castNeedsSight {
		return true
	}
	for _, c := range [4][2]int{{col, row}, {col + 1, row}, {col + 1, row + 1}, {col, row + 1}} {
		if v.fogAt(c[0], c[1]) == FogVisible {
			return true
		}
	}
	return false
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
