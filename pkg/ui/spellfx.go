package ui

import (
	"image/color"

	"againrom/pkg/render/terrain"
)

// SpellSchoolColors is the authored diagnostic palette indexed by the marking
// spell's school. Unknown schools take entry 0. Installed effect sprites draw
// independently of these cell rims.
var SpellSchoolColors = [6]color.RGBA{
	{R: 0xd8, G: 0xd8, B: 0xd8, A: 0xff}, // 0 — no school stated
	{R: 0xff, G: 0x7a, B: 0x1e, A: 0xff}, // 1 — fire
	{R: 0x5a, G: 0xc8, B: 0xff, A: 0xff}, // 2 — water
	{R: 0xc0, G: 0x8a, B: 0x4a, A: 0xff}, // 3 — earth
	{R: 0xe6, G: 0xe6, B: 0x6a, A: 0xff}, // 4 — air
	{R: 0x8c, G: 0xff, B: 0x9a, A: 0xff}, // 5 — life
}

// spellEffectPasses groups diagnostic rims by school in ascending order. It
// shares the placed-unit opt-in and fog gate with the other unit diagnostics.
func (v *Viewer) spellEffectPasses() []overlayPass {
	if !v.showUnits {
		return nil
	}
	var byShool [len(SpellSchoolColors)][]MapEntity
	any := false
	for _, e := range v.entities {
		if e.SpellFX <= 0 {
			continue
		}
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		school := e.SpellFXSchool
		if school < 0 || school >= len(SpellSchoolColors) {
			school = 0
		}
		byShool[school] = append(byShool[school], e)
		any = true
	}
	if !any {
		return nil
	}
	var passes []overlayPass
	for school, ents := range byShool {
		if len(ents) == 0 {
			continue
		}
		if rects := v.entityGlyphRects(ents, terrain.SelectionMarkerRects); len(rects) > 0 {
			passes = append(passes, overlayPass{Color: SpellSchoolColors[school], Rects: rects})
		}
	}
	return passes
}
