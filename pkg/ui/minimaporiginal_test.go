package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// originalLookup is a synthetic 52x14 MiniMapData stand-in whose every sample
// differs from every other.
func originalLookup() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 52, 14))
	for y := 0; y < 14; y++ {
		for x := 0; x < 52; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 16), B: 200, A: 255})
		}
	}
	return img
}

// originalCrystal is a 160x158 background whose pixels name their position.
func originalCrystal() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 160, 158))
	for y := 0; y < 158; y++ {
		for x := 0; x < 160; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 77, A: 255})
		}
	}
	return img
}

func originalShades() [16]color.RGBA {
	var out [16]color.RGBA
	for k := range out {
		out[k] = color.RGBA{R: uint8(10 * k), G: uint8(250 - 10*k), B: uint8(3*k + 1), A: 255}
	}
	return out
}

// originalTile is the tile word that samples lookup (x, y).
func originalTile(x, y int) uint16 { return uint16(x<<4 | y) }

// originalViewer is a w x h map with the engine's eight-cell margin, every
// cell's tile naming a distinct lookup sample, dressed with the crystal and
// the first game's paint.
func originalViewer(t *testing.T, w, h int) *Viewer {
	t.Helper()
	g := grid(w, h)
	g.Block = make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g.Tiles[y*w+x] = originalTile(x%52, y%14)
			if x < 8 || y < 8 || x >= w-8 || y >= h-8 {
				g.Block[y*w+x] = 2
			}
		}
	}
	v, err := NewViewer("mm-original", g, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.Layout(MissionFrameW, MissionFrameH)
	v.SetDialogFrame(&DialogFrame{Minimap: originalCrystal(), MinimapSeam: solidPic(16, 158, color.RGBA{A: 255})})
	v.SetMinimapPaint(&MinimapPaint{Colours: originalLookup(), OwnerShades: originalShades()})
	if !v.minimapOriginalActive() {
		t.Fatal("first game's paint is not active")
	}
	return v
}

// originalContent is the composed picture's pixel at content position p,
// with the geometry it was composed under.
func originalContent(t *testing.T, v *Viewer, p image.Point) color.RGBA {
	t.Helper()
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no minimap geometry")
	}
	pic, at, ok := v.minimapCompose(g)
	if !ok {
		t.Fatal("no minimap picture")
	}
	q := g.Content.Min.Add(p).Sub(at)
	return pic.RGBAAt(q.X, q.Y)
}

// symbolicContent is the same pixel with the paint removed: the minimap this
// build drew before the first game's paint, the loss control of every rule.
func symbolicContent(t *testing.T, v *Viewer, p image.Point) color.RGBA {
	t.Helper()
	paint := v.minimapOriginal.paint
	v.SetMinimapPaint(nil)
	defer v.SetMinimapPaint(paint)
	return originalContent(t, v, p)
}

func gained(c color.RGBA, factor int) color.RGBA {
	return color.RGBA{R: minimapGain(c.R, factor), G: minimapGain(c.G, factor), B: minimapGain(c.B, factor), A: 255}
}

// A 32x32 map has a 16x16 interior, filled to 128 pixels: eight per cell.
func TestOriginalMinimapTerrainIsTheLookupTimesBrightness(t *testing.T) {
	v := originalViewer(t, 32, 32)
	g, _ := v.minimapGeometry()
	if g.Origin != image.Pt(8, 8) || g.Cols != 16 || g.Rows != 16 || g.Content.Size() != image.Pt(128, 128) {
		t.Fatalf("interior geometry %+v", g)
	}
	if off := g.Content.Min.Sub(g.Box.Min); off != image.Pt(72-64, 82-64) {
		t.Fatalf("picture origin in the widget %v, want (8,18)", off)
	}
	lookup := originalLookup()
	// Cell (11,9) is content cell (3,1): pixels 24..31 x 8..15.
	want := lookup.RGBAAt(11, 9)
	if got := originalContent(t, v, image.Pt(27, 12)); got != want {
		t.Fatalf("no level plane: pixel %v, want the lookup sample %v", got, want)
	}
	if got := symbolicContent(t, v, image.Pt(27, 12)); got == want {
		t.Fatal("loss control: the symbolic minimap also drew the lookup sample")
	}

	// A level plane: the cell's four corners 40, 44, 50, 54 average 47, so the
	// factor is 49 and each channel is scaled by 49/32.
	v.levels = make([]uint8, 32*32)
	for i := range v.levels {
		v.levels[i] = 46
	}
	v.levels[9*32+11], v.levels[9*32+12], v.levels[10*32+11], v.levels[10*32+12] = 40, 44, 50, 54
	if got, want := originalContent(t, v, image.Pt(27, 12)), gained(want, 49); got != want {
		t.Fatalf("level plane: pixel %v, want %v", got, want)
	}
	if got, want := originalContent(t, v, image.Pt(4, 4)), gained(lookup.RGBAAt(8, 8), 50); got != want {
		t.Fatalf("flat level 46: pixel %v, want gain 50/32 %v", got, want)
	}
	if minimapGain(200, 80) != 255 || minimapGain(16, 0) != 0 {
		t.Fatal("gain does not saturate")
	}
}

