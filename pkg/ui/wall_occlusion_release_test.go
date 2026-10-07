package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// The installed Fur1 tree is the positive Earth discriminator from mission101.
// Reusing its art for Fire is a declared overlap control; the authored tick24
// Fire trap has zero painted static overlaps in the bounded scene diagnosis.
func releaseOcclusionTree(t *testing.T) (*terrain.StaticClass, *spr256.Sprite) {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: wall occlusion needs installed art")
	}
	src, err := vfs.Open([]string{filepath.Join(root, "graphics.res")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := src.ReadFile("graphics/objects/objects.reg")
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadObjectClasses(r)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := classes.ByCode(10)
	if !ok {
		t.Fatal("installed object byte10 missing")
	}
	raw, err = src.ReadFile("graphics/" + c.SpritePath())
	if err != nil {
		t.Fatal(err)
	}
	sheet, err := spr256.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !sheet.HasPalette || len(sheet.Palette) != 256 || c.Index < 0 || int(c.Index) >= len(sheet.Frames) {
		t.Fatal("installed tree sheet refused")
	}
	var frames []*terrain.StaticFrame
	for _, f := range sheet.Frames {
		frame := &terrain.StaticFrame{Width: f.Width, Height: f.Height, Pixels: make([]terrain.StaticPixel, len(f.Pixels))}
		for i, p := range f.Pixels {
			frame.Pixels[i] = terrain.StaticPixel{Index: p.Index, Opaque: p.Opaque}
		}
		for i, p := range sheet.Palette {
			frame.Palette[i] = color.RGBA{p.R, p.G, p.B, 255}
		}
		frames = append(frames, frame)
	}
	return &terrain.StaticClass{Width: int(c.Width), Height: int(c.Height), CenterX: int(c.CenterX), CenterY: int(c.CenterY), Index: int(c.Index), Frame: frames[c.Index], Frames: frames}, sheet
}

func occlusionViewer(t *testing.T, tree *terrain.StaticClass, cell image.Point) (*Viewer, *spellPassTarget) {
	t.Helper()
	g := terrain.Grid{Width: 12, Height: 12, Tiles: make([]uint16, 144), Overlay: make([]uint8, 144)}
	g.Overlay[cell.Y*12+cell.X] = 10
	statics := &terrain.StaticSet{}
	statics.Classes[10] = tree
	v, err := NewViewerWithStatics("occlusion", g, &terrain.Tileset{}, statics, true, false, terrain.AnimGateAll, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	view := image.Rect(0, 0, 256, 256)
	v.cam.X, v.cam.Y, v.cam.Zoom, v.cam.ViewW, v.cam.ViewH = 0, 0, 1, view.Dx(), view.Dy()
	v.unshaded = true
	v.sun.Theta, v.sun.ShroudObject, v.sun.ShroudUnit = 0.05, 8, 8
	r := &spellPassTarget{t: t, v: v, pixels: image.NewRGBA(view), effects: make(map[*terrain.EffectFrame]string), bodies: map[*terrain.StaticFrame]string{tree.Frame: "tree"}}
	for y := 0; y < view.Dy(); y++ {
		for x := 0; x < view.Dx(); x++ {
			r.pixels.SetRGBA(x, y, color.RGBA{16, 24, 32, 255})
		}
	}
	return v, r
}

func overOcclusion(d, s color.RGBA) color.RGBA {
	a := 1 - float64(s.A)/255
	return color.RGBA{uint8(math.Round(float64(s.R) + float64(d.R)*a)), uint8(math.Round(float64(s.G) + float64(d.G)*a)), uint8(math.Round(float64(s.B) + float64(d.B)*a)), uint8(math.Round(float64(s.A) + float64(d.A)*a))}
}

func TestReleaseWallsOccludeInstalledTreeWithinTheirCell(t *testing.T) {
	art := loadReleasePassArt(t)
	tree, rawTree := releaseOcclusionTree(t)
	for _, name := range []string{"earth", "fire"} {
		for _, clipped := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/clipped%v", name, clipped), func(t *testing.T) {
				v, observed := occlusionViewer(t, tree, image.Pt(4, 4))
				_, base := occlusionViewer(t, tree, image.Pt(4, 4))
				base.v = v
				if clipped {
					v.cam.X, v.cam.Y, v.cam.ViewW, v.cam.ViewH = 132, 112, 28, 40
					observed.pixels, base.pixels = image.NewRGBA(image.Rect(0, 0, 28, 40)), image.NewRGBA(image.Rect(0, 0, 28, 40))
				}
				// Capture the same admitted tree/shadow backdrop with no wall.
				// The raw wall oracle below never reads drawArt's pass ordering.
				v.drawArt(base)
				a := art[name]
				observed.effects[a.sheet.Frames[0]] = name
				v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: image.Pt(4*ShotScale, 4*ShotScale), Sheet: a.sheet, Pass: SpellOverlayA}})
				v.drawArt(observed)
				frame := a.raw.Frames[a.frame]
				painted, holes, overlap, changed := 0, 0, 0, 0
				for y := observed.pixels.Rect.Min.Y; y < observed.pixels.Rect.Max.Y; y++ {
					for x := observed.pixels.Rect.Min.X; x < observed.pixels.Rect.Max.X; x++ {
						wx, wy := x+int(v.cam.X), y+int(v.cam.Y)
						sx, sy := wx-144+a.sheet.CenterX, wy-144+a.sheet.CenterY
						want := base.pixels.RGBAAt(x, y)
						if sx >= 0 && sx < frame.Width && sy >= 0 && sy < frame.Height {
							p := frame.Pixels[sy*frame.Width+sx]
							if p.Painted {
								painted++
								alpha := (int(p.Level) + 1) * 255 / 16
								c := a.raw.Palette[p.Index]
								wall := color.RGBA{uint8(int(c.R) * alpha / 255), uint8(int(c.G) * alpha / 255), uint8(int(c.B) * alpha / 255), uint8(alpha)}
								want = overOcclusion(want, wall)
								// Installed tree dimensions/anchor are explicit in
								// the diagnosis. No sprite placement helper is an oracle.
								tx, ty := wx-80, wy-48
								tf := rawTree.Frames[tree.Index]
								if tx >= 0 && tx < tf.Width && ty >= 0 && ty < tf.Height && tf.Pixels[ty*tf.Width+tx].Opaque {
									overlap++
									if want != base.pixels.RGBAAt(x, y) {
										changed++
									}
								}
							} else {
								holes++
							}
						}
						if got := observed.pixels.RGBAAt(x, y); got != want {
							t.Fatalf("(%d,%d): combined=%v raw wall over backdrop=%v", wx, wy, got, want)
						}
					}
				}
				if painted == 0 || holes == 0 || overlap == 0 || changed == 0 {
					t.Fatalf("non-discriminating fixture: painted=%d holes=%d overlap=%d changed=%d", painted, holes, overlap, changed)
				}
				t.Logf("%s: %d compared pixels, wall painted=%d transparent=%d tree overlap=%d changed=%d, active tree shadow, clipped=%v", name, observed.pixels.Rect.Dx()*observed.pixels.Rect.Dy(), painted, holes, overlap, changed, clipped)
			})
		}
	}
}

