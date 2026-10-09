package ui

import (
	"image"

	"againrom/pkg/render/terrain"
)

// LightStamp is one vertex write into the light-stamp grid, lower brighter
// (MAGIC-269..271). Point marks a point-helper stamp, which Dynamic lighting
// off skips; a path stamp reads no option (MAGIC-273).
type LightStamp struct {
	Vertex image.Point
	Level  uint8
	Point  bool
}

// lightStampFloor is what the unit merge subtracts per corner (MAGIC-273).
const lightStampFloor = 32

// SetLightStamps replaces the grid; a later stamp overwrites (MAGIC-270).
func (v *Viewer) SetLightStamps(stamps []LightStamp) {
	v.lightStamps, v.boltLightStamps = nil, nil
	for _, s := range stamps {
		if v.lightStamps == nil {
			v.lightStamps = make(map[image.Point]uint8)
		}
		v.lightStamps[s.Vertex] = s.Level
		if !s.Point {
			if v.boltLightStamps == nil {
				v.boltLightStamps = make(map[image.Point]uint8)
			}
			v.boltLightStamps[s.Vertex] = s.Level
		}
	}
}

// activeLightStamps is the grid read now: path stamps only with Dynamic
// lighting off (MAGIC-273).
func (v *Viewer) activeLightStamps() map[image.Point]uint8 {
	if v.graphics.DisableLighting {
		return v.boltLightStamps
	}
	return v.lightStamps
}

// lightStampTerrainScale is a stamped vertex's scale. Terrain takes
// min(stamp, level) only with Dynamic lighting on (MAGIC-273); the caller
// composes the scale by max (DIV-2662). The bit-set branch is absent (DIV-2660).
func (v *Viewer) lightStampTerrainScale(vertex image.Point) (float32, bool) {
	if v.graphics.DisableLighting || !v.Lit() {
		return 0, false
	}
	stamp, ok := v.lightStamps[vertex]
	if !ok {
		return 0, false
	}
	return terrain.ShadeScale(int(stamp)), true
}

// lightStampSpriteRow is the unit level of a cell with a stamped corner:
// min(sum>>4, ambient>>2) over max(stamp-32,0), ambient for an unstamped
// corner. It reads no Lighting option (MAGIC-273).
func (v *Viewer) lightStampSpriteRow(cell image.Point) (int, bool) {
	stamps := v.activeLightStamps()
	if len(stamps) == 0 || v.unshaded {
		return 0, false
	}
	ambient := int(v.sun.Ambient)
	sum, stamped := 0, false
	for _, corner := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		s, ok := stamps[cell.Add(corner)]
		if !ok {
			sum += ambient
			continue
		}
		stamped = true
		sum += max(int(s)-lightStampFloor, 0)
	}
	if !stamped {
		return 0, false
	}
	row := min(sum>>4, ambient>>2)
	return min(max(row, 0), terrain.SpriteRowCount-1), true
}
