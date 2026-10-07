package main

// The raster tool's half of the object cycle (0031 AC-7, SC-6).
//
// EVERY FIXTURE IS SYNTHETIC: a .res archive and an .alm built byte by byte by
// internal/synth, a three-frame .256 sheet and a one-class objects.reg. No game
// install is read, and nothing here is evidence about a shipped file.
//
// THE COMPARISON IS AGAINST THE WINDOW'S OWN PIXELS, not against a second walk
// of the palette written here: the window uploads (*terrain.StaticFrame).RGBA()
// and RGBALit(), so those are what the raster's pixels are compared to, byte for
// byte, over the sprite's whole rectangle. A palette walk of this file's own
// would be a third answer that could agree with neither.

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The cycling fixture: one class, one sheet of three frames, each a different
// solid palette index AND a different size, on one cell.
//
// The sizes differ so the re-anchor is exercised by the picture itself: a frame
// drawn at another frame's top-left lands in the wrong place and the rectangle
// comparison below fails on the terrain it uncovers.
const (
	cycleByte = 1 // ID 0

	cycleCol, cycleRow = 1, 1
	cycleCanvas        = 32
	cycleCentre        = 16
)

var cycleFrames = []struct {
	w, h  int
	index uint8
}{
	{20, 30, 1},
	{12, 18, 2},
	{26, 10, 3},
}

// cycleColors are the palette entries those indices select, none of them a
// terrain or marker colour.
var cycleColors = []color.RGBA{
	{},                          // index 0, unused
	{R: 0x10, G: 0x20, B: 0x30}, // index 1
	{R: 0x40, G: 0x50, B: 0x60}, // index 2
	{R: 0x70, G: 0x80, B: 0x90}, // index 3
}

func cyclePalette() []color.RGBA { return append([]color.RGBA(nil), cycleColors...) }

// cycleSheet is the three-frame sheet, each frame wholly opaque in its own
// index — so a pixel assertion is about placement and selection alone.
func cycleSheet() []byte {
	frames := make([]synth.Frame256, len(cycleFrames))
	for i, f := range cycleFrames {
		pixels := make([]synth.Pixel256, f.w*f.h)
		for j := range pixels {
			pixels[j] = synth.Pixel256{Index: f.index, Opaque: true}
		}
		frames[i] = synth.Frame256{Width: f.w, Height: f.h, Pixels: pixels}
	}
	return synth.Sheet256(synth.Sheet256Options{Palette: cyclePalette(), Frames: frames})
}

// cycleRegistry is one class at Index 0 carrying the pair (1,1,1) over (0,1,2):
// the timeline [0 1 2], period 3, so three consecutive counters select three
// different frames of the sheet.
func cycleRegistry() []byte {
	key := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	ints := func(name string, v ...int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x06, Ints: v}
	}
	return synth.ObjectsReg([]string{`trees\cycle`}, []synth.RegNode{
		key("ID", 0), key("File", 0), key("Index", 0),
		key("Width", cycleCanvas), key("Height", cycleCanvas),
		key("CenterX", cycleCentre), key("CenterY", cycleCentre),
		ints("AnimationTime", 1, 1, 1), ints("AnimationFrame", 0, 1, 2),
	})
}

// writeCycleFixtures lays a graphics.res and a 4x3 map with the class on one
// cell. gateBits selects whether the map's tile words carry both of the cycle
// gate's bits, so the DECODED gate can be driven through the tool and not only
// the diagnostic.
//
// The bits are bits 15..14, which the tile split does not read, so setting them
// moves no terrain pixel — asserted below rather than assumed.
func writeCycleFixtures(t *testing.T, dir string, gateBits bool) string {
	t.Helper()

	archive := synth.Archive([]synth.File{
		{Path: "terrain/tile1-00.bmp", Data: solidStrip(14, landBase)},
		{Path: "terrain/tile3-00.bmp", Data: solidStrip(8, waterBase)},
		{Path: "terrain/dirt.bmp", Data: solidStrip(4, dirtBase)},
		{Path: "objects/objects.reg", Data: cycleRegistry()},
		{Path: "objects/trees/cycle.256", Data: cycleSheet()},
	})
	if err := os.WriteFile(filepath.Join(dir, "graphics.res"), archive, 0o644); err != nil {
		t.Fatal(err)
	}

	tiles := fixtureTiles()
	if gateBits {
		for i := range tiles {
			tiles[i] |= 0xc000
		}
	}
	overlay := make([]uint8, 4*3)
	overlay[cycleRow*4+cycleCol] = cycleByte

	name := "cycle.alm"
	if gateBits {
		name = "cycle-gated.alm"
	}
	path := filepath.Join(dir, name)
	data := synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3,
		Type1Payload: tilePayload(tiles),
		Overlay:      overlay,
	})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// cycleRectAt is the world rectangle the cell's sprite occupies when frame f is