// originalTerrainOnly is the first game's picture without marks or outline.
func originalTerrainOnly(t *testing.T, v *Viewer) *image.RGBA {
	t.Helper()
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatal("no minimap geometry")
	}
	p := v.minimapOriginalPlan(g)
	p.marks, p.view = nil, image.Rectangle{}
	return composeMinimapPlan(p, g.extent)
}

// The eight-cell margin never reaches the picture: the first content pixel is
// cell (8,8), the last cell (23,23), and no margin sample appears anywhere.
func TestOriginalMinimapExcludesTheEightCellBorder(t *testing.T) {
	v := originalViewer(t, 32, 32)
	lookup := originalLookup()
	pic := originalTerrainOnly(t, v)
	originalContent := func(_ *testing.T, _ *Viewer, p image.Point) color.RGBA { return pic.RGBAAt(p.X, p.Y) }
	if got, want := originalContent(t, v, image.Pt(0, 0)), lookup.RGBAAt(8, 8); got != want {
		t.Fatalf("first pixel %v, want cell (8,8) %v", got, want)
	}
	if got, want := originalContent(t, v, image.Pt(127, 127)), lookup.RGBAAt(23, 9); got != want {
		t.Fatalf("last pixel %v, want cell (23,23) %v", got, want)
	}
	// Loss control: a picture of the whole grid would put cell (0,0) here.
	if originalContent(t, v, image.Pt(0, 0)) == lookup.RGBAAt(0, 0) {
		t.Fatal("margin cell drawn at the picture origin")
	}
	margin := map[color.RGBA]bool{}
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			if x < 8 || y < 8 || x >= 24 || y >= 24 {
				margin[lookup.RGBAAt(x%52, y%14)] = true
			}
		}
	}
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			delete(margin, lookup.RGBAAt(x%52, y%14))
		}
	}
	for y := 0; y < 128; y += 4 {
		for x := 0; x < 128; x += 4 {
			if c := originalContent(t, v, image.Pt(x, y)); margin[c] {
				t.Fatalf("pixel (%d,%d) shows a margin-only sample %v", x, y, c)
			}
		}
	}
}

// Fog per block: the masked OR of cells (x,y), (x+1,y), (x+1,y+1) and
// (x+2,y+1). Zero shows the CrystalR pixel, exactly 0x8000 averages it with
// terrain, any bit 14 keeps terrain.
func TestOriginalMinimapFogCases(t *testing.T) {
	v := originalViewer(t, 32, 32)
	plane := make([]byte, 32*32)
	set := func(x, y int, s uint8) { plane[y*32+x] = s }
	// Cell (12,12): its whole block quad unseen.
	// Cell (12,14): explored in its own cell only.
	set(12, 14, FogExplored)
	// Cell (12,16): only (14,17), the shifted lower-right word, visible.
	set(14, 17, FogVisible)
	// Cell (12,18): only (12,19), the ordinary quad's lower-left, visible; the
	// block loop does not read it.
	set(12, 19, FogVisible)
	v.SetFog(plane, 32, 32)
	lookup := originalLookup()
	back := originalCrystal()
	g, _ := v.minimapGeometry()
	off := g.Content.Min.Sub(g.Box.Min)
	at := func(cell image.Point) image.Point { return cell.Sub(g.Origin).Mul(8).Add(image.Pt(3, 5)) }
	bg := func(p image.Point) color.RGBA { return back.RGBAAt(off.X+p.X, off.Y+p.Y) }

	unseen := at(image.Pt(12, 12))
	if got, want := originalContent(t, v, unseen), bg(unseen); got != want {
		t.Errorf("OR 0: %v, want the background pixel %v", got, want)
	}
	// A pure black background pixel is copied opaque inside the picture.
	dark := unseen.Add(image.Pt(1, 0))
	v.dialogFrame.Minimap.SetRGBA(off.X+dark.X, off.Y+dark.Y, color.RGBA{A: 255})
	if got := originalContent(t, v, dark); got != (color.RGBA{A: 255}) {
		t.Errorf("OR 0 over black background: %v, want opaque black", got)
	}
	if got := symbolicContent(t, v, dark); got.A != 0 {
		t.Errorf("loss control: symbolic unseen black pixel is opaque %v", got)
	}
	explored := at(image.Pt(12, 14))
	terr := lookup.RGBAAt(12, 0)
	b := bg(explored)
	wantAvg := color.RGBA{R: b.R>>1 + terr.R>>1, G: b.G>>1 + terr.G>>1, B: b.B>>1 + terr.B>>1, A: 255}
	if got := originalContent(t, v, explored); got != wantAvg {
		t.Errorf("OR 0x8000: %v, want the average %v", got, wantAvg)
	}
	shifted := at(image.Pt(12, 16))
	if got, want := originalContent(t, v, shifted), lookup.RGBAAt(12, 2); got != want {
		t.Errorf("OR 0xc000 from (x+2,y+1): %v, want terrain %v", got, want)
	}
	lowerLeft := at(image.Pt(12, 18))
	if got, want := originalContent(t, v, lowerLeft), bg(lowerLeft); got != want {
		t.Errorf("(x,y+1) visible alone: %v, want the background %v", got, want)
	}

	// Loss controls: the symbolic minimap halves an explored pixel and reads
	// only the cell's own fog.
	if got := symbolicContent(t, v, explored); got == wantAvg {
		t.Error("loss control: symbolic explored pixel equals the average")
	}
	if got := symbolicContent(t, v, shifted); got == lookup.RGBAAt(12, 2) {
		t.Error("loss control: symbolic shifted-quad pixel equals terrain")
	}
}

