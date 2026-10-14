package game

import (
	"image"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// firstGameTeleportFront opens mission 101 with the hero holding the shipped
// Astral book, read. A second-game root has no mission 101 and no such book.
func firstGameTeleportFront(t *testing.T) (*FrontEnd, *ui.App) {
	t.Helper()
	f := stoneStillFront(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the first game's mission 101 and Astral book")
	}
	return teleportJumpFrontOn(t, f, 101, false)
}

// teleportBehindCell is a free cell four to six cells south of (x, y), behind
// a caster facing north, with no unit anchored within two cells of it.
func teleportBehindCell(w *sim.World, planes sim.Terrain, width, height int, x, y int32) (int32, int32, bool) {
	for r := int32(4); r <= 6; r++ {
		for _, dx := range []int32{0, 1, -1, 2, -2} {
			cx, cy := x+dx, y+r
			if cx < 1 || cy < 1 || cx >= int32(width)-1 || cy >= int32(height)-1 || planes.Block[int(cy)*width+int(cx)] != 0 {
				continue
			}
			near := false
			for _, e := range w.Entities() {
				near = near || max(e.X-cx, cx-e.X, e.Y-cy, cy-e.Y) <= 2
			}
			if !near {
				return cx, cy, true
			}
		}
	}
	return 0, 0, false
}

// TestReleaseBookTeleportMovesOnTheCountdownAndDrawsBothEnds: on mission 101
// the hero, facing north, casts the shipped Astral book's Teleport at a free
// cell ahead and at one behind. The cast is admitted on the tick P the hero
// faces the destination: P is S ahead and later behind, where the hero turns
// first (MOVE-107, MAGIC-340). P reports the cast and draws both picture-60
// records; the hero stands through P+7 in its Attack run and moves on P+8,
// where the records draw phase 8 and the run step 8 (MAGIC-TELEPORT-370,
// -371, ANIM-TELEPORT-172).
func TestReleaseBookTeleportMovesOnTheCountdownAndDrawsBothEnds(t *testing.T) {
	if stoneStillFront(t).Archives.Game() != base.GameROM1 {
		t.Skip("the first game's mission 101 and Astral book")
	}
	for _, side := range []string{"ahead", "behind"} {
		t.Run(side, func(t *testing.T) {
			f, _ := firstGameTeleportFront(t)
			mw := f.live
			casterID := mw.mission.ids[0]
			caster := releaseEntity(t, mw, casterID)
			if caster.Facing != 0 {
				t.Fatalf("instrument: the hero faces %d at the mission start, want north (0)", caster.Facing)
			}
			from := image.Pt(int(caster.X), int(caster.Y))
			m := mw.mission.state.Map
			planes := mapload.Planes(m, f.Table)
			var to image.Point
			if side == "ahead" {
				for radius := int32(4); radius <= 6 && to == (image.Point{}); radius++ {
					if x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y, radius); found {
						to = image.Pt(int(x), int(y))
					}
				}
			} else if x, y, found := teleportBehindCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y); found {
				to = image.Pt(int(x), int(y))
			}
			if to == (image.Point{}) {
				t.Fatalf("installed mission has no free Teleport destination %s four to six cells away", side)
			}
			charge := max(int(caster.AttackCharge), 8)
			mw.attackOrCast(uint32(casterID), 0, 26, to.X, to.Y, true)
			if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCastAt {
				t.Fatal("Teleport book did not queue its cell cast", mw.pending)
			}
			presented := -1
			for k := 0; presented < 0 && k < 30 || presented >= 0 && k <= presented+charge+4; k++ {
				mw.tickWithCastSink(func(events []sim.CastEvent) {
					for _, ev := range events {
						if ev.Caster == casterID && ev.Spell == 26 {
							if presented >= 0 || int(ev.FromX) != from.X || int(ev.ToX) != to.X || int(ev.ToY) != to.Y {
								t.Errorf("tick S+%d reported Teleport %+v, want the one cast from %v to %v", k, ev, from, to)
							}
							presented = k
						}
					}
				})
				e := releaseEntity(t, mw, casterID)
				if presented < 0 {
					if image.Pt(int(e.X), int(e.Y)) != from || mw.casting(casterID) || len(flightRecords(mw)) != 0 {
						t.Fatalf("tick S+%d, before the cast: hero on (%d,%d), casting %v, %d records", k, e.X, e.Y, mw.casting(casterID), len(flightRecords(mw)))
					}
					continue
				}
				run := k - presented
				want := from
				if run >= charge {
					want = to
				}
				if image.Pt(int(e.X), int(e.Y)) != want {
					t.Fatalf("tick P+%d (S+%d): hero on (%d,%d), want %v (charge %d)", run, k, e.X, e.Y, want, charge)
				}
				if run < 12 && (!mw.casting(casterID) || mw.swing[casterID] != run) {
					t.Errorf("tick P+%d: Attack run casting=%v step %d, want step %d", run, mw.casting(casterID), mw.swing[casterID], run)
				}
				cells := map[image.Point]int{}
				for _, r := range flightRecords(mw) {
					if r.Picture == 60 {
						cells[image.Pt(int(r.X)/256, int(r.Y)/256)]++
						if r.Phase != int32(run) {
							t.Errorf("tick P+%d: picture-60 record %d draws phase %d", run, r.ID, r.Phase)
						}
					}
				}
				if len(cells) != 2 || cells[from] != 1 || cells[to] != 1 {
					t.Fatalf("tick P+%d: picture-60 records in cells %v, want one on %v and one on %v", run, cells, from, to)
				}
				drawn := map[image.Point]int{}
				for _, d := range mw.savedProjectileDraws() {
					drawn[d.Cell]++
				}
				if drawn[from] == 0 || drawn[to] == 0 {
					t.Fatalf("tick P+%d: drawn record cells %v, want both %v and %v", run, drawn, from, to)
				}
				teleportJumpLogHash(t, mw, "countdown-"+side)
			}
			if presented < 0 || (presented == 0) != (side == "ahead") {
				t.Fatalf("%s: cast reported on S+%d; want S ahead and a later tick behind", side, presented)
			}
			t.Logf("%s %v: cast on S+%d, relocation on S+%d", side, to, presented, presented+charge)
		})
	}
}

