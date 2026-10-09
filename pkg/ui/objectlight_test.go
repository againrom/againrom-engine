package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func lightStampViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("light-stamps", cliffGrid(), litTileset())
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	if !v.Lit() {
		t.Fatal("fixture must be lit")
	}
	return v
}

// MAGIC-273: min(stamp, terrain level) at a stamped vertex.
func TestALightStampTakesTheBrighterOfStampAndTerrain(t *testing.T) {
	v := lightStampViewer(t)
	cell := image.Pt(1, 1)
	base := baseScales(v, cliffW, cliffH, cell.X, cell.Y)
	level := int(terrain.CornerLevels(v.levels, cliffW, cliffH, cell.X, cell.Y)[0])
	if level < 10 {
		t.Fatalf("fixture vertex level %d leaves no room for a brighter stamp", level)
	}
	v.SetLightStamps([]LightStamp{{Vertex: cell, Level: uint8(level - 10)}})
	if got := v.cornerScales(cell.X, cell.Y); got[0] != terrain.ShadeScale(level-10) || got[3] != base[3] {
		t.Errorf("brighter stamp: scales %v, want corner 0 at %v and the rest %v", got, terrain.ShadeScale(level-10), base)
	}
	v.SetLightStamps([]LightStamp{{Vertex: cell, Level: uint8(level + 10)}})
	if got := v.cornerScales(cell.X, cell.Y); got != base {
		t.Errorf("darker stamp: scales %v, want the terrain's %v", got, base)
	}
	v.SetLightStamps([]LightStamp{{Vertex: cell, Level: 0}, {Vertex: cell, Level: uint8(level + 10)}})
	if got := v.cornerScales(cell.X, cell.Y); got != base {
		t.Errorf("overwritten stamp: scales %v, want %v", got, base)
	}
	v.SetLightStamps(nil)
	if got := v.cornerScales(cell.X, cell.Y); got != base {
		t.Errorf("cleared grid: scales %v, want %v", got, base)
	}
}

// MAGIC-273 at ambient 48: four corners at 40 give level 2; one at 0 gives
// min((0+3*48)>>4, 12) = 9.
func TestTheUnitMergeOfLightStamps(t *testing.T) {
	v := lightStampViewer(t)
	v.sun = terrain.Light{Ambient: 48}
	normal := terrain.ShadeScale(4*12 + 32)
	cell := image.Pt(2, 2)
	var four []LightStamp
	for _, c := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		four = append(four, LightStamp{Vertex: cell.Add(c), Level: 40})
	}
	v.SetLightStamps(four)
	if got, want := v.spellSpriteFactor(cell), terrain.ShadeScale(4*2+32)/normal; got != want {
		t.Errorf("four corners at 40: factor %v, want %v", got, want)
	}
	v.SetLightStamps([]LightStamp{{Vertex: cell, Level: 0}})
	if got, want := v.spellSpriteFactor(cell), terrain.ShadeScale(4*9+32)/normal; got != want {
		t.Errorf("one corner at 0: factor %v, want %v", got, want)
	}
	if got := v.spellSpriteFactor(image.Pt(5, 5)); got != 1 {
		t.Errorf("an unstamped cell: factor %v, want 1", got)
	}
	v.SetSpellLighting([]SpellLightCell{{Cell: cell, Terrain: 0.5, Sprite: 0.5}})
	v.SetLightStamps(four)
	if got, want := v.spellSpriteFactor(cell), terrain.ShadeScale(4*2+32)/normal; got != want {
		t.Errorf("Darkness under a bolt: factor %v, want %v", got, want)
	}
}

// MAGIC-273: with the option off a path stamp lights units, not ground.
func TestDynamicLightingOffLightsUnitsAndNotGround(t *testing.T) {
	v := lightStampViewer(t)
	v.sun = terrain.Light{Ambient: 48}
	cell := image.Pt(1, 1)
	base := baseScales(v, cliffW, cliffH, cell.X, cell.Y)
	var path, point []LightStamp
	for _, c := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		path = append(path, LightStamp{Vertex: cell.Add(c), Level: 0})
		point = append(point, LightStamp{Vertex: cell.Add(c), Level: 0, Point: true})
	}
	v.SetLightStamps(path)
	lit := v.spellSpriteFactor(cell)
	if lit <= 1 || v.cornerScales(cell.X, cell.Y) == base {
		t.Fatalf("Dynamic lighting on: factor %v and ground %v; want both lit", lit, v.cornerScales(cell.X, cell.Y))
	}

	v.SetGraphicsOptions(GraphicsOptions{DisableLighting: true})
	if got := v.cornerScales(cell.X, cell.Y); got != base {
		t.Errorf("Dynamic lighting off: ground %v, want unlit %v", got, base)
	}
	if got := v.spellSpriteFactor(cell); got != lit {
		t.Errorf("Dynamic lighting off: unit factor %v, want the bolt's %v", got, lit)
	}
	v.SetLightStamps(point)
	if got := v.spellSpriteFactor(cell); got != 1 {
		t.Errorf("Dynamic lighting off: a point stamp lit the unit, factor %v", got)
	}
	v.SetGraphicsOptions(GraphicsOptions{})
	if got := v.spellSpriteFactor(cell); got != lit {
		t.Errorf("Dynamic lighting on: point stamp unit factor %v, want %v", got, lit)
	}
}
