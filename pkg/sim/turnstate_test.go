package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTurnStateSurvivesBinaryAndActionContinuations(t *testing.T) {
	b := Bounds{Width: 7, Height: 7}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{turnActor(1, 3, 3, 0, 21)})
	Step(w, []Command{MoveTo(1, CellPoint{X: 3, Y: 5})})
	Step(w, nil)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var binaryWorld, actionWorld World
	for _, cold := range []*World{&binaryWorld, &actionWorld} {
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
	}
	a := actionCopy(t, w.Actions())
	actionWorld.entities[0].TurnState = TurnState{}
	if err := actionWorld.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	for tick := 0; tick < 12; tick++ {
		for _, cold := range []*World{&binaryWorld, &actionWorld} {
			if cold.Hash() != w.Hash() || cold.entities[0].DrawnFacing() != w.entities[0].DrawnFacing() {
				t.Fatalf("tick %d: cold turn differs", tick)
			}
			Step(cold, nil)
		}
		Step(w, nil)
	}
}

func TestTurnReadsTheNewRateWithoutChangingTheClientCount(t *testing.T) {
	w := &World{entities: []Entity{turnActor(1, 3, 3, 0, 16)}}
	w.entities[0].requestFacing(128)
	w.entities[0].RotationSpeed = 32
	w.advanceTurns()
	e := w.entities[0]
	if e.Facing != 48 || e.TurnRemaining != 4 || e.TurnTotal != 8 || e.DrawnFacing() != 32 {
		t.Fatalf("server/client state after rate change: %+v", e)
	}
}

func TestStoppedActiveTurnKeepsTheLeafForItsNextShortArc(t *testing.T) {
	e := turnActor(1, 3, 3, 0, 16)
	e.requestFacing(128)
	e.clearTurn()
	if !e.TurnState.Active || e.Facing != 16 {
		t.Fatalf("stop lost the active mover flag: %+v", e)
	}
	e.requestFacing(48)
	if e.Facing != 32 || e.TurnRemaining != 2 {
		t.Fatalf("active short arc snapped instead of stepping: %+v", e)
	}
}

func TestTurnStateDecoderRejectsCorruptCountAndTarget(t *testing.T) {
	w := mustWorld(t, 1, Bounds{7, 7}, []Entity{turnActor(1, 3, 3, 0, 16)})
	w.entities[0].requestFacing(128)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(form) - 9 - int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], ^uint32(0)) },
		func(b []byte) { b[start+4+7] = 16 },
	} {
		bad := bytes.Clone(form)
		mutate(bad)
		var cold World
		if cold.UnmarshalBinary(bad) == nil {
			t.Fatal("corrupt turn state accepted")
		}
	}
}

func TestHumanTurnRateUsesOwnSpeedAndRetainsCreatureRate(t *testing.T) {
	human := turnActor(1, 3, 3, 0, 99)
	human.Humanoid, human.Speed, human.GroupSpeed = true, 21, 8
	human.deriveNativeHumanSpeed()
	if human.RotationSpeed != 21 {
		t.Fatalf("formation replaced own turn rate with %d", human.RotationSpeed)
	}
	creature := turnActor(2, 4, 3, 0, 12)
	creature.Speed = 24
	creature.deriveNativeHumanSpeed()
	if creature.RotationSpeed != 12 {
		t.Fatalf("creature rate changed to %d", creature.RotationSpeed)
	}
}

func TestSelfCastWindupBypassesAnActiveFacingGate(t *testing.T) {
	spell := SpellRule{ID: 18, ManaCost: 5, School: 1, MaxRange: 8, TargetsUnit: true,
		Defensive: true, EffectKind: EffectAbsorption, EffectMagnitude: 5}
	caster := spMage(1, 3, 3, 40, 100, 100, 1<<18)
	w := spWorld(t, 1, []SpellRule{spell}, caster)
	if !w.beginBookSpellOnce(0, 1, 18) || len(w.bookCasts) != 1 {
		t.Fatal("self cast was not admitted")
	}
	w.entities[0].RotationSpeed = 16
	w.entities[0].requestFacing(128)
	before := w.bookCasts[0].Remaining
	drawnBefore := w.entities[0].TurnState.DrawRemaining
	Step(w, nil)
	if !w.entities[0].Turning() || w.bookCasts[0].Remaining != before-1 {
		t.Fatal("self cast was held by the non-self facing gate")
	}
	if w.entities[0].TurnState.DrawRemaining != drawnBefore-1 || w.entities[0].DrawnFacing() != 32 {
		t.Fatal("self cast replaced an active client turn")
	}
}

