package ui

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/render/refraction"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestReleaseBackgroundDeformationCopiesInstalledTerrain(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: deformation witness requires installed terrain")
	}
	source, err := vfs.Open([]string{filepath.Join(root, "graphics.res")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tiles := terrain.LoadTileset(source)
	if tiles.Slot(0).SubCell(0) == nil {
		t.Fatal("installed terrain slot0/sub0 absent")
	}
	for _, gpu := range []bool{false, true} {
		t.Run(map[bool]string{false: "cpu", true: "gpu-submission"}[gpu], func(t *testing.T) {
			v, err := NewViewer("deformation", grid(8, 8), tiles)
			if err != nil {
				t.Fatal(err)
			}
			v.SetUnshaded(true)
			v.cam.X, v.cam.Y, v.cam.ViewW, v.cam.ViewH = 0, 0, 256, 256
			r := &deformationTarget{spellPassTarget: &spellPassTarget{t: t, v: v, pixels: image.NewRGBA(image.Rect(0, 0, 256, 256))}}
			if gpu {
				r.gpu = ebiten.NewImage(256, 256)
				t.Cleanup(r.gpu.Dispose)
				recordDeformationGPU(t, r)
			}
			v.drawFlat(r)
			before := deformationClone(r.pixels)
			var previous *image.RGBA
			for _, pos := range []image.Point{{4 * ShotScale, 4 * ShotScale}, {4*ShotScale + ShotScale/2, 4 * ShotScale}} {
				r.pixels = deformationClone(before)
				v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: pos, Effect: SpellBackgroundDeformation}})
				v.drawArt(r)
				center := image.Pt(pos.X*32/256+16, pos.Y*32/256+16)
				expected := deformationClone(before)
				deformationOracle(expected, refraction.Picture7Map(), center, image.Rect(0, 0, 256, 256))
				if !bytes.Equal(r.pixels.Pix, expected.Pix) {
					t.Fatal("production terrain/effect submissions differ from source-copy oracle")
				}
				changed := 0
				for y := 0; y < 256; y++ {
					for x := 0; x < 256; x++ {
						if r.pixels.RGBAAt(x, y) != before.RGBAAt(x, y) {
							changed++
						}
					}
				}
				if changed == 0 {
					t.Fatal("installed terrain is a flat negative control")
				}
				if previous != nil && bytes.Equal(previous.Pix, r.pixels.Pix) {
					t.Fatal("moving effect did not move its pixel footprint")
				}
				if !gpu && previous == nil {
					deformationArtifacts(t, "installed-"+filepath.Base(root), before, r.pixels, center)
				}
				previous = deformationClone(r.pixels)
				t.Logf("installed %s, center%v: %d source pixels changed through production terrain/art calls", filepath.Base(root), center, changed)
			}
			v.SetSpellBolts(nil)
			r.pixels = deformationClone(before)
			v.drawArt(r)
			if !bytes.Equal(before.Pix, r.pixels.Pix) {
				t.Fatal("installed no-effect control changed terrain")
			}
		})
	}
}
