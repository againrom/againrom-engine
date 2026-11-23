package sim

import "testing"

// The second game's altars 76 and 77 toggle +0x42 between 0 and 1 as the
// switches do (R2-ENGINE-1695); the first game places no such kind.
func TestSecondGameAltarsToggleLikeSwitches(t *testing.T) {
	for _, kind := range []uint16{76, 77} {
		if !(Structure{Kind: kind, Width: 2, Height: 2}).Usable() {
			t.Fatalf("altar kind %d is not usable", kind)
		}
	}
	for _, kind := range []uint16{67, 75, 78} {
		if (Structure{Kind: kind, Width: 2, Height: 2}).Usable() {
			t.Fatalf("kind %d became usable", kind)
		}
	}
	w := scriptWorld(t, secondGameScript(t, nil, nil, nil), []Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}})
	w.structures = []Structure{
		{ID: 0, Kind: 76, Width: 2, Height: 2, Col: 5, Row: 5, Field42: 1, MaxHealth: 1},
		{ID: 1, Kind: 77, Width: 2, Height: 2, Col: 9, Row: 5, Field42: 0, MaxHealth: 1},
	}
	for _, want := range []uint16{0, 1, 0} {
		w.applyStructureUse(0, 0)
		if got := w.structures[0].Field42; got != want {
			t.Fatalf("altar 76 after use: +0x42 = %d, want %d", got, want)
		}
	}
	w.applyStructureUse(0, 1)
	if got := w.structures[1].Field42; got != 1 {
		t.Fatalf("altar 77 after use: +0x42 = %d, want 1", got)
	}
}

// A well spends one charge and applies its potion at once; at zero charges it
// gives nothing (R2-ENGINE-1695).
func TestSecondGameWellSpendsOneChargeAndHealsAtOnce(t *testing.T) {
	w := scriptWorld(t, secondGameScript(t, nil, nil, nil), []Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 300}})
	w.structures = []Structure{{ID: 0, Kind: 15, Width: 2, Height: 2, Col: 5, Row: 5, Field42: 1, MaxHealth: 3, UseAmount: 100}}
	w.applyStructureUse(0, 0)
	if e := w.entities[0]; e.HP != 110 || w.structures[0].Field42 != 0 || len(w.carried[0]) != 0 {
		t.Fatalf("after a use: hp %d charges %d bag %v, want 110 0 empty", e.HP, w.structures[0].Field42, w.carried[0])
	}
	w.applyStructureUse(0, 0)
	if e := w.entities[0]; e.HP != 110 || w.structures[0].Field42 != 0 {
		t.Fatalf("an empty well acted: hp %d charges %d", e.HP, w.structures[0].Field42)
	}
}

// Checks 1, 5 and 8 count a living or unconscious unit and not a dead one in
// the second game (R2-ENGINE-1696); the first game leaves the downed unit out.
func TestSecondGameCountsUnconsciousUnitsAndNotTheDead(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Group: 3, Owner: 2},
		{ID: 2, X: 3, Y: 1, HP: 10, MaxHP: 10, Group: 3, Owner: 2},
		{ID: 3, X: 5, Y: 1, HP: 10, MaxHP: 10, Group: 3, Owner: 2},
	}
	checks := []ScriptCheck{
		groupCheck(0, 3),
		{Op: ScriptCheckAlive, Register: 1, Unit: 2, HasUnit: true},
		{Op: ScriptCheckPopulation, Register: 2, Player: 2, HasPlayer: true},
		{Op: ScriptCheckAlive, Register: 3, Unit: 1, HasUnit: true},
	}
	for _, game := range []struct {
		name   string
		script *Script
		want   [4]int32
	}{
		{"second", secondGameScript(t, checks, nil, nil), [4]int32{2, 1, 2, 0}},
		{"first", mustScript(t, checks, nil, nil), [4]int32{1, 0, 1, 0}},
	} {
		w := scriptWorld(t, game.script, ents)
		downGroupMember(t, w, 2)
		killGroupMember(t, w, 1)
		w.scriptPass(nil)
		for r, want := range game.want {
			if got := w.ScriptRegister(int32(r)); got != want {
				t.Errorf("%s game: register %d = %d, want %d", game.name, r, got, want)
			}
		}
	}
}

// A check naming a late reference is skipped undispatched, is no gap, and the
// mark survives the byte form (R2-ENGINE-1697).
func TestSecondGameSkipsACheckNamingALateReference(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckVariable, Register: 0, Args: [10]int32{500}, Late: true},
		{Op: ScriptCheckVariable, Register: 1, Args: [10]int32{500}},
		{Op: 99, Register: 2, Late: true},
	}, []ScriptInstant{{Op: ScriptInstantSetVariable, Args: [10]int32{500, 7}}}, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("late check reported as gap: %+v", gaps)
	}
	w := scriptWorld(t, s, nil)
	w.runInstant(ScriptInstant{Op: ScriptInstantSetVariable, Args: [10]int32{500, 7}})
	tr := &ScriptTrace{}
	w.scriptPass(tr)
	if w.ScriptRegister(0) != 0 || w.ScriptRegister(1) != 7 {
		t.Fatalf("late check wrote: reg0=%d reg1=%d, want 0 7", w.ScriptRegister(0), w.ScriptRegister(1))
	}
	if len(tr.Checks) != 3 || tr.Checks[0].Dispatched || tr.Checks[0].Silence != ScriptSilenceLateReference {
		t.Fatalf("late check trace %+v", tr.Checks)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if got := back.script.checks; len(got) != 3 || !got[0].Late || got[1].Late || !got[2].Late || back.Hash() != w.Hash() {
		t.Fatalf("late marks after round trip: %+v", got)
	}
	plain := scriptWorld(t, secondGameScript(t, []ScriptCheck{{Op: ScriptCheckVariable, Register: 1, Args: [10]int32{500}}}, nil, nil), nil)
	pf, err := plain.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	plain.script.checks[0].Late = true
	marked, err := plain.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if len(marked) != len(pf)+8 {
		t.Fatalf("late tail adds %d bytes, want 8", len(marked)-len(pf))
	}
}

