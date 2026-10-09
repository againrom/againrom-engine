package game

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseStatusBarsSpanALargeUnitsSelectionBox opens installed mission 81
// with the default fighter, selects him through the production press, turns on
// the view's debug reveal, and composes the viewport on the CPU twice. Mission
// 81 places the fighter at cell (18,60), a dragon (class 71, three cells wide)
// at (38,61) and an ogre (class 66, two cells wide) at (19,19).
//
// The fighter's health bar is the one-cell picture, 32 columns by 4, opaque.
// The dragon's bars are 96 columns centred on its body's anchor, the health
// bar's top 50 rows above it, as the owner's capture of the original shows
// (DIV-1460). The ogre's bar spans its installed units.reg selection box, 60
// columns, 2 rows above the box's top edge (DIV-1481). A wide bar has the
// measured cap at both ends and its interior filled in proportion to the pool,
// half blended because the unit is not selected; a unit with no mana has no
// mana bar. The three units stand on their placed cells throughout, and
// composing does not move the world hash. AGAINROM_STATUSBAR_FRAMES, when set,
// names the directory the frames are written to for inspection.
func TestReleaseStatusBarsSpanALargeUnitsSelectionBox(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Markers = Markers{}
	app := f.App("large unit status bars")
	defer app.StopAudio()
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(81)); err != nil {
		t.Fatalf("open mission 81: %v", err)
	}
	live := f.live
	if live == nil || len(live.mission.ids) != 1 {
		t.Fatal("mission 81 opened without the one default hero")
	}
	if !live.view.HealthBarsShown() || live.view.Mode() != ui.ModeDisplaced {
		t.Fatalf("show health %v, mode %v; want the fresh map's show health on and the displaced view",
			live.view.HealthBarsShown(), live.view.Mode())
	}
	addr, _ := MissionMap(81)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatalf("decode %s: %v", addr, err)
	}
	minV := releaseGroundMinVertex(m)

	// The class geometry is read from the installed units.reg through the data
	// tier, not from the render tier the frame is drawn with.
	regRaw, err := f.Archives.Containers.ReadFile(UnitRegistry)
	if err != nil {
		t.Fatalf("read %s: %v", UnitRegistry, err)
	}
	registry, err := reg.Parse(regRaw)
	if err != nil {
		t.Fatalf("parse %s: %v", UnitRegistry, err)
	}
	classes, err := data.LoadUnitClasses(registry)
	if err != nil {
		t.Fatalf("load %s: %v", UnitRegistry, err)
	}

	type unit struct {
		name  string
		class int32
		cell  image.Point
		id    sim.EntityID
		// health is the health bar's rectangle from the anchor pixel, the
		// class centre's pixel on the anchor cell; zero for the one-cell rule.
		health image.Rectangle
	}
	units := []*unit{
		{name: "fighter", cell: image.Pt(18, 60), id: live.mission.ids[0]},
		{name: "dragon", class: 71, cell: image.Pt(38, 61)},
		{name: "ogre", class: 66, cell: image.Pt(19, 19)},
	}
	for _, e := range live.world.EntityView() {
		for _, u := range units[1:] {
			if e.Alive() && e.Class == u.class && int(e.X) == u.cell.X && int(e.Y) == u.cell.Y {
				u.id = e.ID
			}
		}
	}
	for _, u := range units[1:] {
		c, ok := classes.ByID(u.class)
		if u.id == 0 || !ok || c.TileSize < 2 {
			t.Fatalf("mission 81 has no living class %d %s at %v wider than one cell", u.class, u.name, u.cell)
		}
		u.health = image.Rect(int(c.SelectionX1-c.CenterX), int(c.SelectionY1-c.CenterY)-2,
			int(c.SelectionX2-c.CenterX), int(c.SelectionY1-c.CenterY)+2)
		t.Logf("%s: class %d TileSize %d centre (%d,%d) box x %d..%d y %d..%d; health bar %v from the anchor",
			u.name, u.class, c.TileSize, c.CenterX, c.CenterY, c.SelectionX1, c.SelectionX2, c.SelectionY1, c.SelectionY2, u.health)
	}
	// The capture's dragon: 96.7 +- 2 columns centred on the body's anchor,
	// the health bar's top 50.4 +- 0.5 rows above it.
	if want := image.Rect(-48, -50, 48, -46); units[1].health != want {
		t.Fatalf("the installed dragon's box puts its health bar at %v from the anchor; the capture shows %v", units[1].health, want)
	}

	// Every frame the witness steps, the three units must stand on their
	// placed cells, so the bars carry no walking displacement.
	still := func(when string) {
		for _, u := range units {
			e, ok := live.entity(u.id)
			fx, fy, fine := live.world.ActorFinePosition(u.id)
			if !ok || int(e.X) != u.cell.X || int(e.Y) != u.cell.Y || e.Transit != 0 ||
				live.world.ActorMotionActive(u.id) || fine && (fx != 128 || fy != 128) {
				t.Fatalf("%s: the %s is at (%d,%d) transit %d motion %v fine %v (%d,%d); want it standing on %v",
					when, u.name, e.X, e.Y, e.Transit, live.world.ActorMotionActive(u.id), fine, fx, fy, u.cell)
			}
		}
	}
	frames := 0
	step := func() {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		frames++
		still(fmt.Sprintf("frame %d", frames))
	}
	still("at open")

	live.view.SetFogReveal(true)
	cam := live.view.Camera()
	settle := func() {
		for i, same := 0, 0; same < 8; i++ {
			if i == 240 {
				t.Fatalf("the view is still changing height after %d frames", i)
			}
			h := cam.ViewH
			step()
			if cam.ViewH == h {
				same++
			} else {
				same = 0
			}
		}
	}
	lift := func(p image.Point) int { return statusBarAnchorLift(m, minV, p.X, p.Y) }
	centre := func(x, y float64) {
		cam.CenterOn(x, y)
		cam.X, cam.Y = math.Floor(cam.X), math.Floor(cam.Y)
		cam.Clamp()
	}
	centre(float64(units[0].cell.X*32+16), float64(units[0].cell.Y*32+16+lift(units[0].cell)))
	if err := app.HeadlessSelectEntity(uint32(units[0].id)); err != nil {
		t.Fatalf("select the fighter through the production press: %v", err)
	}
	settle()
	if got, ok := live.view.SelectedUnit(); !ok || got != uint32(units[0].id) {
		t.Fatalf("the selection is %d (%v), want the fighter", got, ok)
	}
	t.Logf("view %dx%d", cam.ViewW, cam.ViewH)

	dir := os.Getenv("AGAINROM_STATUSBAR_FRAMES")
	lang := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	for _, shot := range []struct {
		name  string
		shown []*unit
	}{
		{"fighter-and-dragon", units[:2]},
		{"ogre", units[2:]},
	} {
		var sx, sy float64
		for _, u := range shot.shown {
			sx += float64(u.cell.X*32 + 16)
			sy += float64(u.cell.Y*32 + 16 + lift(u.cell))
		}
		centre(sx/float64(len(shot.shown)), sy/float64(len(shot.shown)))
		before := live.world.Hash()
		start := time.Now()
		frame, under, err := live.view.HeadlessStatusBarFrame()
		if err != nil {
			t.Fatalf("compose the %s: %v", shot.name, err)
		}
		t.Logf("%s: composed in %v, camera (%v,%v)", shot.name, time.Since(start).Round(time.Millisecond), cam.X, cam.Y)
		if after := live.world.Hash(); after != before {
			t.Fatalf("composing moved the world hash from %#016x to %#016x", before, after)
		}
		if cam.Zoom != 1 || cam.X != math.Floor(cam.X) || cam.Y != math.Floor(cam.Y) {
			t.Fatalf("camera at (%v,%v) zoom %v; the witness reads whole native pixels", cam.X, cam.Y, cam.Zoom)
		}
		crop := image.Rectangle{}
		for _, u := range shot.shown {
			e, _ := live.entity(u.id)
			if e.MaxHP <= 0 {
				t.Fatalf("the %s has no health pool: %+v", u.name, e)
			}
			faded := u.id != units[0].id
			anchor := image.Pt(u.cell.X*32+16-int(cam.X), u.cell.Y*32+16+lift(u.cell)-int(cam.Y))
			health, width := anchor.Add(image.Pt(-16, -32)), 32
			if u.health != (image.Rectangle{}) {
				health, width = anchor.Add(u.health.Min), u.health.Dx()
			} else if e.TokenSize != 1 {
				t.Fatalf("the %s has token size %d; want a one-cell unit", u.name, e.TokenSize)
			}
			mana := health.Add(image.Pt(0, 4))
			fill := func(v, most int32, minimum bool) int {
				n := (width - 8) * int(v) / int(most)
				if n == 0 && minimum {
					n = 1
				}
				return n
			}
			t.Logf("%s: HP %d/%d mana %d/%d, health bar at %v, %d wide", u.name, e.HP, e.MaxHP, e.Mana, e.MaxMana, health, width)
			report := func(what string, at image.Point, bad []string) {
				if len(bad) > 0 {
					t.Errorf("%s (faded %v) %s at %v, %d wide: %d of %d pixels differ; first %s",
						u.name, faded, what, at, width, len(bad), 4*width, strings.Join(bad[:min(3, len(bad))], "; "))
				}
			}
			healthRows := statusBarWantHealthAt(e.HP, e.MaxHP)
			report("health bar", health, statusBarCheck(frame, under, health, width, &healthRows, fill(e.HP, e.MaxHP, e.HP != 0), faded))
			if e.MaxMana > 0 {
				report("mana bar", mana, statusBarCheck(frame, under, mana, width, &statusBarWantMana, fill(e.Mana, e.MaxMana, true), faded))
			} else {
				report("band under health, which must hold no bar,", mana, statusBarCheck(frame, under, mana, width, nil, 0, false))
			}
			crop = crop.Union(image.Rect(health.X-16, health.Y-16, health.X+width+16, anchor.Y+40))
		}
		if dir != "" {
			name := fmt.Sprintf("%s-%s", lang, shot.name)
			writeStatusBarFrame(t, filepath.Join(dir, name+"-frame.png"), frame)
			writeStatusBarFrame(t, filepath.Join(dir, name+"-crop4x.png"), statusBarScaled(frame, crop.Intersect(frame.Bounds()), 4))
		}
	}
}
