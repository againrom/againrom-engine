package game

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// TestReleaseMissionMinimapPaintsTheInstalledPicture checks every pixel of the
// first game's minimap on mission 10, after the party walks, against inputs
// read here: interface/MiniMapData.bmp, interface/CrystalR.bmp, tile words and
// raw fog. The picture spans the map without its eight-cell border. A block
// with fog OR 0 is the CrystalR pixel, 0x8000 the per-channel average of that
// pixel and the terrain, a visible block the MiniMapData sample times a
// brightness factor (MISSION-083, MISSION-084). The rest is the white outline
// (MISSION-086), an owner mark: palette entry 164 of an owner table plus one
// common sky tint (MISSION-085), or a loot dot. The loss control is the
// symbolic picture, drawn without the installed paint.
func TestReleaseMissionMinimapPaintsTheInstalledPicture(t *testing.T) {
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the first game's minimap picture")
	}
	f.SetDeterministicFrames(true)
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	app := f.App("mission minimap picture")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	if app.HeadlessNoticeOpen() || app.Screen() != ui.ScreenMap || f.live == nil {
		t.Fatal("mission did not settle on the map")
	}
	opened, err := app.HeadlessMinimap()
	if err != nil {
		t.Fatal(err)
	}
	opened = clonePicture(opened)

	m := f.live.mission.state.Map
	start, ok := f.live.entity(f.live.mission.ids[0])
	if !ok {
		t.Fatal("party leader has no entity")
	}
	goal := image.Pt(m.Width/2, m.Height/2)
	if d := goal.Sub(image.Pt(int(start.X), int(start.Y))); d.X*d.X+d.Y*d.Y < 400 {
		goal = image.Pt(m.Width-12, m.Height-12)
	}
	for _, id := range f.live.mission.ids {
		if e, ok := f.live.entity(id); ok && e.Alive() {
			f.live.enqueue(uint32(id), goal.X, goal.Y)
		}
	}
	// The world runs through ordinary presentation frames. The picture is
	// read on the first frame after 240 on which it changes: that frame
	// composed it from the fog the same frame pushed.
	var pic *image.RGBA
	prev, last, changes, closest := opened, 0, 0, 1<<30
	for step := 1; step <= 900 && pic == nil; step++ {
		advance := app.HeadlessStep
		if app.HeadlessNoticeOpen() {
			advance = func() error { return app.HeadlessKey("enter") }
		}
		if err := advance(); err != nil {
			t.Fatal(err)
		}
		now, err := app.HeadlessMinimap()
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(now.Pix, prev.Pix) {
			continue
		}
		changes++
		if last != 0 {
			closest = min(closest, step-last)
		}
		last, prev = step, clonePicture(now)
		if step >= 240 {
			pic = prev
		}
	}
	if pic == nil {
		t.Fatal("the minimap did not change after 240 frames of walking")
	}
	t.Logf("%d picture changes over %d frames; closest two %d frames apart", changes, last, closest)
	if changes > 1 && closest < 11 {
		t.Errorf("the picture changed twice within %d frames", closest)
	}

	got := minimapPictureCensus(t, f, pic)
	t.Logf("mission 10, map %dx%d, interior %v, picture %v at %v: unseen %d/%d, explored %d/%d, visible %d/%d, outline %d, marks %d (tint %v), loot %d, other %d",
		m.Width, m.Height, got.window, got.size, got.origin, got.unseenOK, got.unseen, got.exploredOK, got.explored, got.visibleOK, got.visible, got.outline, got.marks, got.tint, got.loot, got.other)
	if got.unseen == 0 || got.explored == 0 || got.visible == 0 {
		t.Fatalf("the picture lacks a fog class: unseen %d, explored %d, visible %d", got.unseen, got.explored, got.visible)
	}
	if got.other != 0 {
		t.Errorf("%d pixels follow no rule; first at %v", got.other, got.firstOther)
	}
	if got.outline == 0 || got.marks == 0 {
		t.Errorf("outline %d and mark %d pixels: both need one", got.outline, got.marks)
	}

	// Loss control: the same census of the picture drawn without the
	// installed paint.
	f.live.view.SetMinimapPaint(nil)
	symbolic, err := app.HeadlessMinimap()
	f.live.view.SetMinimapPaint(missionMinimapPaint(&f.InstallResources))
	if err != nil {
		t.Fatal(err)
	}
	loss := minimapPictureCensus(t, f, clonePicture(symbolic))
	t.Logf("loss control, symbolic picture: unseen %d/%d, explored %d/%d, visible %d/%d, other %d",
		loss.unseenOK, loss.unseen, loss.exploredOK, loss.explored, loss.visibleOK, loss.visible, loss.other)
	if loss.other == 0 || loss.unseenOK == loss.unseen && loss.exploredOK == loss.explored {
		t.Error("the symbolic picture passes the installed-picture census")
	}
}

func clonePicture(p *image.RGBA) *image.RGBA {
	out := image.NewRGBA(p.Bounds())
	copy(out.Pix, p.Pix)
	return out
}

type minimapPictureCount struct {
	window                          image.Rectangle
	size, origin                    image.Point
	unseen, explored, visible       int
	unseenOK, exploredOK, visibleOK int
	outline, marks, loot, other     int
	tint                            [3]int
	tintSet                         bool
	firstOther                      image.Point
}

