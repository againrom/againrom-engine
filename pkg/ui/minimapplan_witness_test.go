package ui

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"againrom/pkg/render/terrain"
)

// The minimap plan witness: the composed picture and the marks of every
// minimap a viewer can draw from a synthetic map, recorded in
// testdata/minimap_plan.golden before the producers became one. Every line
// holds only numbers: the case, the geometry, two picture hashes and the
// mark rectangles in the composer's paint order.
const minimapPlanGolden = "testdata/minimap_plan.golden"

var minimapWitnessMaps = []struct {
	w, h   int
	margin bool
}{
	{16, 16, false}, {40, 30, false}, // above one pixel per cell, whole map
	{32, 32, true}, {25, 47, true}, // above one pixel per cell, blocked margin
	{156, 156, false}, {128, 128, false}, // at one pixel per cell
	{144, 144, true}, {172, 172, true}, // at one pixel per cell, margin
	{300, 260, false}, {316, 276, true}, // below one pixel per cell
}

// minimapWitnessDressings are the viewer's paint and crystal combinations:
// none, the crystal alone, the lookup without the crystal, the first game's
// paint with the crystal, and a paint value with no lookup under the crystal.
const minimapWitnessDressings = 5

func minimapWitnessDress(v *Viewer, dressing int) {
	frame := &DialogFrame{Minimap: originalCrystal(), MinimapSeam: solidPic(16, 158, color.RGBA{A: 255})}
	switch dressing {
	case 1:
		v.SetDialogFrame(frame)
	case 2:
		v.SetMinimapPaint(&MinimapPaint{Colours: originalLookup(), OwnerShades: originalShades()})
	case 3:
		v.SetDialogFrame(frame)
		v.SetMinimapPaint(&MinimapPaint{Colours: originalLookup(), OwnerShades: originalShades()})
	case 4:
		v.SetDialogFrame(frame)
		v.SetMinimapPaint(&MinimapPaint{})
	}
}

// minimapWitnessTileset is a tileset whose sub-cells name their slot in a
// palette, one sub-cell in three absent, so the sampled colours and the
// placeholder both reach the picture.
func minimapWitnessTileset() *terrain.Tileset {
	set := &terrain.Tileset{}
	for slot := range set.Slots {
		strip := &terrain.Strip{}
		for sub := range 16 {
			if sub%3 == 2 {
				strip.SubCells = append(strip.SubCells, nil)
				continue
			}
			pal := color.Palette{color.RGBA{A: 255}, color.RGBA{R: uint8(slot*2 + sub), G: uint8(255 - slot), B: uint8(sub * 16), A: 255}}
			img := image.NewPaletted(image.Rect(0, 0, 32, 32), pal)
			img.SetColorIndex(16, 16, 1)
			strip.SubCells = append(strip.SubCells, img)
		}
		set.Slots[slot] = strip
	}
	return set
}

// minimapWitnessViewer builds the map of one case with every mark kind on it:
// units over each owner, corpse stage and life, structures of a fixed class, a
// variable class, an unknown class and no owner, and sacks whose frames
// resolve and do not, standing inside the window, on its edges, in the margin
// and off the map.
func minimapWitnessViewer(t *testing.T, set *terrain.Tileset, w, h int, margin bool) *Viewer {
	t.Helper()
	g := grid(w, h)
	if margin {
		g.Block = make([]uint8, w*h)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g.Tiles[y*w+x] = originalTile(x%52, y%14)
			if margin && (x < 8 || y < 8 || x >= w-8 || y >= h-8) {
				g.Block[y*w+x] = borderBit
			}
		}
	}
	v, err := NewViewer("mm-witness", g, set)
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.Layout(MissionFrameW, MissionFrameH)
	v.SetLocalOwner(1)
	points := []image.Point{
		{w / 2, h / 2}, {w/2 + 1, h / 2}, {3, 3}, {w - 4, h - 4}, {9, 9}, {0, 0}, {w - 1, h - 1},
		{w - 1, 0}, {0, h - 1}, {9, h / 2}, {w / 2, 9}, {w - 9, h - 9}, {-1, 3}, {w, h / 2}, {w + 2, -2},
	}
	for k := 1; k <= 6; k++ {
		points = append(points, image.Pt(w*k/7, h*(7-k)/7))
	}
	owners := []uint32{1, 2, 3, 0, 5}
	var ents []MapEntity
	for i, p := range points {
		ents = append(ents, MapEntity{
			ID: uint32(i + 1), Cell: p, Owner: owners[i%5],
			Life: uint8(i % 3), CorpseStage: uint8(i % 4), Restorable: i%4 == 1,
		})
	}
	v.SetEntities(ents)
	v.structureSet = originalStructureSet()
	var builds []MapStructure
	kinds := []uint16{5, 33, 9, 5, 33}
	for i, p := range points {
		size := image.Pt(4, 1)
		if i%3 == 0 {
			size = image.Point{}
		}
		builds = append(builds, MapStructure{ID: uint32(i + 1), Cell: p, Kind: kinds[i%5], Owner: []uint32{2, 1, 0, 3}[i%4], Size: size})
	}
	v.SetStructures(builds)
	v.SetSackFrames([]*terrain.StaticFrame{{Width: 4, Height: 4}, nil})
	var sacks []MapSack
	for i, p := range points {
		sacks = append(sacks, MapSack{Cell: p, FrameIndex: []int{0, 1, 5, 0}[i%4]})
	}
	v.SetSacks(sacks)
	return v
}

