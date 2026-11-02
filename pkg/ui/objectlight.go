package ui

import (
	"image"

	"againrom/pkg/render/terrain"
)

// LightStamp is one vertex write into the light-stamp grid, lower brighter
// (MAGIC-269..271, TERR-LIGHT-286). Point marks a stamp Dynamic lighting off
// skips: the point helper's (MAGIC-273) and a structure's (TERR-LIGHT-285);
// a path stamp reads no option.
type LightStamp struct {
	Vertex image.Point
	Level  uint8
	Point  bool
}

// lightStampFloor is what the unit merge subtracts per corner (MAGIC-273).
const lightStampFloor = 32

// LightKind is where a light source row writes.
type LightKind uint8

const (
	// LightCell is a cell-bit row (MAGIC-UNITLIGHT-057): Sprite is the
	// cell's unit level, Terrain goes to the cell's four vertices.
	LightCell LightKind = iota
	// LightVertex is a light-stamp row: Level at the vertex.
	LightVertex
)

// LightMerge is how a row meets an earlier row's value at the same place.
type LightMerge uint8

const (
	// LightMergeMax keeps the larger value, so an intersecting Light and
	// Darkness cell does not depend on record order.
	LightMergeMax LightMerge = iota
	// LightMergeLater lets the later row overwrite (MAGIC-270).
	LightMergeLater
)

// LightSource is one row of a frame's light source list. A LightCell row
// carries Terrain and Sprite, absolute scales on their decoded ladders; a
// LightVertex row carries Level and Point as a LightStamp does. Merge is the
// row's rule at its place, a cell's unit level or a vertex's stamp; a cell
// row's Terrain overwrites at the vertices. A producer resolves a light's
// pulse and phase into the row's level.
type LightSource struct {
	Kind            LightKind
	At              image.Point
	Terrain, Sprite float32
	Level           uint8
	Point           bool
	Merge           LightMerge
}

// LightSources is a frame's light source list: the cell-bit rows, each by
// the maximum, then the stamps in order, each overwriting.
func LightSources(cells []SpellLightCell, stamps []LightStamp) []LightSource {
	out := make([]LightSource, 0, len(cells)+len(stamps))
	for _, c := range cells {
		out = append(out, LightSource{Kind: LightCell, At: c.Cell, Terrain: c.Terrain, Sprite: c.Sprite, Merge: LightMergeMax})
	}
	for _, s := range stamps {
		out = append(out, LightSource{Kind: LightVertex, At: s.Vertex, Level: s.Level, Point: s.Point, Merge: LightMergeLater})
	}
	return out
}

// lightPlane is the frame's merged light: the cell rows' unit levels and
// vertex terrain scales, every stamp, and the stamps Dynamic lighting off
// keeps. Presentation-only, neither hashed nor persisted.
type lightPlane struct {
	cellLevels   map[image.Point]float32
	cellTerrain  map[image.Point]float32
	vertexStamps map[image.Point]uint8
	pathStamps   map[image.Point]uint8
}

// lightCornerOffsets is a cell's four vertices: TL, TR, BL, BR.
var lightCornerOffsets = [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}}

// SetLightSources replaces the frame's light plane with the merge of rows,
// so a push omitting a source restores ordinary light.
func (v *Viewer) SetLightSources(rows []LightSource) {
	v.frameLight = buildLightPlane(rows)
}

// buildLightPlane merges rows in order, each by its own rule.
func buildLightPlane(rows []LightSource) lightPlane {
	var p lightPlane
	for _, r := range rows {
		switch r.Kind {
		case LightCell:
			if r.Sprite > 0 {
				p.cellLevels = mergeLightValue(p.cellLevels, r.At, r.Sprite, r.Merge)
			}
			if r.Terrain > 0 {
				for _, c := range lightCornerOffsets {
					p.cellTerrain = mergeLightValue(p.cellTerrain, r.At.Add(c), r.Terrain, LightMergeLater)
				}
			}
		case LightVertex:
			p.vertexStamps = mergeLightValue(p.vertexStamps, r.At, r.Level, r.Merge)
			if !r.Point {
				p.pathStamps = mergeLightValue(p.pathStamps, r.At, r.Level, r.Merge)
			}
		}
	}
	return p
}

