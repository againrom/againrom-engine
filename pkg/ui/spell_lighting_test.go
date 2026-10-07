package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestSpellTerrainLightingWritesOneCellsFourSharedVertices(t *testing.T) {
	v := overlayViewer(t, 8, 8, 8*terrain.CellSize, 8*terrain.CellSize)
	cell := image.Pt(3, 3)
	ordinary := [4]float32{1, 1, 1, 1}
	v.SetSpellLighting([]SpellLightCell{{Cell: cell, Terrain: 3, Sprite: 2}})

	tests := []struct {
		name string
		cell image.Point
		want [4]float32
	}{
		{"covered", image.Pt(3, 3), [4]float32{3, 3, 3, 3}},
		{"left shares right edge", image.Pt(2, 3), [4]float32{1, 3, 1, 3}},
		{"right shares left edge", image.Pt(4, 3), [4]float32{3, 1, 3, 1}},
		{"above shares bottom edge", image.Pt(3, 2), [4]float32{1, 1, 3, 3}},
		{"below shares top edge", image.Pt(3, 4), [4]float32{3, 3, 1, 1}},
		{"diagonal shares one vertex", image.Pt(2, 2), [4]float32{1, 1, 1, 3}},
		{"distant vertices stay ordinary", image.Pt(5, 5), ordinary},
	}
	for _, tc := range tests {
		if got := v.cornerScales(tc.cell.X, tc.cell.Y); got != tc.want {
			t.Errorf("%s tile %v scales = %v, want %v", tc.name, tc.cell, got, tc.want)
		}
	}

	v.SetSpellLighting(nil)
	if got := v.cornerScales(cell.X, cell.Y); got != ordinary {
		t.Fatalf("expired Light left scales %v, want restored %v", got, ordinary)
	}
}

func spellLightPopulationViewer(t *testing.T) (*Viewer, image.Point, map[string]*terrain.StaticFrame) {
	t.Helper()
	cell := image.Pt(1, 1)
	g := grid(4, 4)
	g.Overlay = make([]uint8, 16)
	g.Overlay[cell.Y*g.Width+cell.X] = 1
	g.Structures = []terrain.StructureRecord{
		{X: uint32(cell.X << 8), Y: uint32(cell.Y << 8), Key: 1},
		{X: uint32(cell.X << 8), Y: uint32(cell.Y << 8), Key: 2},
	}

	frames := map[string]*terrain.StaticFrame{
		"actor":     spriteLightFrame(7, 11),
		"object":    spriteLightFrame(8, 8),
		"sack":      spriteLightFrame(9, 9),
		"structure": spriteLightFrame(terrain.CellSize, terrain.CellSize),
		"flat":      spriteLightFrame(terrain.CellSize, terrain.CellSize),
	}
	statics := new(terrain.StaticSet)
	statics.Classes[1] = &terrain.StaticClass{
		Width: terrain.CellSize, Height: terrain.CellSize,
		CenterX: terrain.CellSize / 2, CenterY: terrain.CellSize / 2,
		Frame: frames["object"],
	}
	structures := new(terrain.StructureSet)
	structures.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{frames["structure"]},
	}
	structures.Classes[2] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1, Flat: true,
		Frames: []*terrain.StaticFrame{frames["flat"]},
	}
	v, err := NewViewerWithStatics("spell-light-population", g, &terrain.Tileset{},
		statics, true, false, terrain.AnimGateTiles, structures, true)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	layoutViewport(v, 4*terrain.CellSize, 4*terrain.CellSize)
	actor := &terrain.UnitClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60,
		Frames: []*terrain.StaticFrame{frames["actor"]}}
	v.SetEntities([]MapEntity{{Cell: cell, Art: actor, Frame: frames["actor"]}})
	v.SetSackFrames([]*terrain.StaticFrame{frames["sack"]})
	v.SetSacks([]MapSack{{Cell: cell, FrameIndex: 0}})
	return v, cell, frames
}

