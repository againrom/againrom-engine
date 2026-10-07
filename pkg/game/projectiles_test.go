package game_test

import (
	"errors"
	"image"
	"image/color"
	"io/fs"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The projectile art loader. Every fixture is synthetic — internal/synth
// writes the registry and the sheet — so nothing here reads an install.

func projNode(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func projFile(path string) synth.RegNode {
	return synth.RegNode{Name: "File", Kind: 0x00, Str: path}
}

// projArchive is a graphics container carrying a projectiles.reg built from rows
// plus whatever sheets the caller names.
func projArchive(t *testing.T, rows [][]synth.RegNode, sheets map[string][]byte) *terrainSource {
	t.Helper()
	files := []synth.File{{
		Path: graphicsEntry(t, game.ProjectileRegistry),
		Data: synth.ProjectilesReg(int32(len(rows)), rows...),
	}}
	for path, data := range sheets {
		files = append(files, synth.File{Path: graphicsEntry(t, "graphics/"+path), Data: data})
	}
	return &terrainSource{openContainers(t, synth.Archive(files))}
}

// terrainSource adapts the container filesystem to the one method the loader
// takes, which is the same seam LoadUnits reads through.
type terrainSource struct {
	fs interface{ ReadFile(string) ([]byte, error) }
}

func (s *terrainSource) ReadFile(name string) ([]byte, error) { return s.fs.ReadFile(name) }

// oneFrameSheet is a .16a stream of one painted 2x2 frame, built through the
// same control-word grammar cursor_test.go spells out.
func oneFrameSheet() []byte {
	return curSheet(2, 2, []uint16{
		curOpLiteral | 4,
		curPixel(5, 15), curPixel(6, 15), curPixel(7, 15), curPixel(8, 15),
	})
}

// indexedSheet is a .256 stream of one 2x1 frame, index 5 then a hole,
// carrying table as its own colours, or no table at all for nil.
func indexedSheet(table []color.RGBA) []byte {
	return synth.Sheet256(synth.Sheet256Options{
		Palette: table, NoPalette: table == nil,
		Frames: []synth.Frame256{{Width: 2, Height: 1,
			Pixels: []synth.Pixel256{{Index: 5, Opaque: true}, {}}}},
	})
}

// TestAnIndexedRowDrawsThroughTheTableItsPaletteBitNames: a row with A16 0
// names a .256 sheet (REG-PROJ-087). Palette 0 draws it through the shared
// projectiles.pal even when the sheet carries a table of its own, and
// Palette 1 through the sheet's own (ANIM-CAST-027, PAL-PROJ-011); a hole
// stays transparent.
func TestAnIndexedRowDrawsThroughTheTableItsPaletteBitNames(t *testing.T) {
	t.Parallel()

	own := make([]color.RGBA, 6)
	own[5] = color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc}
	shared := make(color.Palette, 256)
	for i := range shared {
		shared[i] = color.RGBA{A: 0xff}
	}
	shared[5] = color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}
	src := projArchive(t,
		[][]synth.RegNode{
			{projNode("ID", 1), projFile("archer\\arrow"), projNode("Phases", 1)},
			{projNode("ID", 4), projFile("goblin\\arrow"), projNode("Phases", 1),
				projNode("RotationPhases", 1), projNode("Palette", 1)},
		},
		map[string][]byte{
			"projectiles/archer/arrow.256": indexedSheet(own),
			"projectiles/goblin/arrow.256": indexedSheet(own),
			"projectiles/projectiles.pal":  synth.BMP8(image.NewPaletted(image.Rect(0, 0, 1, 1), shared)),
		},
	)
	set, err := game.LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	for _, tc := range []struct {
		picture int
		want    color.RGBA
	}{
		{1, color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}},
		{4, color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}},
	} {
		sheet := set.Sheet(tc.picture)
		if sheet == nil || len(sheet.Frames) != 1 {
			t.Errorf("picture %d built %+v, want one frame", tc.picture, sheet)
			continue
		}
		f := sheet.Frames[0]
		if f.Width != 2 || f.Height != 1 || f.Pixels[0] != tc.want || f.Pixels[1] != (color.RGBA{}) {
			t.Errorf("picture %d frame is %dx%d %v, want %v then a hole", tc.picture, f.Width, f.Height, f.Pixels, tc.want)
		}
		// The registry's Width and Height default to 64 (REG-PROJ-086).
		if sheet.CenterX != 32 || sheet.CenterY != 32 {
			t.Errorf("picture %d centres on %d/%d, want the defaults' halves 32/32", tc.picture, sheet.CenterX, sheet.CenterY)
		}
	}
}

