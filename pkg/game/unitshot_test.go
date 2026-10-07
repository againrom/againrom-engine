package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A ranged swing's own shot. The shooter's units.reg class names a
// projectiles.reg row and the swing tick the shot leaves on
// (ANIM-STATE-023); the shot draws that row's installed sheet while it flies,
// and the orange mark is left to a swing whose class names no projectile.
// Every fixture is loaded through the production loaders from a synthetic
// archive, so nothing here reads an install.

const (
	shotClassArcher = 14 // the arrow row, released on swing tick shotArcherDelay
	shotClassLate   = 15 // the arrow row, released after its own blow lands
	shotClassNow    = 16 // the arrow row, released in the swing's first tick
	shotClassBat    = 70 // row 7, whose draw arm blits no sheet
	shotClassPlain  = 1  // no projectile at all

	shotPictureArrow = 1
	shotPictureBat   = 7
	shotArcherDelay  = 2
	shotLateDelay    = 6
)

// shotArchive loads the fixture classes and projectile art: an indexed arrow
// sheet with no colour table of its own, drawn through the shared
// projectiles.pal, and a row 7 sheet carrying its own table.
func shotArchive(t *testing.T) (*terrain.UnitSet, *terrain.EffectSet) {
	t.Helper()
	i := func(name string, v int32) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x02, Int: v} }
	s := func(name, v string) synth.RegNode { return synth.RegNode{Name: name, Kind: 0x00, Str: v} }
	class := func(id, projectile, delay int32) []synth.RegNode {
		return []synth.RegNode{i("ID", id), i("Width", 16), i("Height", 16), i("CenterX", 8),
			i("CenterY", 14), i("Projectile", projectile), i("ShootDelay", delay)}
	}
	frames := func(n int) []synth.Frame256 {
		out := make([]synth.Frame256, n)
		for k := range out {
			out[k] = synth.Frame256{Width: 3, Height: 1, Pixels: []synth.Pixel256{
				{Index: 5, Opaque: true}, {Index: 5, Opaque: true}, {}}}
		}
		return out
	}
	own := make([]color.RGBA, 6)
	own[5] = color.RGBA{R: 0x80, G: 0x60, B: 0x40}
	shared := make(color.Palette, 256)
	for k := range shared {
		shared[k] = color.RGBA{A: 0xff}
	}
	shared[5] = color.RGBA{R: 0xc0, G: 0xc8, B: 0xd0, A: 0xff}
	src := sackGraphicsContainers(t, []synth.File{
		{Path: "units/units.reg", Data: synth.UnitsReg(nil,
			class(shotClassArcher, shotPictureArrow, shotArcherDelay),
			class(shotClassLate, shotPictureArrow, shotLateDelay),
			class(shotClassNow, shotPictureArrow, 0),
			class(shotClassBat, shotPictureBat, 2),
			class(shotClassPlain, 0, 0))},
		{Path: "projectiles/projectiles.reg", Data: synth.ProjectilesReg(2,
			[]synth.RegNode{i("ID", shotPictureArrow), s("File", `archer\arrow`), i("Phases", 1)},
			[]synth.RegNode{i("ID", shotPictureBat), s("File", `catap2\sprites`), i("Phases", 1),
				i("Palette", 1)})},
		{Path: "projectiles/archer/arrow.256", Data: synth.Sheet256(synth.Sheet256Options{
			NoPalette: true, Frames: frames(16)})},
		{Path: "projectiles/catap2/sprites.256", Data: synth.Sheet256(synth.Sheet256Options{
			Palette: own, Frames: frames(16)})},
		{Path: "projectiles/projectiles.pal", Data: synth.BMP8(image.NewPaletted(image.Rect(0, 0, 1, 1), shared))},
	})
	units, err := LoadUnits(src)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	set, err := LoadProjectiles(src)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	return units, set
}

// shotWorld is a driver over a hand-built world with the loaded bundles.
func shotWorld(t *testing.T, units *terrain.UnitSet, set *terrain.EffectSet, spells []sim.SpellRule,
	ents ...sim.Entity) *mapWorld {
	t.Helper()
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil, ents, nil, spells)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return shotMapWorld(t, w, units, set)
}