// minimapPictureCensus classes every pixel of the picture's interior
// rectangle by the fog OR of its block and checks it against the installed
// inputs. A pixel its fog class does not explain counts as the outline when
// white, as a mark when it is an owner shade plus the picture's one tint, as a
// loot dot in its unchanged colour (DIV-2954), and otherwise as other.
func minimapPictureCensus(t *testing.T, f *FrontEnd, pic *image.RGBA) minimapPictureCount {
	t.Helper()
	m := f.live.mission.state.Map
	g := terrain.Grid{Width: m.Width, Height: m.Height, Block: mapload.PassabilityWith(m, f.Table)}
	window := image.Rectangle{Min: image.Pt(m.Width, m.Height)}
	for row := 0; row < m.Height; row++ {
		for col := 0; col < m.Width; col++ {
			if !g.BorderCell(col, row) {
				window.Min.X, window.Min.Y = min(window.Min.X, col), min(window.Min.Y, row)
				window.Max.X, window.Max.Y = max(window.Max.X, col+1), max(window.Max.Y, row+1)
			}
		}
	}
	if window.Empty() {
		t.Fatal("no installed terrain window")
	}
	lookup, err := readInstallBMP(f.Archives.Containers, "graphics/interface/minimapdata.bmp")
	if err != nil {
		t.Fatal(err)
	}
	crystal, err := readInstallBMP(f.Archives.Containers, "graphics/interface/crystalr.bmp")
	if err != nil {
		t.Fatal(err)
	}
	seam := pic.Bounds().Dx() - crystal.Bounds().Dx()
	den := max(window.Dx(), window.Dy())
	size := image.Pt(window.Dx()*128/den, window.Dy()*128/den)
	origin := image.Pt(seam+72-size.X>>1, 82-size.Y>>1)
	fog := f.live.fog.project()
	word := func(col, row int) int {
		if col < 0 || row < 0 || col >= m.Width || row >= m.Height {
			return 0
		}
		switch fog[row*m.Width+col] {
		case ui.FogVisible:
			return 0xc000
		case ui.FogExplored:
			return 0x8000
		}
		return 0
	}
	var shades []color.RGBA
	for k := range 16 {
		c := f.Units.OwnerPalettes[k][164]
		shades = append(shades, color.RGBA{c.R, c.G, c.B, 255})
	}
	out := minimapPictureCount{window: window, size: size, origin: origin}
	terrainAt := func(cell image.Point, c color.RGBA) bool {
		tile := int(m.Tiles[cell.Y*m.Width+cell.X] & 0x1fff)
		s := lookup.RGBAAt(tile>>4, tile&15)
		for factor := 1; factor <= 96; factor++ {
			gain := func(v uint8) uint8 { return uint8(min(255, int(v)*factor>>5)) }
			if c == (color.RGBA{gain(s.R), gain(s.G), gain(s.B), 255}) {
				return true
			}
		}
		return false
	}
	averageAt := func(cell image.Point, bg, c color.RGBA) bool {
		tile := int(m.Tiles[cell.Y*m.Width+cell.X] & 0x1fff)
		s := lookup.RGBAAt(tile>>4, tile&15)
		for factor := 1; factor <= 96; factor++ {
			gain := func(v uint8) uint8 { return uint8(min(255, int(v)*factor>>5)) }
			if c == (color.RGBA{bg.R>>1 + gain(s.R)>>1, bg.G>>1 + gain(s.G)>>1, bg.B>>1 + gain(s.B)>>1, 255}) {
				return true
			}
		}
		return false
	}
	mark := func(c color.RGBA) bool {
		for _, s := range shades {
			d := [3]int{int(c.R) - int(s.R), int(c.G) - int(s.G), int(c.B) - int(s.B)}
			if d[0] < 0 || d[1] < 0 || d[2] < 0 {
				continue
			}
			if !out.tintSet {
				out.tint, out.tintSet = d, true
			}
			if d == out.tint {
				return true
			}
		}
		return false
	}
	for y := range size.Y {
		for x := range size.X {
			col, row := x*den/128, y*den/128
			if 128 >= den {
				col, row = ((x+1)*den-1)/128, ((y+1)*den-1)/128
			}
			cell := window.Min.Add(image.Pt(col, row))
			or := word(cell.X, cell.Y) | word(cell.X+1, cell.Y) | word(cell.X+1, cell.Y+1) | word(cell.X+2, cell.Y+1)
			px := origin.Add(image.Pt(x, y))
			c := pic.RGBAAt(px.X, px.Y)
			bg := crystal.RGBAAt(px.X-seam, px.Y)
			bg.A = 255
			switch {
			case or == 0:
				out.unseen++
				if c == bg {
					out.unseenOK++
					continue
				}
			case or == 0x8000:
				out.explored++
				if averageAt(cell, bg, c) {
					out.exploredOK++
					continue
				}
			default:
				out.visible++
				if terrainAt(cell, c) {
					out.visibleOK++
					continue
				}
			}
			switch {
			case c == (color.RGBA{255, 255, 255, 255}):
				out.outline++
			case mark(c):
				out.marks++
			case c == (color.RGBA{0xff, 0xc8, 0x20, 0xff}):
				out.loot++
			default:
				if out.other == 0 {
					out.firstOther = px
				}
				out.other++
			}
		}
	}
	return out
}