// minimapWitnessFog is a plane of one state, or the mixed three-state plane
// for state above FogVisible.
func minimapWitnessFog(w, h int, state uint8) []byte {
	plane := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			plane[y*w+x] = state
			if state > FogVisible {
				plane[y*w+x] = []byte{FogUnseen, FogExplored, FogVisible}[(x+y)%3]
			}
		}
	}
	return plane
}

// minimapWitnessVariants apply one state of fog and options to a viewer.
var minimapWitnessVariants = []func(v *Viewer, w, h int){
	func(v *Viewer, w, h int) {},
	func(v *Viewer, w, h int) { v.SetFog(minimapWitnessFog(w, h, FogVisible), w, h) },
	func(v *Viewer, w, h int) { v.SetFog(minimapWitnessFog(w, h, FogExplored), w, h) },
	func(v *Viewer, w, h int) { v.SetFog(minimapWitnessFog(w, h, FogUnseen), w, h) },
	func(v *Viewer, w, h int) { v.SetFog(minimapWitnessFog(w, h, FogVisible+1), w, h) },
	func(v *Viewer, w, h int) {
		v.SetFog(minimapWitnessFog(w, h, FogVisible+1), w, h)
		v.minimapHeld = true
	},
	func(v *Viewer, w, h int) {
		v.SetFog(minimapWitnessFog(w, h, FogVisible+1), w, h)
		v.graphics.HideLootDots = true
	},
	func(v *Viewer, w, h int) {
		v.SetFog(minimapWitnessFog(w, h, FogVisible+1), w, h)
		v.SetFogReveal(true)
	},
	func(v *Viewer, w, h int) {
		v.SetFog(minimapWitnessFog(w, h, FogVisible), w, h)
		v.cam.CenterOn(float64(w)*48, float64(h)*48)
	},
	func(v *Viewer, w, h int) {
		v.SetFog(minimapWitnessFog(w, h, FogVisible+1), w, h)
		v.levels = make([]uint8, w*h)
		for i := range v.levels {
			v.levels[i] = uint8(i%17 + 30)
		}
		v.SetPlayerColours([]uint32{4, 3, 2, 1, 7, 6})
		v.SetMissionScreen(MissionScreen{MinimapLocal: color.RGBA{R: 9, G: 8, B: 7, A: 255}})
	},
}

// minimapWitnessHash is one picture's size and pixels as a decimal number.
func minimapWitnessHash(pic *image.RGBA) uint64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%v", pic.Bounds().Size())
	if pic.Bounds().Dx() > 0 {
		h.Write(pic.Pix)
	}
	return h.Sum64()
}

// minimapWitnessPaintOrder is the composer's order: Under marks, then loot,
// then the rest, each in plan order.
func minimapWitnessPaintOrder(marks []minimapMark) []minimapMark {
	var out []minimapMark
	for pass := range 3 {
		for _, m := range marks {
			if (pass == 0) != m.Under || (pass == 1) != (m.Loot && !m.Under) {
				continue
			}
			out = append(out, m)
		}
	}
	return out
}

// minimapWitnessContent is the composed content picture of the viewer's plan.
func minimapWitnessContent(v *Viewer, g minimapGeom) *image.RGBA {
	return composeMinimapPlan(v.minimapProducePlan(g), g.extent)
}

// minimapWitnessMarks is the viewer's plan marks.
func minimapWitnessMarks(v *Viewer, g minimapGeom) []minimapMark {
	return v.minimapProducePlan(g).marks
}