// drawn, worked out by hand from the anchor formula over a 32x32 canvas centred
// (16,16) on the flat geometry:
//
//	anchorX = (16 - 16) + w/2      destX = 1*32 + 16 - anchorX
//	anchorY = (16 - 16) + h/2      destY = 1*32 + 16 - anchorY
//
//	frame 0 (20x30): anchor (10,15) top-left (38, 33) rect [38,58) x [33,63)
//	frame 1 (12x18): anchor ( 6, 9) top-left (42, 39) rect [42,54) x [39,57)
//	frame 2 (26x10): anchor (13, 5) top-left (35, 43) rect [35,61) x [43,53)
//
// The ground point is (48, 48) at every one of them, which is the whole content
// of the re-anchor.
func cycleRectAt(f int) image.Rectangle {
	w, h := cycleFrames[f].w, cycleFrames[f].h
	ax, ay := w/2, h/2
	x := cycleCol*terrain.CellSize + terrain.CellSize/2 - ax
	y := cycleRow*terrain.CellSize + terrain.CellSize/2 - ay
	return image.Rect(x, y, x+w, y+h)
}

// checkCycleRect compares the rendered image against the window's own pixels for
// one frame, over that frame's whole rectangle, byte for byte.
func checkCycleRect(t *testing.T, label string, img image.Image, want *image.RGBA, r image.Rectangle) {
	t.Helper()
	if want.Bounds().Dx() != r.Dx() || want.Bounds().Dy() != r.Dy() {
		t.Fatalf("%s: the window's image is %v, the raster rectangle %v", label, want.Bounds(), r)
	}
	for y := 0; y < r.Dy(); y++ {
		for x := 0; x < r.Dx(); x++ {
			got := color.RGBAModel.Convert(img.At(r.Min.X+x, r.Min.Y+y)).(color.RGBA)
			w := want.RGBAAt(x, y)
			if got != w {
				t.Fatalf("%s: pixel (%d,%d) of the sprite = %v, the window's is %v",
					label, x, y, got, w)
			}
		}
	}
}

// cycleBundle loads the fixture bundle the way the tool does, so the frames the
// window would upload come from the same decode the raster's blit read.
func cycleBundle(t *testing.T, dir string) *terrain.StaticClass {
	t.Helper()
	containers, err := game.OpenContainers(filepath.Join(dir, "graphics.res"))
	if err != nil {
		t.Fatalf("OpenContainers: %v", err)
	}
	set, err := game.LoadStatics(containers)
	if err != nil {
		t.Fatalf("LoadStatics: %v", err)
	}
	c := set.Classes[cycleByte]
	if c == nil || len(c.Frames) != len(cycleFrames) {
		t.Fatalf("the fixture class carries %v frames, want %d", c, len(cycleFrames))
	}
	if want := []int{0, 1, 2}; len(c.Timeline) != len(want) {
		t.Fatalf("the fixture timeline is %v, want %v", c.Timeline, want)
	}
	return c
}

// TestRenderStaticsAtAChosenTick is AC-7 and SC-6: the raster's pixels and the
// window's own pixels compared byte for byte at one counter and at a second
// counter that selects a different frame, unshaded and lit.
//
// The counters are chosen from the contract's own step, so the frame each one
// selects is stated here rather than read back from the tool.
func TestRenderStaticsAtAChosenTick(t *testing.T) {
	dir := t.TempDir()
	mapPath := writeCycleFixtures(t, dir, true)
	class := cycleBundle(t, dir)

	// cell (1,1): step = (counter + col*(row+1)) mod 3 = (counter + 2) mod 3.
	// Counter 0 -> step 2 -> timeline value 2 -> frame 2.
	// Counter 2 -> step 1 -> timeline value 1 -> frame 1.
	for _, tc := range []struct {
		tick  uint32
		frame int
	}{
		{0, 2},
		{2, 1},
	} {
		step := terrain.ObjectStep(cycleCol, cycleRow, tc.tick, len(class.Timeline))
		if class.Timeline[step] != tc.frame {
			t.Fatalf("tick %d selects frame %d, not the %d this case is written for",
				tc.tick, class.Timeline[step], tc.frame)
		}
		r := cycleRectAt(tc.frame)
		frame := class.Frames[tc.frame]

		unshadedLabel := fmt.Sprintf("tick %d unshaded", tc.tick)
		summary, img := runTool(t, dir, mapPath, fmt.Sprintf("cycle-%d-unshaded", tc.tick),
			"-statics", "-objectanim", "-unshaded", "-tick", fmt.Sprint(tc.tick))
		if !strings.Contains(summary, ", statics 1 (animated 1)") {
			t.Errorf("%s: summary %q does not report one open cycle", unshadedLabel, summary)
		}
		checkCycleRect(t, unshadedLabel, img, frame.RGBA(), r)

		litLabel := fmt.Sprintf("tick %d lit", tc.tick)
		_, litImg := runTool(t, dir, mapPath, fmt.Sprintf("cycle-%d-lit", tc.tick),
			"-statics", "-objectanim", "-tick", fmt.Sprint(tc.tick))
		light := terrain.DefaultDaytime
		checkCycleRect(t, litLabel, litImg,
			frame.RGBALit(light.SkyTint, terrain.SpriteRow(light)), r)
	}
}