// shotMapWorld is a driver over w with the loaded bundles.
func shotMapWorld(t *testing.T, w *sim.World, units *terrain.UnitSet, set *terrain.EffectSet) *mapWorld {
	t.Helper()
	v, err := ui.NewViewer("shot", terrain.Grid{
		Width: swingW, Height: swingH, Tiles: make([]uint16, swingW*swingH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := newMapWorld(w, nil, units, v)
	mw.projectiles = set
	return mw
}

// shotShooter has reach 4 and deals nothing, so its victim outlives every
// blow and the order stands for the whole run.
func shotShooter(id sim.EntityID, class int32, x, y int32, charge int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: class, HP: 200, MaxHP: 200, Owner: 1,
		Reach: 4, AttackCharge: charge, AttackRelax: 8}
}

func shotVictim(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: shotClassPlain, HP: 200, MaxHP: 200, Owner: 2}
}

// shotFlight is one drawn tick of the shooter's own shot.
type shotFlight struct {
	swing int
	phase sim.AttackPhase
	bolt  ui.SpellBolt
}

// flyOneShot orders entity 1 onto entity 2 and ticks until the first flight
// of sheet has come and gone, failing on any tick that hands the viewer the
// orange mark for the shooter.
func flyOneShot(t *testing.T, mw *mapWorld, sheet *terrain.EffectSheet) []shotFlight {
	t.Helper()
	mw.strike(1, 2)
	var out []shotFlight
	for tick := 0; tick < 64; tick++ {
		mw.tick()
		if d := swingDraw(t, mw, 1); d.Shot != nil {
			t.Errorf("tick %d (swing %d): the shooter carries the orange mark at %v", tick, mw.swing[1], *d.Shot)
		}
		drawn := false
		for _, b := range mw.boltDraws(mw.world.EntityView()) {
			if sheet != nil && b.Sheet == sheet {
				e, _ := mw.entity(1)
				out = append(out, shotFlight{swing: mw.swing[1], phase: e.AttackPhase, bolt: b})
				drawn = true
			}
		}
		if !drawn && len(out) > 0 {
			break
		}
	}
	return out
}

// TestARangedSwingDrawsItsClassShotFromTheInstalledSheet: an archer four
// cells west of its victim draws the arrow row's sheet from the swing tick
// its class names, for the ftol(distance)/200 ticks the shot lives
// (ANIM-PROJ-025), at the east facing (ANIM-PROJ-026), arriving on the
// victim's cell; and it never draws the orange mark.
func TestARangedSwingDrawsItsClassShotFromTheInstalledSheet(t *testing.T) {
	units, set := shotArchive(t)
	arrow := set.Sheet(shotPictureArrow)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassArcher, 1, 4, 8), shotVictim(2, 5, 4))
	flight := flyOneShot(t, mw, arrow)
	if arrow == nil {
		t.Fatal("the arrow row, indexed and palette-less, built no sheet")
	}
	const life = 4 * 256 / 200
	if len(flight) != life {
		t.Fatalf("the arrow was drawn on %d ticks, want %d: %+v", len(flight), life, flight)
	}
	for k, f := range flight {
		// Built on the swing count ShootDelay less one (SAV-1153).
		if f.swing != shotArcherDelay-1+k {
			t.Errorf("flight tick %d stands at swing %d, want %d", k, f.swing, shotArcherDelay-1+k)
		}
		if f.bolt.Frame != terrain.EffectFacing(4, 0) || f.bolt.Mirror {
			t.Errorf("flight tick %d draws frame %d mirror %t, want the east facing %d",
				k, f.bolt.Frame, f.bolt.Mirror, terrain.EffectFacing(4, 0))
		}
		if at := image.Pt(f.bolt.Pos.X/ui.ShotScale, f.bolt.Pos.Y/ui.ShotScale); f.bolt.Cell != at || f.bolt.To != at {
			t.Errorf("flight tick %d is drawn on cell %v to %v at %v, want the record's own cell", k, f.bolt.Cell, f.bolt.To, f.bolt.Pos)
		}
		if k > 0 && f.bolt.Pos.X <= flight[k-1].bolt.Pos.X {
			t.Errorf("flight tick %d stands at x %d after %d: it does not advance", k, f.bolt.Pos.X, flight[k-1].bolt.Pos.X)
		}
	}
	if last := flight[len(flight)-1].bolt.Pos; last != image.Pt(5*ui.ShotScale+ui.ShotScale/2, 4*ui.ShotScale+ui.ShotScale/2) {
		t.Errorf("the arrow's last drawn point is %v, want the victim's point", last)
	}
}