// minimapWitnessLine is one case's row: the geometry, the content hash, the
// framed picture hash and place, and each mark's rectangle and colour.
func minimapWitnessLine(t *testing.T, index int, v *Viewer) string {
	t.Helper()
	g, ok := v.minimapGeometry()
	if !ok {
		t.Fatalf("case %d: no geometry", index)
	}
	content := minimapWitnessContent(v, g)
	pic, at, ok := v.minimapCompose(g)
	if !ok || pic == nil {
		t.Fatalf("case %d: no picture", index)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d %d %d %d %d %d %d %d %d %d", index, g.Cols, g.Rows, g.Num, g.Den, g.Content.Min.X, g.Content.Min.Y, minimapWitnessHash(content), minimapWitnessHash(pic), at.X+at.Y*10000)
	marks := minimapWitnessPaintOrder(minimapWitnessMarks(v, g))
	fmt.Fprintf(&b, " %d", len(marks))
	for _, m := range marks {
		r := minimapMarkRect(m, g.Cols, g.Rows, g.Num, g.Den, content.Bounds())
		fmt.Fprintf(&b, " %d %d %d %d %d %d %d %d", r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, m.Colour.R, m.Colour.G, m.Colour.B, m.Colour.A)
	}
	return b.String()
}

// A mod group's own colour reaches the viewer with no paint data: the cell of
// the group shows it under the symbolic fog rule, with and without the
// crystal, while the other cells keep their sampled colour. Loss control:
// without the paint the same cell shows the placeholder.
func TestAModGroupColourReachesAViewerWithNoPaintData(t *testing.T) {
	const group = 20
	own := color.RGBA{R: 12, G: 200, B: 34, A: 255}
	for _, crystal := range []bool{false, true} {
		words := make([]uint16, 8*8)
		words[8*2+3] = group<<6 | 5
		v, err := NewViewer("mm-group", grid(8, 8, words...), &terrain.Tileset{})
		if err != nil {
			t.Fatalf("NewViewer: %v", err)
		}
		v.Layout(MissionFrameW, MissionFrameH)
		v.SetEntities(nil)
		if crystal {
			v.SetDialogFrame(&DialogFrame{Minimap: originalCrystal()})
		}
		paint := &MinimapPaint{}
		paint.GroupColours[group] = own
		v.SetMinimapPaint(paint)
		if v.minimapOriginalActive() {
			t.Fatal("a paint with no lookup turned the installed picture on")
		}
		cellColour := func(cell image.Point) color.RGBA {
			t.Helper()
			g, ok := v.minimapGeometry()
			if !ok {
				t.Fatal("no minimap geometry")
			}
			img := composeMinimapPlan(v.minimapProducePlan(g), g.extent)
			r := minimapMarkRect(minimapMark{Cell: cell}, g.Cols, g.Rows, g.Num, g.Den, img.Bounds())
			return img.RGBAAt(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
		}
		if got := cellColour(image.Pt(3, 2)); got != own {
			t.Errorf("crystal %t: mod group's cell shows %v, want its own colour %v", crystal, got, own)
		}
		if got := cellColour(image.Pt(4, 4)); got != terrain.PlaceholderColor {
			t.Errorf("crystal %t: a plain cell shows %v, want the sampled placeholder", crystal, got)
		}
		v.SetFog(minimapWitnessFog(8, 8, FogExplored), 8, 8)
		if got, want := cellColour(image.Pt(3, 2)), (color.RGBA{R: own.R / 2, G: own.G / 2, B: own.B / 2, A: 255}); got != want {
			t.Errorf("crystal %t: explored mod group's cell shows %v, want the symbolic half %v", crystal, got, want)
		}
		v.SetFog(nil, 0, 0)
		v.SetMinimapPaint(nil)
		if got := cellColour(image.Pt(3, 2)); got == own {
			t.Errorf("crystal %t: loss control: the viewer with no paint drew the mod colour", crystal)
		}
	}
}

func minimapWitnessLines(t *testing.T) []string {
	set := minimapWitnessTileset()
	var out []string
	index := 0
	for dressing := range minimapWitnessDressings {
		for _, m := range minimapWitnessMaps {
			for _, variant := range minimapWitnessVariants {
				v := minimapWitnessViewer(t, set, m.w, m.h, m.margin)
				minimapWitnessDress(v, dressing)
				variant(v, m.w, m.h)
				out = append(out, minimapWitnessLine(t, index, v))
				index++
			}
		}
	}
	return out
}

func readMinimapPlanGolden(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(minimapPlanGolden)
	if err != nil {
		t.Fatalf("minimap plan golden: %v", err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if strings.Trim(sc.Text(), "0123456789 -") != "" {
			t.Fatalf("golden line holds more than numbers: %q", sc.Text())
		}
		out = append(out, sc.Text())
	}
	return out
}

func diffMinimapPlan(want, got []string) string {
	if len(want) != len(got) {
		return fmt.Sprintf("%d golden lines, %d computed", len(want), len(got))
	}
	for i := range want {
		if want[i] != got[i] {
			return fmt.Sprintf("line %d:\n golden   %s\n computed %s", i+1, want[i], got[i])
		}
	}
	return ""
}

// TestMinimapPlanWitness holds every case's picture and marks byte equal to
// the golden recorded before the producers became one.
// AGAINROM_MINIMAP_PLAN_RECORD=1 rewrites it.
func TestMinimapPlanWitness(t *testing.T) {
	got := minimapWitnessLines(t)
	if os.Getenv("AGAINROM_MINIMAP_PLAN_RECORD") == "1" {
		if err := os.WriteFile(minimapPlanGolden, []byte(strings.Join(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if d := diffMinimapPlan(readMinimapPlanGolden(t), got); d != "" {
		t.Fatalf("a minimap picture or mark moved: %s", d)
	}
}

// TestMinimapPlanWitnessFailsAPlantedFault checks the comparison itself: one
// number changed anywhere is a failure.
func TestMinimapPlanWitnessFailsAPlantedFault(t *testing.T) {
	want := readMinimapPlanGolden(t)
	got := append([]string(nil), want...)
	got[0] = strings.Replace(got[0], "0", "1", 1) + " 7"
	if diffMinimapPlan(want, got) == "" {
		t.Fatal("a planted change to one witness number was not reported")
	}
}
