package game_test

import (
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The undying fixture: one unit on a blank map built through the editor
// library, two triggers and three actions. Trigger 1 (constant == constant)
// sets the unit undying on the first script pass when undying is true, and
// runs nothing otherwise. Trigger 2 (health below -25 == constant) clears the
// state and writes variable 7 = 1. Conditions take registers in list order:
// the constant is register 0, the health-below condition register 1.
const (
	undyingVariable  = 7
	undyingThreshold = -25
	// undyingFiredTick is the world clock after the first trigger pass that
	// follows the blow on tick 83: the pass runs once per 16 sub-ticks, on
	// phase 6 (TRIG-EVAL-001), here the step from tick 86.
	undyingFiredTick = 87
)

type undyingFixture struct {
	m    *alm.Map
	unit uint16
}

func undyingTriggerRecord(name string, left, right uint32, acts ...uint32) []byte {
	rec := make([]byte, 184)
	copy(rec, name)
	binary.LittleEndian.PutUint32(rec[0x80:], left)
	binary.LittleEndian.PutUint32(rec[0x84:], right)
	for i, a := range acts {
		binary.LittleEndian.PutUint32(rec[0x98+4*i:], a)
	}
	binary.LittleEndian.PutUint32(rec[0xb4:], 1)
	return rec
}

func buildUndyingMap(t *testing.T, undying bool) undyingFixture {
	t.Helper()
	e, err := mapedit.NewMap(alm.NewMapSpec{Width: 40, Height: 40, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := e.NewUnitRecord(20<<8|128, 20<<8|128, 27, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlaceUnit(rec); err != nil {
		t.Fatal(err)
	}
	if err := e.SetDropCell(5, 5); err != nil {
		t.Fatal(err)
	}
	view, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	unit := view.Units[len(view.Units)-1].UnitID
	place := func(k mapedit.Kind, label string, op uint32, params ...mapedit.ScriptParam) uint32 {
		t.Helper()
		id, err := e.NextScriptID(k)
		if err != nil {
			t.Fatal(err)
		}
		node, err := mapedit.NewScriptNode(label, op, id, params...)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Place(k, node); err != nil {
			t.Fatal(err)
		}
		return id
	}
	unitParam := mapedit.ScriptParam{Type: mapload.ScriptUnit, Value: uint32(unit)}
	one := place(mapedit.KindCondition, "One", mapload.ScriptConstant, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
	threshold := int32(undyingThreshold)
	below := place(mapedit.KindCondition, "Below", mapload.AgainromScriptOperation(true), unitParam,
		mapedit.ScriptParam{Type: mapload.ScriptInt, Value: uint32(threshold)})
	set := place(mapedit.KindAction, "Set", mapload.AgainromScriptOperation(false), unitParam, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
	clearState := place(mapedit.KindAction, "Clear", mapload.AgainromScriptOperation(false), unitParam, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 0})
	mark := place(mapedit.KindAction, "Mark", uint32(sim.ScriptInstantSetVariable),
		mapedit.ScriptParam{Type: mapload.ScriptInt, Value: undyingVariable}, mapedit.ScriptParam{Type: mapload.ScriptInt, Value: 1})
	var first []uint32
	if undying {
		first = append(first, set)
	}
	if _, err := e.Place(mapedit.KindTrigger, undyingTriggerRecord("Start", one, one, first...)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place(mapedit.KindTrigger, undyingTriggerRecord("Below", below, one, clearState, mark)); err != nil {
		t.Fatal(err)
	}
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	return undyingFixture{m: m, unit: unit}
}

func startUndyingMission(t *testing.T, f undyingFixture) (*game.Mission, sim.EntityID) {
	t.Helper()
	ms, err := game.StartMissionFrom(f.m, "fixture.alm", 10, nil, mapload.DifficultyNormal, game.MissionParty(nil, data.BodyList{}, nil))
	if err != nil {
		t.Fatal(err)
	}
	if ms.World.Script() == nil || len(ms.World.Script().Unsupported()) != 0 {
		t.Fatalf("the fixture's script is absent or has gaps: %+v", ms.World.Script())
	}
	id, ok := mapload.ScriptUnits(f.m, nil)[f.unit]
	if !ok {
		t.Fatal("the fixture unit has no entity")
	}
	return ms, id
}

func undyingEntity(t *testing.T, w *sim.World, id sim.EntityID) (sim.Entity, bool) {
	t.Helper()
	return w.Entity(id)
}

// The undying unit takes damage past zero and past -10, stays alive, keeps no
// corpse stage and still obeys a move order; the health-below -25 condition
// fires on the first script pass after its health falls below -25, and the
// clearing action lets the ordinary death path take it: the fallen stage at
// once, the teardown after its dwell. The control map, the same without the
// undying action, takes the death transition on the blow that reaches health
// <= 0 (HERO-DEATH-026).
func TestScriptUndyingKeepsAUnitAliveUntilHealthBelowClearsIt(t *testing.T) {
	ms, id := startUndyingMission(t, buildUndyingMap(t, true))
	w := ms.World
	for i := 0; i < 16; i++ {
		sim.Step(w, nil)
	}
	e, _ := undyingEntity(t, w, id)
	if !e.Undying {
		t.Fatalf("after one script cycle (tick %d) the unit is not undying: %+v", w.Tick(), e.HP)
	}
	sim.Step(w, []sim.Command{sim.Damage(id, e.HP+20)})
	e, _ = undyingEntity(t, w, id)
	if e.HP != -20 || !e.Alive() || e.Dead() || e.Decay != sim.DecayNone {
		t.Fatalf("undying at -20: hp %d alive %v decay %d", e.HP, e.Alive(), e.Decay)
	}
	from := e.X
	sim.Step(w, []sim.Command{sim.MoveTo(id, sim.CellPoint{X: 30, Y: 20})})
	for i := 0; i < 64; i++ {
		sim.Step(w, nil)
	}
	e, _ = undyingEntity(t, w, id)
	if e.X <= from || e.HP >= 0 || !e.Alive() || e.Decay != sim.DecayNone {
		t.Fatalf("undying unit at -20 did not act: x %d -> %d, hp %d, decay %d", from, e.X, e.HP, e.Decay)
	}
	if w.ScriptRegister(undyingVariable) != 0 {
		t.Fatal("health-below -25 fired at -20")
	}
	sim.Step(w, []sim.Command{sim.Damage(id, e.HP+35)})
	damaged := w.Tick()
	fired := uint64(0)
	for i := 0; i < 32 && fired == 0; i++ {
		sim.Step(w, nil)
		if w.ScriptRegister(undyingVariable) == 1 {
			fired = w.Tick()
		}
	}
	if fired != undyingFiredTick {
		t.Fatalf("health-below -25 after the blow on tick %d fired on tick %d, want %d", damaged, fired, undyingFiredTick)
	}
	e, _ = undyingEntity(t, w, id)
	if e.Undying || e.Alive() || e.Decay != sim.DecayFallen || e.HP >= undyingThreshold {
		t.Fatalf("cleared below -25: undying %v alive %v decay %d hp %d", e.Undying, e.Alive(), e.Decay, e.HP)
	}
	t.Logf("damaged to -35 on tick %d; health-below fired and cleared on tick %d", damaged, fired)
	torn := undyingTeardown(t, w, id)
	t.Logf("teardown %d ticks after the clear", torn-fired)

	control, cid := startUndyingMission(t, buildUndyingMap(t, false))
	for i := 0; i < 16; i++ {
		sim.Step(control.World, nil)
	}
	c, _ := undyingEntity(t, control.World, cid)
	sim.Step(control.World, []sim.Command{sim.Damage(cid, c.HP+20)})
	c, _ = undyingEntity(t, control.World, cid)
	if c.Undying || c.Alive() || c.Decay != sim.DecayFallen {
		t.Fatalf("control at -20: undying %v alive %v decay %d", c.Undying, c.Alive(), c.Decay)
	}
	died := control.World.Tick()
	t.Logf("control: died on tick %d, teardown %d ticks later", died, undyingTeardown(t, control.World, cid)-died)
}

// undyingTeardown steps until the body leaves the fallen stage for the
// teardown (HERO-DWELL-065) and returns that tick.
func undyingTeardown(t *testing.T, w *sim.World, id sim.EntityID) uint64 {
	t.Helper()
	for i := 0; i < 400; i++ {
		e, ok := w.Entity(id)
		if !ok || e.Decay >= sim.DecayBones {
			return w.Tick()
		}
		sim.Step(w, nil)
	}
	e, _ := w.Entity(id)
	t.Fatalf("no teardown within 400 ticks: %+v", e.Decay)
	return 0
}

// The binder refuses an Againrom node that names no unit, and the program
// compiler refuses a health-below X outside the signed 16-bit health field and
// an undying state other than 0 or 1.
func TestScriptUndyingRefusals(t *testing.T) {
	src := alm.Script{
		Conditions: []alm.ScriptNode{{ID: 1, Opcode: mapload.AgainromScriptOperation(true), Type: [10]uint32{mapload.ScriptInt}, Value: [10]uint32{5}}},
	}
	if _, _, err := mapload.CompileScriptFrom(src, mapload.ScriptRefs{}); err == nil || !strings.Contains(err.Error(), "names no unit") {
		t.Fatalf("health-below with no unit: %v", err)
	}
	src = alm.Script{Actions: []alm.ScriptNode{{ID: 1, Opcode: mapload.AgainromScriptOperation(false), Type: [10]uint32{mapload.ScriptInt}, Value: [10]uint32{1}}}}
	if _, _, err := mapload.CompileScriptFrom(src, mapload.ScriptRefs{}); err == nil || !strings.Contains(err.Error(), "names no unit") {
		t.Fatalf("undying with no unit: %v", err)
	}
	for _, x := range []int32{-32769, 32768} {
		if _, err := sim.NewScript([]sim.ScriptCheck{{Op: sim.ScriptCheckHealthBelow, Args: [10]int32{x}}}, nil, nil); err == nil {
			t.Fatalf("health-below %d accepted", x)
		}
	}
	for _, x := range []int32{-32768, 32767, -25} {
		if _, err := sim.NewScript([]sim.ScriptCheck{{Op: sim.ScriptCheckHealthBelow, Args: [10]int32{x}}}, nil, nil); err != nil {
			t.Fatalf("health-below %d refused: %v", x, err)
		}
	}
	if _, err := sim.NewScript(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantUndying, Args: [10]int32{2}}}, nil); err == nil {
		t.Fatal("undying state 2 accepted")
	}
}