func TestTurnProjectionStoresCurrentByteFlagCounterAndEstimate(t *testing.T) {
	e := turnActor(1, 3, 3, 0, 21)
	e.requestFacing(128)
	m, err := ProjectActorMotion(e, SavedActorMotion{}, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if m.Mover[0] != 21 || m.Mover[1] != 128 || m.Mover[10] != 21 || m.Mover[0x9d] != 1 || m.Mover[0xa4] != 7 || binary.LittleEndian.Uint32(m.Mover[0xa0:]) != 1 || m.ActorAction != 1 {
		t.Fatal("SAV mover did not retain the current turn operands")
	}
}

func TestHumanZeroRateDerivationSettlesAnActiveTurnBeforeSave(t *testing.T) {
	w := mustWorld(t, 1, Bounds{7, 7}, []Entity{turnActor(1, 3, 3, 0, 16)})
	w.entities[0].Humanoid, w.entities[0].Speed = true, 16
	w.entities[0].requestFacing(128)
	if landed, ok := w.applyEffectDelta(0, EffectSpeed, 240); !ok || landed != 240 {
		t.Fatal("speed effect did not land")
	}
	e := w.entities[0]
	if e.RotationSpeed != 0 || e.Turning() || e.Facing != 128 {
		t.Fatal("zero rate retained a non-persistent active turn")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

func TestClientTurnKeepsItsLastFrameAfterTheServerFinishes(t *testing.T) {
	w := &World{entities: []Entity{turnActor(1, 3, 3, 0, 16)}}
	w.entities[0].requestFacing(128)
	w.entities[0].RotationSpeed = 128
	for call := 2; call <= 8; call++ {
		w.advanceTurns()
		if !w.entities[0].DrawingTurn() || w.entities[0].DrawnFacing() != uint8(call*16) {
			t.Fatalf("client call %d did not keep its standing frame", call)
		}
	}
	w.advanceTurns()
	if w.entities[0].DrawingTurn() {
		t.Fatal("completed client run remained active")
	}
}

func TestClientTurnExpiresAtItsCountWhenTheServerSlowsDown(t *testing.T) {
	w := &World{entities: []Entity{turnActor(1, 3, 3, 0, 16)}}
	w.entities[0].requestFacing(128)
	w.entities[0].RotationSpeed = 4
	for call := 2; call <= 8; call++ {
		w.advanceTurns()
	}
	if !w.entities[0].DrawingTurn() || w.entities[0].DrawnFacing() != 128 {
		t.Fatal("last client call lost its target frame")
	}
	w.advanceTurns()
	if !w.entities[0].Turning() || w.entities[0].DrawingTurn() || w.entities[0].DrawnFacing() != 128 {
		t.Fatal("client count followed the slower server instead of expiring")
	}
}

func TestSavedTurnRetainsItsRoundedClientHeadingAfterCompletion(t *testing.T) {
	w := savedTurnWorldForTest(t, 0, 253, 16, 0, 0)
	w.savedOrder(7).Raw[8] = 0xb
	for call := 0; call < 3; call++ {
		Step(w, nil)
		e := w.entities[0]
		if e.Facing != 253 || e.DrawnFacing() != 0 || call > 0 && e.DrawingTurn() {
			t.Fatal("completed saved turn lost its rounded client heading")
		}
		cold := worldRoundTripForTest(t, w)
		if cold.Hash() != w.Hash() || cold.entities[0].DrawnFacing() != 0 {
			t.Fatal("rounded client heading changed after reload")
		}
	}
}

func TestSavedTurnRateChangeKeepsTheMessageDeadline(t *testing.T) {
	w := savedTurnWorldForTest(t, 0, 128, 21, 1, 0)
	w.savedOrder(7).Raw[8] = 0xb
	w.entities[0].HealthRegenPeriod = 100
	w.initializeActionClocks()
	messageClock := uint32(w.tick)
	Step(w, nil)
	deadline := w.entities[0].ActionClock.End
	if deadline != messageClock+7 {
		t.Fatal("new turn did not write its message deadline")
	}
	w.motionFor(7).Mover[10] = 32
	Step(w, nil)
	if w.entities[0].ActionClock.End != deadline {
		t.Fatal("continuing turn rewrote the delivered message deadline")
	}
}
