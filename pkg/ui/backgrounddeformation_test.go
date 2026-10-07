package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/render/refraction"
	"againrom/pkg/render/terrain"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestBackgroundDeformationAdmittedWithoutSheet(t *testing.T) {
	v := commandViewer(t)
	v.cam.X, v.cam.Y = 0, 0
	v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4),
		Pos: image.Pt(4*ShotScale, 4*ShotScale), Effect: SpellBackgroundDeformation}})
	if placements := v.spellArtPlacements(); len(placements) != 1 {
		t.Fatalf("sprite-free background deformation admitted %d placements, want 1", len(placements))
	}
}

type deformationTarget struct {
	*spellPassTarget
	gpu     *ebiten.Image
	centers []image.Point
}

func (r *deformationTarget) DrawBackgroundDeformation(d backgroundDeformationDraw) {
	r.calls = append(r.calls, "deformation")
	r.centers = append(r.centers, d.Center)
	if r.gpu == nil {
		refraction.Apply(r.pixels, d.Mapping, d.Center, d.Clip)
	} else {
		drawBackgroundDeformationGPU(r.gpu, d)
	}
}

func (r *deformationTarget) DrawTriangles(vertices []ebiten.Vertex, _ []uint16, img *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	var source *image.RGBA
	for key, texture := range r.v.cache {
		if texture == img {
			src := r.v.set.Slot(key.slot).SubCell(key.sub)
			if src == nil || key.dirt != 0 {
				r.t.Fatal("terrain witness selected absent/impassable tile")
			}
			source = paddedCellPixels(cellPixels(src, key.tint))
			break
		}
	}
	if source == nil || len(vertices) != 4 || op.Filter != ebiten.FilterNearest {
		r.t.Fatal("unidentified production terrain submission")
	}
	a, b, c := vertices[0], vertices[1], vertices[2]
	for _, v := range vertices {
		if v.ColorR != 1 || v.ColorG != 1 || v.ColorB != 1 || v.ColorA != 1 {
			r.t.Fatal("terrain witness requires unshaded vertices", v)
		}
	}
	x0, x1 := max(0, int(math.Ceil(float64(a.DstX)-0.5))), min(r.pixels.Rect.Max.X, int(math.Ceil(float64(b.DstX)-0.5)))
	y0, y1 := max(0, int(math.Ceil(float64(a.DstY)-0.5))), min(r.pixels.Rect.Max.Y, int(math.Ceil(float64(c.DstY)-0.5)))
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			u, w := (float32(x)+0.5-a.DstX)/(b.DstX-a.DstX), (float32(y)+0.5-a.DstY)/(c.DstY-a.DstY)
			sx, sy := a.SrcX+u*(b.SrcX-a.SrcX)+w*(c.SrcX-a.SrcX), a.SrcY+u*(b.SrcY-a.SrcY)+w*(c.SrcY-a.SrcY)
			p := image.Pt(int(math.Floor(float64(sx))), int(math.Floor(float64(sy))))
			if !p.In(source.Rect) {
				r.t.Fatal("terrain source outside submitted texture", p)
			}
			r.pixels.SetRGBA(x, y, source.RGBAAt(p.X, p.Y))
		}
	}
}

