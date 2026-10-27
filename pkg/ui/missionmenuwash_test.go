package ui

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// washViewerApp is an App whose mission menu stands over a viewer that draws
// the right column (filler, statistics card, command panel) and the open pack
// bar, under the given menu profile.
func washViewerApp(t *testing.T, profile GameMenuProfile) (*App, *Viewer) {
	t.Helper()
	a := newTestApp(t, appRows(1), okLoader(t))
	a.flow = openMissionMenu(t)
	a.flow.menuFont = chargenTestFont()
	v := columnViewer(t)
	v.SetPackShown(true)
	v.DeferPointer(true)
	a.flow.viewer = v
	a.SetGameMenuProfile(profile)
	a.Layout(MissionFrameW, MissionFrameH)
	return a, v
}

// frameValues reads every cell of the viewer's last frame from its canvas log.
func frameValues(v *Viewer) ([]color.RGBA, []bool) {
	b := v.canvasLog.bounds
	vals := make([]color.RGBA, b.Dx()*b.Dy())
	known := make([]bool, len(vals))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := (y-b.Min.Y)*b.Dx() + x - b.Min.X
			vals[i], known[i] = v.canvasLog.value(len(v.canvasLog.ops), x, y)
		}
	}
	return vals, known
}

// The mission menu's backdrop darkens every layer of the mission frame in the
// second game, and the menu panel is composed after that frame. The first
// game's notice dim keeps its place under the right column and the pack bar.
func TestMissionMenuBackdropDarkensEveryMissionLayer(t *testing.T) {
	wash := color.RGBA{A: 64}
	for _, tc := range []struct {
		name    string
		profile GameMenuProfile
		darkens bool
	}{
		{"second", GameMenuProfile{NativeMissionWindows: true, MissionBackdrop: wash, WindowsCloseMenu: true}, true},
		{"first", GameMenuProfile{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := washViewerApp(t, tc.profile)
			layers := recordColumnLayers(t)
			screen := ebiten.NewImage(MissionFrameW, MissionFrameH)
			defer screen.Dispose()

			a.flow.setMenuUp(false)
			v.Draw(screen)
			down, downKnown := frameValues(v)

			a.flow.setMenuUp(true)
			var order []string
			prevFrame, prevMenu := blitFrameCanvas, blitMenuOverMap
			blitFrameCanvas = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions) {
				order = append(order, "frame")
				prevFrame(dst, img, op)
			}
			blitMenuOverMap = func(dst, img *ebiten.Image, op *ebiten.DrawImageOptions) {
				order = append(order, "menu")
				prevMenu(dst, img, op)
			}
			defer func() { blitFrameCanvas, blitMenuOverMap = prevFrame, prevMenu }()
			a.Draw(screen)
			if fmt.Sprint(order) != "[frame menu]" {
				t.Fatalf("composition order %v, want the mission frame then the menu", order)
			}
			up, upKnown := frameValues(v)

			pack, _, packShown := v.packBarShown()
			if !packShown {
				t.Fatal("pack bar not shown")
			}
			rects := map[string]image.Rectangle{"pack": pack}
			for _, name := range []string{"filler", "missionCard", "controlPanel"} {
				rec, ok := layers[name]
				if !ok {
					t.Fatalf("layer %q was not drawn", name)
				}
				rects[name] = image.Rectangle{Min: rec.origin, Max: rec.origin.Add(rec.size)}
			}
			// The backdrop is one solid source-over rectangle over the whole
			// frame. Its blend with these layers' channels is not an integer,
			// so the log cannot read the result; the test reads the frame under
			// the backdrop instead and asserts nothing is painted above it.
			ops := v.canvasLog.ops
			washAt := -1
			for i, op := range ops {
				if op.kind == pixelOver && op.pic == nil && op.layer == nil && op.solid == wash && op.rect == v.canvasLog.bounds {
					if washAt >= 0 {
						t.Fatalf("two whole-frame backdrops, ops %d and %d", washAt, i)
					}
					washAt = i
				}
			}
			if tc.darkens != (washAt >= 0) {
				t.Fatalf("whole-frame backdrop at op %d, want darkened %v", washAt, tc.darkens)
			}
			w := v.canvasLog.bounds.Dx()
			for name, r := range rects {
				checked := 0
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						i := y*w + x
						if !downKnown[i] {
							continue
						}
						got, known := up[i], upKnown[i]
						if tc.darkens {
							got, known = v.canvasLog.value(washAt, x, y)
							p := image.Pt(x, y)
							for j := washAt + 1; j < len(ops); j++ {
								if !ops[j].covers(p) {
									continue
								}
								if s, ok := ops[j].source(x, y); ok && s.A == 0 {
									continue
								}
								t.Fatalf("%s pixel %v is painted by op %d after the backdrop at op %d", name, p, j, washAt)
							}
						}
						if !known || got != down[i] {
							t.Fatalf("%s pixel (%d,%d) under the menu %v (known %v), with the menu down %v", name, x, y, got, known, down[i])
						}
						checked++
					}
				}
				if checked == 0 {
					t.Fatalf("%s %v: no known pixel", name, r)
				}
			}

			for _, op := range a.menuLog.ops {
				if op.kind != pixelUnknown && op.pic == nil && op.layer == nil && op.solid == wash {
					t.Fatal("the menu canvas holds the mission backdrop")
				}
			}
			if !tc.darkens {
				h := sha256.New()
				for i := range up {
					c := up[i]
					var k byte
					if upKnown[i] {
						k = 1
					}
					h.Write([]byte{c.R, c.G, c.B, c.A, k})
				}
				binary.Write(h, binary.LittleEndian, int64(len(v.canvasLog.ops)))
				t.Logf("first-game frame with the menu up: %x", h.Sum(nil))
			}
		})
	}
}
