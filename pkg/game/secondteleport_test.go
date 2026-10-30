package game

import (
	"image"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The second game's Teleport is row 23 and picture 54, 21 phases of 96 x 96
// (R2-ENGINE-445, R2-ASSET-101). The mage is units.reg type 24 and the
// creature type 71, with their AttackAnimTime sums, AttackDelay, Center and
// Selection (R2-ASSET-180). Both casters wind up for 8 ticks.
const (
	secondTeleportRow     = 23
	secondTeleportPicture = 54
	secondTeleportPhases  = 21
	secondTeleportSound   = 554
)

type secondTeleportClass struct {
	id, run, delay   int
	centerX, centerY int
	selection        image.Rectangle
}

var (
	secondMageClass     = secondTeleportClass{id: 24, run: 12, delay: 8, centerX: 64, centerY: 78, selection: image.Rect(48, 48, 80, 90)}
	secondCreatureClass = secondTeleportClass{id: 71, run: 14, delay: 8, centerX: 80, centerY: 80, selection: image.Rect(32, 32, 128, 128)}
)

// anchor is the claimed image offset: Center minus half Selection, in fine
// units (R2-ENGINE-703).
func (c secondTeleportClass) anchor() image.Point {
	return image.Pt(c.selection.Dx()/2-c.centerX, c.selection.Dy()/2-c.centerY)
}

func secondTeleportArt(c secondTeleportClass) *terrain.UnitClass {
	const attackBase = 24
	track := make([]int, c.run)
	for i := range track {
		track[i] = i
	}
	art := worldFixtureArt(128, 128, c.centerX, c.centerY, 4, 4, attackBase+8*c.run+8)
	art.Selection, art.TileSize = c.selection, 1
	art.Anim = terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 0, AttackBase: attackBase, DyingBase: attackBase + 8*c.run, TailBase: 8,
		MoveSlot: 1, IdleSlot: 1, AttackSlot: 6, Total: attackBase + 8*c.run + 8,
		MoveTrack: []int{0}, MoveOK: true,
		IdleTrack: []int{0}, IdleOK: true,
		AttackTrack: track, AttackOK: true}
	art.Corpse = art
	return art
}

// secondTeleportWorld is a second-game map with one caster of class c at
// (8,8), a Teleport row of Delivery 1 as the installed row (R2-ENGINE-705)
// and the caster's Teleport order to (12,8), whose cell blocked closes.
func secondTeleportWorld(t *testing.T, c secondTeleportClass, humanoid, blocked bool) *mapWorld {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: int32(c.id), HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << secondTeleportRow,
		AttackCharge: tpCharge, AttackRelax: 4, ScanRange: 30, TokenSize: 1, Humanoid: humanoid}
	block := make([]byte, 32*32)
	if blocked {
		block[8*32+12] = 1
	}
	rules := []sim.SpellRule{{ID: secondTeleportRow, ManaCost: 3, School: 5, MaxRange: 20, Delivery: 1}}
	mapload.SecondGameSpellArms(rules)
	script, err := sim.NewROM2Script(missionChecks(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		sim.Terrain{Block: block}, []sim.Entity{caster}, script, sim.Relations{}, nil, nil, rules)
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	v, err := ui.NewViewer("second-teleport", terrain.Grid{Width: 32, Height: 32, Tiles: make([]uint16, 32*32)}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{int32(c.id): secondTeleportArt(c)}}, v)
	edition := base.GameROM2.Edition()
	mw.mission = &missionNotices{table: &mapload.Table{Edition: &edition, SpellArms: mapload.SecondGameSpellArms}}
	mw.sounds = map[int32]UnitSound{int32(c.id): {AttackDelay: int32(c.delay)}}
	mw.projectiles = &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{
		secondTeleportPicture: {Frames: spFrames(secondTeleportPhases, 96), Phases: secondTeleportPhases, RotationPhases: 1, CenterX: 48, CenterY: 48},
	}, Pictures: map[int]bool{secondTeleportPicture: true}}
	mw.sched = [][]sim.Command{{sim.CastAt(1, secondTeleportRow, sim.CellPoint{X: 12, Y: 8})}}
	return mw
}

type secondTeleportTick struct {
	casts   int
	cell    image.Point
	run     int // the cast run's step, -1 outside it
	records []sim.SavedProjectile
	sounds  []int
	mana    int32
}

func runSecondTeleport(t *testing.T, mw *mapWorld, n int) []secondTeleportTick {
	t.Helper()
	var heard []int
	mw.spellSound = func(slot int, _ uint32, _ image.Point) { heard = append(heard, slot) }
	var out []secondTeleportTick
	for range n {
		heard = nil
		var casts []sim.CastEvent
		mw.tickWithCastSink(func(events []sim.CastEvent) { casts = events })
		e, ok := mw.entity(1)
		if !ok {
			t.Fatal("caster left the world")
		}
		run := -1
		if mw.casting(1) {
			run = mw.swing[1]
		}
		out = append(out, secondTeleportTick{casts: len(casts), cell: image.Pt(int(e.X), int(e.Y)), run: run,
			records: flightRecords(mw), sounds: heard, mana: e.Mana})
	}
	return out
}