func deformationArtifacts(t *testing.T, label string, before, after *image.RGBA, center image.Point) {
	t.Helper()
	dir := os.Getenv("AGAINROM_DEFORMATION_ARTIFACTS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name   string
		pixels *image.RGBA
	}{{"before", before}, {"after", after}} {
		crop := image.Rect(center.X-32, center.Y-32, center.X+32, center.Y+32).Intersect(item.pixels.Rect)
		out := image.NewRGBA(image.Rect(0, 0, crop.Dx()*4, crop.Dy()*4))
		for y := 0; y < out.Rect.Dy(); y++ {
			for x := 0; x < out.Rect.Dx(); x++ {
				out.SetRGBA(x, y, item.pixels.RGBAAt(crop.Min.X+x/4, crop.Min.Y+y/4))
			}
		}
		file, err := os.Create(filepath.Join(dir, label+"-"+item.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, out); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func deformationCoordinates(rect image.Rectangle) *image.RGBA {
	pixels := image.NewRGBA(rect)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			pixels.SetRGBA(x, y, color.RGBA{uint8(x), uint8(y), uint8(x ^ y), 255})
		}
	}
	return pixels
}

func deformationClone(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	draw.Draw(dst, dst.Rect, src, src.Rect.Min, draw.Src)
	return dst
}

func deformationOracle(dst *image.RGBA, m *refraction.Map, center image.Point, clip image.Rectangle) {
	for x := m.Size - 1; x >= 0; x-- {
		for y := m.Size - 1; y >= 0; y-- {
			p := m.Offsets[x*m.Size+y]
			for _, copy := range [][4]int{{x, y, p.X, p.Y}, {x, -y, p.X, -p.Y}, {-x, y, -p.X, p.Y}, {-x, -y, -p.X, -p.Y}} {
				d, s := center.Add(image.Pt(copy[0], copy[1])), center.Add(image.Pt(copy[2], copy[3]))
				if d.In(clip) && s.In(clip) {
					dst.SetRGBA(d.X, d.Y, dst.RGBAAt(s.X, s.Y))
				}
			}
		}
	}
}

func recordDeformationGPU(t *testing.T, r *deformationTarget) {
	t.Helper()
	oldCopy, oldSubmit := copyDeformationSnapshot, submitDeformationPixels
	t.Cleanup(func() { copyDeformationSnapshot, submitDeformationPixels = oldCopy, oldSubmit })
	var texture *ebiten.Image
	var pixels *image.RGBA
	copied := false
	copyDeformationSnapshot = func(snapshot, target *ebiten.Image, region image.Rectangle) {
		if target != r.gpu || region.Intersect(image.Rect(0, 0, r.v.cam.ViewW, r.v.cam.ViewH)) != region {
			t.Fatal("snapshot sampled outside the map buffer", region)
		}
		if snapshot == target || snapshot.Bounds().Size() != region.Size() {
			t.Fatal("snapshot reads its draw destination or has wrong size")
		}
		texture, pixels = snapshot, image.NewRGBA(image.Rect(0, 0, region.Dx(), region.Dy()))
		draw.Draw(pixels, pixels.Rect, r.pixels, region.Min, draw.Src)
		copied = true
	}
	submitDeformationPixels = func(target, snapshot *ebiten.Image, vertices []ebiten.Vertex, indices []uint16, op *ebiten.DrawTrianglesOptions) {
		if !copied || target != r.gpu || texture != snapshot || op.Filter != ebiten.FilterNearest || op.Blend != ebiten.BlendCopy {
			t.Fatal("GPU submission did not use its immutable snapshot with nearest/copy")
		}
		if len(vertices)%4 != 0 || len(indices)*2 != len(vertices)*3 {
			t.Fatal("GPU pixel quad topology")
		}
		clip := image.Rect(0, 0, r.v.cam.ViewW, r.v.cam.ViewH)
		for i := 0; i < len(vertices); i += 4 {
			v := vertices[i]
			d, s := image.Pt(int(v.DstX), int(v.DstY)), image.Pt(int(v.SrcX), int(v.SrcY))
			if float32(d.X) != v.DstX || float32(d.Y) != v.DstY || !d.In(clip) || !s.In(pixels.Rect) {
				t.Fatal("GPU source/destination pixel outside clip", d, s)
			}
			for k, c := range [4]image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				got := vertices[i+k]
				if got.DstX != v.DstX+float32(c.X) || got.DstY != v.DstY+float32(c.Y) ||
					got.SrcX != v.SrcX+float32(c.X) || got.SrcY != v.SrcY+float32(c.Y) ||
					got.ColorR != 1 || got.ColorG != 1 || got.ColorB != 1 || got.ColorA != 1 {
					t.Fatal("GPU quad changed source pixels", got)
				}
			}
			for k, want := range [6]uint16{0, 1, 2, 1, 3, 2} {
				if indices[i/4*6+k] != uint16(i)+want {
					t.Fatal("GPU index order")
				}
			}
			r.pixels.SetRGBA(d.X, d.Y, pixels.RGBAAt(s.X, s.Y))
		}
	}
}

func TestBackgroundDeformationDrawCopiesSourcesThroughCPUAndGPUMapBuffers(t *testing.T) {
	for _, gpu := range []bool{false, true} {
		t.Run(map[bool]string{false: "cpu", true: "gpu-submission"}[gpu], func(t *testing.T) {
			v, base := newSpellPassFixture(t)
			v.SetEntities(nil)
			v.cam.ViewW, v.cam.ViewH = 200, 180
			r := &deformationTarget{spellPassTarget: base}
			if gpu {
				r.gpu = ebiten.NewImage(256, 256)
				t.Cleanup(r.gpu.Dispose)
				recordDeformationGPU(t, r)
			}
			for _, m := range []*refraction.Map{refraction.Picture7Map(), {Size: 2, Offsets: []image.Point{{0, 0}, {1, 0}, {0, 0}, {1, 0}}}} {
				for _, center := range []image.Point{{144, 144}, {0, 0}, {199, 179}, {-5, 90}, {200, 90}, {90, -2}} {
					r.pixels = deformationCoordinates(image.Rect(0, 0, 256, 256))
					before, expected := deformationClone(r.pixels), deformationClone(r.pixels)
					v.cam.X, v.cam.Y = float64(144-center.X), float64(144-center.Y)
					v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: image.Pt(4*ShotScale, 4*ShotScale), Effect: SpellBackgroundDeformation, Mapping: m}})
					v.drawArt(r)
					deformationOracle(expected, m, center, image.Rect(0, 0, 200, 180))
					if !bytes.Equal(r.pixels.Pix, expected.Pix) {
						t.Fatalf("center%v size%d differs from source-pixel oracle", center, m.Size)
					}
					if !gpu && m.Size == 19 && center == image.Pt(144, 144) {
						deformationArtifacts(t, "synthetic-coordinates", before, r.pixels, center)
					}
					for y := 0; y < 256; y++ {
						for x := 0; x < 256; x++ {
							if (x >= 200 || y >= 180) && before.RGBAAt(x, y) != r.pixels.RGBAAt(x, y) {
								t.Fatal("HUD pixel changed", x, y)
							}
						}
					}
				}
			}
		})
	}
}

