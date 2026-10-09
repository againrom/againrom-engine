package game

import (
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// castLaunchScale is the fine units one sprite pixel of a class offset stands
// for (MAGIC-263 Medium; DIV-2656).
const castLaunchScale = 8

// castLaunch is a cast object's start from its caster's cell centre, in
// ShotScale units: A+8*(ShootOffset pair-Center), or the Selection fallback
// for an empty array and Teleport (MAGIC-261). A adds 128*(TileSize-1).
func castLaunch(c *terrain.UnitClass, facing uint8, picture int) image.Point {
	if c == nil {
		return image.Point{}
	}
	a := (max(c.TileSize, 1) - 1) * ui.ShotScale / 2
	at := castLaunchPair(facing)
	if picture != teleportPicture && len(c.ShootOffset) >= at+2 {
		return image.Pt(a+castLaunchScale*(c.ShootOffset[at]-c.CenterX),
			a+castLaunchScale*(c.ShootOffset[at+1]-c.CenterY))
	}
	return image.Pt(a, a).Add(castSelectionDelta(c))
}

// castLaunchPair is (facing>>4-8)&14; odd facings share the even pair before.
func castLaunchPair(facing uint8) int {
	return (int(facing>>4) - 8) & 14
}

// castSelectionDelta is trunc((Selection2-Selection1)/2)-Center per axis.
func castSelectionDelta(c *terrain.UnitClass) image.Point {
	return image.Pt((c.Selection.Max.X-c.Selection.Min.X)/2-c.CenterX,
		(c.Selection.Max.Y-c.Selection.Min.Y)/2-c.CenterY)
}

// casterClass is the caster's class at the observation: equipment-selected
// for a hero (MAGIC-262, DIV-2657). nil when absent.
func (mw *mapWorld) casterClass(id sim.EntityID) *terrain.UnitClass {
	if mw.units == nil {
		return nil
	}
	e, ok := mw.entity(id)
	if !ok {
		return nil
	}
	return mw.units.Classes[mw.spellClientClass(id, e.Class)]
}

func (mw *mapWorld) castLaunchFor(id sim.EntityID, facing uint8, picture int) image.Point {
	return castLaunch(mw.casterClass(id), facing, picture)
}
