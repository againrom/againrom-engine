package game

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var launchDirections = [8]image.Point{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}}

var launchDirectionNames = [8]string{"n", "ne", "e", "se", "s", "sw", "w", "nw"}

// launchWitness opens mission 41 with the installed mage, holding his staff or
// bare-handed, who knows Lightning and has the mana for it.
func launchWitness(t *testing.T, staff bool) (*FrontEnd, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(true, nil, f.Bodies, f.Table)
	if !staff {
		party[0].Weapon = nil
		party[0].Worn[0] = 0
	}
	app := f.App("cast launch")
	if err := app.OpenMission(f.MissionOpenerWith(41, party)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	hero := mw.mission.ids[0]
	e, _ := mw.entity(hero)
	rule, ok := mw.world.Spell(spLightning)
	if !ok {
		t.Fatal("no installed Lightning row")
	}
	book := e.Book
	book.State = sim.BookPresent
	book.Slots[spLightning-1] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
	if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: hero, KnownSpells: e.KnownSpells | 1<<spLightning, Book: book}}); err != nil {
		t.Fatal(err)
	}
	return f, hero
}

func launchRefill(t *testing.T, mw *mapWorld, hero sim.EntityID) {
	t.Helper()
	e, _ := mw.entity(hero)
	if err := mw.world.ImportOriginalActorPools([]sim.OriginalActorPools{{ID: hero, HP: e.MaxHP, MaxHP: e.MaxHP, Mana: 1000, MaxMana: 1000}}); err != nil {
		t.Fatal(err)
	}
}

// launchObserve turns the mage to facing and observes his Lightning cast
// three cells along dir, then advances the path object to age 4, the phase-0
// call.
func launchObserve(t *testing.T, mw *mapWorld, hero sim.EntityID, facing uint8, dir image.Point) spellBolt {
	t.Helper()
	if err := mw.world.ImportOriginalActorFacings([]sim.OriginalActorFacing{{ID: hero, Facing: facing}}); err != nil {
		t.Fatal(err)
	}
	e, _ := mw.entity(hero)
	at := image.Pt(int(e.X), int(e.Y)).Add(dir.Mul(3))
	mw.bolts = nil
	mw.observeCasts([]sim.CastEvent{{Caster: hero, Spell: spLightning, Owner: e.Owner, FromX: e.X, FromY: e.Y,
		ToX: int32(at.X), ToY: int32(at.Y), Facing: e.Facing}})
	if len(mw.bolts) != 1 || mw.bolts[0].picture != 34 {
		t.Fatalf("the observed Lightning cast spawned %d objects", len(mw.bolts))
	}
	for range 4 {
		mw.advanceBolts()
	}
	return mw.bolts[0]
}

// launchCast orders Lightning at the nearest unit the book admits and ticks
// until its path object stands at age 4.
func launchCast(t *testing.T, mw *mapWorld, hero sim.EntityID) spellBolt {
	t.Helper()
	mw.bolts = nil
	launchRefill(t, mw, hero)
	h, _ := mw.entity(hero)
	victim, best := sim.EntityID(0), -1
	for _, e := range mw.world.Entities() {
		if e.ID == hero || !e.Alive() || mw.world.BookSpellRefusal(hero, e.ID, spLightning) != "" {
			continue
		}
		d := max(launchAbs(int(e.X-h.X)), launchAbs(int(e.Y-h.Y)))
		if best < 0 || d < best {
			victim, best = e.ID, d
		}
	}
	if best < 0 {
		t.Fatal("no unit the book admits as a Lightning victim")
	}
	mw.pending = append(mw.pending, sim.Cast(hero, victim, spLightning))
	for tick := 0; tick < 240; tick++ {
		mw.tick()
		for _, b := range mw.bolts {
			if b.picture == 34 && b.age == 4 {
				return b
			}
		}
	}
	t.Fatalf("no Lightning object reached age 4 at victim %d", victim)
	return spellBolt{}
}

