package game

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The creature classes of mission 130 by the one spell their slot holds. Each
// spell's arm fixes the target: the creature itself, the victim, the victim's
// cell, or the cell one step toward the victim.
const (
	creatureClassSelf   = 73 // spell 15, aimed at the caster
	creatureClassStep   = 76 // spell 9, Acid Stream, aimed one step toward the victim
	creatureClassTele   = 80 // spell 26, Teleport, aimed one step toward the victim
	creatureClassCell   = 70 // spell 17, aimed at the victim's cell
	creatureClassVictim = 74 // spell 1, aimed at the victim
)

// creatureStepOracle is the cell one step from (cx, cy) toward (vx, vy) along
// the nearest of the eight headings, computed with a bearing in floating point
// and not with the engine's integer sector test.
func creatureStepOracle(cx, cy, vx, vy int32) (int32, int32) {
	angle := math.Atan2(float64(vx-cx), -float64(vy-cy))
	sector := int(math.Round(angle/(math.Pi/4))+8) % 8
	dx := [8]int32{0, 1, 1, 1, 0, -1, -1, -1}
	dy := [8]int32{-1, -1, 0, 1, 1, 1, 0, -1}
	return cx + dx[sector], cy + dy[sector]
}

// creatureRingCell is a passable, unoccupied cell at Chebyshev distance r from
// (x, y), or false.
func creatureRingCell(w *sim.World, planes sim.Terrain, width, height int, x, y, r int32) (int32, int32, bool) {
	occupied := map[[2]int32]bool{}
	for _, e := range w.Entities() {
		occupied[[2]int32{e.X, e.Y}] = true
	}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if max(dx, -dx, dy, -dy) != r {
				continue
			}
			cx, cy := x+dx, y+dy
			if cx < 1 || cy < 1 || cx >= int32(width)-1 || cy >= int32(height)-1 {
				continue
			}
			if planes.Block[int(cy)*width+int(cx)] == 0 && !occupied[[2]int32{cx, cy}] {
				return cx, cy, true
			}
		}
	}
	return 0, 0, false
}

// creatureAlone takes every unit except the two named off the map, so the
// creature's one possible victim is the hero.
func creatureAlone(hero, creature sim.EntityID) func(*sim.Entity) {
	return func(e *sim.Entity) {
		if e.ID != hero && e.ID != creature {
			e.OffMap = true
		}
	}
}

type creatureObservation struct {
	rec              sim.BookContinuation
	tick             int
	casterX, casterY int32
	heroX, heroY     int32
}

// creatureObserve steps w until caster holds a book record in one of the
// wanted phases and returns it with the positions the decision read.
func creatureObserve(w *sim.World, caster, hero sim.EntityID, ticks int, phases ...uint8) (creatureObservation, bool) {
	position := func(id sim.EntityID) (int32, int32) {
		for _, e := range w.Entities() {
			if e.ID == id {
				return e.X, e.Y
			}
		}
		return 0, 0
	}
	for tick := 0; tick < ticks; tick++ {
		cx, cy := position(caster)
		hx, hy := position(hero)
		sim.Step(w, nil)
		if rec, ok := castOrderBook(w, caster); ok {
			for _, p := range phases {
				if rec.Phase == p {
					return creatureObservation{rec, tick, cx, cy, hx, hy}, true
				}
			}
		}
	}
	return creatureObservation{}, false
}