// TestAShotLeavingAfterItsBlowStillFliesToItsTarget: a class whose swing
// releases the shot after the countdown has struck keeps drawing it through
// the relax, as a projectile object outlives the swing that spawned it.
func TestAShotLeavingAfterItsBlowStillFliesToItsTarget(t *testing.T) {
	units, set := shotArchive(t)
	arrow := set.Sheet(shotPictureArrow)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassLate, 1, 4, 4), shotVictim(2, 5, 4))
	flight := flyOneShot(t, mw, arrow)
	if arrow == nil {
		t.Fatal("the arrow row built no sheet")
	}
	const life = 4 * 256 / 200
	if len(flight) != life {
		t.Fatalf("the arrow was drawn on %d ticks, want %d: %+v", len(flight), life, flight)
	}
	relaxed := 0
	for k, f := range flight {
		if f.swing != shotLateDelay-1+k {
			t.Errorf("flight tick %d stands at swing %d, want %d", k, f.swing, shotLateDelay-1+k)
		}
		if f.phase != sim.AttackCharging {
			relaxed++
		}
	}
	if relaxed == 0 {
		t.Errorf("no flight tick fell after the blow; the fixture does not test the relax: %+v", flight)
	}
}

// firstRecord ticks mw's shooter until its first World record exists and
// answers it with the swing count that tick stands at.
func firstRecord(t *testing.T, mw *mapWorld) (sim.SavedProjectile, int) {
	t.Helper()
	mw.strike(1, 2)
	for tick := 0; tick < 64; tick++ {
		mw.tick()
		if items := mw.world.SavedProjectiles().Items; len(items) > 0 {
			return items[0], mw.swing[1]
		}
	}
	t.Fatal("no record was built")
	return sim.SavedProjectile{}, 0
}

// A ShootDelay 0 record has made two driver calls when it is built, a record
// with a delay one (SAV-1153); each case controls the other.
func TestAShotRecordHasMadeTheDriverCallsOfItsCreationTick(t *testing.T) {
	units, set := shotArchive(t)
	for _, tc := range []struct {
		name  string
		class int32
		swing int
		calls int32
	}{
		{"delay 0", shotClassNow, 0, 2},
		{"delay 2", shotClassArcher, shotArcherDelay - 1, 1},
	} {
		mw := shotWorld(t, units, set, nil, shotShooter(1, tc.class, 1, 4, 8), shotVictim(2, 5, 4))
		p, swing := firstRecord(t, mw)
		if swing != tc.swing || p.ActionPhase != tc.calls || p.ActionSegments != 5-tc.calls {
			t.Errorf("%s: record built at swing %d with phase %d and %d segments left, want swing %d, phase %d, %d left",
				tc.name, swing, p.ActionPhase, p.ActionSegments, tc.swing, tc.calls, 5-tc.calls)
		}
	}
}

// A wind-up restored mid-run counts the native swing clock and builds its shot
// on the same tick; the action clock counts from the original's swing start,
// one tick before the native zero.
func TestARestoredWindUpCountsTheNativeSwingClockAndBuildsItsShotOnTheSameTick(t *testing.T) {
	units, set := shotArchive(t)
	shooter := shotShooter(1, shotClassLate, 1, 4, 8)
	shooter.HealthRegenPeriod, shooter.ManaRegenPeriod = 100, 100
	native := shotWorld(t, units, set, nil, shooter, shotVictim(2, 5, 4))
	native.strike(1, 2)
	for native.swing[1] < 3 {
		native.tick()
	}
	raw, err := native.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold sim.World
	if err := cold.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	restored := shotMapWorld(t, &cold, units, set)
	restored.seedRestoredRuns()
	if restored.swing[1] != native.swing[1] {
		t.Fatalf("restored swing %d, native %d", restored.swing[1], native.swing[1])
	}
	built := func(mw *mapWorld) int {
		for k := 1; k < 32; k++ {
			mw.tick()
			if len(mw.world.SavedProjectiles().Items) > 0 {
				return k
			}
		}
		return -1
	}
	if n, r := built(native), built(restored); n != shotLateDelay-1-3 || r != n {
		t.Errorf("shot built %d ticks on after the sample natively and %d restored, want %d for both", n, r, shotLateDelay-1-3)
	}
}