// TestReleaseBookTeleportSavedInTheCountdownRelocatesOnTheSameTick: a menu
// SAVE two ticks after the admission writes the cast in flight in the actor's
// original fields (state 14, sub-phase 5, countdown 6, destination;
// SAV-1254), and the LOAD relocates the hero on the tick the live game does.
func TestReleaseBookTeleportSavedInTheCountdownRelocatesOnTheSameTick(t *testing.T) {
	f, a := firstGameTeleportFront(t)
	mw := f.live
	casterID := mw.mission.ids[0]
	caster := releaseEntity(t, mw, casterID)
	m := mw.mission.state.Map
	planes := mapload.Planes(m, f.Table)
	x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y, 5)
	if !found {
		t.Fatal("installed mission has no free Teleport destination five cells away")
	}
	mw.attackOrCast(uint32(casterID), 0, 26, int(x), int(y), true)
	mw.tick()
	mw.tick()
	mw.tick()
	_, _, raw := menuSAVE(t, f, a, OriginalStore{})
	saved := mw.world.Tick()
	live := uint64(0)
	for range 16 {
		mw.tick()
		if e := releaseEntity(t, mw, casterID); live == 0 && e.X == x && e.Y == y {
			live = mw.world.Tick()
		}
	}
	if live == 0 {
		t.Fatal("the live hero did not relocate")
	}
	ms, _, err := loadOriginalMission(f, raw)
	if err != nil {
		t.Fatalf("LOAD: %v", err)
	}
	if ms.World.Tick() != saved {
		t.Fatalf("loaded at tick %d, saved at %d", ms.World.Tick(), saved)
	}
	fields := map[string]uint32{}
	for _, b := range ms.savedDocument.Actors {
		if b.EntityID != casterID {
			continue
		}
		r := &ms.savedDocument.Document.Objects[b.ObjectIndex-1]
		for _, name := range []string{"U60", "U61", "U64", "U6C"} {
			fields[name], _ = savedStructureValue(r, name)
		}
		for _, name := range []string{"U54", "U58"} {
			if v, err := savedMotionRaw(r, name, 4); err == nil {
				fields[name] = uint32(v[0]) | uint32(v[1])<<8 | uint32(v[2])<<16 | uint32(v[3])<<24
			}
		}
	}
	spell := uint32(0)
	for i := range ms.savedDocument.Document.Objects {
		r := &ms.savedDocument.Document.Objects[i]
		if this, err := savedStructureValue(r, "This"); r.Class == "Spell" && err == nil && this == fields["U64"] {
			spell, _ = savedStructureValue(r, "S08")
		}
	}
	// U64 is 0 when the Spell was learned in the mission: its key is minted
	// after the cast fields are written (DIV-3330).
	if fields["U54"] != 14 || fields["U58"] != 5 || fields["U6C"] != uint32(live-saved) ||
		fields["U60"] != uint32(x) || fields["U61"] != uint32(y) || fields["U64"] != 0 && spell != 26 {
		t.Fatalf("saved caster fields %v, Spell %d; want state 14, sub-phase 5, countdown %d, (%d,%d), Spell 26", fields, spell, live-saved, x, y)
	}
	for ms.World.Tick() < live+2 {
		sim.StepReported(ms.World, nil)
		e, _ := ms.World.Entity(casterID)
		if moved := e.X == x && e.Y == y; moved != (ms.World.Tick() >= live) {
			t.Fatalf("loaded tick %d: hero on (%d,%d), the live hero moved on %d", ms.World.Tick(), e.X, e.Y, live)
		}
	}
	t.Logf("saved at %d, relocation at %d live and loaded", saved, live)
}
