package ui

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"github.com/hajimehoshi/ebiten/v2"
)

// painterPlanPieceSizes are two nine-piece size sets: the measured lm.256
// window (DLG-PANEL-035) and a smaller set whose edges leave remainders.
var painterPlanPieceSizes = [2][9]image.Point{
	{{96, 64}, {48, 48}, {96, 48}, {48, 48}, {48, 64}, {48, 64}, {48, 48}, {96, 48}, {48, 48}},
	{{20, 14}, {12, 10}, {18, 10}, {12, 10}, {12, 15}, {12, 15}, {12, 10}, {18, 10}, {12, 10}},
}

// painterPlanFrame is nine patterned pieces of sizes with holes, so a shadow
// mask differs from its rectangle.
func painterPlanFrame(sizes [9]image.Point, seed int) *DialogFrame {
	art := &DialogFrame{}
	for i, s := range sizes {
		p := image.NewRGBA(image.Rectangle{Max: s})
		for y := 0; y < s.Y; y++ {
			for x := 0; x < s.X; x++ {
				if (x*3+y*5+i+seed)%7 == 0 {
					continue
				}
				p.SetRGBA(x, y, color.RGBA{uint8(40 + 20*i + x), uint8(200 - 9*i + y), uint8(seed*31 + x + y), 255})
			}
		}
		art.Pieces[i] = p
	}
	return art
}

// painterPlanPicture is an opaque gradient with a transparent band, so a
// shadow falls on both a picture and on empty pixels.
func painterPlanPicture(r image.Rectangle, seed int) *image.RGBA {
	p := image.NewRGBA(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if (x/37+y/29+seed)%5 == 0 {
				continue
			}
			p.SetRGBA(x, y, color.RGBA{uint8(x*7 + seed), uint8(y*3 + seed*11), uint8(x ^ y), 255})
		}
	}
	return p
}

func newPainterPlanTexture(p *image.RGBA) *ebiten.Image { return ebiten.NewImageFromImage(p) }

type painterDigest struct{ lines []string }

func (d *painterDigest) add(name string, parts ...[]byte) {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p)
	}
	d.lines = append(d.lines, fmt.Sprintf("%s %x", name, h.Sum(nil)[:8]))
}

func painterPix(p *image.RGBA) []byte {
	if p == nil {
		return []byte("nil")
	}
	return append([]byte(p.Rect.String()), p.Pix...)
}

func painterLog(l *pixelLog, r image.Rectangle) []byte {
	var out []byte
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c, ok := l.value(len(l.ops), x, y)
			k := byte(0)
			if ok {
				k = 1
			}
			out = append(out, c.R, c.G, c.B, c.A, k)
		}
	}
	return out
}

func painterCalls(calls []text.DrawCall) []byte {
	var out []byte
	for _, c := range calls {
		out = binary.LittleEndian.AppendUint32(out, uint32(int32(c.X)))
		out = binary.LittleEndian.AppendUint32(out, uint32(int32(c.Y)))
		for _, v := range c.RasterColors {
			out = append(out, v.R, v.G, v.B, v.A)
		}
		out = append(out, '|')
		for _, v := range c.Under {
			out = append(out, v.R, v.G, v.B, v.A)
		}
		out = append(out, fmt.Sprint(c.Tint, c.SourceOver, c.Flat, c.Erased)...)
	}
	return out
}

// painterPlanDigest is the digest of every line TestPainterPlansKeepEveryPixel
// writes, taken at the base before the frame shadows and the native-to-wide
// composites were planned; the lines are in the hotfix's notes.
const painterPlanDigest = "a851da0b2257eda2ec36eef751f885d05d74a5f4cb9b9fbd9dd6a396f942e1dc"

