package game

import (
	"image"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Every object in flight is one World projectile record: a cast, a staged
// burst, a Fire_Ball burst, a unit's shot at a unit or a structure, and a
// record a LOAD restored. The World's producer builds each one; this tier
// keeps the look a record has no field for and draws it.

// flightLook is the presentation a record carries no field for: a path
// picture's figure seed, its link tag and the victims a Prismatic Spray links.
// A LOAD starts every record without one, as the original's LOAD loses a
// record's word list and point array (SAV-1202).
type flightLook struct {
	seed  uint32
	tag   int
	links []image.Point
}

func (mw *mapWorld) keepFlightLook(ids []uint16, look flightLook) {
	if len(ids) == 0 {
		return
	}
	if mw.flights == nil {
		mw.flights = make(map[uint16]flightLook)
	}
	for _, id := range ids {
		mw.flights[id] = look
	}
}

func (mw *mapWorld) keepCastLook(ev sim.CastEvent, index int) {
	look := flightLook{seed: mw.visualCastSeed(ev.Caster, ev.Target, int(ev.Spell), ev.AtCell), tag: index % chainTagCount}
	for _, v := range ev.Victims {
		look.links = append(look.links, image.Pt(int(v.X), int(v.Y)))
	}
	if len(look.links) == 0 && mw.spells().DrawsPath(int32(data.CastPicture(int(ev.Spell)))) {
		look.links = []image.Point{image.Pt(int(ev.ToX), int(ev.ToY))}
	}
	mw.keepFlightLook(ev.Records, look)
}

func (mw *mapWorld) keepScriptCastLook(ev sim.ScriptCastEvent, seed uint32) {
	look := flightLook{seed: seed}
	if mw.world.SpellArm(ev.Spell) == 13 {
		look.links = []image.Point{image.Pt(int(ev.ToX), int(ev.ToY))}
	} else {
		for _, v := range ev.Victims {
			look.links = append(look.links, image.Pt(int(v.X), int(v.Y)))
		}
	}
	mw.keepFlightLook(ev.Records, look)
}

// releaseObservedCasts hands the World casts its step built no records for.
func (mw *mapWorld) releaseObservedCasts(events []sim.CastEvent) []sim.ProjectileMark {
	mw.bindFlightInputs()
	r := mw.world.ReleaseCasts(events)
	mw.keepFlightStarts(r.FlightStarts)
	return r.ProjectileMarks
}

func (mw *mapWorld) releaseObservedScriptCasts(events []sim.ScriptCastEvent) {
	mw.bindFlightInputs()
	mw.keepFlightStarts(mw.world.ReleaseScriptCasts(events).FlightStarts)
}

func (mw *mapWorld) releaseObservedPaints(paints []sim.AreaPaint) {
	mw.bindFlightInputs()
	mw.world.ReleaseAreaPaints(paints)
}

// recordPhases is the phase count a picture's installed sheet holds, the
// driver's frame clock, or 0 for a picture with no usable sheet.
func (mw *mapWorld) recordPhases(picture int) uint16 {
	if sheet := mw.projectiles.Sheet(picture); sheet != nil && sheet.Phases > 0 && sheet.Phases <= 65535 {
		return uint16(sheet.Phases)
	}
	return 0
}

// spells is the spell presentation set of the World's own game.
func (mw *mapWorld) spells() *data.SpellPresentation {
	if mw.world == nil {
		return data.FirstSpellPresentation()
	}
	return data.SpellPresentationOf(mw.world.SpellGame())
}

// recordPaths is a path record's figures this call, one per link, each as the
// path producer reads it. A record with no look draws one figure to its aim
// point, and a Prismatic Spray record none: LOAD empties its word list
// (SAV-1202).
func (mw *mapWorld) recordPaths(p sim.SavedProjectile, owner uint32) []spellBolt {
	look, ok := mw.flights[p.ID]
	if !ok {
		if mw.spells().Effect(p.Picture).Chain {
			return nil
		}
		look = flightLook{seed: uint32(p.ID) + 1}
		look.links = []image.Point{image.Pt(floorDiv(int(p.ActionX), ui.ShotScale), floorDiv(int(p.ActionY), ui.ShotScale))}
	}
	cell := image.Pt(floorDiv(int(p.X), ui.ShotScale), floorDiv(int(p.Y), ui.ShotScale))
	base := spellBolt{from: cell, picture: int(p.Picture), owner: owner, age: int(p.ActionPhase), phase: int(p.Phase),
		launch: image.Pt(int(p.X)-cell.X*ui.ShotScale-ui.ShotScale/2, int(p.Y)-cell.Y*ui.ShotScale-ui.ShotScale/2)}
	out := make([]spellBolt, 0, len(look.links))
	for k, to := range look.links {
		b := base
		b.to = to
		b.seed = look.seed + uint32(k)*2654435761
		b.tag = (look.tag + k) % chainTagCount
		if len(look.links) > 1 {
			b.tag = k % chainTagCount
		}
		out = append(out, b)
	}
	return out
}

// forgetFlights drops the looks of records the World no longer holds.
func (mw *mapWorld) forgetFlights(records map[uint16]bool) {
	for id := range mw.flights {
		if !records[id] {
			delete(mw.flights, id)
		}
	}
}
