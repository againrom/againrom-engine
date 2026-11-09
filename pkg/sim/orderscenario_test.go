package sim

import (
	"encoding/binary"
	"testing"
)

func TestSecondGameEmptyRouteScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1108/1019: an empty non-move route reaches activity 6.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Reach, self.Facing = 2, 64
	grid := make([]byte, 64*64)
	for dy := int32(-1); dy <= 1; dy++ {
		for dx := int32(-1); dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			grid[(40+dy)*64+30+dx] = blockGround
		}
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{Block: grid},
		[]Entity{self, asFoe(asFoeA, 32, 40)}, secondGameScript(t, nil, nil, nil),
		engRel(t, [3]uint32{SelfSlot, asSide, 1}, [3]uint32{asSide, SelfSlot, 1}), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.commandGroup([]int{0}, orderNone, cell{})
	w.entities[0].ActorState = actorStateGuard
	w.entities[0].PostX, w.entities[0].PostY = 45, 40
	o := w.syncSavedActorCommand(0)
	binary.LittleEndian.PutUint16(o.Raw[:], 45|40<<8)
	w.walkToCell(0, 45, 40, false)
	w.syncSavedDestination(0)
	cold := worldRoundTripForTest(t, w)
	for range 160 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("blocked-route continuation diverged after synthetic LOAD")
		}
		if e := w.entities[0]; e.AcquirePursuit {
			if !e.HasAttackTarget || e.AttackTarget != asFoeA || e.HasTarget || w.savedOrder(asHero).Raw[8] != 6 {
				t.Fatal("empty route did not become the shared idle-attack activity")
			}
			return
		}
	}
	t.Fatal("blocked guard route never reached re-pick")
}

func TestSecondGameComputerLookAroundScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-504: support turns in place and retains an unfinished heading.
	self := asFighter(asHero, asSide, 30, 40)
	self.RotationSpeed = 1
	w := asWorld(t, true, self)
	w.commandGroup([]int{0}, orderNone, cell{})
	o := w.savedOrder(asHero)
	o.Raw[8], o.Raw[0x54] = 0xb, 1
	wakeGroups(w)
	Step(w, nil)
	if o.Raw[0x54] != 0 || !w.entities[0].Turning() {
		t.Fatal("computer support did not consume its alarm and start a turn")
	}
	state, desired := w.RandomState(), w.entities[0].DesiredFacing
	cold := worldRoundTripForTest(t, w)
	for range 10 {
		Step(w, nil)
		Step(cold, nil)
		e := w.entities[0]
		if w.Hash() != cold.Hash() || w.RandomState() != state || e.DesiredFacing != desired || e.X != 30 || e.Y != 40 {
			t.Fatal("look-around changed heading, moved or diverged after synthetic LOAD")
		}
	}
}

func TestSecondGameIdleBodyScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-502: body promotion reaches the normal activity-6 producer.
	self := asFighter(asHero, SelfSlot, 30, 40)
	self.Facing, self.DamageBase = 64, 0
	a, b := asFoe(asFoeA, 31, 40), asFoe(asFoeB, 30, 41)
	a.HP, a.Domain, b.HP = -1, DomainAir, -1
	w := asWorld(t, true, self, a, b)
	w.commandDefend([]int{0}, asHero)
	for range 7 {
		Step(w, nil)
	}
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeB || e.X != 30 || e.Y != 40 {
		t.Fatalf("standing body choice did not survive Step: target=%v/%d cell=%d,%d", e.HasAttackTarget, e.AttackTarget, e.X, e.Y)
	}
	if o := w.savedOrder(asHero); o.State != 12 || o.Raw[8] != 6 {
		t.Fatalf("standing order/activity=%d/%d, want 12/6", o.State, o.Raw[8])
	}
}

func TestSecondGamePlayerOrderScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1042, R2-ENGINE-1076, R2-ENGINE-1105/1106/1107.
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 34, 41), asFoe(asFoeB, 45, 40))
	Step(w, []Command{GroupMoveTo(asHero, CellPoint{X: 33, Y: 40}, 1)})
	arrived := false
	for n := 0; n < 240; n++ {
		e := w.entities[0]
		atGoal := e.X == 33 && e.Y == 40 && e.Transit == 0
		if !atGoal && e.HasAttackTarget {
			t.Fatalf("walk acquired %d before arrival", e.AttackTarget)
		}
		if atGoal && e.HasAttackTarget {
			if o := w.savedOrder(asHero); o.State != 12 || o.Raw[8] != 6 {
				t.Fatalf("arrival order/activity=%d/%d, want 12/6", o.State, o.Raw[8])
			}
			arrived = true
			break
		}
		Step(w, nil)
	}
	if !arrived {
		t.Fatal("move did not reach adjacent idle attack")
	}
	Step(w, []Command{Attack(asHero, asFoeB), MoveTo(asFoeB, CellPoint{X: 55, Y: 40})})
	for range 90 {
		Step(w, nil)
		id, _, ok := w.entities[0].RequestedAttackTarget()
		if !ok || id != asFoeB {
			t.Fatalf("automatic choice replaced explicit moving target with %v/%d", ok, id)
		}
	}
	if w.entities[0].X <= 33 {
		t.Fatal("explicit attack did not pursue the moving target")
	}
	Step(w, []Command{MoveTo(asHero, CellPoint{X: 32, Y: 45})})
	if e := w.entities[0]; !e.HasTarget || e.TargetX != 32 || e.TargetY != 45 {
		t.Fatalf("replacement click not attached immediately: %+v", e)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	finished := false
	for n := 0; n < 400; n++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("replacement continuation diverged at step %d", n)
		}
		if e := w.entities[0]; e.X == 32 && e.Y == 45 && e.Transit == 0 && !e.HasTarget && e.ActorState == actorStateAcquire {
			finished = true
			break
		}
	}
	if !finished {
		e := w.entities[0]
		t.Fatalf("replacement did not finish: cell=(%d,%d) destination=%v state=%d", e.X, e.Y, e.HasTarget, e.ActorState)
	}
}

func TestSecondGameComputerChoiceUsesTheSharedMachine(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1111: assignment keeps least score, then the first full tie.
	self := asFighter(asHero, asSide, 30, 40)
	self.Facing = 64
	w := asWorld(t, true, self, asFighter(asFoeA, SelfSlot, 30, 39), asFighter(asFoeB, SelfSlot, 31, 40))
	w.commandGroup([]int{0}, orderStandGround, cell{})
	for range 7 {
		Step(w, nil)
	}
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeB || e.AcquirePursuit {
		t.Fatalf("computer target=%v/%d acquisition=%v, want assigned %d", e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, asFoeB)
	}
}
