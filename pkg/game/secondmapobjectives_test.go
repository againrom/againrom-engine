package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// pageScript builds a second-game script from a map page's wiring: checks by
// their IDs as registers, actions by their IDs, triggers in list order. A
// condition whose arm the page leaves Medium is a probe of bank slot 900+ID,
// so the test sets what it measures; constants, register reads and every
// action stand as the page states them.
type pageScript struct {
	checks   []sim.ScriptCheck
	instants []sim.ScriptInstant
	actions  map[int32]int32
	triggers []sim.ScriptTrigger
}

func (p *pageScript) probe(ids ...int32) {
	for _, id := range ids {
		p.checks = append(p.checks, sim.ScriptCheck{Op: sim.ScriptCheckScenario, Register: id, Args: [10]int32{900 + id}})
	}
}

func (p *pageScript) constant(id, v int32) {
	p.checks = append(p.checks, sim.ScriptCheck{Op: sim.ScriptCheckConstant, Register: id, Args: [10]int32{v}})
}

func (p *pageScript) variable(id, register int32) {
	p.checks = append(p.checks, sim.ScriptCheck{Op: sim.ScriptCheckVariable, Register: id, Args: [10]int32{register}})
}

func (p *pageScript) action(id, op int32, args ...int32) {
	if p.actions == nil {
		p.actions = map[int32]int32{}
	}
	in := sim.ScriptInstant{Op: op}
	copy(in.Args[:], args)
	p.actions[id] = int32(len(p.instants))
	p.instants = append(p.instants, in)
}

// trigger is trigger t: once, its pairs (left, cmp, right) and its actions.
func (p *pageScript) trigger(t int32, once bool, pairs [][3]int32, actions ...int32) {
	tr := sim.ScriptTrigger{Once: once, Latch: t}
	for k, pair := range pairs {
		tr.Pairs[k] = sim.ScriptPair{Left: pair[0], Cmp: pair[1], Right: pair[2], Used: true}
	}
	for k := range tr.Instants {
		tr.Instants[k] = sim.ScriptNone
	}
	for k, a := range actions {
		tr.Instants[k] = p.actions[a]
	}
	p.triggers = append(p.triggers, tr)
}

// pageRun drives a page script: set writes probe slots, run steps the world
// over several runner passes and answers the messages raised, and state is
// the objective slots 753, 754 and the window's rows.
type pageRun struct {
	t  *testing.T
	w  *sim.World
	mw *mapWorld
}

func newPageRun(t *testing.T, p *pageScript, probes map[int32]int32) *pageRun {
	t.Helper()
	s, err := sim.NewROM2Script(p.checks, p.instants, p.triggers)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), nil, s)
	if err != nil {
		t.Fatal(err)
	}
	r := &pageRun{t: t, w: w, mw: &mapWorld{world: w, mission: &missionNotices{
		table: readUnder(base.GameROM2, &mapload.Table{}), objectiveLabels: []string{"", "objective 1", "objective 2"},
	}}}
	r.set(probes)
	return r
}

func (r *pageRun) set(probes map[int32]int32) {
	bank, ok := r.w.ROM2ScenarioState()
	if !ok {
		r.t.Fatal("no scenario bank")
	}
	for id, v := range probes {
		bank[900+id] = v
	}
	r.w.SetROM2ScenarioState(bank)
}

func (r *pageRun) run() []int32 {
	var messages []int32
	for i := 0; i < 96; i++ {
		messages = append(messages, sim.StepReported(r.w, nil).ScriptMessages...)
	}
	return messages
}

// expect runs the passes and checks the messages, the two objective slots
// and the window's rows with their marks.
func (r *pageRun) expect(step string, messages []int32, slot753, slot754 int32, rows ...ui.QuestObjective) {
	r.t.Helper()
	got := r.run()
	bank, _ := r.w.ROM2ScenarioState()
	if !reflect.DeepEqual(got, messages) && !(len(got) == 0 && len(messages) == 0) {
		r.t.Fatalf("%s: messages %v, want %v", step, got, messages)
	}
	if bank[753] != slot753 || bank[754] != slot754 {
		r.t.Fatalf("%s: slots 753, 754 = %d, %d, want %d, %d", step, bank[753], bank[754], slot753, slot754)
	}
	if shown := r.mw.missionObjectives(); !reflect.DeepEqual(shown, rows) && !(len(shown) == 0 && len(rows) == 0) {
		r.t.Fatalf("%s: the window shows %v, want %v", step, shown, rows)
	}
}

const (
	pageEQ = sim.ScriptCmpEQ
	pageNE = sim.ScriptCmpNE
	pageGT = sim.ScriptCmpGT
	pageGE = sim.ScriptCmpGE
	pageLE = sim.ScriptCmpLE
)

func pageRow(n, mark int) ui.QuestObjective {
	return ui.QuestObjective{Text: []string{"", "objective 1", "objective 2"}[n], Mark: mark}
}