func TestRenderStaticsTickDefaultsToZero(t *testing.T) {
	dir := t.TempDir()
	mapPath := writeCycleFixtures(t, dir, true)

	_, implicit := runTool(t, dir, mapPath, "implicit", "-statics", "-objectanim", "-unshaded")
	_, explicit := runTool(t, dir, mapPath, "explicit", "-statics", "-objectanim", "-unshaded", "-tick", "0")
	if at, ok := differsOutside(implicit, explicit); !ok {
		t.Errorf("an omitted -tick and -tick 0 differ at %v", at)
	}
}

func TestRenderStaticsDecodedGateNeedsTheTileBits(t *testing.T) {
	dir := t.TempDir()
	gated := writeCycleFixtures(t, dir, true)
	plain := writeCycleFixtures(t, dir, false)

	// The gated map, decoded gate, no diagnostic: the counter moves the picture.
	gatedSummary, atZero := runTool(t, dir, gated, "gated-0", "-statics", "-unshaded", "-tick", "0")
	if !strings.Contains(gatedSummary, ", statics 1 (animated 1)") {
		t.Fatalf("the gated map reports %q, want one open cycle", strings.TrimSpace(gatedSummary))
	}
	_, atTwo := runTool(t, dir, gated, "gated-2", "-statics", "-unshaded", "-tick", "2")
	if _, ok := differsOutside(atZero, atTwo); ok {
		t.Error("the decoded gate opened a cycle but two counters drew the same image")
	}

	// The ungated map: 0 open cycles, and the same image at every counter — the
	// one the layer drew before this story, which is the class's Index frame.
	baseSummary, base := runTool(t, dir, plain, "plain-0", "-statics", "-unshaded")
	if !strings.Contains(baseSummary, ", statics 1 (animated 0)") {
		t.Fatalf("the ungated map reports %q, want no open cycle", strings.TrimSpace(baseSummary))
	}
	class := cycleBundle(t, dir)
	checkCycleRect(t, "the pre-story picture", base, class.Frames[class.Index].RGBA(), cycleRectAt(class.Index))

	for _, tick := range []string{"1", "2", "3", "97", "4294967295"} {
		_, img := runTool(t, dir, plain, "plain-"+tick, "-statics", "-unshaded", "-tick", tick)
		if at, ok := differsOutside(img, base); !ok {
			t.Errorf("-tick %s moved a pixel at %v on a map no cell of which opens the gate", tick, at)
		}
	}

	// The diagnostic is what makes the arm visible on that same map, and it is
	// the ONLY thing that does.
	diagSummary, diag := runTool(t, dir, plain, "plain-diag", "-statics", "-unshaded", "-objectanim", "-tick", "2")
	if !strings.Contains(diagSummary, ", statics 1 (animated 1)") {
		t.Errorf("-objectanim reports %q, want one open cycle", strings.TrimSpace(diagSummary))
	}
	if _, ok := differsOutside(diag, base); ok {
		t.Error("-objectanim drew the pre-story picture; the diagnostic opened nothing")
	}
}

// TestRenderStaticsGateBitsMoveNoTerrain pins the premise the fixture above
// rests on: bits 15..14 of a tile word are not read by the tile split, so
// setting them on every cell moves no terrain pixel. Without this the test above
// could pass on a picture that changed for the wrong reason.
func TestRenderStaticsGateBitsMoveNoTerrain(t *testing.T) {
	dir := t.TempDir()
	gated := writeCycleFixtures(t, dir, true)
	plain := writeCycleFixtures(t, dir, false)

	_, withBits := runTool(t, dir, gated, "terrain-gated", "-unshaded")
	_, without := runTool(t, dir, plain, "terrain-plain", "-unshaded")
	if at, ok := differsOutside(withBits, without); !ok {
		t.Errorf("setting the gate bits moved a terrain pixel at %v", at)
	}
}
