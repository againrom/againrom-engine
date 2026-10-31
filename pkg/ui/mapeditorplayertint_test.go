package ui

import (
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

// tintFixture is a 20x20 flat map of three players: five units (owners 1, 2,
// 3, none and a player the map lacks) that share one picture, three structures
// (owners 2, none and 3, the last one flat), one static object and one sack.
type tintFixture struct {
	e                                 *MapEditor
	unit, plain, flat, object, sack   *terrain.StaticFrame
	records                           []InspectionRecord
	mk                                func(cells []image.Point) *Viewer
	unitOwners, structureOwners       []int
	unitCells, structureCells, sackAt []image.Point
}

func tintFrame(w, h int, index uint8, c color.RGBA) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: index, Opaque: true}
	}
	f.Pixels[0].Opaque = false // one hole in every picture
	f.Palette[index] = c
	return f
}

func newTintFixture(t *testing.T) *tintFixture {
	t.Helper()
	x := &tintFixture{
		unit:           tintFrame(10, 6, 5, color.RGBA{100, 150, 200, 255}),
		plain:          tintFrame(32, 32, 7, color.RGBA{200, 60, 40, 255}),
		flat:           tintFrame(32, 32, 9, color.RGBA{30, 200, 90, 255}),
		object:         tintFrame(10, 6, 1, color.RGBA{90, 90, 90, 255}),
		sack:           tintFrame(10, 6, 2, color.RGBA{220, 200, 20, 255}),
		unitOwners:     []int{1, 2, 3, 0, 9},
		unitCells:      []image.Point{{6, 1}, {8, 1}, {10, 1}, {12, 1}, {14, 1}},
		structureCells: []image.Point{{4, 4}, {8, 4}, {12, 4}},
		sackAt:         []image.Point{{2, 8}},
	}
	x.structureOwners = []int{2, 0, 3}
	x.records = []InspectionRecord{
		{Kind: "Player", Section: 5, Index: 0, Label: "1: Red"},
		{Kind: "Player", Section: 5, Index: 1, Label: "2: Blue"},
		{Kind: "Player", Section: 5, Index: 2, Label: "3: Wild"},
	}
	for i, c := range x.unitCells {
		x.records = append(x.records, InspectionRecord{Kind: "Unit", Section: 6, Index: i, Cell: c, Spatial: true, Size: image.Pt(1, 1), Player: x.unitOwners[i]})
	}
	for i, c := range x.structureCells {
		x.records = append(x.records, InspectionRecord{Kind: "Structure", Section: 4, Index: i, Cell: c, Spatial: true, Size: image.Pt(1, 1), Player: x.structureOwners[i]})
	}
	x.records = append(x.records, InspectionRecord{Kind: "Sack", Section: 8, Index: 0, Cell: x.sackAt[0], Spatial: true, Size: image.Pt(1, 1)})
	art := entityArtA()
	art.Frames = []*terrain.StaticFrame{x.unit}
	x.mk = func(cells []image.Point) *Viewer {
		overlay := make([]uint8, 400)
		overlay[2*20+2] = 1
		g := terrain.Grid{Width: 20, Height: 20, Tiles: make([]uint16, 400), Overlay: overlay, Structures: []terrain.StructureRecord{
			{ID: 0, X: 4 << 8, Y: 4 << 8, Key: 1}, {ID: 1, X: 8 << 8, Y: 4 << 8, Key: 1}, {ID: 2, X: 12 << 8, Y: 4 << 8, Key: 2}}}
		statics := &terrain.StaticSet{}
		statics.Classes[1] = &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: x.object, Frames: []*terrain.StaticFrame{x.object}}
		structures := &terrain.StructureSet{}
		structures.Classes[1] = &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1, Frames: []*terrain.StaticFrame{x.plain}}
		structures.Classes[2] = &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1, Flat: true, Frames: []*terrain.StaticFrame{x.flat}}
		v, err := NewViewerWithStatics("tint", g, &terrain.Tileset{}, statics, true, false, terrain.AnimGateTiles, structures, true)
		if err != nil {
			t.Fatal(err)
		}
		var entities []MapEntity
		for i, c := range cells {
			entities = append(entities, MapEntity{ID: uint32(i), Cell: c, Art: art, Frame: x.unit, Owner: uint32(x.unitOwners[i])})
		}
		v.SetEntities(entities)
		v.SetSackFrames([]*terrain.StaticFrame{x.sack})
		v.SetSacks([]MapSack{{Cell: x.sackAt[0]}})
		return v
	}
	x.e = NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
		return &InspectionDocument{Viewer: x.mk(x.unitCells), Width: 20, Height: 20, Records: x.records}, nil
	})
	if err := x.e.Open("tint"); err != nil {
		t.Fatal(err)
	}
	x.e.Layout(1024, 768)
	v := x.e.doc.Viewer
	v.cam.SetZoom(1)
	v.cam.X, v.cam.Y = 0, 0
	return x
}