func TestABatShotTravelsWithoutSheetOrMark(t *testing.T) {
	units, set := shotArchive(t)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassBat, 1, 4, 8), shotVictim(2, 5, 4))
	mw.strike(1, 2)
	charged := false
	var flight []ui.SpellBolt
	for tick := 0; tick < 24; tick++ {
		mw.tick()
		if e, _ := mw.entity(1); e.AttackPhase == sim.AttackCharging {
			charged = true
		}
		if d := swingDraw(t, mw, 1); d.Shot != nil {
			t.Errorf("tick %d (swing %d): the bat carries the orange mark at %v", tick, mw.swing[1], *d.Shot)
		}
		for _, b := range mw.boltDraws(mw.world.EntityView()) {
			if b.Sheet != nil || b.Effect != ui.SpellBackgroundDeformation {
				t.Errorf("tick %d: the bat drew a sprite", tick)
			}
			flight = append(flight, b)
		}
		if len(flight) > 0 && len(mw.world.SavedProjectiles().Items) == 0 {
			break
		}
	}
	if !charged {
		t.Fatal("the bat never charged; the fixture does not reach a shot")
	}
	if len(flight) != 4*256/200 {
		t.Fatalf("the bat has %d travelling draws, want 5", len(flight))
	}
	for i := 1; i < len(flight); i++ {
		if flight[i].Pos.X <= flight[i-1].Pos.X {
			t.Fatal("the bat effect did not travel", flight)
		}
	}
}

// TestAClassNamingNoProjectileKeepsTheMark is the fallback's own control: a
// ranged swing whose class names no projectile still carries the mark while
// it charges, and draws no sheet.
func TestAClassNamingNoProjectileKeepsTheMark(t *testing.T) {
	units, set := shotArchive(t)
	mw := shotWorld(t, units, set, nil, shotShooter(1, shotClassPlain, 1, 4, 8), shotVictim(2, 5, 4))
	mw.strike(1, 2)
	marked := false
	for tick := 0; tick < 24; tick++ {
		mw.tick()
		e, _ := mw.entity(1)
		if d := swingDraw(t, mw, 1); d.Shot != nil && e.AttackPhase == sim.AttackCharging {
			marked = true
		}
		for _, b := range mw.boltDraws(mw.world.EntityView()) {
			if b.Cell == image.Pt(1, 4) {
				t.Errorf("tick %d: a class naming no projectile draws %+v", tick, b)
			}
		}
	}
	if !marked {
		t.Error("a charging ranged swing whose class names no projectile carried no mark")
	}
}

// TestAWeaponBorneCastCarriesNoMark: a staff that casts Fire Arrow winds up in
// the casting phase, whose picture is the spell's own (weaponBoltDraws). No
// tick of that run, its relax included, carries the orange mark, whatever the
// caster's class names, and the class's own shot is not released by it.
func TestAWeaponBorneCastCarriesNoMark(t *testing.T) {
	units, set := shotArchive(t)
	fireArrow := []sim.SpellRule{{ID: 1, School: 1, MaxRange: 5, TargetsUnit: true, Damaging: true}}
	for _, class := range []int32{shotClassPlain, shotClassArcher} {
		caster := shotShooter(1, class, 1, 4, 8)
		caster.MaxMana, caster.Mana, caster.WeaponSpell, caster.WeaponSpellLevel = 50, 50, 1, 1
		caster.ScanRange = 8
		mw := shotWorld(t, units, set, fireArrow, caster, shotVictim(2, 5, 4))
		mw.strike(1, 2)
		cast := false
		for tick := 0; tick < 24; tick++ {
			mw.tick()
			if e, _ := mw.entity(1); e.AttackPhase == sim.AttackCasting {
				cast = true
			}
			if d := swingDraw(t, mw, 1); d.Shot != nil {
				t.Errorf("class %d tick %d (swing %d): the caster carries the orange mark at %v",
					class, tick, mw.swing[1], *d.Shot)
			}
			for _, b := range mw.boltDraws(mw.world.EntityView()) {
				if b.Sheet == set.Sheet(shotPictureArrow) {
					t.Errorf("class %d tick %d: a casting run released the class's own shot", class, tick)
				}
			}
		}
		if !cast {
			t.Fatalf("class %d never wound up a cast; the fixture does not reach one", class)
		}
	}
}

