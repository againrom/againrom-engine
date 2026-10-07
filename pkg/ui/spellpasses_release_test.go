package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr16"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

type passReleaseArt struct {
	sheet *terrain.EffectSheet
	raw   *spr16.SpriteA
	frame int
}

func loadReleasePassArt(t *testing.T) map[string]passReleaseArt {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: spell pass composition needs a lawful install")
	}
	src, err := vfs.Open([]string{filepath.Join(root, "graphics.res")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := src.ReadFile("graphics/projectiles/projectiles.reg")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := reg.Parse(registry)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := data.LoadProjectiles(parsed)
	if err != nil {
		t.Fatal(err)
	}
	art := make(map[string]passReleaseArt)
	for _, item := range []struct {
		name           string
		picture, frame int
	}{
		{"fire", 15, 4}, {"earth", 47, 0}, {"meteor", 51, 8},
		{"freezing", 23, 2}, {"poison", 25, 2},
	} {
		row, ok := rows.ByID(int32(item.picture))
		if !ok {
			t.Fatalf("missing installed picture %d", item.picture)
		}
		payload, err := src.ReadFile("graphics/" + row.SpritePath())
		if err != nil {
			t.Fatal(err)
		}
		raw, err := spr16.DecodeA(payload, true)
		if err != nil || raw == nil || item.frame >= len(raw.Frames) {
			t.Fatalf("picture %d frame %d: decoded=%v error=%v", item.picture, item.frame, raw != nil, err)
		}
		// Internal UI tests cannot import pkg/game. Load the same installed
		// frame type at this seam, without introducing a production API solely
		// for the witness. The expected pixels below read raw palette/coverage,
		// independently of these converted frames and EffectFrame.RGBA.
		f := raw.Frames[item.frame]
		frame := &terrain.EffectFrame{Width: f.Width, Height: f.Height, Pixels: make([]color.RGBA, len(f.Pixels))}
		for i, p := range f.Pixels {
			if !p.Painted {
				continue
			}
			c := raw.Palette[p.Index]
			a := (int(p.Level) + 1) * 255 / 16
			frame.Pixels[i] = color.RGBA{uint8(int(c.R) * a / 255), uint8(int(c.G) * a / 255), uint8(int(c.B) * a / 255), uint8(a)}
		}
		art[item.name] = passReleaseArt{sheet: &terrain.EffectSheet{Frames: []*terrain.EffectFrame{frame}, Phases: 1,
			CenterX: int(row.Width) / 2, CenterY: int(row.Height) / 2}, raw: raw, frame: item.frame}
	}
	return art
}

func TestReleaseSpellPassesComposeInstalledArtAroundShadowsAndBodies(t *testing.T) {
	art := loadReleasePassArt(t)
	for _, cloud := range []string{"freezing", "poison"} {
		for _, category := range []terrain.UnitCategory{terrain.UnitOrdinary, terrain.UnitAlternate, terrain.UnitAir} {
			t.Run(fmt.Sprintf("%s/category%d", cloud, category), func(t *testing.T) {
				v, target := newSpellPassFixture(t)
				for i := range v.entities {
					v.entities[i].DrawCategory = category
				}
				var bolts []SpellBolt
				for _, item := range []struct {
					name string
					pass SpellPass
				}{{"meteor", SpellProjectiles}, {"fire", SpellOverlayA}, {cloud, SpellOverlayB}, {"earth", SpellOverlayA}} {
					a := art[item.name]
					target.effects[a.sheet.Frames[0]] = item.name
					bolts = append(bolts, SpellBolt{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: image.Pt(4*ShotScale, 4*ShotScale), Sheet: a.sheet, Pass: item.pass})
				}
				v.SetSpellBolts(bolts)
				v.drawArt(target)
				want := []string{"fire", "earth", "shadow body1", "shadow body2", "meteor", cloud, "body1", "body2"}
				sequence := []string{"fire", "earth", "shadow", "shadow", "meteor", cloud, "body1", "body2"}
				if category != terrain.UnitAir {
					want = []string{"shadow body1", "body1", "shadow body2", "body2", "fire", "earth", "meteor", cloud}
					sequence = []string{"shadow", "body1", "shadow", "body2", "fire", "earth", "meteor", cloud}
				}
				if !reflect.DeepEqual(target.calls, want) {
					t.Fatalf("production drawArt calls=%v want=%v", target.calls, want)
				}
				checked, changed := 0, 0
				for y := 134; y < 155; y++ {
					for x := 134; x < 155; x++ {
						// Both solid 32x32 fixture units and their near-vertical
						// shadows cover this literal interior. Installed art is
						// sampled from the raw decoded palette at the same cell.
						correct := releasePassPixel(art, x, y, sequence)
						old := releasePassPixel(art, x, y, []string{"shadow", "shadow", "body1", "body2", "meteor", "fire", cloud, "earth"})
						if got := target.pixels.RGBAAt(x, y); got != correct {
							t.Fatalf("installed composition at (%d,%d)=%v want raw oracle=%v", x, y, got, correct)
						}
						checked++
						if old != correct {
							changed++
						}
					}
				}
				if changed == 0 {
					t.Fatal("installed art does not distinguish the old final spell band")
				}
				t.Logf("production drawArt: %d raw-oracle pixels checked; %d differ from the old final spell band", checked, changed)
				if dir := os.Getenv("AGAINROM_SPELL_PASSES_PNG"); dir != "" {
					if err := os.MkdirAll(dir, 0o755); err != nil {
						t.Fatal(err)
					}
					out, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-category%d-passes.png", cloud, category)))
					if err != nil {
						t.Fatal(err)
					}
					if err := png.Encode(out, target.pixels); err != nil {
						out.Close()
						t.Fatal(err)
					}
					if err := out.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

// releasePassPixel is the independent straight-color/raw-coverage oracle.
// Its explicit sequence never reads production tags, placement or draw lists.
func releasePassPixel(art map[string]passReleaseArt, x, y int, sequence []string) color.RGBA {
	c := [3]float64{16, 24, 32}
	for _, name := range sequence {
		var rgb [3]float64
		var alpha float64
		switch name {
		case "shadow":
			// An owner's invisible unit casts one silhouette at the unit-path
			// level, which the fixture sets to 4 of 16.
			alpha = 0.25
		case "body1":
			rgb, alpha = [3]float64{100, 20, 0}, 0.5
		case "body2":
			rgb, alpha = [3]float64{0, 60, 120}, 0.5
		default:
			a := art[name]
			f := a.raw.Frames[a.frame]
			sx, sy := x-144+a.sheet.CenterX, y-144+a.sheet.CenterY
			if sx < 0 || sy < 0 || sx >= f.Width || sy >= f.Height {
				continue
			}
			p := f.Pixels[sy*f.Width+sx]
			if !p.Painted {
				continue
			}
			coverage := (int(p.Level) + 1) * 255 / 16
			pal := a.raw.Palette[p.Index]
			rgb = [3]float64{float64(int(pal.R) * coverage / 255), float64(int(pal.G) * coverage / 255), float64(int(pal.B) * coverage / 255)}
			alpha = float64(coverage) / 255
		}
		for i := range c {
			c[i] = math.Round(rgb[i] + c[i]*(1-alpha))
		}
	}
	return color.RGBA{uint8(c[0]), uint8(c[1]), uint8(c[2]), 255}
}