func (x *tintFixture) draw() *shadowRecorder {
	x.e.applyPlayerTint()
	var r shadowRecorder
	x.e.doc.Viewer.drawArt(&r)
	return &r
}

// blend is the test's own statement of the mask: the sprite's colour at
// 55 % and the player's at 45 %, rounded half up.
func blend(under, over uint8) uint8 { return uint8((int(under)*55 + int(over)*45 + 50) / 100) }

func TestPlayerColoursTintOwnedObjectsOnly(t *testing.T) {
	x := newTintFixture(t)
	v := x.e.doc.Viewer
	off := x.draw()
	if x.e.view.playerColours {
		t.Fatal("player colours start on")
	}
	if err := x.e.Key("playertint"); err != nil || !x.e.view.playerColours {
		t.Fatalf("Key playertint: %v on=%v", err, x.e.view.playerColours)
	}
	on := x.draw()
	if !reflect.DeepEqual(off.geo, on.geo) || !reflect.DeepEqual(off.blend, on.blend) || !reflect.DeepEqual(off.alpha, on.alpha) || len(off.imgs) != len(on.imgs) {
		t.Fatal("the mask moved, resized or re-blended a draw")
	}
	shadows := 0
	for i := range on.imgs {
		if on.blend[i] == shadowBlend {
			shadows++
			if on.imgs[i] != off.imgs[i] {
				t.Fatalf("shadow draw %d changed under the mask", i)
			}
		}
	}
	if shadows < 3 {
		t.Fatalf("only %d shadow draws; the fixture lost its casters", shadows)
	}
	count := func(r *shadowRecorder) map[*ebiten.Image]int {
		m := map[*ebiten.Image]int{}
		for i, img := range r.imgs {
			if r.blend[i] != shadowBlend {
				m[img]++
			}
		}
		return m
	}
	tint := editorPlayerTints
	want := map[*ebiten.Image]int{
		v.tintedImage(x.unit, false, tint[0]):  1,
		v.tintedImage(x.unit, false, tint[1]):  1,
		v.tintedImage(x.unit, false, tint[2]):  1,
		v.staticImage(x.unit):                  2, // no owner, and an owner the map has no player for
		v.tintedImage(x.plain, false, tint[1]): 1,
		v.staticImage(x.plain):                 1,
		v.tintedImage(x.flat, false, tint[2]):  1,
		v.staticImage(x.object):                1,
		v.staticImage(x.sack):                  1,
	}
	if got := count(on); !reflect.DeepEqual(got, want) {
		t.Fatalf("mask on drew %d distinct pictures, want the %d of the owners", len(got), len(want))
	}
	if got, base := count(off), map[*ebiten.Image]int{v.staticImage(x.unit): 5, v.staticImage(x.plain): 2, v.staticImage(x.flat): 1, v.staticImage(x.object): 1, v.staticImage(x.sack): 1}; !reflect.DeepEqual(got, base) {
		t.Fatal("mask off does not draw today's pictures")
	}
	_ = x.e.Key("playertint")
	if again := x.draw(); !reflect.DeepEqual(again, off) {
		t.Fatal("mask off again is not byte for byte the first frame")
	}
}

