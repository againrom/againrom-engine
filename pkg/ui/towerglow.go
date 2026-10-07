package ui

import (
	"image"
	"math"

	"againrom/pkg/render/terrain"
)

// Structure light masks: a pulsing lighting contribution around every
// structure whose registry class carries a light source (owner report 2,
// hotfix DIV-1313, amended in the R3 batch: "Glow вокруг источников света
// сделан неправильно, должна быть просто маска освещения (как от заклинания
// свет) вокруг объекта с пульсацией" — a light MASK, like the Light spell,
// not a drawn ring).
//
// SELECTION IS SHIPPED DATA, NOT A GUESS, AND IT IS NOT ONLY THE TOWERS.
// structures/structures.reg's own LightRadius/LightPulse keys (pkg/data's
// decode; REG-STR-040) are nonzero on five of the 66 structure classes, with
// identical values in both roots (`classdump graphics.res`):
//
//	ID 12 "Tower 1"   LightRadius 1  LightPulse 8
//	ID 13 "Tower 2"   LightRadius 1  LightPulse 8
//	ID 15 "Well 2"    LightRadius 1  LightPulse 0
//	ID 16 "Well 3"    LightRadius 1  LightPulse 4
//	ID 53 "Campfire"  LightRadius 1  LightPulse 6
//
// Every other class carries zero, INCLUDING the other tower-named ones this
// registry ships: Guard Tower 1/2, Skrakan Tower 1/2 and Mage's tower. The
// owner has SEPARATELY CONFIRMED this from the live game ("у фонтана и огня
// тоже есть подсветка, ты прав" — R3 batch, mid-task correction): the well
// and the campfire glow alongside the towers. That is owner OBSERVATION of
// the original now, not only an inference from the registry's own selection.
//
// THE RADIUS IS SHIPPED; ITS GEOMETRY AND UNIT ARE PARTLY OWNER OBSERVATION.
// Research does not establish what unit LightRadius or LightPulse are stored
// in, or how the original consumed them (Unknown; DIV-1313 states it). The
// owner's own further observation settles the SHAPE this pass reads
// LightRadius as: "выглядит так что такие
// подсветки все имеют радиус 1 от края
// объекта" — radius measured from the EDGE of the object, not its
// anchor cell or centre. A multi-cell class (both towers are 2x2) therefore
// lights its whole footprint DILATED by the radius, not a disc around one
// cell — a 2x2 footprint with radius 1 lights a 4x4 region.
//
// THE SAME SHAPE THE DECODED MECHANISM USES. The decoded fact is the
// existence and per-cell form of that stamp; the owner's words are the
// authority for reading LightRadius through it. The two are stated
// separately and the second is never offered as proof of the first.
//
// THE PEAK BRIGHTNESS IS OURS, sized equal to the Light spell's own actor
// and terrain levels (spellLightingCells' lightTerrainBrightness/
// lightSpriteBrightness, pkg/game/world.go) because the owner's own words
// for this pass are "как от заклинания свет" — the SAME mechanism at the
// SAME magnitude, composed with it by that function's own max rule so
// overlapping sources and source order cannot disagree (see
// refreshStructureLighting below). LightPulse is read as cadence in the one
// relation the shipped values themselves order — zero does not pulse
// (Well 2 holds a steady mask) and a smaller value cycles faster — using
// towerGlowElapsed/towerGlowSwing below, this file's own presentation clock
// math, unchanged from the ring this pass replaces.
//
// UNCONDITIONAL, NOT FOG-GATED. DIV-1313 originally gated the (then) ring on
// FogVisible ("only if there is line of sight"); the owner's new words are
// "освещение работает всегда" (item 2's own report: "the object itself must
// not animate out of view — LIGHTING always works"), and TERR-LIGHT-061's
// three-stage terrain sweep and MAGIC-UNITLIGHT-057's unit-level splat carry
// no sight test of their own. structurePlacements() still requires a
// structure be EXPLORED at least once to draw at all — a structure never
// revealed casts no light because it never places — and the existing shroud
// (drawShroud) already hides whatever the local player cannot presently see;
// unconditional light under that shroud is both what TERR-LIGHT-061 states
// and what the owner now asks for.
const (
	// structureLightPeakTerrain/Sprite are the absolute brightness-ladder
	// scales (SpellLightCell's own scale, overlay.go) a lit structure's mask
	// reaches at the peak of its pulse — equal to pkg/game/world.go's
	// lightTerrainBrightness/lightSpriteBrightness, the Light spell's own
	// values, per this file's own header comment above.
	structureLightPeakTerrain = float32(3)
	structureLightPeakSprite  = float32(2)
	// structureLightFloor is the fraction of the peak the mask holds at the
	// trough of its pulse — never fully dark, so "работает всегда" reads as
	// a light that breathes rather than one that blinks off. A class whose
	// shipped LightPulse is zero (Well 2) holds the MIDPOINT between floor
	// and peak, steady, for towerGlowSwing's own reason: zero pulse returns
	// swing 0 always, never -1.
	structureLightFloor = float32(0.6)
	// structureLightEdge is the fraction of the mask's own EXCESS OVER AMBIENT
	// that survives at the radius edge — 1 at the source cell falling to this
	// at distance `radius`, so the mask reads as a light with a centre rather
	// than a uniform block with a hard rim.
	//
	// OWNER REPORT, R4: "слишком большой радиус, по ощущениям сделали 2,
	// вместо 1". Asked whether the 1x1 Campfire (a 3x3 mask, unambiguously
	// radius 1) or only the 2x2 towers (a 4x4 mask) read too large, the owner
	// answered BOTH. That rules the geometry out as the cause — 3x3 around one
	// cell IS radius 1 — and leaves the brightness profile: before this, every
	// cell inside the stamp held the full peak and the mask ended in a hard
	// edge, which reads as a larger radius than a light that falls off.
	//
	// THE CURVE IS OURS. Linear in Euclidean distance is an authored choice,
	// named as one.
	structureLightEdge = float32(0.35)
)

