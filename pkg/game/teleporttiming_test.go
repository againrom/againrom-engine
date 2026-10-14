package game

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The Teleport fixture: a caster at (8,8) with charge 8, AttackDelay 8 and a
// twelve-step Attack timeline, the values SAV-1254 and MAGIC-TELEPORT-371
// read for the hero mage classes; the cast goes east to (12,8).
const (
	tpAttackBase = 24
	tpAttackRun  = 12
	tpEast       = 6
	tpCharge     = 8
)

func teleportTimingAnim() terrain.UnitAnim {
	track := make([]int, tpAttackRun)
	for i := range track {
		track[i] = i
	}
	return terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 0, AttackBase: tpAttackBase, DyingBase: tpAttackBase + 8*tpAttackRun, TailBase: 8,
		MoveSlot: 1, IdleSlot: 1, AttackSlot: tpAttackRun, Total: tpAttackBase + 8*tpAttackRun + 8,
		MoveTrack: []int{0}, MoveOK: true,
		IdleTrack: []int{0}, IdleOK: true,
		AttackTrack: track, AttackOK: true}
}

func teleportTimingWorld(t *testing.T, blocked bool) *mapWorld {
	t.Helper()
	return teleportTimingWorldTo(t, blocked, 0, image.Pt(12, 8))
}

// teleportTimingWorldTo is the fixture with the caster's turn rate and the
// cast's destination named; the caster faces north (facing 0).
func teleportTimingWorldTo(t *testing.T, blocked bool, rate int32, to image.Point) *mapWorld {
	t.Helper()
	caster := sim.Entity{ID: 1, X: 8, Y: 8, Class: 1, HP: 100, MaxHP: 100,
		Owner: 7, Mind: 40, Mana: 100, MaxMana: 100, KnownSpells: 1 << 26,
		AttackCharge: tpCharge, AttackRelax: 4, ScanRange: 30, TokenSize: 1, RotationSpeed: rate}
	block := make([]byte, 32*32)
	if blocked {
		block[8*32+12] = 1
	}
	rule := sim.SpellRule{ID: 26, ManaCost: 3, School: 5, MaxRange: 20}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical,
		sim.Terrain{Block: block}, []sim.Entity{caster}, nil, sim.Relations{}, nil, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	c := worldFixtureArt(16, 16, 8, 14, 4, 4, tpAttackBase+8*tpAttackRun+8)
	c.Anim = teleportTimingAnim()
	c.Corpse = c
	v, err := ui.NewViewer("teleport-timing", terrain.Grid{
		Width: 32, Height: 32, Tiles: make([]uint16, 32*32),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	mw := newMapWorld(w, nil, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: c}}, v)
	mw.sounds = map[int32]UnitSound{1: {AttackDelay: tpCharge}}
	mw.projectiles = sbSheets()
	mw.sched = [][]sim.Command{{sim.CastAt(1, 26, sim.CellPoint{X: int32(to.X), Y: int32(to.Y)})}}
	return mw
}

// teleportTick is one production tick and what it reports.
type teleportTick struct {
	casts   []sim.CastEvent
	cell    image.Point
	step    int // the attack step drawn, -1 for none
	run     int // the Attack run's step, -1 outside a cast run
	records []sim.SavedProjectile
}

func runTeleportTicks(t *testing.T, mw *mapWorld, n int) []teleportTick {
	t.Helper()
	art := mw.units.Classes[1]
	var out []teleportTick
	for range n {
		var casts []sim.CastEvent
		mw.tickWithCastSink(func(events []sim.CastEvent) { casts = events })
		e, ok := mw.entity(1)
		if !ok {
			t.Fatal("caster left the world")
		}
		d := swingDraw(t, mw, 1)
		step := -1
		for dir := range 8 {
			for s := range tpAttackRun {
				if d.Frame == art.Frames[tpAttackBase+dir*tpAttackRun+s] {
					step = s
				}
			}
		}
		run := -1
		if mw.casting(1) {
			run = mw.swing[1]
		}
		out = append(out, teleportTick{casts: casts, cell: image.Pt(int(e.X), int(e.Y)), step: step, run: run, records: flightRecords(mw)})
	}
	return out
}