func TestTheBundleIsKeyedByPictureIDAndCarriesTheRegistrysScalars(t *testing.T) {
	t.Parallel()

	src := projArchive(t,
		[][]synth.RegNode{{
			projNode("ID", 10), projFile("firebolt\\sprites"),
			projNode("Phases", 4), projNode("RotationPhases", 16),
			projNode("Width", 128), projNode("Height", 96),
			projNode("Flip", 1), projNode("A16", 1), projNode("Palette", 1),
		}},
		map[string][]byte{"projectiles/firebolt/sprites.16a": oneFrameSheet()},
	)

	set, err := game.LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	sheet := set.Sheet(10)
	if sheet == nil {
		t.Fatal("the bundle holds no sheet at picture 10")
	}
	if sheet.Phases != 4 || sheet.RotationPhases != 16 || !sheet.Flip {
		t.Errorf("the sheet carries %+v, want the registry's own 4/16/flip", sheet)
	}
	// The registry's Width and Height are the draw's centring halves.
	if sheet.CenterX != 64 || sheet.CenterY != 48 {
		t.Errorf("the centring halves are %d/%d, want 64/48", sheet.CenterX, sheet.CenterY)
	}
	if len(sheet.Frames) != 1 || sheet.Frames[0].Width != 2 || sheet.Frames[0].Height != 2 {
		t.Errorf("the sheet holds %d frames, want one 2x2", len(sheet.Frames))
	}
	if sheet.Clock != terrain.EffectClockHalf {
		t.Errorf("picture 10 runs clock %v, want the default", sheet.Clock)
	}
	if set.Sheet(11) != nil {
		t.Error("a picture the registry does not name resolved a sheet")
	}
}

func TestTheTwoOverridePicturesCarryTheirOwnClock(t *testing.T) {
	t.Parallel()

	src := projArchive(t,
		[][]synth.RegNode{
			{projNode("ID", 60), projFile("teleport\\sprites"), projNode("Phases", 21),
				projNode("RotationPhases", 1), projNode("A16", 1)},
			{projNode("ID", 51), projFile("Meteor\\sprites"), projNode("Phases", 9),
				projNode("RotationPhases", 1), projNode("A16", 1)},
		},
		map[string][]byte{
			"projectiles/teleport/sprites.16a": oneFrameSheet(),
			"projectiles/Meteor/sprites.16a":   oneFrameSheet(),
		},
	)
	set, err := game.LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	if got := set.Sheet(60).Clock; got != terrain.EffectClockDirect {
		t.Errorf("picture 60 runs clock %v, want the direct one", got)
	}
	if got := set.Sheet(51).Clock; got != terrain.EffectClockRaw {
		t.Errorf("picture 51 runs clock %v, want the raw one", got)
	}
}

func TestEveryUnreadableRowIsASkipAndOnlyTheRegistryIsAnError(t *testing.T) {
	t.Parallel()

	src := projArchive(t,
		[][]synth.RegNode{
			// An indexed sheet with no table of its own, on an archive with no
			// shared projectiles.pal to draw it through.
			{projNode("ID", 1), projFile("archer\\arrow"), projNode("Phases", 1)},
			// An indexed row whose Palette bit claims a table its sheet lacks.
			{projNode("ID", 4), projFile("goblin\\arrow"), projNode("Phases", 1),
				projNode("Palette", 1)},
			// The entry is not in the archive.
			{projNode("ID", 20), projFile("healing\\sprites"), projNode("Phases", 7),
				projNode("A16", 1)},
			// The stream is not a sheet.
			{projNode("ID", 30), projFile("Drain\\sprites"), projNode("Phases", 9),
				projNode("A16", 1)},
			// And one that reads, so the walk is shown to continue past the
			// refusals rather than stopping at the first.
			{projNode("ID", 34), projFile("lightnin\\sprites"), projNode("Phases", 5),
				projNode("RotationPhases", 1), projNode("A16", 1)},
		},
		map[string][]byte{
			"projectiles/archer/arrow.256":     indexedSheet(nil),
			"projectiles/goblin/arrow.256":     indexedSheet(nil),
			"projectiles/Drain/sprites.16a":    []byte("not a sheet"),
			"projectiles/lightnin/sprites.16a": oneFrameSheet(),
		},
	)
	set, err := game.LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	for _, picture := range []int{1, 4, 20, 30} {
		if set.Sheet(picture) != nil {
			t.Errorf("picture %d built a sheet it should have skipped", picture)
		}
	}
	if set.Sheet(34) == nil {
		t.Error("the readable row was dropped with the skips")
	}
}

// TestAnAbsentRegistryIsReportedAsTheAddressItReadFrom is the one failure, and
// its error is returned unwrapped so a -check run can test it by kind.
func TestAnAbsentRegistryIsReportedAsTheAddressItReadFrom(t *testing.T) {
	t.Parallel()

	empty := &terrainSource{openContainers(t, synth.Archive(nil))}
	if _, err := game.LoadProjectiles(empty); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an install with no projectile registry gave %v, want a not-exist error", err)
	}
	if _, err := game.LoadProjectiles(nil); err == nil {
		t.Error("a nil source loaded without error")
	}
}

func TestTwoRowsNamingOneSheetShareItsFrames(t *testing.T) {
	t.Parallel()

	src := projArchive(t,
		[][]synth.RegNode{
			{projNode("ID", 6), projFile("catap2\\sprites"), projNode("Phases", 1),
				projNode("RotationPhases", 1), projNode("A16", 1)},
			{projNode("ID", 7), projFile("catap2\\sprites"), projNode("Phases", 1),
				projNode("RotationPhases", 1), projNode("A16", 1)},
		},
		map[string][]byte{"projectiles/catap2/sprites.16a": oneFrameSheet()},
	)
	set, err := game.LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	a, b := set.Sheet(6), set.Sheet(7)
	if a == nil || b == nil {
		t.Fatal("one of the two rows naming one file built no sheet")
	}
	if a.Frames[0] != b.Frames[0] {
		t.Error("two rows naming one file hold two copies of its frames")
	}
}