func TestReleaseWallsFollowInstalledCellDepth(t *testing.T) {
	art := loadReleasePassArt(t)
	tree, rawTree := releaseOcclusionTree(t)
	for _, name := range []string{"earth", "fire"} {
		for _, tc := range []struct {
			name      string
			cell      image.Point
			wallFront bool
		}{
			{"earlier-row", image.Pt(4, 3), true}, {"later-row", image.Pt(4, 5), false},
			{"earlier-column", image.Pt(5, 4), true}, {"later-column", image.Pt(3, 4), false},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				v, target := occlusionViewer(t, tree, tc.cell)
				a := art[name]
				target.effects[a.sheet.Frames[0]] = name
				v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: image.Pt(4*ShotScale, 4*ShotScale), Sheet: a.sheet, Pass: SpellOverlayA}})
				v.drawArt(target)
				wf, tf := a.raw.Frames[a.frame], rawTree.Frames[tree.Index]
				checked := 0
				for y := 0; y < 256; y++ {
					for x := 0; x < 256; x++ {
						sx, sy := x-144+a.sheet.CenterX, y-144+a.sheet.CenterY
						tx, ty := x-(tc.cell.X*32-48), y-(tc.cell.Y*32-80)
						if sx < 0 || sx >= wf.Width || sy < 0 || sy >= wf.Height || tx < 0 || tx >= tf.Width || ty < 0 || ty >= tf.Height {
							continue
						}
						wp, tp := wf.Pixels[sy*wf.Width+sx], tf.Pixels[ty*tf.Width+tx]
						if !wp.Painted || !tp.Opaque {
							continue
						}
						treeRGB := rawTree.Palette[tp.Index]
						want := color.RGBA{treeRGB.R, treeRGB.G, treeRGB.B, 255}
						if tc.wallFront {
							wallRGB := a.raw.Palette[wp.Index]
							alpha := (int(wp.Level) + 1) * 255 / 16
							want = overOcclusion(want, color.RGBA{uint8(int(wallRGB.R) * alpha / 255), uint8(int(wallRGB.G) * alpha / 255), uint8(int(wallRGB.B) * alpha / 255), uint8(alpha)})
						}
						if got := target.pixels.RGBAAt(x, y); got != want {
							t.Fatalf("(%d,%d)=%v want raw cell-order pixel=%v", x, y, got, want)
						}
						checked++
					}
				}
				if checked == 0 {
					t.Fatal("installed control has no painted overlap")
				}
				t.Logf("raw installed cell-depth oracle: %d painted overlap pixels", checked)
			})
		}
	}
}