// creatureClassCases lists the creatures of one class in mission 130 the
// witnesses may use, with the map and the started mission.
func creatureClassCases(t *testing.T, f *FrontEnd, class int) (*alm.Map, *Mission, []sim.Entity) {
	t.Helper()
	m := releaseMissionMap(t, f, creatureWitnessMission)
	ms, err := StartMissionFrom(m, "witness.alm", creatureWitnessMission, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
	if err != nil {
		t.Fatal(err)
	}
	var out []sim.Entity
	for _, e := range ms.World.Entities() {
		if int(e.Class) == class && e.Alive() && e.Owner != sim.SelfSlot && e.CreatureSpells[0].ID != 0 {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		t.Fatalf("mission %d places no creature of class %d", creatureWitnessMission, class)
	}
	return m, ms, out
}

// TestReleaseCreatureSpellsAreAimedByTheirArm drives one creature of each arm
// beside the party in mission 130 and reads the book record its first draw
// writes: a cast at itself, at the victim, at the victim's cell, and one step
// toward the victim for Acid Stream and Teleport.
func TestReleaseCreatureSpellsAreAimedByTheirArm(t *testing.T) {
	f := releaseFront(t)
	for _, tc := range []struct {
		name  string
		class int
	}{
		{"caster arm", creatureClassSelf},
		{"step cell arm Acid Stream", creatureClassStep},
		{"step cell arm Teleport", creatureClassTele},
		{"victim cell arm", creatureClassCell},
		{"victim arm", creatureClassVictim},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, ms, creatures := creatureClassCases(t, f, tc.class)
			hero := ms.Start.IDs[0]
			planes := mapload.Planes(m, f.Table)
			tried := 0
			for _, c := range creatures {
				t.Logf("class %d creature %d slots %v", tc.class, c.ID, c.CreatureSpells)
				x, y, ok := creatureFreeCellNear(ms.World, planes, m.Width, m.Height, c.X, c.Y)
				if !ok {
					continue
				}
				if tried == 10 {
					break
				}
				tried++
				w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, creatureAlone(hero, c.ID))
				got, ok := creatureObserve(w, c.ID, hero, 1500, 0, 1, sim.BookPhaseApproach)
				if !ok {
					continue
				}
				rec := got.rec
				if uint32(rec.Spell) != c.CreatureSpells[0].ID || !rec.Retained {
					t.Fatalf("creature %d began %+v, want its slot spell %d as a retained order", c.ID, rec, c.CreatureSpells[0].ID)
				}
				switch tc.class {
				case creatureClassSelf:
					if rec.AtCell || rec.Target != c.ID {
						t.Fatalf("creature %d cast %+v, want a cast at itself", c.ID, rec)
					}
				case creatureClassVictim:
					if rec.AtCell || rec.Target != hero {
						t.Fatalf("creature %d cast %+v, want a cast at the hero %d", c.ID, rec, hero)
					}
				case creatureClassCell:
					if !rec.AtCell || rec.X != got.heroX || rec.Y != got.heroY {
						t.Fatalf("creature %d cast %+v, want a cast at the hero's cell (%d,%d)", c.ID, rec, got.heroX, got.heroY)
					}
				default:
					ex, ey := creatureStepOracle(got.casterX, got.casterY, got.heroX, got.heroY)
					if !rec.AtCell || rec.X != ex || rec.Y != ey {
						t.Fatalf("creature %d at (%d,%d) cast %+v at the hero (%d,%d), want the cell (%d,%d) one step toward it",
							c.ID, got.casterX, got.casterY, rec, got.heroX, got.heroY, ex, ey)
					}
				}
				t.Logf("class %d creature %d: tick %d record %+v", tc.class, c.ID, got.tick, rec)
				return
			}
			t.Fatalf("none of %d class %d creatures began a cast within 1500 ticks", tried, tc.class)
		})
	}
}

// creatureApproachPlacement finds a class creature and a hero cell at which an
// out-of-range draw arms an approach, and returns the world that shows it.
func creatureApproachPlacement(t *testing.T, f *FrontEnd, m *alm.Map, ms *Mission, creatures []sim.Entity) (sim.Entity, int32, int32, creatureObservation) {
	t.Helper()
	hero := ms.Start.IDs[0]
	planes := mapload.Planes(m, f.Table)
	for _, c := range creatures {
		for r := int32(6); r <= 9; r++ {
			x, y, ok := creatureRingCell(ms.World, planes, m.Width, m.Height, c.X, c.Y, r)
			if !ok {
				continue
			}
			w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, creatureAlone(hero, c.ID))
			if got, ok := creatureObserve(w, c.ID, hero, 600, sim.BookPhaseApproach); ok {
				return c, x, y, got
			}
		}
	}
	t.Fatal("no placement of a class creature armed an approach")
	return sim.Entity{}, 0, 0, creatureObservation{}
}