// Map 21's objective wiring as R2-ENGINE-894 states it: T0 opens objective 1
// on the first pass, T1 opens objective 2 when the hero is within 4 of unit
// 46, T4 completes it there once T3 has raised register 70 while the sack
// stands, and T9 completes objective 1 when the sack at (60, 18) is gone.
// c1 is the hero's distance to unit 46, c4 the sack test, c12 unit 39's
// distance to (61, 18).
func TestSecondMapTwentyOneObjectivesOpenAndCompleteAsThePageSays(t *testing.T) {
	p := &pageScript{}
	p.probe(1, 4, 12)
	p.constant(6, 4)
	p.constant(7, 0)
	p.variable(8, 70)
	p.action(6, sim.ScriptInstantMessage, 1)
	p.action(7, sim.ScriptInstantMessage, 2)
	p.action(10, sim.ScriptInstantMessage, 5)
	p.action(11, sim.ScriptInstantIncVariable, 70)
	p.action(18, 36, 1, 1)
	p.action(19, 36, 1, 2)
	p.action(20, 36, 2, 1)
	p.action(21, 36, 2, 2)
	p.trigger(0, true, [][3]int32{{7, pageEQ, 7}}, 6, 18)
	p.trigger(1, true, [][3]int32{{1, pageLE, 6}}, 7, 20)
	p.trigger(3, true, [][3]int32{{12, pageGT, 7}}, 11)
	p.trigger(4, true, [][3]int32{{1, pageLE, 6}, {4, pageNE, 7}, {8, pageNE, 7}}, 10, 11, 21)
	p.trigger(9, true, [][3]int32{{4, pageEQ, 7}}, 19)
	r := newPageRun(t, p, map[int32]int32{1: 255, 4: 1, 12: 0})

	r.expect("T0 first pass", []int32{1}, 1, 0, pageRow(1, 0))
	r.set(map[int32]int32{1: 3})
	r.expect("T1 near unit 46", []int32{2}, 1, 1, pageRow(1, 0), pageRow(2, 0))
	r.set(map[int32]int32{12: 5})
	r.expect("T3 then T4", []int32{5, sim.ObjectiveEventDone}, 1, 3, pageRow(1, 0), pageRow(2, 1))
	r.set(map[int32]int32{4: 0})
	r.expect("T9 sack gone", []int32{sim.ObjectiveEventDone}, 3, 3, pageRow(1, 1), pageRow(2, 1))
	r.expect("no trigger repeats", nil, 3, 3, pageRow(1, 1), pageRow(2, 1))
}

// Map 31's objective wiring as R2-ENGINE-895 states it: T2 opens objective 1
// near unit 101 and T3 objective 2 near unit 102 while player 4 has units;
// with player 4 gone T8 raises register 76 near unit 102, T6 raises 75 near
// unit 100 and T13 completes objective 2; T5 completes objective 1 near unit
// 101 with event 250, and T16 then ends the mission complete in the same
// pass. c1, c3, c7 are the hero's distances to units 100, 101, 102 and c18
// player 4's population.
func TestSecondMapThirtyOneObjectivesOpenAndCompleteAsThePageSays(t *testing.T) {
	p := &pageScript{}
	p.probe(1, 3, 7, 18)
	p.constant(9, 0)
	p.constant(10, 4)
	p.constant(28, 1)
	for id, register := range map[int32]int32{11: 70, 12: 71, 13: 72, 14: 73, 17: 74, 19: 75, 21: 76, 27: 77} {
		p.variable(id, register)
	}
	for id, event := range map[int32]int32{5: 4, 9: 8, 11: 10, 19: 12, 26: 14, 46: 250} {
		p.action(id, sim.ScriptInstantMessage, event)
	}
	for id, register := range map[int32]int32{14: 71, 16: 73, 17: 74, 18: 75, 24: 76} {
		p.action(id, sim.ScriptInstantIncVariable, register)
	}
	p.action(21, sim.ScriptInstantWin)
	p.action(40, 36, 1, 1)
	p.action(41, 36, 1, 2)
	p.action(42, 36, 2, 2)
	p.action(43, 36, 2, 1)
	p.trigger(2, false, [][3]int32{{3, pageLE, 10}, {12, pageEQ, 9}, {18, pageNE, 9}}, 5, 14, 40)
	p.trigger(3, false, [][3]int32{{7, pageLE, 10}, {14, pageEQ, 9}, {18, pageNE, 9}}, 9, 16, 43)
	p.trigger(5, false, [][3]int32{{3, pageLE, 10}, {17, pageEQ, 9}, {18, pageEQ, 9}}, 19, 17, 46, 41)
	p.trigger(6, false, [][3]int32{{1, pageLE, 10}, {19, pageEQ, 9}, {21, pageNE, 9}}, 11, 18)
	p.trigger(8, false, [][3]int32{{7, pageLE, 10}, {21, pageEQ, 9}, {18, pageEQ, 9}}, 26, 24)
	p.trigger(13, true, [][3]int32{{19, pageEQ, 28}}, 42, 18)
	p.trigger(16, false, [][3]int32{{17, pageGT, 9}}, 21)
	r := newPageRun(t, p, map[int32]int32{1: 255, 3: 255, 7: 255, 18: 3})

	r.expect("far from every unit", nil, 0, 0)
	r.set(map[int32]int32{3: 2})
	r.expect("T2 near unit 101", []int32{4}, 1, 0, pageRow(1, 0))
	r.set(map[int32]int32{3: 255, 7: 2})
	r.expect("T3 near unit 102", []int32{8}, 1, 1, pageRow(1, 0), pageRow(2, 0))
	r.set(map[int32]int32{18: 0})
	r.expect("T8 player 4 gone", []int32{14}, 1, 1, pageRow(1, 0), pageRow(2, 0))
	r.set(map[int32]int32{7: 255, 1: 2})
	r.expect("T6 then T13", []int32{10, sim.ObjectiveEventDone}, 1, 3, pageRow(1, 0), pageRow(2, 1))
	if r.w.Outcome() != sim.OutcomeUndecided {
		t.Fatal("the mission ended before objective 1")
	}
	r.set(map[int32]int32{1: 255, 3: 2})
	r.expect("T5 near unit 101", []int32{12, 250, sim.ObjectiveEventDone}, 3, 3, pageRow(1, 1), pageRow(2, 1))
	if r.w.Outcome() != sim.OutcomeWon {
		t.Fatalf("T16 did not end the mission complete: %v", r.w.Outcome())
	}
}