// TestPainterPlansKeepEveryPixel renders every frame shadow and composite the
// painter-twin census names through the CPU path, and through the GPU path's
// pixel log, mask upload and submission (no GPU readback runs headless), and
// compares the digest with the base's.
func TestPainterPlansKeepEveryPixel(t *testing.T) {
	var d painterDigest
	screen := image.Rect(0, 0, 640, 480)
	policies := []DialogueBackdrop{
		{},
		{Layout: backdrop.RGB555, Mode: backdrop.Reduced},
		{Layout: backdrop.RGB565, Mode: backdrop.Reduced, FrameClipped: true, FrameClip: image.Rect(60, 40, 410, 330), Clipped: true, Clip: image.Rect(30, 20, 500, 400)},
	}
	bodies := []image.Rectangle{
		image.Rect(0, 0, 40, 40),
		image.Rect(0, 0, 96+96*2, 96+64*2),
		image.Rect(5, 7, 305, 177),
		image.Rect(0, 0, 632, 472),
		image.Rect(500, 400, 800, 600),
	}
	// Widget kit frames, over an empty destination and over a picture.
	for ai, sizes := range painterPlanPieceSizes {
		art := painterPlanFrame(sizes, ai+1)
		for _, kind := range []frameKind{frameWindow, frameTip, framePanel} {
			for bi, body := range bodies {
				for pi, pol := range policies {
					for _, over := range []bool{false, true} {
						dst := image.NewRGBA(screen)
						if over {
							dst = painterPlanPicture(screen, bi+pi)
						}
						s := frameSpec{Kind: kind, Rect: image.Rect(body.Min.X, body.Min.Y, body.Max.X+8, body.Max.Y+8), Art: art, Policy: pol,
							Fill: color.RGBA{10, 20, 30, 255}, Border: color.RGBA{200, 100, 50, 255}}
						drawFrame(dst, s)
						d.add(fmt.Sprintf("frame art=%d kind=%d body=%d policy=%d over=%v", ai, kind, bi, pi, over), painterPix(dst))
					}
				}
			}
		}
		snap := frameSpec{Kind: frameWindow, Rect: image.Rect(0, 0, 300, 200), Snap: true, Art: art}
		dst := painterPlanPicture(screen, 9)
		drawFrame(dst, snap)
		d.add(fmt.Sprintf("frame art=%d snapped", ai), painterPix(dst))
	}
	// The notice's dialogue shadow on the CPU.
	for ai, sizes := range painterPlanPieceSizes {
		for pi, pol := range policies {
			for _, at := range []image.Point{{}, {30, 60}} {
				l := dialogueClaimLayout()
				l.Frame = painterPlanFrame(sizes, ai+3)
				l.DialogueBackdrop = pol
				room := painterPlanPicture(screen, pi)
				ComposeDialogueNotice(room, l, dialogueClaimFont(), "AB", nil, l.Box.Min.Add(at))
				d.add(fmt.Sprintf("notice art=%d policy=%d at=%v", ai, pi, at), painterPix(room))
			}
		}
	}
	// The dialogue window's shadow on the CPU and on the GPU, with captured
	// text under it.
	small := image.Rect(0, 0, 320, 240)
	f := dialogueClaimFont()
	for ai, sizes := range painterPlanPieceSizes {
		art := painterPlanFrame(sizes, ai+5)
		for bi, body := range []image.Rectangle{image.Rect(0, 0, 200, 150), image.Rect(0, 0, 30, 20), image.Rect(0, 0, 250, 180)} {
			for pi, pol := range policies {
				for _, at := range []image.Point{{}, {40, 30}, {100, 90}} {
					room := painterPlanPicture(small, bi+pi)
					calls := text.Record(func() { f.Draw(room, "AAAA", at.X+body.Dx()-6, at.Y+body.Dy()/2, color.RGBA{248, 252, 248, 255}) })
					cpu := image.NewRGBA(room.Rect)
					copy(cpu.Pix, room.Pix)
					s := dialogueBackdropState{policy: pol}
					cpuCalls := append([]text.DrawCall(nil), calls...)
					s.applyFrameShadows(cpu, art, body, at, cpuCalls)
					d.add(fmt.Sprintf("dialogue cpu art=%d body=%d policy=%d at=%v", ai, bi, pi, at), painterPix(cpu), painterCalls(cpuCalls))
					for _, scale := range []float64{1, 0.5, 2} {
						g := dialogueBackdropState{policy: pol}
						var log pixelLog
						log.reset(room.Bounds())
						log.upload(room)
						var submitted []byte
						old := submitDialogueShadow
						submitDialogueShadow = func(dst *ebiten.Image, vertices []ebiten.Vertex, shader *ebiten.Shader, op *ebiten.DrawTrianglesShaderOptions) {
							submitted = append(submitted, fmt.Sprint(dst.Bounds(), vertices, op.Uniforms["GreenBits"], op.Uniforms["MaskAt"], op.Blend)...)
							old(dst, vertices, shader, op)
						}
						gpuCalls := append([]text.DrawCall(nil), calls...)
						g.drawFrameShadows(newPainterPlanTexture(room), &log, art, body, at, scale, gpuCalls)
						submitDialogueShadow = old
						d.add(fmt.Sprintf("dialogue gpu art=%d body=%d policy=%d at=%v scale=%v", ai, bi, pi, at, scale),
							painterLog(&log, room.Bounds()), painterPix(g.shadowMask), submitted, painterCalls(gpuCalls))
					}
				}
			}
		}
	}
	// The dialogue backdrop remap on the CPU and on the GPU.
	for pi, pol := range policies {
		for _, shows := range []uint64{1, 2, 40} {
			room := painterPlanPicture(small, pi)
			calls := text.Record(func() { f.Draw(room, "AAAA", 50, 60, color.RGBA{248, 252, 248, 255}) })
			s := dialogueBackdropState{policy: pol}
			cpu := image.NewRGBA(room.Rect)
			copy(cpu.Pix, room.Pix)
			s.apply(cpu, shows)
			var log pixelLog
			log.reset(room.Bounds())
			log.upload(room)
			gpuCalls := append([]text.DrawCall(nil), calls...)
			g := dialogueBackdropState{policy: pol}
			g.drawWithCalls(newPainterPlanTexture(room), &log, shows, gpuCalls)
			d.add(fmt.Sprintf("backdrop policy=%d shows=%d", pi, shows), painterPix(cpu), painterLog(&log, room.Bounds()), painterCalls(gpuCalls))
		}
	}
	// Native-to-wide composites: the CPU layout, the GPU route's pixel log,
	// and the detached dialogue on both.
	native := painterPlanPicture(screen, 4)
	dialogue := painterPlanPicture(image.Rect(0, 0, 580, 240), 6)
	for _, width := range []int{640, 854, 960, 1280} {
		for _, mode := range []wideFrameMode{wideFrameCentered, wideFrameStretched, wideFrameTownColumns} {
			layout := newWideFrameLayout(width, mode)
			wide := layout.compose(native)
			d.add(fmt.Sprintf("compose width=%d mode=%d", width, mode), painterPix(wide))
			a := newTestApp(t, appRows(1), okLoader(t))
			a.canvas = newPainterPlanTexture(native)
			a.canvasLog.reset(native.Bounds())
			a.canvasLog.upload(native)
			present := a.prepareWideCanvas(layout)
			d.add(fmt.Sprintf("wide log width=%d mode=%d", width, mode), []byte(present.Bounds().String()), painterLog(&a.presentLog, present.Bounds()))
			cpu := layout.compose(native)
			overlayDetachedTownDialogue(cpu, dialogue)
			for pi, pol := range policies {
				var log pixelLog
				log.reset(wide.Bounds())
				log.upload(wide)
				paintDetachedTownDialogue(newPainterPlanTexture(wide), dialogue, &log, pol)
				d.add(fmt.Sprintf("detached width=%d mode=%d policy=%d", width, mode, pi), painterPix(cpu), painterLog(&log, wide.Bounds()))
			}
		}
	}
	// Whole frames: both games' menu placements over the mission and the
	// town, the headless panel, the menu layer over the map, and a town
	// dialogue through HeadlessFrame and Draw.
	for _, size := range []image.Point{{1024, 768}, {1920, 1080}} {
		for _, g := range gameMenuProfiles {
			for _, state := range []string{"menu", "load", "mod", "town", "map"} {
				a := placementOpsApp(t, g.profile, state, size)
				pix, note, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame})
				name := fmt.Sprintf("app %v %s %s", size, g.game, state)
				d.add(name+" headless", painterPix(pix), []byte(note.Note), []byte(fmt.Sprint(err != nil)))
				a.Draw(ebiten.NewImage(size.X, size.Y))
				d.add(name+" draw", painterLog(&a.menuLog, image.Rect(0, 0, 640, 480)), painterLog(&a.presentLog, a.presentLog.bounds))
			}
		}
	}
	for _, width := range []int{640, 960} {
		for _, smooth := range []bool{false, true} {
			l := dialogueClaimLayout()
			l.Frame = painterPlanFrame(painterPlanPieceSizes[0], 7)
			l.Button, l.Portrait = image.Rectangle{}, image.Rectangle{}
			town := &frameDialogueTown{fakeShopDialogueTown: fakeShopDialogueTown{pic: image.NewRGBA(image.Rect(0, 0, 488, 232))}, layout: l}
			a := newTestApp(t, appRows(3), okLoader(t))
			a.SetTown(town)
			a.flow.showTown("")
			a.Layout(width, 480)
			a.SetTextSmoothing(smooth)
			pix, _, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame})
			if err != nil {
				t.Fatal(err)
			}
			a.Draw(ebiten.NewImage(width, 480))
			var layers []byte
			for _, tl := range a.textLayers {
				layers = append(layers, painterCalls(tl.calls)...)
			}
			d.add(fmt.Sprintf("town dialogue width=%d smooth=%v", width, smooth), painterPix(pix), painterLog(&a.presentLog, a.presentLog.bounds), layers)
		}
	}
	joined := strings.Join(d.lines, "\n")
	if sum := fmt.Sprintf("%x", sha256.Sum256([]byte(joined))); sum != painterPlanDigest {
		t.Fatalf("%d lines, digest %s, want %s:\n%s", len(d.lines), sum, painterPlanDigest, joined)
	}
}