// structureLightSource is one currently placed light-source structure: the
// distinct footprint cells its own placements name (deduplicated — a
// rectangle cell of a multi-row overhang can contribute more than one
// StructurePlacement entry, TERR-STRUCT-102's own "one entry per image row"
// shape), its class's own shipped LightRadius and LightPulse.
type structureLightSource struct {
	footprint []image.Point
	radius    int
	pulse     int
}

// structureLightSources collects every LightRadius-positive structure
// structurePlacements() gives this frame, one entry per StructureID — no fog
// gate of its own beyond what structurePlacements() already applies (this
// file's own header comment, "UNCONDITIONAL, NOT FOG-GATED"), and no
// showStructureArt gate either: that toggle is whether a structure's own
// SPRITE paints (viewer.go's own doc on the field), and the owner's words for
// this whole batch are that the light is a separate thing from the object
// ("освещение работает всегда") — the ring this replaces read the same list
// with no such check, and this keeps that.
func (v *Viewer) structureLightSources() []structureLightSource {
	var out []structureLightSource
	index := make(map[uint32]int)
	seen := make(map[uint32]map[image.Point]bool)
	for _, p := range v.structurePlacements() {
		if p.Class == nil || p.Class.LightRadius <= 0 {
			continue
		}
		i, ok := index[p.StructureID]
		if !ok {
			i = len(out)
			index[p.StructureID] = i
			out = append(out, structureLightSource{radius: p.Class.LightRadius, pulse: p.Class.LightPulse})
			seen[p.StructureID] = make(map[image.Point]bool)
		}
		if !seen[p.StructureID][p.Cell] {
			seen[p.StructureID][p.Cell] = true
			out[i].footprint = append(out[i].footprint, p.Cell)
		}
	}
	return out
}

// towerGlowElapsed is v.anim's own clock in whole ticks plus the fractional
// one in progress — so the mask's pulse does not step once per tick, it
// moves every frame the tick itself is subdivided into.
func towerGlowElapsed(count uint32, remainder, period int) float64 {
	frac := 0.0
	if period > 0 && remainder > 0 {
		frac = float64(remainder) / float64(period)
	}
	return float64(count) + frac
}