func TestBackgroundDeformationDrawKeepsProjectileOrderAndIgnoresFrameFields(t *testing.T) {
	v, base := newSpellPassFixture(t)
	v.SetEntities(nil)
	r := &deformationTarget{spellPassTarget: base}
	first, last, cloud := solidPassSheet(color.RGBA{220, 40, 20, 255}), solidPassSheet(color.RGBA{20, 120, 240, 255}), solidPassSheet(color.RGBA{10, 240, 20, 255})
	r.effects[first.Frames[0]], r.effects[last.Frames[0]], r.effects[cloud.Frames[0]] = "first", "last", "cloud"
	pos := image.Pt(4*ShotScale, 4*ShotScale)
	bolt := func(sheet *terrain.EffectSheet, pass SpellPass) SpellBolt {
		return SpellBolt{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: pos, Sheet: sheet, Pass: pass}
	}
	v.SetSpellBolts([]SpellBolt{bolt(first, SpellProjectiles), {Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: pos, Effect: SpellBackgroundDeformation}, bolt(last, SpellProjectiles), bolt(cloud, SpellOverlayB)})
	r.pixels = deformationCoordinates(image.Rect(0, 0, 256, 256))
	v.drawArt(r)
	if !reflect.DeepEqual(r.calls, []string{"first", "deformation", "last", "cloud"}) {
		t.Fatal("projectile composition order", r.calls)
	}
	if r.pixels.RGBAAt(127, 144) != (color.RGBA{220, 40, 20, 255}) || r.pixels.RGBAAt(144, 144) != (color.RGBA{10, 240, 20, 255}) {
		t.Fatal("deformation did not read earlier projectile or rewrote following cloud")
	}
	var previous *image.RGBA
	for _, phase := range []int{0, 7, 100} {
		r.pixels = deformationCoordinates(image.Rect(0, 0, 256, 256))
		v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4), Pos: pos, Effect: SpellBackgroundDeformation, Phase: phase, Frame: phase, Mirror: phase != 0, Sheet: uiSheet(1, 1, 1, phase, phase)}})
		v.drawArt(r)
		if previous != nil && !bytes.Equal(previous.Pix, r.pixels.Pix) {
			t.Fatal("phase/facing/registry extents changed callback output")
		}
		previous = deformationClone(r.pixels)
	}
	before := deformationCoordinates(image.Rect(0, 0, 256, 256))
	r.pixels = deformationClone(before)
	v.SetSpellBolts(nil)
	v.drawArt(r)
	if !bytes.Equal(before.Pix, r.pixels.Pix) {
		t.Fatal("no-effect control changed background")
	}
}

func TestBackgroundDeformationUsesTheProjectileFogGate(t *testing.T) {
	v := commandViewer(t)
	v.cam.X, v.cam.Y = 0, 0
	v.SetLocalOwner(1)
	for _, tc := range []struct {
		fog   byte
		owner uint32
		want  int
	}{{FogUnseen, 2, 0}, {FogExplored, 2, 0}, {FogVisible, 2, 1}, {FogUnseen, 1, 1}} {
		plane := make([]byte, 64)
		plane[4*8+4] = tc.fog
		v.SetFog(plane, 8, 8)
		v.SetSpellBolts([]SpellBolt{{Cell: image.Pt(4, 4), To: image.Pt(4, 4),
			Pos: image.Pt(4*ShotScale, 4*ShotScale), Owner: tc.owner, Effect: SpellBackgroundDeformation}})
		if got := len(v.spellArtPlacements()); got != tc.want {
			t.Fatalf("fog %d owner %d admitted %d, want %d", tc.fog, tc.owner, got, tc.want)
		}
	}
}