func originalStructureSet() *terrain.StructureSet {
	set := &terrain.StructureSet{}
	set.Classes[5] = &terrain.StructureClass{ID: 5, TileWidth: 3, TileHeight: 2}
	set.Classes[33] = &terrain.StructureClass{ID: 33, TileWidth: 1, TileHeight: 1, VariableSize: true}
	return set
}

// Marks: a structure spans its class's tile extents, a wooden bridge its own
// size, a unit one cell, each in its owner's shade; the fog quad gates every
// mark and a unit past corpse stage 1 has none.
func TestOriginalMinimapMarkExtentsColoursAndGates(t *testing.T) {
	v := originalViewer(t, 32, 32)
	v.structureSet = originalStructureSet()
	v.SetLocalOwner(1)
	plane := make([]byte, 32*32)
	for y := 8; y < 24; y++ {
		for x := 8; x < 16; x++ {
			plane[y*32+x] = FogVisible
		}
	}
	v.SetFog(plane, 32, 32)
	shades := originalShades()
	v.SetStructures([]MapStructure{
		{ID: 1, Cell: image.Pt(9, 9), Kind: 5, Owner: 2, Size: image.Pt(7, 7)},
		{ID: 2, Cell: image.Pt(9, 14), Kind: 33, Owner: 3, Size: image.Pt(2, 4)},
		{ID: 3, Cell: image.Pt(20, 9), Kind: 5, Owner: 4},  // under fog
		{ID: 4, Cell: image.Pt(12, 20), Kind: 5, Owner: 0}, // no owner
	})
	v.SetEntities([]MapEntity{
		{ID: 10, Cell: image.Pt(13, 9), Owner: 1, Life: LifeAlive},
		{ID: 11, Cell: image.Pt(13, 11), Owner: 18, Life: LifeAlive},
		{ID: 12, Cell: image.Pt(13, 13), Owner: 1, Life: LifeDead, CorpseStage: 2},
		{ID: 13, Cell: image.Pt(20, 20), Owner: 2, Life: LifeAlive}, // under fog
	})
	px := func(cell image.Point, dx, dy int) image.Point {
		return cell.Sub(image.Pt(8, 8)).Mul(8).Add(image.Pt(dx, dy))
	}
	for _, tc := range []struct {
		name string
		p    image.Point
		want color.RGBA
	}{
		{"structure top-left", px(image.Pt(9, 9), 0, 0), shades[2]},
		{"structure far corner", px(image.Pt(11, 10), 7, 7), shades[2]},
		{"bridge far corner", px(image.Pt(10, 17), 7, 7), shades[3]},
		{"local unit", px(image.Pt(13, 9), 4, 4), shades[1]},
		{"owner 18 takes table 2", px(image.Pt(13, 11), 0, 7), shades[2]},
	} {
		if got := originalContent(t, v, tc.p); got != tc.want {
			t.Errorf("%s at %v: %v, want %v", tc.name, tc.p, got, tc.want)
		}
		if got := symbolicContent(t, v, tc.p); got == tc.want {
			t.Errorf("loss control: symbolic %s also %v", tc.name, got)
		}
	}
	lookup := originalLookup()
	for _, tc := range []struct {
		name string
		p    image.Point
		want color.RGBA
	}{
		{"past the structure's width", px(image.Pt(12, 9), 0, 0), lookup.RGBAAt(12, 9)},
		{"past the structure's height", px(image.Pt(9, 11), 0, 0), lookup.RGBAAt(9, 11)},
		{"past the unit", px(image.Pt(14, 9), 0, 0), lookup.RGBAAt(14, 9)},
		{"corpse past stage 1", px(image.Pt(13, 13), 4, 4), lookup.RGBAAt(13, 13)},
	} {
		if got := originalContent(t, v, tc.p); got != tc.want {
			t.Errorf("%s at %v: %v, want terrain %v", tc.name, tc.p, got, tc.want)
		}
	}
	for _, tc := range []struct {
		name string
		p    image.Point
		not  color.RGBA
	}{
		{"structure under fog", px(image.Pt(20, 9), 2, 2), shades[4]},
		{"unit under fog", px(image.Pt(20, 20), 2, 2), shades[2]},
		{"ownerless structure", px(image.Pt(12, 20), 2, 2), shades[0]},
	} {
		if got := originalContent(t, v, tc.p); got == tc.not {
			t.Errorf("%s drew its mark %v", tc.name, got)
		}
	}
	local, other, err := func() (bool, bool, error) {
		a := &App{flow: &flow{viewer: v, screen: ScreenMap}}
		return a.HeadlessMinimapMarkAt(13, 9)
	}()
	if err != nil || !local || other {
		t.Fatalf("headless mark at the local unit: local=%v other=%v %v", local, other, err)
	}
}