// mergeLightValue writes value at at under rule.
func mergeLightValue[V float32 | uint8](m map[image.Point]V, at image.Point, value V, rule LightMerge) map[image.Point]V {
	if m == nil {
		m = make(map[image.Point]V)
	}
	if earlier, ok := m[at]; ok && rule == LightMergeMax && earlier >= value {
		return m
	}
	m[at] = value
	return m
}

// lightRead is what mergeLight answers.
type lightRead uint8

const (
	readLightCorners lightRead = iota
	readLightSprite
)

// mergeLight reads the frame's light plane at cell. readLightCorners gives
// the four corner scales (TL, TR, BL, BR) over the cell's shading, only with
// Dynamic lighting on: a cell row's terrain scale replaces the shading, which
// lets Darkness darken; a stamp then takes min(stamp, level), the larger
// scale (MAGIC-273, DIV-2662), outside the black margin. The bit-set branch
// is absent (DIV-2660). readLightSprite gives the unit sprite factor: a
// stamped corner's merged level, min(sum>>4, ambient>>2) over
// max(stamp-32,0), replaces the cell row's level as the stamp sweep
// overwrites the cell-bit stage. That merge reads no Lighting option, so a
// bolt lights units with Dynamic lighting off, and point stamps then drop.
func (v *Viewer) mergeLight(read lightRead, cell image.Point) (corners [4]float32, sprite float32) {
	if read == readLightCorners {
		sc := v.cornerShading(cell.X, cell.Y)
		if v.graphics.DisableLighting {
			return sc, 0
		}
		stampLit := v.Lit() && !v.borderDimmed(cell.X, cell.Y)
		for i, c := range lightCornerOffsets {
			vertex := cell.Add(c)
			if brightness, ok := v.frameLight.cellTerrain[vertex]; ok {
				sc[i] = brightness
			}
			if stamp, ok := v.frameLight.vertexStamps[vertex]; ok && stampLit {
				if scale := terrain.ShadeScale(int(stamp)); scale > sc[i] {
					sc[i] = scale
				}
			}
		}
		return sc, 0
	}
	// SpriteRow's decoded gain equals this terrain-ladder expression.
	normal := terrain.ShadeScale(4*v.spriteRow() + 32)
	stamps := v.frameLight.vertexStamps
	if v.graphics.DisableLighting {
		stamps = v.frameLight.pathStamps
	}
	stamped, row := false, 0
	if len(stamps) != 0 && !v.unshaded {
		ambient, sum := int(v.sun.Ambient), 0
		for _, c := range lightCornerOffsets {
			s, ok := stamps[cell.Add(c)]
			if !ok {
				sum += ambient
				continue
			}
			stamped = true
			sum += max(int(s)-lightStampFloor, 0)
		}
		row = min(max(min(sum>>4, ambient>>2), 0), terrain.SpriteRowCount-1)
	}
	if v.graphics.DisableLighting {
		if stamped {
			return corners, terrain.ShadeScale(4*row+32) / normal
		}
		return corners, 1
	}
	level := v.frameLight.cellLevels[cell]
	if stamped {
		level = terrain.ShadeScale(4*row + 32)
	}
	if level > 0 {
		return corners, level / normal
	}
	return corners, 1
}

// StructureClass is the decoded class of a structures.reg kind, nil when the
// viewer holds no structure set or no class at that kind.
func (v *Viewer) StructureClass(kind uint16) *terrain.StructureClass {
	if v.structureSet == nil {
		return nil
	}
	return v.structureSet.Classes[kind&0xff]
}