// towerGlowSwing is one class's own position on its pulse, in [-1,1]. A class
// whose shipped LightPulse is zero does not pulse at all and holds the mean:
// Well 2 ships exactly that.
func towerGlowSwing(elapsed float64, pulse int) float64 {
	if pulse <= 0 {
		return 0
	}
	cycle := elapsed / (float64(pulse) * towerGlowPulseTicksPerUnit)
	cycle -= math.Floor(cycle)
	return math.Sin(cycle * 2 * math.Pi)
}

// towerGlowPulseTicksPerUnit converts one unit of a class's own shipped
// LightPulse into v.anim ticks. The scalar's real unit is Unknown
// (DIV-1313), so this factor is ours: it preserves the shipped ORDER —
// Well 3's 4 cycles twice as fast as a tower's 8 — and puts the towers
// themselves at ten ticks, a slow readable pulse rather than a flicker.
const towerGlowPulseTicksPerUnit = 1.25

// structureLightAttenuation is the mask's own falloff: 1 at the source cell,
// structureLightEdge at exactly `radius`, linear in Euclidean distance between
// them. A zero radius is a single cell and never attenuates. The caller applies
// it to the mask's excess over ambient, not to the absolute level.
func structureLightAttenuation(dist float64, radius int) float32 {
	if radius <= 0 || dist <= 0 {
		return 1
	}
	frac := float32(dist / float64(radius))
	if frac > 1 {
		frac = 1
	}
	return 1 - (1-structureLightEdge)*frac
}

// structureLightLevel is one absolute-ladder peak scaled by a pulse's own
// swing: the mean of floor and peak at swing 0 (a steady mask, Well 2's own
// case), the floor at swing -1, and the full peak at swing +1.
func structureLightLevel(peak float32, swing float64) float32 {
	t := float32((swing + 1) / 2)
	return peak * (structureLightFloor + (1-structureLightFloor)*t)
}

