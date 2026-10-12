package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

type secondBlizzardPulse struct {
	Tick             uint64
	X, Y             int32
	AnchorX, AnchorY int32
}

type secondBlizzardArea struct {
	Key     uint16
	Landing uint64
	Power   uint16
	Done    uint64
}

// secondBlizzardWatch follows every Blizzard area of one World tick by tick
// and fails on any departure from the claims' schedule: a new area stands at
// pulse 1 with countdown 2; k ticks later it holds pulse 1+k/3 and countdown
// 2-k%3; it is gone after its 32nd pulse, 93 ticks after the landing; every
// painted cell lies -2..3 from a standing anchor (R2-ENGINE-391,
// R2-ENGINE-392).
type secondBlizzardWatch struct {
	areas  map[uint16]*secondBlizzardArea
	order  []uint16
	pulses []secondBlizzardPulse
}

func (b *secondBlizzardWatch) observe(t *testing.T, w *sim.World, report sim.Report) {
	t.Helper()
	if b.areas == nil {
		b.areas = map[uint16]*secondBlizzardArea{}
	}
	states, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	now := uint64(w.Tick())
	standing := map[uint16]sim.AreaSaveState{}
	for _, s := range states {
		if s.Spell != 7 {
			continue
		}
		if _, twice := standing[s.Key]; twice {
			t.Fatalf("tick %d: two Blizzards stand on anchor %04x", now, s.Key)
		}
		standing[s.Key] = s
		a, ok := b.areas[s.Key]
		if !ok || a.Done != 0 {
			a = &secondBlizzardArea{Key: s.Key, Landing: now}
			b.areas[s.Key] = a
			b.order = append(b.order, s.Key)
		}
		k := now - a.Landing
		if s.Mode != sim.AreaModeRing || uint64(s.Stage) != 1+k/3 || uint64(s.Remaining) != 2-k%3 || k >= 93 {
			t.Fatalf("tick %d: Blizzard %04x landed at %d holds mode %d pulse %d countdown %d", now, s.Key, a.Landing, s.Mode, s.Stage, s.Remaining)
		}
		a.Power = s.Policy.Power
	}
	for key, a := range b.areas {
		if _, ok := standing[key]; !ok && a.Done == 0 {
			if now-a.Landing != 93 {
				t.Fatalf("tick %d: Blizzard %04x landed at %d ended after %d ticks, want 93", now, key, a.Landing, now-a.Landing)
			}
			a.Done = now
		}
	}
	for _, paint := range report.AreaPaints {
		if paint.Spell != 7 {
			continue
		}
		if len(paint.Cells) != 1 {
			t.Fatalf("tick %d: a Blizzard pulse painted %d cells", now, len(paint.Cells))
		}
		c := paint.Cells[0]
		owner := -1
		for i, key := range b.order {
			a := b.areas[key]
			ax, ay := int32(key&0xff), int32(key>>8)
			if dx, dy := c.X-ax, c.Y-ay; dx >= -2 && dx <= 3 && dy >= -2 && dy <= 3 && (now-a.Landing)%3 == 0 && now-a.Landing <= 93 {
				owner = i
			}
		}
		if owner < 0 {
			t.Fatalf("tick %d: Blizzard cell %d,%d belongs to no pulsing area", now, c.X, c.Y)
		}
		key := b.order[owner]
		b.pulses = append(b.pulses, secondBlizzardPulse{Tick: now, X: c.X, Y: c.Y, AnchorX: int32(key & 0xff), AnchorY: int32(key >> 8)})
	}
}

// secondBlizzardStruck checks that a live actor standing still on a painted
// cell lost at least max(trunc(B*(100-p)/100 + 0.75), 0) for its water
// resistance p (R2-ENGINE-393), and counts the actors it checked.
func secondBlizzardStruck(t *testing.T, before, after []sim.Entity, paints []sim.AreaPaint, power uint16) int {
	t.Helper()
	f := 1 + float64(min(power, 255))/30
	base := int64(uint8(math.Trunc(10 * f)))
	struck := map[sim.CellPoint]int{}
	for _, p := range paints {
		if p.Spell == 7 {
			for _, c := range p.Cells {
				struck[c]++
			}
		}
	}
	checked := 0
	for i, a := range before {
		if i >= len(after) || after[i].ID != a.ID || !a.Alive() || after[i].X != a.X || after[i].Y != a.Y || a.TokenSize > 1 {
			continue
		}
		n := struck[sim.CellPoint{X: a.X, Y: a.Y}]
		if n == 0 {
			continue
		}
		p := int64(min(max(a.Protection[1], 0), 100))
		least := int64(n) * max(int64(math.Trunc(float64(base*(100-p))/100+0.75)), 0)
		if lost := int64(a.HP) - int64(after[i].HP); lost < least && after[i].HP > 0 {
			t.Fatalf("actor %d on a struck cell lost %d health, want at least %d", a.ID, lost, least)
		}
		checked++
	}
	return checked
}

