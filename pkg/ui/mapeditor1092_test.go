package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func editor1092(t *testing.T) *MapEditor {
	t.Helper()
	open := func(source string) (*InspectionDocument, error) {
		if source == "bad" {
			return nil, errors.New("refused map")
		}
		v, err := NewViewer("editor", terrain.Grid{Width: 50, Height: 50, Tiles: make([]uint16, 2500)}, &terrain.Tileset{})
		if err != nil {
			return nil, err
		}
		d := &InspectionDocument{Viewer: v, Source: source, Width: 50, Height: 50, SourceBytes: []byte("untouched")}
		for i := 0; i < 40; i++ {
			d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: i, Section: 6, Cell: image.Pt(i+1, 20), Spatial: true, Size: image.Pt(1, 1)})
		}
		d.Records[30].Kind = "Sack"
		return d, nil
	}
	e := NewMapEditor(panelFont(), []EditorMap{{Key: "first", Label: "First"}, {Key: "bad", Label: "Bad"}}, open)
	if err := e.Open("first"); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestMapEditor1092NativePixelsSelectionFilteringAndTransactionalOpen(t *testing.T) {
	e := editor1092(t)
	e.Layout(800, 600)
	v := e.doc.Viewer
	bar := e.barHeight()
	if v.cam.ViewW != 800 || v.cam.ViewH != 600-2*bar || v.place.Scale() != 1 {
		t.Fatal("the map frame is not the window between the bar and the strip")
	}
	if _, oy := v.place.Origin(); int(oy) != bar {
		t.Fatalf("the map frame starts at %v, not below the bar", oy)
	}
	e.filter, e.listShown = 1, true
	e.Select(30, true)
	e.listSpec()
	if top, n := e.model.Visible(); e.model.Selection() < top || e.model.Selection() >= top+n {
		t.Fatal("selected row outside the list popup")
	}
	before := *v.cam
	e.Select(30, false)
	if e.filter != 1 {
		t.Fatal("selection silently replaced user's filter")
	}
	old := e.doc
	for _, source := range []string{"bad", "bad"} {
		if err := e.Open(source); err == nil || e.doc != old || e.message == "" {
			t.Fatal("failed open was not transactional")
		}
	}
	if !reflect.DeepEqual(before, *v.cam) {
		t.Fatal("failed open moved previous camera")
	}
	if !bytes.Equal(e.doc.SourceBytes, []byte("untouched")) {
		t.Fatal("inspection wrote source")
	}
	e.prompt.kind = promptOpen
	if err := e.Key("escape"); err != nil || e.prompt.open() || e.message != "" {
		t.Fatal("path cancellation closed editor")
	}
	e.catalog = true
	if err := e.Key("escape"); err != nil || e.catalog {
		t.Fatal("catalogue cancellation lost current map")
	}
}

func TestMapEditor1092CameraAndCataloguePointerDoors(t *testing.T) {
	e := editor1092(t)
	e.Layout(1024, 768)
	v := e.doc.Viewer
	e.Select(20, true)
	before := *v.cam
	e.showRecords(0)
	e.Select(20, true)
	before = *v.cam
	e.listSpec()
	row := listRowPoint(t, e, e.model.Selection())
	in := Input{CursorX: row.X, CursorY: row.Y, WheelY: -2}
	sel := e.model.Selection()
	e.Pointer(in, time.Unix(1, 0))
	if !reflect.DeepEqual(before, *v.cam) {
		t.Fatal("list popup wheel zoomed/panned canvas")
	}
	if e.model.Selection() == sel {
		t.Fatal("list popup wheel did not move the list")
	}
	e.closeList()
	zoom := v.cam.Zoom
	e.Pointer(Input{CursorX: 350, CursorY: 350, WheelY: 1}, time.Unix(1, 0))
	if v.cam.Zoom <= zoom {
		t.Fatal("canvas wheel did not zoom")
	}
	// Select exact cell through the rendered camera, not an index-only API.
	wx, wy := e.cellCenter(e.doc.Records[20].Cell)
	sx, sy := v.cam.WorldToScreen(wx, wy)
	e.selected = -1
	e.objectHidden = true
	at := mapPoint(e, image.Pt(int(sx), int(sy)))
	in = Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}
	e.Pointer(in, time.Unix(1, 0))
	in.PrimaryDown = false
	e.Pointer(in, time.Unix(1, 0))
	if e.selected != 20 {
		t.Fatalf("canvas selected %d want20", e.selected)
	}
	x := v.cam.X
	e.Pointer(Input{PanRight: true, CursorX: 350, CursorY: 350}, time.Unix(1, 0))
	if v.cam.X <= x {
		t.Fatal("arrow pan missing")
	}
	if v.AnimationCounter() != 0 {
		t.Fatal("read-only inspection advanced animation")
	}
}