// TestTeleportIsPresentedAtAdmissionAndMovesOnTheCountdown: tick S reports the
// cast, builds both records and starts the Attack run; the caster moves on
// S+8, drawing steps 0..7 at the start and 8..11 at the destination; the
// records draw phase k on S+k in place; the AttackDelay cue sounds on S+8 at
// the destination (MAGIC-TELEPORT-370, -371, -358, ANIM-TELEPORT-172).
func TestTeleportIsPresentedAtAdmissionAndMovesOnTheCountdown(t *testing.T) {
	mw := teleportTimingWorld(t, false)
	plays := recordSpellSounds(mw)
	ticks := runTeleportTicks(t, mw, 14)
	start, dest := image.Pt(8, 8), image.Pt(12, 8)
	if len(ticks[0].casts) != 1 || ticks[0].casts[0].FromX != 8 || ticks[0].casts[0].ToX != 12 {
		t.Fatalf("admitting tick reported %+v, want the one Teleport from (8,8) to (12,8)", ticks[0].casts)
	}
	if len(ticks[0].records) != 2 {
		t.Fatalf("admitting tick holds %d records, want both picture-60 records", len(ticks[0].records))
	}
	points := [2]image.Point{}
	for i, r := range ticks[0].records {
		points[i] = image.Pt(int(r.X)/256, int(r.Y)/256)
	}
	if points != [2]image.Point{start, dest} {
		t.Fatalf("the records stand in cells %v, want the start %v and the destination %v", points, start, dest)
	}
	for k, tk := range ticks {
		if k > 0 && len(tk.casts) != 0 {
			t.Errorf("tick S+%d reported a second cast %+v", k, tk.casts)
		}
		wantCell, wantStep := start, k
		if k >= tpCharge {
			wantCell = dest
		}
		if k >= tpAttackRun {
			wantStep = -1
		}
		if tk.cell != wantCell {
			t.Errorf("tick S+%d: caster on %v, want %v", k, tk.cell, wantCell)
		}
		if tk.step != wantStep {
			t.Errorf("tick S+%d: drawn attack step %d, want %d", k, tk.step, wantStep)
		}
		if k <= 20 && len(tk.records) == 2 {
			for i, r := range tk.records {
				if r.Phase != int32(k) || r.ActionPhase+r.ActionSegments != 21 {
					t.Errorf("tick S+%d record %d: phase %d, actionphase %d, remaining %d; want phase %d and 21 calls", k, i, r.Phase, r.ActionPhase, r.ActionSegments, k)
				}
				if image.Pt(int(r.X)/256, int(r.Y)/256) != points[i] {
					t.Errorf("tick S+%d record %d moved to (%d,%d)", k, i, r.X, r.Y)
				}
			}
		} else if k <= 13 {
			t.Errorf("tick S+%d holds %d records, want 2", k, len(tk.records))
		}
	}
	if len(*plays) != 2 || (*plays)[0].cell != start || (*plays)[1].cell != dest {
		t.Fatalf("sounds %+v, want the cast message at the start and the AttackDelay cue at the destination", *plays)
	}
	if e, _ := mw.entity(1); e.Mana != 97 {
		t.Fatalf("mana %d, want one payment of 3", e.Mana)
	}
}

// TestABlockedTeleportShowsBothRecordsAndPaysWithoutMoving is MAGIC-TELEPORT-174
// and MAGIC-TELEPORT-356: the destination veto falls at the relocation, after
// the admission built both records and the cost was paid.
func TestABlockedTeleportShowsBothRecordsAndPaysWithoutMoving(t *testing.T) {
	mw := teleportTimingWorld(t, true)
	ticks := runTeleportTicks(t, mw, 12)
	if len(ticks[0].casts) != 1 || len(ticks[0].records) != 2 {
		t.Fatalf("a blocked Teleport reported %d casts and %d records on admission, want 1 and 2", len(ticks[0].casts), len(ticks[0].records))
	}
	for k, tk := range ticks {
		if tk.cell != image.Pt(8, 8) {
			t.Fatalf("tick S+%d: a blocked Teleport moved its caster to %v", k, tk.cell)
		}
	}
	if e, _ := mw.entity(1); e.Mana != 97 {
		t.Fatalf("mana %d, want the cost of 3 paid", e.Mana)
	}
}

// TestATurningTeleportDrawsItsRunAcrossTheMove is MOVE-107, MAGIC-340,
// MAGIC-TELEPORT-370 and ANIM-TELEPORT-172 for a caster that turns first: for
// a destination ahead, to the side and behind a caster with turn rate 22, the
// tick P that reports the cast builds both records and starts the Attack run;
// the caster moves on P+8, where the records draw phase 8 and the run step 8.
// On P a caster that turned still draws its turn's last frame.
func TestATurningTeleportDrawsItsRunAcrossTheMove(t *testing.T) {
	start := image.Pt(8, 8)
	for _, to := range []image.Point{{8, 4}, {12, 4}, {12, 8}, {12, 12}, {8, 12}, {4, 12}} {
		mw := teleportTimingWorldTo(t, false, 22, to)
		ticks := runTeleportTicks(t, mw, 30)
		presented := -1
		for k, tk := range ticks {
			if len(tk.casts) == 0 {
				continue
			}
			if presented >= 0 || len(tk.casts) != 1 {
				t.Fatalf("to %v: casts %+v on S+%d after the cast on S+%d", to, tk.casts, k, presented)
			}
			presented = k
		}
		if presented < 0 {
			t.Fatalf("to %v: no cast reported", to)
		}
		for k, tk := range ticks {
			wantCell, run := start, k-presented
			if run >= tpCharge {
				wantCell = to
			}
			if tk.cell != wantCell {
				t.Errorf("to %v: tick S+%d (P+%d): caster on %v, want %v", to, k, run, tk.cell, wantCell)
			}
			if run < 0 {
				if len(tk.records) != 0 || tk.step != -1 || tk.run != -1 {
					t.Errorf("to %v: tick S+%d, before the cast: %d records, attack step %d, run %d", to, k, len(tk.records), tk.step, tk.run)
				}
				continue
			}
			wantStep := run
			if run >= tpAttackRun {
				wantStep = -1
			}
			if tk.run != wantStep {
				t.Errorf("to %v: tick P+%d: Attack run step %d, want %d", to, run, tk.run, wantStep)
			}
			if tk.step != wantStep && (run > 0 || presented == 0 && to.X == start.X) {
				t.Errorf("to %v: tick P+%d: drawn attack step %d, want %d", to, run, tk.step, wantStep)
			}
			if run <= 20 {
				if len(tk.records) != 2 {
					t.Fatalf("to %v: tick P+%d holds %d records, want 2", to, run, len(tk.records))
				}
				for i, r := range tk.records {
					if r.Phase != int32(run) {
						t.Errorf("to %v: tick P+%d record %d draws phase %d", to, run, i, r.Phase)
					}
				}
			}
		}
		t.Logf("to %v: cast on S+%d, relocation on S+%d", to, presented, presented+tpCharge)
	}
}