type secondBlizzardResult struct {
	Pulses []secondBlizzardPulse
	Areas  []secondBlizzardArea
	Hash   uint64
}

// secondBlizzardRun plays mission 102 from its start: the hero attacks the
// first Blizzard caster, and the creatures cast Blizzard through their own
// cast route. It watches every Blizzard for 1400 ticks.
func secondBlizzardRun(t *testing.T) (secondBlizzardResult, int) {
	t.Helper()
	w, _, hero := secondSpellMission(t, 102)
	caster := secondBlizzardCaster(t, w, hero)
	sim.Step(w, []sim.Command{sim.Attack(hero, caster)})
	var watch secondBlizzardWatch
	checked := 0
	for range 1400 {
		before := w.Entities()
		report := sim.StepReported(w, nil)
		watch.observe(t, w, report)
		for _, key := range watch.order {
			if a := watch.areas[key]; a.Done == 0 {
				checked += secondBlizzardStruck(t, before, w.Entities(), report.AreaPaints, a.Power)
				break
			}
		}
	}
	out := secondBlizzardResult{Pulses: watch.pulses, Hash: w.Hash()}
	for _, key := range watch.order {
		out.Areas = append(out.Areas, *watch.areas[key])
	}
	return out, checked
}

func secondBlizzardCaster(t *testing.T, w *sim.World, hero sim.EntityID) sim.EntityID {
	t.Helper()
	for _, e := range w.Entities() {
		for _, s := range e.CreatureSpells {
			if e.ID != hero && s.ID == 7 {
				return e.ID
			}
		}
	}
	t.Fatal("no placed creature casts Blizzard")
	return 0
}

// TestReleaseSecondGameBlizzardCast: on mission 102 the creatures cast
// Blizzard (row 7) at the hero through their own cast route. Each area pulses
// 32 times three ticks apart, paints one cell -2..3 from its anchor per
// admitted pulse and costs a struck actor at least its water-resisted base;
// a second run from the same start repeats every pulse and the World hash.
func TestReleaseSecondGameBlizzardCast(t *testing.T) {
	first, checked := secondBlizzardRun(t)
	completed := 0
	for _, a := range first.Areas {
		if a.Done != 0 {
			completed++
		}
	}
	if completed == 0 || len(first.Pulses) == 0 {
		t.Fatalf("no Blizzard completed: %+v", first.Areas)
	}
	again, _ := secondBlizzardRun(t)
	if !reflect.DeepEqual(first, again) {
		t.Fatal("a second seeded run from the same start cast another Blizzard")
	}
	t.Logf("%d Blizzard areas (%d complete), %d painted cells, %d struck actors checked", len(first.Areas), completed, len(first.Pulses), checked)
}

type secondBlizzardProof struct {
	Key    uint16
	Hashes []uint64
	Stages []uint8
}