func TestMapEditor1092LargeMapFitAndLocalZoomAtSupportedSizes(t *testing.T) {
	for _, cells := range []int{128, 256} {
		for _, size := range []image.Point{image.Pt(640, 480), image.Pt(800, 600), image.Pt(1280, 800)} {
			g := terrain.Grid{Width: cells, Height: cells, Tiles: make([]uint16, cells*cells)}
			v, err := NewViewer("fit", g, &terrain.Tileset{})
			if err != nil {
				t.Fatal(err)
			}
			e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
				return &InspectionDocument{Viewer: v, Width: cells, Height: cells}, nil
			})
			e.Layout(size.X, size.Y)
			if err := e.Open("large"); err != nil {
				t.Fatal(err)
			}
			viewH := size.Y - 2*e.barHeight()
			want := math.Min(float64(size.X)/float64(cells*32), float64(viewH)/float64(cells*32))
			check := func() {
				if math.Abs(v.cam.Zoom-want) > 1e-12 {
					t.Fatalf("%dx%d at%v zoom=%v want%v", cells, cells, size, v.cam.Zoom, want)
				}
				for _, p := range []image.Point{{0, 0}, {cells * 32, 0}, {0, cells * 32}, {cells * 32, cells * 32}} {
					x, y := v.cam.WorldToScreen(float64(p.X), float64(p.Y))
					if x < -1e-8 || y < -1e-8 || x > float64(size.X)+1e-8 || y > float64(viewH)+1e-8 {
						t.Fatalf("edge %v outside %v: %v,%v", p, size, x, y)
					}
				}
			}
			check() // initial Open must fit too
			e.Pointer(Input{CursorX: 150, CursorY: 240, WheelY: 1}, time.Unix(1, 0))
			if math.Abs(v.cam.Zoom-want*WheelZoomStep) > 1e-12 {
				t.Fatal("first zoom jumped to game minimum")
			}
			_ = e.Key("fit")
			check()
			e.Pointer(Input{CursorX: 150, CursorY: 240, WheelY: -1}, time.Unix(1, 0))
			want = math.Max(math.Min(0.125, want), want/WheelZoomStep)
			check() // zoom floor remains editor-local after shared camera input
			ordinary, err := NewViewer("ordinary", g, &terrain.Tileset{})
			if err != nil {
				t.Fatal(err)
			}
			ordinary.cam.SetZoom(0.001)
			if ordinary.cam.Zoom != 0.125 {
				t.Fatal("ordinary Viewer minimum changed")
			}
		}
	}
}

func TestMapEditor1092FractionalWheelOwnsTravelPerPopup(t *testing.T) {
	e := editor1092(t)
	e.Layout(640, 480)
	e.showRecords(0)
	e.Select(30, true)
	for i := 0; i < 40; i++ {
		e.doc.Records[30].Details = append(e.doc.Records[30].Details, fmt.Sprintf("item %02d", i))
	}
	e.ChromeImage()
	e.model.Select(0)
	cameraBefore := *e.doc.Viewer.cam
	listAt := listRowPoint(t, e, 0)
	listAt.X = e.listAt.X + popupPad + 4
	_, l := objectLayout(t, e)
	objectAt := center(l.Rows[0].Rect)
	if e.chromeHit(listAt) != popList || e.chromeHit(objectAt) != popObject {
		t.Fatalf("setup: list %v object %v", e.chromeHit(listAt), e.chromeHit(objectAt))
	}
	wheel := func(p image.Point, delta float64, n int) {
		for i := 0; i < n; i++ {
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: delta}, time.Unix(1, 0))
			e.ChromeImage()
		}
	}
	wheel(listAt, -0.25, 20)
	if e.model.Selection() != 15 || e.objectTop != 0 {
		t.Fatalf("list=%d object=%d", e.model.Selection(), e.objectTop)
	}
	wheel(listAt, 0.25, 20)
	if e.model.Selection() != 0 {
		t.Fatal("fractional list cannot return")
	}
	wheel(objectAt, -0.25, 20)
	if e.objectTop != 15 || e.model.Selection() != 0 {
		t.Fatalf("object=%d list=%d", e.objectTop, e.model.Selection())
	}
	wheel(objectAt, -0.25, 400)
	s, _ := objectLayout(t, e)
	if e.objectTop+s.MaxRows != len(s.Rows) {
		t.Fatal("cannot reach the last detail")
	}
	wheel(objectAt, 0.25, 400)
	if e.objectTop != 0 || e.model.Selection() != 0 || !reflect.DeepEqual(cameraBefore, *e.doc.Viewer.cam) {
		t.Fatal("reverse detail traversal moved list/camera or failed")
	}
	// Neither popup may borrow the other's uncommitted fractional rows.
	e.listWheel, e.objectWheel = 0, 0
	wheel(listAt, -0.1, 3)
	wheel(objectAt, -0.1, 1)
	if e.model.Selection() != 0 || e.objectTop != 0 {
		t.Fatal("fractional remainder crossed popup boundary")
	}
	wheel(listAt, -0.1, 1)
	if e.model.Selection() != 1 || e.objectTop != 0 {
		t.Fatal("list did not retain its own remainder")
	}
}

