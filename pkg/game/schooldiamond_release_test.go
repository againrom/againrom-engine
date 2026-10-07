package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/ui"
)

func TestReleaseSchoolDiamondFramesMatchInstalledPixels(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("production school art: %v", f.TownSchoolArt.Err())
	}
	v := ui.TownSurfaceView{Kind: ui.TownSurfaceSchool, SchoolArt: f.TownSchoolArt.Value(),
		SchoolClass: -1, HoverCell: -1, SchoolDiamondActive: true}
	for frame := 0; frame < 9; frame++ {
		// Read the literal claim address independently of LoadTownSchoolArt.
		// The expected pixels do not come from its array or RGBA conversion.
		name := fmt.Sprintf("graphics/interface/training/diamond/on%04d.bmp", frame)
		raw, err := f.Archives.Containers.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want, err := bmp.Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		if want.Width != 80 || want.Height != 76 {
			t.Fatalf("%s: %dx%d, want 80x76", name, want.Width, want.Height)
		}
		v.SchoolDiamondFrame = frame
		got := ui.ComposeTownSurface(v)
		for y := 0; y < 76; y++ {
			for x := 0; x < 80; x++ {
				c := want.At(x, y)
				if pix := got.RGBAAt(200+x, 60+y); pix != (color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}) {
					t.Fatalf("%s: source %d,%d composed as %v, want %v opaque at (200,60)", name, x, y, pix, c)
				}
			}
		}
	}
	t.Log("school diamond: nine installed frames, 54,720 pixels matched at (200,60), opaque")
}

func TestReleaseSchoolDiamondTrainPaintLifecycle(t *testing.T) {
	f := releaseFront(t)
	if f.TownSchoolArt.Value() == nil {
		t.Fatalf("production school art: %v", f.TownSchoolArt.Err())
	}
	f.Carried = f.NextParty()
	f.Town.gold = 100000
	s := f.TownScreen().(*townScreen)
	s.Choose(2)
	for i := 0; s.room == roomTalk && i < 64; i++ {
		s.AdvanceTownDialogue()
	}
	if s.room != roomSchool || len(f.Carried) == 0 {
		t.Fatal("production school entry has no training subject")
	}
	s.CloseTip()
	before := paintDiamondSchool(t, s)
	writeSchoolDiamondWitness(t, f, "idle", before)
	cell := 0
	if f.Carried[0].Mage {
		cell = 5
	}
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: cell}, false)
	gold := f.Town.Gold()
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false)
	if f.Town.Gold() >= gold {
		t.Fatal("production Train did not spend its quoted price")
	}
	changed := 0
	for i, frame := range []int{1, 2, 3, 4, 5, 6, 7, 8, 7, 6, 5, 4, 3, 2, 1, 0} {
		pix := paintDiamondSchool(t, s)
		if s.TownSurface().SchoolDiamondActive != (i < 15) || s.TownSurface().SchoolDiamondFrame != frame {
			t.Fatalf("paint %d did not publish phase %d / active=%v", i+1, frame, i < 15)
		}
		var expected *bmp.Image
		if i < 15 {
			raw, err := f.Archives.Containers.ReadFile(fmt.Sprintf("graphics/interface/training/diamond/on%04d.bmp", frame))
			if err != nil {
				t.Fatal(err)
			}
			expected, err = bmp.Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		for y := 0; y < 76; y++ {
			for x := 0; x < 80; x++ {
				want := before.RGBAAt(x+200, y+60)
				if expected != nil {
					c := expected.At(x, y)
					want = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
				}
				if got := pix.RGBAAt(x+200, y+60); got != want {
					t.Fatalf("paint %d pixel %d,%d: %v, want %v", i+1, x+200, y+60, got, want)
				}
				if i == 7 && want != before.RGBAAt(x+200, y+60) {
					changed++
				}
			}
		}
		if i == 7 {
			writeSchoolDiamondWitness(t, f, "active", pix)
		} else if i == 15 {
			writeSchoolDiamondWitness(t, f, "completed", pix)
		}
	}
	if changed == 0 {
		t.Fatal("the installed peak frame changes no school pixel")
	}
	t.Logf("school diamond: admitted Train, 16 paints, 15 eligible frames; peak changes %d of 6080 pixels; completion restores all 6080", changed)
}

func writeSchoolDiamondWitness(t *testing.T, f *FrontEnd, state string, pix *image.RGBA) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SCHOOL_DIAMOND_PNG")
	if dir == "" {
		return
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	// An optional owner-review capture must never write inside its install.
	rel, err := filepath.Rel(f.Archives.Root, dir)
	if err != nil || rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("school capture directory must be outside the install: %q", dir)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(dir, "school-"+state+".png"))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(out, pix)
	closeErr := out.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	t.Logf("school witness: %s", out.Name())
}