// TestReleaseSecondGameBlizzardSaveContinuation: a named SAV taken in the
// middle of a Blizzard on mission 102 keeps the area; a cold LOAD in this
// process and another in a fresh process continue to the same remaining
// pulses, cells and damage as the continuous run.
func TestReleaseSecondGameBlizzardSaveContinuation(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_SAVE_INPUT"); path != "" {
		secondBlizzardColdProof(t, path)
		return
	}
	out := secondMissionSaveDirectory(t)
	f := secondGameFront(t)
	app := f.App("blizzard save")
	app.Layout(1024, 768)
	secondLaterEntry(t, f, app, 102, 100)
	hero := f.live.mission.ids[0]
	caster := secondBlizzardCaster(t, f.live.world, hero)
	f.live.strike(uint32(hero), uint32(caster))
	keep := []sim.EntityID{hero}
	for _, c := range f.live.world.Script().Checks() {
		if c.Op == sim.ScriptCheckAlive && c.HasUnit {
			keep = append(keep, c.Unit)
		}
	}
	stage, key := uint8(0), uint16(0)
	for step := 0; step < 4000 && stage < 12; step++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		dismissSecondMissionDialogue(t, app, f.live)
		// The hero and the units whose death loses the mission stand in the
		// creatures' way; keep them standing until the save.
		for _, id := range keep {
			if err := f.live.world.HeadlessHeal(id); err != nil {
				t.Fatal(err)
			}
		}
		stage, key = secondBlizzardStage(t, f.live.world, 0)
	}
	if stage < 12 || f.live.world.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("no Blizzard reached its twelfth pulse: pulse %d outcome %v", stage, f.live.world.Outcome())
	}
	raw := secondMissionNamedSave(t, f, app, out, "Blizzard")
	cold, a := secondMissionColdAt(t, out, secondSaveFile(t, out, "Blizzard"), 102)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "cold LOAD")
	// Loss control: the same SAV without the sampled cells the areas carry
	// loads to another World.
	carried := 0
	lost := secondChangeSave(t, raw, func(d *currentActionData) {
		for i := range d.NativeAreas {
			carried += len(d.NativeAreas[i].Policy.RingCells)
			d.NativeAreas[i].Policy.RingCells = nil
		}
	})
	if err := os.WriteFile(filepath.Join(out, "Blizzard without cells.sav"), lost, 0600); err != nil {
		t.Fatal(err)
	}
	if carried == 0 {
		t.Fatal("the SAV carries no sampled Blizzard cell")
	}
	if control, _ := secondMissionColdAt(t, out, "Blizzard without cells.sav", 102); control.live.world.Hash() == cold.live.world.Hash() {
		t.Fatal("a SAV without its sampled cells loaded to the same World")
	}
	proof := secondBlizzardProof{Key: key}
	for range 120 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "post-load Blizzard")
		proof.Hashes = append(proof.Hashes, f.live.world.Hash())
		pulse, _ := secondBlizzardStage(t, f.live.world, key)
		proof.Stages = append(proof.Stages, pulse)
	}
	// The 32nd pulse removes the area on its own tick, so 31 is the last
	// count a tick leaves standing.
	if proof.Stages[len(proof.Stages)-1] != 0 || slices.Max(proof.Stages) != 31 {
		t.Fatalf("the saved Blizzard did not run out its pulses within 120 ticks: %v", proof.Stages)
	}
	encoded, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(out, secondSaveFile(t, out, "Blizzard"))
	if err := os.WriteFile(path+".json", encoded, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_SAVE_INPUT")
	t.Logf("burst picture %d registered: %v", data.BurstPicture(7), f.live.projectiles.HasPicture(data.BurstPicture(7)))
	t.Logf("M102 Blizzard at pulse %d -> named SAV -> cold LOAD here and in a fresh process -> 120 identical ticks to its end", stage)
}

// secondBlizzardStage is the pulse count and anchor of the Blizzard standing
// on key, or of the first standing one for key 0; zero when none stands.
func secondBlizzardStage(t *testing.T, w *sim.World, key uint16) (uint8, uint16) {
	t.Helper()
	states, err := w.NativeAreaSaveStates()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range states {
		if s.Spell == 7 && (key == 0 || s.Key == key) {
			return s.Stage, s.Key
		}
	}
	return 0, 0
}

func secondBlizzardColdProof(t *testing.T, path string) {
	raw, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var proof secondBlizzardProof
	if err := json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionColdAt(t, filepath.Dir(path), filepath.Base(path), 102)
	for i, want := range proof.Hashes {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		pulse, _ := secondBlizzardStage(t, f.live.world, proof.Key)
		if got := f.live.world.Hash(); got != want || pulse != proof.Stages[i] {
			t.Fatalf("fresh-process tick %d: hash %x pulse %d, want %x pulse %d", i, got, pulse, want, proof.Stages[i])
		}
	}
	t.Logf("fresh-process LOAD continued the Blizzard through %d identical ticks", len(proof.Hashes))
}
