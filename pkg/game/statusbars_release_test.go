package game

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func statusBarWantRGB(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// The original's bar picture as measured from the owner's screenshot: the
// end cap, rows top to bottom, a zero colour transparent, and the two kinds'
// interior rows. Written out here rather than read from the render tier, so
// the witness does not grade the code by its own tables.
var (
	statusBarWantCap = [4][4]color.RGBA{
		{{}, statusBarWantRGB(0x6b4129), statusBarWantRGB(0x4a2c18), {}},
		{statusBarWantRGB(0x9c6542), statusBarWantRGB(0xce926b), statusBarWantRGB(0x6b4129), statusBarWantRGB(0x422810)},
		{statusBarWantRGB(0x6b4129), statusBarWantRGB(0x9c6542), statusBarWantRGB(0x4a2810), statusBarWantRGB(0x392410)},
		{statusBarWantRGB(0x000400), statusBarWantRGB(0x392410), statusBarWantRGB(0x211408), {}},
	}
	statusBarWantHealth = [4]color.RGBA{statusBarWantRGB(0x008200), statusBarWantRGB(0x00ff00), statusBarWantRGB(0x00c300), statusBarWantRGB(0x008200)}
	statusBarWantMana   = [4]color.RGBA{statusBarWantRGB(0x000084), statusBarWantRGB(0x0000ff), statusBarWantRGB(0x0000c6), statusBarWantRGB(0x000084)}
)

// statusBarWantFaded is a row colour at half opacity over what lies under
// it: the colour's 128/255 plus the ground's 127/255.
func statusBarWantFaded(c, under color.RGBA) color.RGBA {
	mix := func(s, d uint8) uint8 { return uint8((uint32(s)*128+127)/255 + (uint32(d)*127+127)/255) }
	return color.RGBA{R: mix(c.R, under.R), G: mix(c.G, under.G), B: mix(c.B, under.B), A: 0xff}
}

func statusBarNear(a, b color.RGBA) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) <= 1 && d(a.G, b.G) <= 1 && d(a.B, b.B) <= 1
}

// statusBarCheck compares the width by 4 pixels at 'at' with one bar: the
// caps exact and opaque at both ends, fill interior columns in the rows (half
// blended when faded), the unfilled columns and the caps' transparent pixels
// showing what lies under. rows nil asks that no bar is drawn there at all.
func statusBarCheck(frame, under *image.RGBA, at image.Point, width int, rows *[4]color.RGBA, fill int, faded bool) []string {
	var bad []string
	for y := range 4 {
		for x := range width {
			p := at.Add(image.Pt(x, y))
			if !p.In(frame.Bounds()) {
				return append(bad, fmt.Sprintf("pixel (%d,%d) at %v is outside the viewport", x, y, p))
			}
			got, ground := frame.RGBAAt(p.X, p.Y), under.RGBAAt(p.X, p.Y)
			want, near := ground, false
			switch {
			case rows == nil:
			case x < 4:
				if c := statusBarWantCap[y][x]; c.A != 0 {
					want = c
				}
			case x >= width-4:
				if c := statusBarWantCap[y][x-width+4]; c.A != 0 {
					want = c
				}
			case x-4 < fill && faded:
				want, near = statusBarWantFaded(rows[y], ground), true
			case x-4 < fill:
				want = rows[y]
			}
			if got != want && !(near && statusBarNear(got, want)) {
				bad = append(bad, fmt.Sprintf("pixel (%d,%d) is %v, want %v over %v", x, y, got, want, ground))
			}
		}
	}
	return bad
}

// statusBarAnchorLift is how far the displaced view lifts a mark on cell
// (col,row): the mean of the cell's four corner altitudes, read off the
// mission's own map, measured from the canvas origin minV.
func statusBarAnchorLift(m *alm.Map, minV, col, row int) int {
	h := releaseGroundAltitude(m, col, row) + releaseGroundAltitude(m, col+1, row) +
		releaseGroundAltitude(m, col, row+1) + releaseGroundAltitude(m, col+1, row+1)
	return -h/4 - minV
}