// TestReleaseCreatureCastOutOfRangeWalksAndStaysArmed places the party beyond
// a creature's spell range in sight of it. The first draw that selects the
// spell arms an approach instead of engaging; the creature closes the distance
// with that order armed and its cast begins inside the spell range.
func TestReleaseCreatureCastOutOfRangeWalksAndStaysArmed(t *testing.T) {
	f := releaseFront(t)
	m, ms, creatures := creatureClassCases(t, f, creatureClassVictim)
	c, x, y, armed := creatureApproachPlacement(t, f, m, ms, creatures)
	armedAt := int64(max(creatureAbs(armed.casterX-armed.heroX), creatureAbs(armed.casterY-armed.heroY)))
	if armed.rec.Phase != sim.BookPhaseApproach || !armed.rec.Retained {
		t.Fatalf("the out-of-range draw wrote %+v, want a retained armed approach", armed.rec)
	}
	// The same placement again, read to the first wind-up.
	hero := ms.Start.IDs[0]
	w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, creatureAlone(hero, c.ID))
	for tick := 0; tick < 1500; tick++ {
		sim.Step(w, nil)
		rec, ok := castOrderBook(w, c.ID)
		if !ok || rec.Phase != 1 {
			continue
		}
		var e, h sim.Entity
		for _, o := range w.Entities() {
			if o.ID == c.ID {
				e = o
			}
			if o.ID == hero {
				h = o
			}
		}
		d := int64(max(creatureAbs(e.X-h.X), creatureAbs(e.Y-h.Y)))
		if d >= armedAt {
			t.Fatalf("the cast began at distance %d, the draw armed it at %d", d, armedAt)
		}
		t.Logf("armed at distance %d, cast began at distance %d on tick %d", armedAt, d, tick)
		return
	}
	t.Fatal("the armed creature never began its cast")
}

func creatureAbs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestReleaseCreatureRetainedCastEndsAtAMissingDraw follows the first creature
// that casts at the party. A retained cast would repeat every cycle for as long
// as the victim stays in range; the draw each pass makes while the cast is
// pending ends the order at the first pass that selects no spell, so the casts
// stay few.
func TestReleaseCreatureRetainedCastEndsAtAMissingDraw(t *testing.T) {
	f := releaseFront(t)
	m, ms, creatures := creatureClassCases(t, f, creatureClassVictim)
	hero := ms.Start.IDs[0]
	planes := mapload.Planes(m, f.Table)
	const window = 1500
	for _, c := range creatures {
		x, y, ok := creatureFreeCellNear(ms.World, planes, m.Width, m.Height, c.X, c.Y)
		if !ok {
			continue
		}
		w := creatureWitnessWorld(t, ms.World, m, f.Table, hero, x, y, creatureAlone(hero, c.ID))
		casts, first := 0, -1
		for tick := 0; tick < window; tick++ {
			for _, ev := range sim.StepObserved(w, nil) {
				if ev.Caster == c.ID && ev.Target == hero {
					casts++
					if first < 0 {
						first = tick
					}
				}
			}
		}
		if casts == 0 {
			continue
		}
		t.Logf("creature %d cast %d times in %d ticks, first at %d", c.ID, casts, window, first)
		if casts > 6 {
			t.Fatalf("creature %d cast %d times in %d ticks: its retained order was never replaced by a draw", c.ID, casts, window)
		}
		return
	}
	t.Fatal("no class creature cast at the party")
}

// creatureWireOrder is the order block of the Unit record of a map unit in a
// SAV.
func creatureWireOrder(t *testing.T, raw []byte, unit uint16) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Objects {
		if r.Class == "Unit" && savedRecordValueForTest(t, r, "T08") == uint32(unit) {
			return savedRecordRawForTest(t, r, "U158")
		}
	}
	t.Fatalf("the SAV holds no Unit record for map unit %d", unit)
	return nil
}