func launchRenderDir(t *testing.T) string {
	t.Helper()
	out := os.Getenv("AGAINROM_SPELL_LAUNCH_RENDERS")
	if out == "" {
		return ""
	}
	root, err := filepath.Abs(os.Getenv("AGAINROM_ASSETS"))
	if err != nil {
		t.Fatal(err)
	}
	out, err = filepath.Abs(out)
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(root, out); err == nil && !strings.HasPrefix(rel, "..") {
		t.Fatal("render directory is inside the install")
	}
	dir := filepath.Join(out, filepath.Base(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func launchWritePNG(t *testing.T, dir, name string, pix image.Image) {
	t.Helper()
	if dir == "" {
		return
	}
	out, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(out, pix)
	if closeErr := out.Close(); err != nil || closeErr != nil {
		t.Fatal(err, closeErr)
	}
}

// The installed mage holding a staff takes class 24, bare-handed class 23
// (MAGIC-262). Each Lightning cast in eight directions leaves the class's
// launch point for the facing the cast turned him to (MAGIC-261, MAGIC-263).
func TestReleaseLightningLeavesTheStaffTipAndTheHandInEightDirections(t *testing.T) {
	dir := launchRenderDir(t)
	for _, c := range []struct {
		name   string
		staff  bool
		class  int32
		deltas [8]image.Point
		offset []int
	}{
		{"staff", true, 24, launchStaffDeltas, launchStaffOffsets},
		{"bare", false, 23, launchMageDeltas, launchMageOffsets},
	} {
		f, hero := launchWitness(t, c.staff)
		mw := f.live
		e, _ := mw.entity(hero)
		if got := mw.spellClientClass(hero, e.Class); got != c.class {
			t.Fatalf("%s: the mage takes class %d, want %d", c.name, got, c.class)
		}
		class := mw.casterClass(hero)
		if class == nil || class.CenterX != 64 || class.CenterY != 78 || fmt.Sprint(class.ShootOffset) != fmt.Sprint(c.offset) {
			t.Fatalf("%s: installed class geometry %+v differs from MAGIC-263", c.name, class)
		}
		for d, step := range launchDirections {
			b := launchObserve(t, mw, hero, launchFacings[d], step)
			_, _, points := mw.pathFigure(b)
			want := launchDisplay(mw, b.from, b.from.Mul(256).Add(c.deltas[d]))
			if len(points) == 0 || !withinPixel(points[0], want) {
				t.Fatalf("%s %s: the figure starts at %v, want %v", c.name, launchDirectionNames[d], points, want)
			}
			if dir != "" {
				mw.push()
				pix, _, err := mw.view.HeadlessMapFrame(b.from, 224, 224)
				if err != nil {
					t.Fatal(err)
				}
				launchWritePNG(t, dir, fmt.Sprintf("%s-%d-%s.png", c.name, d, launchDirectionNames[d]), pix)
			}
			t.Logf("%s %s: launch %v", c.name, launchDirectionNames[d], c.deltas[d])
		}
		b := launchCast(t, mw, hero)
		e, _ = mw.entity(hero)
		_, _, points := mw.pathFigure(b)
		if want := launchDisplay(mw, b.from, b.from.Mul(256).Add(c.deltas[(castLaunchPair(e.Facing)/2+4)%8])); len(points) == 0 || !withinPixel(points[0], want) {
			t.Fatalf("%s: an ordered cast at facing %d starts at %v, want %v", c.name, e.Facing, points, want)
		}
		t.Logf("%s: ordered cast at facing %d leaves %v", c.name, e.Facing, points[0])
	}
}

// unitScales is the sorted colour scales of the sprites drawn with the
// unit's frame, and the screen boxes of every sprite.
func unitScales(t *testing.T, mw *mapWorld, unit sim.EntityID, draws []ui.HeadlessArtDraw) (string, []image.Rectangle) {
	t.Helper()
	var actor ui.MapEntity
	for _, d := range mw.entityDraws() {
		if d.ID == uint32(unit) {
			actor = d
		}
	}
	var scales []float64
	var boxes []image.Rectangle
	for _, d := range draws {
		if d.Kind != "sprite" {
			continue
		}
		x0, y0 := d.Geometry.Apply(0, 0)
		x1, y1 := d.Geometry.Apply(float64(d.Pixels.Bounds().Dx()), float64(d.Pixels.Bounds().Dy()))
		boxes = append(boxes, image.Rect(int(min(x0, x1))-1, int(min(y0, y1))-1, int(max(x0, x1))+2, int(max(y0, y1))+2))
		if d.Frame == actor.Frame && actor.Frame != nil {
			scales = append(scales, float64(d.ColorScale.R()))
		}
	}
	if len(scales) == 0 {
		t.Fatal("the art pass drew no sprite of the unit")
	}
	sort.Float64s(scales)
	return fmt.Sprint(scales), boxes
}

func inAny(p image.Point, boxes []image.Rectangle) bool {
	for _, b := range boxes {
		if p.In(b) {
			return true
		}
	}
	return false
}

func lumaSum(pix *image.RGBA) int {
	sum := 0
	for i := 0; i < len(pix.Pix); i += 4 {
		sum += int(pix.Pix[i]) + int(pix.Pix[i+1]) + int(pix.Pix[i+2])
	}
	return sum
}

// A phase-0 Lightning path lights the ground and a unit in a stamped cell;
// with Dynamic lighting off only the unit (MAGIC-270, MAGIC-273).
func TestReleaseALightningBoltLightsTheGroundAndAUnitOnItsPath(t *testing.T) {
	dir := launchRenderDir(t)
	f, hero := launchWitness(t, true)
	mw := f.live
	b := launchCast(t, mw, hero)
	mw.push()
	stamps := mw.objectLightStamps(mw.world.EntityView())
	if len(stamps) == 0 {
		t.Fatal("the bolt stamps no light")
	}
	stamped := map[image.Point]bool{}
	for _, s := range stamps {
		if !s.Point && s.Level != 0 {
			t.Fatalf("a phase-0 path stamp at level %d", s.Level)
		}
		stamped[s.Vertex] = true
	}
	unit := sim.EntityID(0)
	for _, e := range mw.world.Entities() {
		c := image.Pt(int(e.X), int(e.Y))
		if e.Alive() && stamped[c] && stamped[c.Add(image.Pt(1, 0))] && stamped[c.Add(image.Pt(0, 1))] && stamped[c.Add(image.Pt(1, 1))] {
			unit = e.ID
			break
		}
	}
	if unit == 0 {
		t.Fatal("no living unit stands in a stamped path cell")
	}
	view := mw.view
	type shot struct {
		pix   *image.RGBA
		scale string
		boxes []image.Rectangle
	}
	frame := func(off, lit, sprites bool) shot {
		view.SetGraphicsOptions(ui.GraphicsOptions{DisableLighting: off})
		mw.push()
		if !lit {
			view.SetLightStamps(nil)
		}
		if !sprites {
			view.SetSpellBolts(nil)
		}
		pix, draws, err := view.HeadlessMapFrame(b.to, 224, 224)
		if err != nil {
			t.Fatal(err)
		}
		scale, boxes := unitScales(t, mw, unit, draws)
		return shot{pix, scale, boxes}
	}
	launchWritePNG(t, dir, "bolt-lit-dynamic-on.png", frame(false, true, true).pix)
	launchWritePNG(t, dir, "bolt-lit-dynamic-off.png", frame(true, true, true).pix)

	litOn, darkOn := frame(false, true, false), frame(false, false, false)
	if on, dark := lumaSum(litOn.pix), lumaSum(darkOn.pix); on <= dark {
		t.Errorf("Dynamic lighting on: lit ground sums %d, unlit %d", on, dark)
	}
	litOff, plainOff := frame(true, true, false), frame(true, false, false)
	if litOn.scale != litOff.scale || litOff.scale == plainOff.scale {
		t.Errorf("unit %d scales: lit on %v, lit off %v, unlit off %v", unit, litOn.scale, litOff.scale, plainOff.scale)
	}
	for y := 0; y < litOff.pix.Bounds().Dy(); y++ {
		for x := 0; x < litOff.pix.Bounds().Dx(); x++ {
			if litOff.pix.RGBAAt(x, y) != plainOff.pix.RGBAAt(x, y) && !inAny(image.Pt(x, y), litOff.boxes) {
				t.Fatalf("Dynamic lighting off: the bolt light changed ground pixel (%d,%d)", x, y)
			}
		}
	}
	view.SetGraphicsOptions(ui.GraphicsOptions{})
	t.Logf("unit %d scales lit %v unlit %v; ground luma lit %d unlit %d; %d stamps",
		unit, litOn.scale, plainOff.scale, lumaSum(litOn.pix), lumaSum(darkOn.pix), len(stamps))
}

func launchAbs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// launchDisplay is a launch point as the figure's native display pixel.
func launchDisplay(mw *mapWorld, cell, pos image.Point) image.Point {
	x, y := mw.boltDisplayPoint(pos, cell)
	return image.Pt(int(x), int(y))
}