// Below one pixel per cell a structure's extents stay unscaled pixels and a
// unit is one pixel; the half-open fill starts at the scaled cell.
func TestOriginalMinimapDownscaledMarks(t *testing.T) {
	v := originalViewer(t, 272, 272) // interior 256: half a pixel per cell
	v.structureSet = originalStructureSet()
	v.SetStructures([]MapStructure{{ID: 1, Cell: image.Pt(40, 40), Kind: 5, Owner: 2}})
	v.SetEntities([]MapEntity{{ID: 9, Cell: image.Pt(100, 100), Owner: 3, Life: LifeAlive}})
	shades := originalShades()
	g, _ := v.minimapGeometry()
	if g.Num >= g.Den {
		t.Fatalf("not downscaled: %d/%d", g.Num, g.Den)
	}
	base := image.Pt(16, 16) // (40-8)/2
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 3; dx++ {
			if got := originalContent(t, v, base.Add(image.Pt(dx, dy))); got != shades[2] {
				t.Errorf("structure pixel +(%d,%d): %v", dx, dy, got)
			}
		}
	}
	if got := originalContent(t, v, base.Add(image.Pt(3, 0))); got == shades[2] {
		t.Error("structure extent scaled past three pixels")
	}
	unit := image.Pt(46, 46) // (100-8)/2
	if got := originalContent(t, v, unit); got != shades[3] {
		t.Errorf("unit pixel %v", got)
	}
	if originalContent(t, v, unit.Add(image.Pt(1, 0))) == shades[3] || originalContent(t, v, unit.Add(image.Pt(0, 1))) == shades[3] {
		t.Error("unit mark wider than one pixel")
	}
}