func TestMapEditor1092MinimumWindowAllSackDetailsReachTheObjectPopup(t *testing.T) {
	e := editor1092(t)
	e.Layout(640, 480)
	e.Select(30, true)
	r := &e.doc.Records[30]
	r.Label = "ground loot"
	for i := 0; i < 30; i++ {
		r.Details = append(r.Details, fmt.Sprintf("%02d: item with authored code %04x", i, 0x1100+i))
	}
	beforeCamera := *e.doc.Viewer.cam
	before := e.ChromeImage()
	if e.doc.Viewer.cam.ViewW != 640 || e.doc.Viewer.place.Scale() != 1 {
		t.Fatal("minimum window shrank glyphs")
	}
	seen := map[string]bool{}
	_, l := objectLayout(t, e)
	at := center(l.Rows[0].Rect)
	for i := 0; i < 60; i++ {
		s, l := objectLayout(t, e)
		if s.MaxRows < 3 {
			t.Fatal("no usable detail viewport")
		}
		for _, rr := range l.Rows {
			if !rr.Rect.In(l.Rect) {
				t.Fatalf("row %d leaves the popup", rr.Index)
			}
			seen[s.Rows[rr.Index].Label] = true
		}
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, time.Unix(1, 0))
	}
	for _, d := range r.Details {
		if !seen[d] {
			t.Fatalf("detail line %q skipped by wheel scroll", d)
		}
	}
	if !reflect.DeepEqual(beforeCamera, *e.doc.Viewer.cam) {
		t.Fatal("detail scroll moved the camera")
	}
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: 100}, time.Unix(1, 0))
	if !bytes.Equal(before.Pix, e.ChromeImage().Pix) {
		t.Fatal("cannot return to first detail")
	}
}

type editorDraw1092 struct{ images []*ebiten.Image }

func (d *editorDraw1092) DrawImage(img *ebiten.Image, _ *ebiten.DrawImageOptions) {
	d.images = append(d.images, img)
}

func TestMapEditor1092ActualFourArtLayersReachDrawAndPreview(t *testing.T) {
	frame := func(index byte) *terrain.StaticFrame {
		f := syntheticStaticFrame(8, 8)
		f.Palette[index] = color.RGBA{index, byte(255 - index), 20, 255}
		for i := range f.Pixels {
			f.Pixels[i] = terrain.StaticPixel{Index: index, Opaque: true}
		}
		return f
	}
	u, b, s, o := frame(40), frame(80), frame(120), frame(160)
	g := terrain.Grid{Width: 20, Height: 20, Tiles: make([]uint16, 400), Overlay: make([]byte, 400), Structures: []terrain.StructureRecord{{ID: 0, X: 4 << 8, Y: 4 << 8, Key: 1}}}
	g.Overlay[6*20+6] = 1
	statics := &terrain.StaticSet{}
	statics.Classes[1] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 4, Frame: o}
	structures := &terrain.StructureSet{}
	structures.Classes[1] = &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1, Frames: []*terrain.StaticFrame{b}}
	v, err := NewViewerWithStatics("all", g, &terrain.Tileset{}, statics, true, false, false, structures, true)
	if err != nil {
		t.Fatal(err)
	}
	v.SetEntities([]MapEntity{{ID: 0, Cell: image.Pt(2, 2), Art: &terrain.UnitClass{Width: 8, Height: 8, CenterX: 4, CenterY: 4, Frames: []*terrain.StaticFrame{u}}, Frame: u}})
	v.SetSackFrames([]*terrain.StaticFrame{s})
	v.SetSacks([]MapSack{{Cell: image.Pt(8, 8)}})
	e := NewMapEditor(panelFont(), nil, func(string) (*InspectionDocument, error) {
		return &InspectionDocument{Viewer: v, Width: 20, Height: 20, Records: []InspectionRecord{{Kind: "Sack", Section: 8, Index: 0, Cell: image.Pt(8, 8), Size: image.Pt(1, 1), Spatial: true, Preview: s}}}, nil
	})
	if err := e.Open("all"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1024, 768)
	v.cam.SetZoom(1)
	v.cam.X, v.cam.Y = 0, 0
	spy := &editorDraw1092{}
	v.drawArt(spy)
	for _, f := range []*terrain.StaticFrame{u, b, s, o} {
		want := v.staticImage(f)
		found := false
		for _, img := range spy.images {
			if img == want {
				found = true
			}
		}
		if !found {
			t.Fatal("actual content draw omitted a unit/building/sack/scenery texture")
		}
	}
	e.Select(0, false)
	pic := e.ChromeImage()
	data := objectRowRect(t, e, e.obj.data).Rect // the picture stands right of the data
	if got := pic.RGBAAt(data.Max.X-4, data.Min.Y+4); got != s.Palette[120] {
		t.Fatalf("selected sack art pixel %v", got)
	}
	if len(e.markerLines()) != 6 {
		t.Fatal("selected record lacks exact anchor and footprint")
	}
	// Wrapping must preserve shipped CP866 bytes rather than decoding them as
	// UTF-8 and replacing each high byte with a replacement character.
	raw := string([]byte{0x81, 0xe0, 0xae, 0xad, 0xa7, 0xae, 0xa2, 0xeb, 0xa9})
	if got := strings.Join(editorWrap(panelFont(), raw, 4), ""); got != raw {
		t.Fatal("RU item name bytes corrupted by wrapping")
	}
}