// refreshStructureLighting rebuilds this frame's structure-light mask: every
// currently placed light-source structure contributes a radius-dilated stamp
// of its own footprint — one square stamp per footprint cell, exactly
// spellLightingCells' own Wall of Fire loop shape (pkg/game/world.go),
// unioned by MAX so overlapping stamps (a multi-cell footprint's own, or two
// neighbouring structures') cannot disagree by order — into this file's own
// v.structureLighting (cell-keyed, SpellLightCell.Sprite's own units) and
// v.structureTerrainLighting (vertex-keyed, SetSpellLighting's own four-
// corner expansion of each stamped cell, so two stamps meeting at a shared
// edge blend through that vertex rather than stopping short of it).
// spellTerrainScale/spellSpriteFactor (overlay.go) then compose these two
// planes with the spell-light plane by the same MAX rule.
//
// IT RUNS ONCE PER FRAME, FROM drawFrame, BEFORE EITHER TERRAIN OR THE PLANE
// SPRITES ARE DRAWN — not lazily from inside spellTerrainScale/
// spellSpriteFactor, because v.anim's own pulse must read the SAME swing at
// every vertex and every sprite one frame draws, and because
// v.structurePlacements() itself is not cheap enough to call once per
// vertex.
func (v *Viewer) refreshStructureLighting() {
	v.structureLighting = nil
	v.structureTerrainLighting = nil
	if v.graphics.DisableLighting {
		return
	}
	sources := v.structureLightSources()
	if len(sources) == 0 {
		return
	}
	elapsed := towerGlowElapsed(v.anim.Count(), v.anim.Remainder(), v.anim.Period())
	sprite := make(map[image.Point]float32)
	vertices := make(map[image.Point]float32)
	// vertexCorners is SetSpellLighting's own corner set (overlay.go): a lit
	// CELL writes its absolute terrain level to all four vertices it shares
	// with its neighbours, so two dilated stamps meeting at an edge blend
	// through the same shared vertex rather than stopping short of it.
	vertexCorners := [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
	// THE TWO BASES THE FALLOFF ATTENUATES TOWARD. Both consumers read this
	// plane as the cell's ABSOLUTE level, not as an addition to it, so the
	// falloff attenuates the mask's EXCESS OVER A BASE and never the absolute
	// level — otherwise the rim would sit below ambient and DARKEN.
	//
	// spriteAmbient is EXACT: spellSpriteFactor divides by this same
	// expression, so it is the real per-frame sprite ambient and nothing under
	// it is ever written.
	//
	// terrainAmbient is a REFERENCE, NOT A CEILING. ShadeScale(46) is the level
	// flat ground holds under the cycle-off daytime sun at any theta
	// (TERR-LIGHT-013/020/030); relief moves a vertex either way, and a swept
	// LevelGrid measurement over nine relief amplitudes and five suns reaches
	// 36 (scale 1.875) on the bright side. So a rim value built over this base
	// CAN sit under a sunlit slope's own shading. cornerScales therefore
	// composes this plane by MAX rather than by replacement
	// (structureTerrainScale, overlay.go), which is what makes "never darkens"
	// hold at the consumer instead of resting on this base being a maximum.
	// The base stays at flat daytime because raising it to the measured 1.875
	// would put the pulse trough (tPeak 1.8) under it and make the whole mask
	// blink off once per cycle.
	spriteAmbient := terrain.ShadeScale(4*v.spriteRow() + 32)
	terrainAmbient := terrain.ShadeScale(46)
	for _, src := range sources {
		swing := towerGlowSwing(elapsed, src.pulse)
		tPeak := structureLightLevel(structureLightPeakTerrain, swing)
		sPeak := structureLightLevel(structureLightPeakSprite, swing)
		// A PLANE WHOSE OWN PEAK IS ALREADY BELOW AMBIENT CONTRIBUTES NOTHING.
		// Writing it would darken, because both consumers treat this value as
		// the cell's absolute level rather than as an addition to it. This is
		// the documented day/night asymmetry, not a special case: a campfire
		// does not visibly light anything at noon, and the same mask brightens
		// plainly at night when the ambient it is compared against is lower.
		//
		// It also removes a defect this falloff work uncovered in the R3 pass
		// this file replaces: with peak 2 and pulse floor 0.6 the sprite plane
		// sat at 1.2 against a 1.625 ambient, so before this guard a unit
		// standing beside a tower was DIMMED for most of every pulse.
		spriteOn := sPeak > spriteAmbient
		terrainOn := tPeak > terrainAmbient
		if !spriteOn && !terrainOn {
			continue
		}
		for _, cell := range src.footprint {
			for dy := -src.radius; dy <= src.radius; dy++ {
				for dx := -src.radius; dx <= src.radius; dx++ {
					// EUCLIDEAN, NOT CHEBYSHEV: a diagonal at radius 1 is
					// 1.41 cells away and falls outside, so the mask is a
					// rounded shape rather than a square. That is the other
					// half of the owner's R4 report — a square stamp reads
					// wider at its corners than the radius it claims.
					dist := math.Hypot(float64(dx), float64(dy))
					if dist > float64(src.radius) {
						continue
					}
					at := cell.Add(image.Pt(dx, dy))
					if at.X < 0 || at.Y < 0 || at.X >= v.grid.Width || at.Y >= v.grid.Height {
						continue
					}
					atten := structureLightAttenuation(dist, src.radius)
					if spriteOn {
						if s := spriteAmbient + (sPeak-spriteAmbient)*atten; s > sprite[at] {
							sprite[at] = s
						}
					}
					if terrainOn {
						t := terrainAmbient + (tPeak-terrainAmbient)*atten
						for _, corner := range vertexCorners {
							vp := at.Add(corner)
							if t > vertices[vp] {
								vertices[vp] = t
							}
						}
					}
				}
			}
		}
	}
	v.structureLighting = sprite
	v.structureTerrainLighting = vertices
}