// The view outline is white, follows the camera, and an edge outside the
// picture is dropped rather than moved onto the picture's border.
func TestOriginalMinimapWhiteOutlineFollowsScroll(t *testing.T) {
	v := originalViewer(t, 64, 64) // interior 48: 128/48 pixels per cell
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	g, _ := v.minimapGeometry()
	check := func(stage string) {
		view := v.minimapViewCells().Sub(g.Origin)
		x0, y0 := minimapCellPixel(view.Min.X, g.Num, g.Den), minimapCellPixel(view.Min.Y, g.Num, g.Den)
		x1 := x0 + minimapCellPixel(view.Dx(), g.Num, g.Den) - 1
		y1 := y0 + minimapCellPixel(view.Dy(), g.Num, g.Den) - 1
		for _, p := range []image.Point{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}, {(x0 + x1) / 2, y0}, {x1, (y0 + y1) / 2}} {
			if !p.In(image.Rect(0, 0, 128, 128)) {
				continue
			}
			if got := originalContent(t, v, p); got != white {
				t.Errorf("%s: outline pixel %v = %v (view %v)", stage, p, got, view)
			}
		}
		inside := image.Pt((x0+x1)/2, (y0+y1)/2)
		if got := originalContent(t, v, inside); got == white {
			t.Errorf("%s: outline is filled", stage)
		}
	}
	v.cam.CenterOn(v.cellWorldCentre(image.Pt(30, 30)))
	check("centre")
	v.cam.CenterOn(v.cellWorldCentre(image.Pt(40, 36)))
	check("scrolled")

	// A range reaching outside the picture: its left and top edges lie
	// outside and draw nothing, rather than moving onto the picture's border;
	// the right and bottom edges remain.
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	strokeMinimapView(img, image.Rect(-3, -3, 5, 5), 2, 1, white)
	for _, tc := range []struct {
		p    image.Point
		want bool
	}{{image.Pt(0, 2), false}, {image.Pt(2, 0), false}, {image.Pt(3, 3), false}, {image.Pt(9, 0), true}, {image.Pt(0, 9), true}, {image.Pt(9, 10), false}} {
		if got := img.RGBAAt(tc.p.X, tc.p.Y) == white; got != tc.want {
			t.Errorf("clipped outline pixel %v white=%v, want %v", tc.p, got, tc.want)
		}
	}
}

// Loot dots keep their amber cell mark on the first game's picture.
func TestOriginalMinimapLootDotUnchanged(t *testing.T) {
	v := originalViewer(t, 32, 32)
	v.SetSackFrames([]*terrain.StaticFrame{{Width: 4, Height: 4}})
	v.SetSacks([]MapSack{{Cell: image.Pt(10, 10), FrameIndex: 0}})
	p := image.Pt(2*8+3, 2*8+3)
	if got := originalContent(t, v, p); got != minimapLootColor {
		t.Fatalf("loot dot %v, want %v", got, minimapLootColor)
	}
	if got := symbolicContent(t, v, p); got != minimapLootColor {
		t.Fatalf("symbolic loot dot %v changed", got)
	}
}

// Draw keeps the last picture until the presentation counter has moved 11
// steps or a repaint key changes.
func TestOriginalMinimapRepaintGate(t *testing.T) {
	v := originalViewer(t, 64, 64)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(12, 12), Owner: 1, Life: LifeAlive}})
	first, _, ok := v.minimapPresent()
	if !ok {
		t.Fatal("no picture")
	}
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(15, 15), Owner: 1, Life: LifeAlive}})
	for n := 1; n < minimapRepaintSteps; n++ {
		v.anim.AdvanceOne()
		if pic, _, _ := v.minimapPresent(); pic != first {
			t.Fatalf("repainted after %d counter steps", n)
		}
	}
	g, _ := v.minimapGeometry()
	fresh, _, _ := v.minimapCompose(g)
	if string(fresh.Pix) == string(first.Pix) {
		t.Fatal("loss control: the moved unit changes no pixel")
	}
	v.anim.AdvanceOne()
	second, _, _ := v.minimapPresent()
	if string(second.Pix) != string(fresh.Pix) {
		t.Fatal("eleventh counter step did not repaint")
	}
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(18, 18), Owner: 1, Life: LifeAlive}})
	v.cam.CenterOn(v.cellWorldCentre(image.Pt(40, 40)))
	if third, _, _ := v.minimapPresent(); third == second {
		t.Fatal("a moved view did not repaint")
	}
}

// HeadlessMinimapComposed reads the state now; HeadlessMinimap keeps the gated picture.
func TestHeadlessMinimapComposedBypassesRepaintGate(t *testing.T) {
	v := originalViewer(t, 64, 64)
	v.SetLocalOwner(1)
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(12, 12), Owner: 1, Life: LifeAlive}})
	a := &App{flow: &flow{viewer: v, screen: ScreenMap}}
	first, err := a.HeadlessMinimap()
	if err != nil {
		t.Fatal(err)
	}
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(15, 15), Owner: 1, Life: LifeAlive}})
	gated, _ := a.HeadlessMinimap()
	if string(gated.Pix) != string(first.Pix) {
		t.Fatal("loss control: the gate repainted within its steps")
	}
	composed, err := a.HeadlessMinimapComposed()
	if err != nil {
		t.Fatal(err)
	}
	if string(composed.Pix) == string(first.Pix) {
		t.Fatal("the composed minimap kept the old unit cell")
	}
}