func TestPlayerColoursBlendEveryOpaquePixel(t *testing.T) {
	x := newTintFixture(t)
	v := x.e.doc.Viewer
	tint := editorPlayerTints
	for _, f := range []*terrain.StaticFrame{x.unit, x.plain, x.flat} {
		src := v.spritePixels(f)
		for p, c := range []color.RGBA{tint[0], tint[1]} {
			got := v.tintedPixels(f, false, c)
			for y := 0; y < f.Height; y++ {
				for X := 0; X < f.Width; X++ {
					s, g := src.RGBAAt(X, y), got.RGBAAt(X, y)
					if s.A == 0 {
						if g != (color.RGBA{}) {
							t.Fatalf("hole (%d,%d) of player %d became %v", X, y, p+1, g)
						}
						continue
					}
					if want := (color.RGBA{blend(s.R, c.R), blend(s.G, c.G), blend(s.B, c.B), s.A}); g != want {
						t.Fatalf("pixel (%d,%d) of player %d is %v, want %v", X, y, p+1, g, want)
					}
				}
			}
		}
		if reflect.DeepEqual(v.tintedPixels(f, false, tint[0]).Pix, v.tintedPixels(f, false, tint[1]).Pix) {
			t.Fatal("two players share a tint")
		}
	}
}

func TestPlayerColourTableHoldsSixteenDistinctColours(t *testing.T) {
	seen := map[color.RGBA]bool{}
	for n, c := range editorPlayerTints {
		if c.A != 255 || seen[c] {
			t.Fatalf("player %d colour %v is repeated or translucent", n+1, c)
		}
		seen[c] = true
	}
	if len(seen) != 16 {
		t.Fatalf("%d colours", len(seen))
	}
}

func TestPlayerLegendListsEveryPlayerInItsColour(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	base := e.ChromeImage()
	if box, rows := e.playerLegend(); !box.Empty() || len(rows) != 0 {
		t.Fatal("legend shown with the mode off")
	}
	_ = e.Key("playertint")
	box, rows := e.playerLegend()
	if len(rows) != 3 {
		t.Fatalf("%d legend rows, want one per player", len(rows))
	}
	area := e.mapArea()
	if !box.In(area) || box.Min.X > area.Min.X+16 || box.Max.Y < area.Max.Y-16 {
		t.Fatalf("legend %v is not in the map view's bottom left corner %v", box, area)
	}
	pic := e.ChromeImage()
	for i, r := range rows {
		if want := x.records[i].Label; r.label != want || r.color != editorPlayerTints[i] {
			t.Fatalf("row %d is %q %v, want %q %v", i, r.label, r.color, want, editorPlayerTints[i])
		}
		mid := image.Pt(r.rect.Min.X+legendSwatch/2, r.rect.Min.Y+r.rect.Dy()/2)
		if got := pic.RGBAAt(mid.X, mid.Y); got != r.color {
			t.Fatalf("row %d swatch pixel %v, want %v", i, got, r.color)
		}
	}
	for y := 0; y < pic.Bounds().Dy(); y++ {
		for px := 0; px < pic.Bounds().Dx(); px++ {
			if !image.Pt(px, y).In(box) && pic.RGBAAt(px, y) != base.RGBAAt(px, y) {
				t.Fatalf("the legend changed pixel (%d,%d) outside its box", px, y)
			}
		}
	}
	_ = e.Key("playertint")
	if got := e.ChromeImage(); !reflect.DeepEqual(got.Pix, base.Pix) {
		t.Fatal("legend left pixels behind after the mode went off")
	}
}

func TestViewMenuRowTogglesPlayerColours(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	n, ok := e.MenuRow("view", "playertint")
	if !ok {
		t.Fatal("the View menu has no player colours row")
	}
	row := func() string { return e.menuItems("view")[n].label }
	off := row()
	if want := markWord(false, e.word("editor.view.playertint")); off != want {
		t.Fatalf("row %q, want %q", off, want)
	}
	e.menuAct("playertint")
	if !e.view.playerColours || row() != markWord(true, e.word("editor.view.playertint")) || row() == off {
		t.Fatalf("row after the click: %q on=%v", row(), e.view.playerColours)
	}
	e.menuAct("playertint")
	if e.view.playerColours || row() != off {
		t.Fatal("a second click did not turn the mode off")
	}
	if e.word("editor.view.playertint") == "editor.view.playertint" || e.word("editor.keys.playertint") == "editor.keys.playertint" {
		t.Log("word table not set on this editor; the table test covers both languages")
	}
}