// checkSecondTeleport is R2-ENGINE-703, -704 and -705 for one cast admitted
// on tick S: two picture-54 records at the caster's point and the target cell
// centre, each moved by Center minus half Selection, standing still and
// drawing frame k on S+k for 21 updates; the caster's run step k on S+k for
// its class's attack list; Sfx 554 once, on S+AttackDelay; the move on S+8,
// the wind-up after the cast routine, unless the cell is refused
// (R2-ENGINE-706).
func checkSecondTeleport(t *testing.T, c secondTeleportClass, ticks []secondTeleportTick, moves bool, mana int32) {
	t.Helper()
	start, dest := image.Pt(8, 8), image.Pt(12, 8)
	if ticks[0].casts != 1 {
		t.Fatalf("the admitting tick reported %d casts, want the one Teleport", ticks[0].casts)
	}
	off := c.anchor()
	want := [2]image.Point{
		image.Pt(start.X*256+128+off.X, start.Y*256+128+off.Y),
		image.Pt(dest.X*256+128+off.X, dest.Y*256+128+off.Y),
	}
	sounded := []int{}
	for k, tk := range ticks {
		if k > 0 && tk.casts != 0 {
			t.Errorf("S+%d reported %d more casts", k, tk.casts)
		}
		if tk.mana != mana {
			t.Errorf("S+%d: mana %d, want %d paid at the cast", k, tk.mana, mana)
		}
		wantCell := start
		if moves && k >= tpCharge {
			wantCell = dest
		}
		if tk.cell != wantCell {
			t.Errorf("S+%d: caster on %v, want %v", k, tk.cell, wantCell)
		}
		wantRun := k
		if k >= c.run {
			wantRun = -1
		}
		if tk.run != wantRun {
			t.Errorf("S+%d: run step %d, want %d", k, tk.run, wantRun)
		}
		for range tk.sounds {
			sounded = append(sounded, k)
		}
		for _, s := range tk.sounds {
			if s != secondTeleportSound {
				t.Errorf("S+%d: sound slot %d, want %d", k, s, secondTeleportSound)
			}
		}
		if k >= secondTeleportPhases {
			if len(tk.records) != 0 {
				t.Errorf("S+%d: %d records past the 21 updates", k, len(tk.records))
			}
			continue
		}
		if len(tk.records) != 2 {
			t.Fatalf("S+%d: %d records, want the two type-54 images", k, len(tk.records))
		}
		for i, r := range tk.records {
			if r.Picture != secondTeleportPicture || image.Pt(int(r.X), int(r.Y)) != want[i] || r.Phase != int32(k) {
				t.Errorf("S+%d record %d: picture %d at (%d,%d) frame %d; want %d at %v frame %d",
					k, i, r.Picture, r.X, r.Y, r.Phase, secondTeleportPicture, want[i], k)
			}
		}
	}
	if len(sounded) != 1 || sounded[0] != c.delay {
		t.Errorf("Sfx 554 sounded on S+%v, want once on S+%d", sounded, c.delay)
	}
}

func TestASecondGameHeroTeleportDrawsSoundsAndMovesOnTheOriginalsClock(t *testing.T) {
	mw := secondTeleportWorld(t, secondMageClass, true, false)
	checkSecondTeleport(t, secondMageClass, runSecondTeleport(t, mw, 24), true, 97)
}

func TestASecondGameCreatureTeleportTakesItsOwnTypesRunAndOffset(t *testing.T) {
	// The creature holds a mana pool, so the mage test of R2-ENGINE-544 passes
	// and it pays as the hero does (R2-ENGINE-707).
	mw := secondTeleportWorld(t, secondCreatureClass, false, false)
	checkSecondTeleport(t, secondCreatureClass, runSecondTeleport(t, mw, 24), true, 97)
}

func TestASecondGameTeleportToARefusedCellKeepsManaImagesAndSound(t *testing.T) {
	mw := secondTeleportWorld(t, secondMageClass, true, true)
	checkSecondTeleport(t, secondMageClass, runSecondTeleport(t, mw, 24), false, 97)
}

// The first game's Teleport keeps its profile: the Attack run and the cast
// message's sound beside the run's hook (ANIM-TELEPORT-172); its records,
// run, sounds and relocation tick are
// TestTeleportIsPresentedAtAdmissionAndMovesOnTheCountdown's.
func TestTheTeleportCasterProfilesAreGameData(t *testing.T) {
	if got, want := firstGameFiles.teleportCast, (teleportCastProfile{run: true, messageSound: true}); got != want {
		t.Errorf("first game's Teleport caster %+v, want %+v", got, want)
	}
	if got, want := secondGameFiles.teleportCast, (teleportCastProfile{run: true}); got != want {
		t.Errorf("second game's Teleport caster %+v, want %+v", got, want)
	}
}