// creatureArmedSession opens mission 130 through the player's opener with the
// n-th Acid Stream creature moved to distance r from the party's start cell and
// ticks the session until that creature holds an armed approach. It reports
// false when the placement arms none within the window.
func creatureArmedSession(t *testing.T, n int, r int32) (*FrontEnd, sim.Entity, bool) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "SlotProbe", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	for i := range party {
		party[i].Hero.Body = 400
	}
	probe, _, creatures := creatureClassCases(t, f, creatureClassCell)
	if n >= len(creatures) {
		return nil, sim.Entity{}, false
	}
	partyStart, err := StartMissionFrom(probe, "probe.alm", creatureWitnessMission, f.Table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatal(err)
	}
	var heroAt sim.Entity
	for _, e := range partyStart.World.Entities() {
		if e.ID == partyStart.Start.IDs[0] {
			heroAt = e
		}
	}
	creature := creatures[n]
	x, y, ok := creatureRingCell(partyStart.World, mapload.Planes(probe, f.Table), probe.Width, probe.Height, heroAt.X, heroAt.Y, r)
	if !ok {
		return nil, sim.Entity{}, false
	}
	app := f.App("armed creature SAVE")
	t.Cleanup(app.StopAudio)
	opener := f.missionOpener(creatureWitnessMission, party, nil, nil, func(m *alm.Map) (*originalFog, error) {
		m.Units[int(creature.ID)].X, m.Units[int(creature.ID)].Y = uint32(x*256+128), uint32(y*256+128)
		return nil, nil
	})
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	live := f.live
	for tick := 0; tick < 300; tick++ {
		live.tick()
		for _, c := range live.world.Actions().Books {
			if c.Phase != sim.BookPhaseApproach {
				continue
			}
			for _, e := range live.world.Entities() {
				if e.ID == c.Caster {
					return f, e, true
				}
			}
		}
	}
	return f, creature, false
}

// TestReleaseCreatureArmedCastSaveColdLoad saves a mission-130 session while a
// creature's out-of-range cast is armed, then loads the file cold. The order
// block carries the cast order at progress 0 with the retention flag set; the
// loaded session keeps the armed cast and both sessions keep one hash through
// the cast landing.
func TestReleaseCreatureArmedCastSaveColdLoad(t *testing.T) {
	var f *FrontEnd
	var creature sim.Entity
	for n := 0; n < 8 && f == nil; n++ {
		for r := int32(5); r <= 7 && f == nil; r++ {
			if g, c, ok := creatureArmedSession(t, n, r); ok {
				f, creature = g, c
			}
		}
	}
	if f == nil {
		t.Fatal("no placement armed an approach in the player's session")
	}
	hero := f.live.mission.ids[0]
	armed, _ := castOrderBook(f.live.world, creature.ID)
	wantKind := byte(8)
	if armed.AtCell {
		wantKind = 9
	}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentWorldSave(snapshot, "armed creature")
	if err != nil {
		t.Fatal(err)
	}
	order := creatureWireOrder(t, saved, creature.MapUnitID)
	if order[8] != wantKind || order[9] != 0 || order[0x60] != 1 ||
		armed.AtCell && binary.LittleEndian.Uint16(order[0x3c:]) != uint16(uint8(armed.X))|uint16(uint8(armed.Y))<<8 {
		t.Fatalf("the armed order block reads kind %d progress %d flag %d cell %#x, want kind %d, progress 0, flag 1 and the aimed cell (%d,%d)",
			order[8], order[9], order[0x60], binary.LittleEndian.Uint16(order[0x3c:]), wantKind, armed.X, armed.Y)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "armed creature.sav"), saved, 0o600); err != nil {
		t.Fatal(err)
	}
	g, _ := castOrderSession(t, dir)
	cold, ok := castOrderBook(g.live.world, creature.ID)
	if !ok || cold != armed {
		t.Fatalf("cold LOAD holds the cast %+v/%v, saved %+v", cold, ok, armed)
	}
	castOrderLockstep(t, f, g, 400, func(h *FrontEnd, events []sim.CastEvent) bool {
		for _, ev := range events {
			if ev.Caster == creature.ID {
				return true
			}
		}
		rec, ok := castOrderBook(h.live.world, creature.ID)
		return !ok || rec.Phase != sim.BookPhaseApproach
	})
	_ = hero
}