// tintProbe is an edit host whose move, undo and redo hand back documents in
// which one unit has moved, the way the model's history does.
type tintProbe struct {
	noUnitAuthoring
	build func(state int) *InspectionDocument // a fresh document of the unit unmoved (0) or moved (1)
	state int
}

func (p *tintProbe) Dirty() bool   { return p.state == 1 }
func (p *tintProbe) CanUndo() bool { return p.state == 1 }
func (p *tintProbe) CanRedo() bool { return p.state == 0 }
func (p *tintProbe) Move(string, int, image.Point) (*InspectionDocument, error) {
	p.state = 1
	return p.build(1), nil
}
func (p *tintProbe) Undo() (*InspectionDocument, error) { p.state = 0; return p.build(0), nil }
func (p *tintProbe) Redo() (*InspectionDocument, error) { p.state = 1; return p.build(1), nil }
func (p *tintProbe) SaveAs(string) (string, error)      { return "", nil }

func TestPlayerColourFollowsAMovedUnitThroughUndoAndRedo(t *testing.T) {
	x := newTintFixture(t)
	e := x.e
	from, to := x.unitCells[1], image.Pt(8, 12) // unit 1 belongs to player 2
	base := *e.doc
	p := &tintProbe{}
	p.build = func(state int) *InspectionDocument {
		cell := []image.Point{from, to}[state]
		d := base
		d.Records = append([]InspectionRecord(nil), base.Records...)
		cells := append([]image.Point(nil), x.unitCells...)
		cells[1] = cell
		for i, r := range d.Records {
			if r.Kind == "Unit" && r.Index == 1 {
				d.Records[i].Cell = cell
			}
		}
		d.Viewer = x.mk(cells)
		d.Edits = p
		return &d
	}
	e.doc.Edits = p
	_ = e.Key("playertint")
	spriteOf := func() staticScreenRect {
		e.applyPlayerTint()
		for _, s := range e.doc.Viewer.planeSprites() {
			if s.Inspection == (InspectionSubject{Kind: InspectionUnit, ID: 1}) {
				return s
			}
		}
		t.Fatal("unit 1 has no sprite")
		return staticScreenRect{}
	}
	check := func(step string, wantCell image.Point) staticScreenRect {
		s := spriteOf()
		if !s.Tint.On || s.Tint.Color != editorPlayerTints[1] {
			t.Fatalf("%s: unit of player 2 drawn with tint %+v", step, s.Tint)
		}
		if e.doc.Records[3+1].Cell != wantCell {
			t.Fatalf("%s: record at %v, want %v", step, e.doc.Records[4].Cell, wantCell)
		}
		cam := e.doc.Viewer.cam
		s.Y = s.Y/cam.Zoom + cam.Y // the world row, whatever the camera did meanwhile
		return s
	}
	before := check("start", from)
	// The drag of the unit from its cell to a free one commits one move.
	e.Select(4, true)
	now := time.Unix(1, 0)
	cx, cy := e.cellCenter(from)
	sx, sy := e.doc.Viewer.cam.WorldToScreen(cx, cy)
	tx, ty := e.cellCenter(to)
	tx, ty = e.doc.Viewer.cam.WorldToScreen(tx, ty)
	press, drop := mapPoint(e, image.Pt(int(sx), int(sy))), mapPoint(e, image.Pt(int(tx), int(ty)))
	e.Pointer(Input{CursorX: press.X, CursorY: press.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: drop.X, CursorY: drop.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: drop.X, CursorY: drop.Y}, now)
	if p.state != 1 {
		t.Fatal("the drag did not move the unit")
	}
	after := check("move", to)
	if after.Y == before.Y {
		t.Fatal("the moved sprite did not move")
	}
	if err := e.Key("undo"); err != nil {
		t.Fatal(err)
	}
	if got := check("undo", from); got.Y != before.Y {
		t.Fatalf("undo did not bring the sprite back: %v then %v, camera %v", before.Y, got.Y, e.doc.Viewer.cam.Y)
	}
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	if got := check("redo", to); got.Y != after.Y {
		t.Fatalf("redo did not move the sprite again: %v, then %v", after.Y, got.Y)
	}
}