// TestSpellLightingAffectsOnlyTerrainAndActors is MAGIC-LIGHTDRAW-055 and
// MAGIC-UNITLIGHT-057 at one mixed cell. The former names the terrain plane;
// the latter names the unit-body input. Static objects, sacks and both structure
// passes share the cell deliberately but are outside both decoded populations.
func TestSpellLightingAffectsOnlyTerrainAndActors(t *testing.T) {
	v, cell, frames := spellLightPopulationViewer(t)
	tests := []struct {
		name        string
		terrain     float32
		sprite      float32
		actorFactor float32
	}{
		{name: "Light", terrain: 3, sprite: 2, actorFactor: 80.0 / 65.0},
		{name: "Darkness", terrain: 0.5, sprite: 0.5, actorFactor: 20.0 / 65.0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v.SetSpellLighting([]SpellLightCell{{Cell: cell, Terrain: tc.terrain, Sprite: tc.sprite}})
			if got, want := v.cornerScales(cell.X, cell.Y),
				([4]float32{tc.terrain, tc.terrain, tc.terrain, tc.terrain}); got != want {
				t.Fatalf("terrain scales = %v, want %v", got, want)
			}

			brightness := make(map[*terrain.StaticFrame]float32)
			for _, sprite := range v.planeSprites() {
				if sprite.Effect == nil {
					brightness[sprite.Frame] = sprite.Brightness
				}
			}
			if got := brightness[frames["actor"]]; got != tc.actorFactor {
				t.Errorf("actor brightness factor = %v, want %v", got, tc.actorFactor)
			}
			for _, kind := range []string{"object", "sack", "structure"} {
				got, ok := brightness[frames[kind]]
				if !ok {
					t.Errorf("%s was not submitted; the normal-brightness assertion has no subject", kind)
					continue
				}
				if got != 0 {
					t.Errorf("%s brightness factor = %v, want normal draw options (0)", kind, got)
				}
			}

			var flat staticRecorder
			v.drawStructuresFlat(&flat)
			if len(flat.red) != 1 || flat.red[0] != 1 {
				t.Errorf("flat structure red scales = %v, want one normal scale 1", flat.red)
			}
		})
	}
}

// TestSpriteOnlySpellLightingReachesItsActorAndNotTerrain is the seam Wall of
// Fire needs: MAGIC-UNITLIGHT-057 supplies an actor-body level without a
// MAGIC-LIGHTDRAW-055 terrain write. A zero Terrain value must therefore mean
// "leave every vertex alone", not "reject the whole input".
func TestSpriteOnlySpellLightingReachesItsActorAndNotTerrain(t *testing.T) {
	v, cell, frames := spellLightPopulationViewer(t)
	outsideCell := image.Pt(2, 2)
	outsideFrame := spriteLightFrame(6, 10)
	outsideArt := &terrain.UnitClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60,
		Frames: []*terrain.StaticFrame{outsideFrame}}
	v.SetEntities(append(v.entities, MapEntity{Cell: outsideCell, Art: outsideArt, Frame: outsideFrame}))

	before := v.cornerScales(cell.X, cell.Y)
	v.SetSpellLighting([]SpellLightCell{{Cell: cell, Sprite: 2}})
	if got := v.cornerScales(cell.X, cell.Y); got != before {
		t.Fatalf("sprite-only input changed terrain from %v to %v", before, got)
	}

	brightness := make(map[*terrain.StaticFrame]float32)
	for _, sprite := range v.planeSprites() {
		if sprite.Effect == nil {
			brightness[sprite.Frame] = sprite.Brightness
		}
	}
	if got := brightness[frames["actor"]]; got != 80.0/65.0 {
		t.Errorf("actor on the lit cell has factor %v, want %v", got, 80.0/65.0)
	}
	if got := brightness[outsideFrame]; got != 0 {
		t.Errorf("actor outside the lit cell has factor %v, want normal draw options (0)", got)
	}
}