// Action 11 moves one unit of the item from U's bag to V's bag
// (R2-ENGINE-1699).
func TestSecondGameAction11MovesOneItemBetweenBags(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: ScriptInstantMoveItem}}, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("action 11 reported as gap: %+v", gaps)
	}
	w, err := NewStockedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10}, {ID: 2, X: 3, Y: 1, HP: 10, MaxHP: 10}}, s, Relations{}, nil,
		[]Stock{{ID: 1, Items: []uint16{9, 9, 5}}, {ID: 2, Items: []uint16{9}}})
	if err != nil {
		t.Fatal(err)
	}
	move := ScriptInstant{Op: ScriptInstantMoveItem, Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true, Item: 9, HasItem: true}
	w.runInstant(move)
	from, _ := w.CarriedStacks(1)
	to, _ := w.CarriedStacks(2)
	if len(from) != 2 || from[0].Code != 9 || from[0].Count != 1 || len(to) != 1 || to[0].Count != 2 {
		t.Fatalf("after one move: U %v V %v, want U 9x1 and 5, V 9x2", from, to)
	}
	w.runInstant(move)
	w.runInstant(move)
	from, _ = w.CarriedStacks(1)
	to, _ = w.CarriedStacks(2)
	if len(from) != 1 || from[0].Code != 5 || len(to) != 1 || to[0].Count != 3 {
		t.Fatalf("after three moves: U %v V %v, want U 5 and V 9x3", from, to)
	}
}

// Action 27 moves the unit to (low8 p0, low8 p1); a computer unit then takes
// the Guard order at its own cell, also when the cell refused it
// (R2-ENGINE-1700).
func TestSecondGameAction27RelocatesAndGuardsAComputerUnit(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: ScriptInstantRelocateUnit}}, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("action 27 reported as gap: %+v", gaps)
	}
	w := scriptWorld(t, s, []Entity{
		{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10, Owner: 2},
		{ID: 2, X: 7, Y: 5, HP: 10, MaxHP: 10, Owner: SelfSlot},
		{ID: 3, X: 20, Y: 20, HP: 10, MaxHP: 10, Owner: 2},
	})
	for i := range w.entities {
		w.entities[i].ActorState = actorStatePatrol
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantRelocateUnit, Unit: 1, HasUnit: true, Args: [10]int32{0x10a, 12}})
	e, _ := w.Entity(1)
	if e.X != 10 || e.Y != 12 || e.ActorState != actorStateGuard || e.PostX != 10 || e.PostY != 12 {
		t.Fatalf("computer unit at (%d,%d) state %#x post (%d,%d), want (10,12) guard post (10,12)", e.X, e.Y, e.ActorState, e.PostX, e.PostY)
	}
	if o := w.savedOrder(1); o == nil || o.State != uint32(actorStateGuard) || o.Raw[0] != 10 || o.Raw[1] != 12 || o.Raw[8] != 0 {
		t.Fatalf("computer unit's saved order %+v", o)
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantRelocateUnit, Unit: 2, HasUnit: true, Args: [10]int32{14, 12}})
	if p, _ := w.Entity(2); p.X != 14 || p.Y != 12 || p.ActorState != actorStatePatrol {
		t.Fatalf("player unit at (%d,%d) state %#x, want (14,12) patrol kept", p.X, p.Y, p.ActorState)
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantRelocateUnit, Unit: 3, HasUnit: true, Args: [10]int32{10, 12}})
	if c, _ := w.Entity(3); c.X != 20 || c.Y != 20 || c.ActorState != actorStateGuard || c.PostX != 20 || c.PostY != 20 {
		t.Fatalf("refused computer unit at (%d,%d) state %#x post (%d,%d), want (20,20) guard post (20,20)", c.X, c.Y, c.ActorState, c.PostX, c.PostY)
	}
}

// A cell-entry record keeps its source bytes: zero sends the cast from cell
// (0, 0), aimed at the entering unit; action 25 with p4 zero names its own
// cell as source and target (R2-ENGINE-1698).
func TestSecondGameCellEntrySourceAndAction25OwnCell(t *testing.T) {
	w := scriptWorld(t, secondGameScript(t, nil, nil, nil), []Entity{{ID: 1, X: 10, Y: 10, HP: 50, MaxHP: 50, Domain: DomainGround}})
	w.setCellTail(ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{3, 20, 10, 10}})
	want := [cellTailLen]byte{3, 20, 10, 10, 10, 10}
	if got := w.CellTails(); len(got) != 1 || got[0].Bytes != want {
		t.Fatalf("action 25 with p4 zero stored %v, want %v", got, want)
	}
	w.spells = []SpellRule{{ID: 3, TargetsUnit: true}}
	w.writeCellTail(tailKey(10, 10), [cellTailLen]byte{3, 20, 0, 0, 0, 0})
	w.requestCellCast(0, 10, 10)
	if len(w.casts) != 1 || w.casts[0].FromX != 0 || w.casts[0].FromY != 0 || !w.casts[0].AtUnit || w.casts[0].Target != 1 {
		t.Fatalf("zero-source record cast %+v, want from (0,0) at unit 1", w.casts)
	}
}