// A delay 0 wind-up restored on its start tick builds no second record.
func TestARestoredDelayZeroWindUpBuildsNoSecondRecord(t *testing.T) {
	units, set := shotArchive(t)
	shooter := shotShooter(1, shotClassNow, 1, 4, 8)
	shooter.HealthRegenPeriod, shooter.ManaRegenPeriod = 100, 100
	native := shotWorld(t, units, set, nil, shooter, shotVictim(2, 5, 4))
	native.strike(1, 2)
	native.tick()
	// The sample on the swing start tick: zero elapsed.
	e, _ := native.entity(1)
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: swingW, Height: swingH}, sim.ModeCanonical, nil,
		[]sim.Entity{e, shotVictim(2, 5, 4)}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !w.ReleaseUnitShot(sim.UnitShot{Shooter: 1, Picture: shotPictureArrow, Phases: 1, Dir: shotDirection, Late: 1}) {
		t.Fatal("record not released")
	}
	restored := shotMapWorld(t, w, units, set)
	restored.seedRestoredRuns()
	el, ok := restored.world.WindUpElapsed(1)
	t.Logf("elapsed %d %v swing %d", el, ok, restored.swing[1])
	restored.tick()
	if got := len(restored.world.SavedProjectiles().Items); got != 1 {
		t.Errorf("restored world holds %d records after one tick (swing %d), want 1", got, restored.swing[1])
	}
}

// A unit shot's damage tick minus its record's creation tick is
// charge - ShootDelay + floor((d*256+128)/200) for a distance d above 1, and
// charge - ShootDelay at d = 1 (SAV-1157). The expected value is worked from
// the claim's formula; each class and distance is a native run.
func TestAUnitShotOffsetFollowsChargeShootDelayAndDistance(t *testing.T) {
	units, set := shotArchive(t)
	const charge = 8
	for _, class := range []struct {
		id    int32
		delay int
	}{{shotClassArcher, shotArcherDelay}, {shotClassLate, shotLateDelay}} {
		for d := int32(1); d <= 6; d++ {
			shooter := shotShooter(1, class.id, 1, 4, charge)
			shooter.Reach = 8
			shooter.DamageBase, shooter.AlwaysHits = 10, true
			shooter.HealthRegenPeriod, shooter.ManaRegenPeriod = 100, 100
			mw := shotWorld(t, units, set, nil, shooter, shotVictim(2, 1+d, 4))
			mw.strike(1, 2)
			created, damaged := -1, -1
			for tick := 0; tick < 64 && (created < 0 || damaged < 0); tick++ {
				mw.tick()
				if created < 0 && len(mw.world.SavedProjectiles().Items) > 0 {
					created = tick
				}
				if v, _ := mw.entity(2); damaged < 0 && v.HP < 200 {
					damaged = tick
				}
			}
			if created < 0 || damaged < 0 {
				t.Fatalf("class %d d %d: record tick %d, damage tick %d", class.id, d, created, damaged)
			}
			want := charge - class.delay
			if d > 1 {
				want += int((d*256 + 128) / 200)
			}
			if got := damaged - created; got != want {
				t.Errorf("class %d (ShootDelay %d) d %d: damage %d ticks after the record, want %d", class.id, class.delay, d, got, want)
			}
		}
	}
}
