package game

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// Both games paint the mission minimap from their own install's
// interface/MiniMapData.bmp and the owner tables of human.pal: one paint
// builder, the screen manifest naming the lookup for each game. A manifest
// without the lookup leaves the second game on the symbolic minimap, which
// draws no structure mark (R2-ENGINE-593).
func TestBothGamesPaintTheMinimapFromTheirLookup(t *testing.T) {
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	lookup := synthBMP(52, 14, color.RGBA{R: 40, G: 90, B: 20, A: 255})
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: "interface/minimapdata.bmp", Data: lookup}}), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	units := &terrain.UnitSet{HasOwnerPalettes: true}
	units.OwnerPalettes[8][minimapShadeEntry] = color.RGBA{R: 33, G: 70, B: 188, A: 255}
	units.OwnerPalettes[3][minimapShadeEntry] = color.RGBA{R: 228, G: 114, B: 87, A: 255}
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		in := &InstallResources{
			Archives: &Archives{Containers: fsys, Base: base.Match{Profile: base.Profile{Game: g}}},
			Units:    units,
		}
		p := missionMinimapPaint(in)
		if p == nil || p.Colours == nil {
			t.Fatalf("%s: no minimap paint from the install's lookup", g)
		}
		if size := p.Colours.Bounds().Size(); size.X != 52 || size.Y != 14 {
			t.Errorf("%s: lookup %v, want 52x14", g, size)
		}
		if c := p.Colours.RGBAAt(3, 5); c != (color.RGBA{R: 40, G: 90, B: 20, A: 255}) {
			t.Errorf("%s: lookup sample %v", g, c)
		}
		if p.OwnerShades[8] != (color.RGBA{R: 33, G: 70, B: 188, A: 255}) || p.OwnerShades[3] != (color.RGBA{R: 228, G: 114, B: 87, A: 255}) {
			t.Errorf("%s: owner shades 8 and 3 = %v %v, want entry %d of tables 8 and 3", g, p.OwnerShades[8], p.OwnerShades[3], minimapShadeEntry)
		}
	}
}