// TestReleaseStatusBarsOnAFighterAndAMage opens installed mission 10 with a
// fighter and a mage, selects each in turn through the production press, and
// composes the viewport on the CPU. The selected unit's bars must be the
// original's opaque picture and the other unit's interiors half blended with
// opaque caps; the fighter has no mana bar; each bar stands 16 (health) or 12
// (mana) rows above its unit's lifted cell; composing does not move the world
// hash. AGAINROM_STATUSBAR_FRAMES, when set, names the directory the frames
// are written to for inspection.
func TestReleaseStatusBarsOnAFighterAndAMage(t *testing.T) {
	f := releaseFront(t)
	witnessDir := effectRimWitnessDir(t, f)
	f.SetDeterministicFrames(true)
	// The shipped game starts without the diagnostic markers.
	f.Markers = Markers{}
	fighter := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	mage := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	mage.ID, mage.StartingHero, mage.Name = "companion-mage", false, "Mage"
	app := f.App("status bars")
	defer app.StopAudio()
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, []mapload.PartyMember{fighter, mage})); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	live := f.live
	if live == nil || len(live.mission.ids) != 2 {
		t.Fatal("mission 10 opened without the fighter and the mage")
	}
	if !live.view.HealthBarsShown() || live.view.Mode() != ui.ModeDisplaced {
		t.Fatalf("show health %v, mode %v; want the fresh map's show health on and the displaced view",
			live.view.HealthBarsShown(), live.view.Mode())
	}
	addr, _ := MissionMap(10)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", addr, err)
	}
	minV := releaseGroundMinVertex(m)

	units := []struct {
		name string
		id   sim.EntityID
	}{{"fighter", live.mission.ids[0]}, {"mage", live.mission.ids[1]}}
	a, _ := live.entity(units[0].id)
	b, _ := live.entity(units[1].id)
	if a.MaxMana != 0 || b.MaxMana <= 0 {
		t.Fatalf("fighter mana maximum %d, mage %d; want none and some", a.MaxMana, b.MaxMana)
	}
	// The party stands near the map's bottom edge, and a selection slides the
	// unit panel up over the view's lower rows, so the camera is centred on the
	// party's lifted cells whenever the view has changed height.
	cam := live.view.Camera()
	centre := func() {
		a, _ := live.entity(units[0].id)
		b, _ := live.entity(units[1].id)
		ya := int(a.Y)*32 + statusBarAnchorLift(m, minV, int(a.X), int(a.Y))
		yb := int(b.Y)*32 + statusBarAnchorLift(m, minV, int(b.X), int(b.Y))
		cam.CenterOn(float64((a.X+b.X)*16+16), float64(ya+yb)/2+16)
		cam.X, cam.Y = math.Floor(cam.X), math.Floor(cam.Y)
		cam.Clamp()
	}
	settle := func() {
		for i, same := 0, 0; same < 8; i++ {
			if i == 240 {
				t.Fatalf("the view is still changing height after %d frames", i)
			}
			h := cam.ViewH
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if cam.ViewH == h {
				same++
			} else {
				same = 0
			}
		}
	}

	// Mission 10 opens on a spoken notice that holds the map's input until it
	// is acknowledged; the witness waits for it and acknowledges it.
	for i := 0; ; i++ {
		if _, _, up := f.LiveNotice(); up {
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			break
		}
		if i == 600 {
			t.Fatal("mission 10's opening notice did not appear in 600 frames")
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}

	dir := os.Getenv("AGAINROM_STATUSBAR_FRAMES")
	lang := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	for i, selected := range units {
		if i > 0 {
			// With a unit selected, a press on another own unit is an order
			// rather than a selection, so the selection is first cancelled by
			// the production right click on open ground: three cells west of
			// the fighter, on its row. The frame is the window at this layout.
			e, _ := live.entity(units[0].id)
			gx := (int(e.X)-3)*32 + 16 - int(cam.X)
			gy := int(e.Y)*32 + statusBarAnchorLift(m, minV, int(e.X)-3, int(e.Y)) + 16 - int(cam.Y)
			for _, edge := range []string{"right-press", "right-release"} {
				if err := app.HeadlessPointer(edge, gx, gy); err != nil {
					t.Fatal(err)
				}
			}
			if got, ok := live.view.SelectedUnit(); ok {
				t.Fatalf("the right click at (%d,%d) left %d selected", gx, gy, got)
			}
			settle()
		}
		centre()
		if err := app.HeadlessSelectEntity(uint32(selected.id)); err != nil {
			t.Fatalf("select the %s through the production press: %v", selected.name, err)
		}
		settle()
		centre()
		if got, ok := live.view.SelectedUnit(); !ok || got != uint32(selected.id) {
			t.Fatalf("the selection is %d (%v), want the %s", got, ok, selected.name)
		}
		before := live.world.Hash()
		start := time.Now()
		frame, under, err := live.view.HeadlessStatusBarFrame()
		if err != nil {
			t.Fatalf("compose with the %s selected: %v", selected.name, err)
		}
		t.Logf("%s selected: composed in %v", selected.name, time.Since(start).Round(time.Millisecond))
		if after := live.world.Hash(); after != before {
			t.Fatalf("composing moved the world hash from %#016x to %#016x", before, after)
		}
		if cam.Zoom != 1 || cam.X != math.Floor(cam.X) || cam.Y != math.Floor(cam.Y) {
			t.Fatalf("camera at (%v,%v) zoom %v; the witness reads whole native pixels", cam.X, cam.Y, cam.Zoom)
		}
		crop := image.Rectangle{}
		for _, u := range units {
			e, ok := live.entity(u.id)
			if !ok || e.TokenSize > 1 || e.MaxHP <= 0 {
				t.Fatalf("the %s is %+v; want a present one-cell unit with health", u.name, e)
			}
			top := image.Pt(int(e.X)*32-int(cam.X), int(e.Y)*32+statusBarAnchorLift(m, minV, int(e.X), int(e.Y))-int(cam.Y))
			faded := u.id != selected.id
			fill := func(v, most int32) int { return min(max(24*int(v)/int(most), 0), 24) }
			health, mana := top.Add(image.Pt(0, -16)), top.Add(image.Pt(0, -12))
			report := func(what string, at image.Point, bad []string) {
				if len(bad) > 0 {
					t.Errorf("%s selected, %s (faded %v) %s at %v: %d of 128 pixels differ; first %s",
						selected.name, u.name, faded, what, at, len(bad), strings.Join(bad[:min(3, len(bad))], "; "))
				}
			}
			report("health bar", health, statusBarCheck(frame, under, health, 32, &statusBarWantHealth, fill(e.HP, e.MaxHP), faded))
			if e.MaxMana > 0 {
				report("mana bar", mana, statusBarCheck(frame, under, mana, 32, &statusBarWantMana, fill(e.Mana, e.MaxMana), faded))
			} else {
				report("band under health, which must hold no bar,", mana, statusBarCheck(frame, under, mana, 32, nil, 0, false))
			}
			crop = crop.Union(image.Rect(top.X-16, top.Y-48, top.X+48, top.Y+40))
		}
		if dir != "" {
			name := fmt.Sprintf("%s-%s-selected", lang, selected.name)
			writeStatusBarFrame(t, filepath.Join(dir, name+"-frame.png"), frame)
			writeStatusBarFrame(t, filepath.Join(dir, name+"-crop4x.png"), statusBarScaled(frame, crop.Intersect(frame.Bounds()), 4))
		}
	}
	t.Run("lasting effect art", func(t *testing.T) { releaseEffectRims(t, f, m, minV, witnessDir) })
}

func statusBarScaled(src *image.RGBA, r image.Rectangle, k int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, r.Dx()*k, r.Dy()*k))
	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			out.SetRGBA(x, y, src.RGBAAt(r.Min.X+x/k, r.Min.Y+y/k))
		}
	}
	return out
}

func writeStatusBarFrame(t *testing.T, path string, img image.Image) {
	t.Helper()
	out, err := os.Create(path)
	if err != nil {
		t.Fatalf("write the frame: %v", err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
}
