package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// overloadHuman is a native Humanoid whose load is twenty times its capacity,
// so the overload penalty reaches the floor of six whatever its base.
func overloadHuman(t *testing.T, load int32) (*World, int) {
	t.Helper()
	w := mustWorld(t, 1, fmBounds, []Entity{{ID: 1, X: 4, Y: 4, HP: 30, MaxHP: 30, Humanoid: true,
		Speed: 20, Capacity: 301, RotationSpeed: 99}})
	w.entities[0].Load = load // construction recomputes Load from an empty inventory
	return w, 0
}

func rearmSpeed(t *testing.T, w *World, speed, modifier int32) {
	t.Helper()
	if !w.SetDerived(1, DerivedBlock{MaxHP: 30, Speed: speed, SpeedModifier: modifier, Capacity: 301, Combat: CombatBlock{Reach: 1}}) {
		t.Fatal("SetDerived refused")
	}
}

func assertNativeSpeed(t *testing.T, e Entity, speed, modifier, word int32, turn uint8) {
	t.Helper()
	if e.Speed != speed || e.SpeedModifier != modifier || aloneSpeed(e) != word || e.RotationSpeed != int32(turn) || e.SpeedWord() != word {
		t.Fatalf("speed %d modifier %d word %d turn %d; want %d, %d, %d, %d",
			e.Speed, e.SpeedModifier, aloneSpeed(e), e.RotationSpeed, speed, modifier, word, turn)
	}
}

func TestNativeHumanFloorBindsBeforeAPositiveModifier(t *testing.T) {
	w, i := overloadHuman(t, 6020)
	rearmSpeed(t, w, 25, 5)
	// Base 20 less 6020/301 floors at 6; the modifier then adds 5. The old
	// order floored 25 less 20 at 6.
	assertNativeSpeed(t, w.entities[i], 25, 5, 11, 11)
	if rate, _, adj, ok := w.StepRate(1, 5, 4); !ok || !adj || rate != rateOf(DomainGround, 11, 0, 0, 0, 0) {
		t.Fatalf("step rate %d, want the rate of speed 11", rate)
	}
}

func TestNativeHumanZeroLoadKeepsBasePlusModifier(t *testing.T) {
	w, i := overloadHuman(t, 0)
	rearmSpeed(t, w, 25, 5)
	assertNativeSpeed(t, w.entities[i], 25, 5, 25, 25)
}

func TestSlowOnAnOverloadedHumanClearsTheModifier(t *testing.T) {
	w, i := overloadHuman(t, 6020)
	rearmSpeed(t, w, 20, 0)
	assertNativeSpeed(t, w.entities[i], 20, 0, 6, 6)
	if landed, ok := w.applyEffectDelta(i, EffectSpeed, -15); !ok || landed != -15 {
		t.Fatal("Slow did not land", landed, ok)
	}
	// 6 - 15 = -9: the word stays negative and the modifier is cleared, so
	// Speed returns to the base. The turn rate is the word's low byte.
	e := w.entities[i]
	assertNativeSpeed(t, e, 20, 0, -9, 247)
	if !rated(e) {
		t.Fatal("a negative word lost its rate")
	}
	if rate, _, adj, ok := w.StepRate(1, 5, 4); !ok || !adj || rate != rateFloor {
		t.Fatalf("step rate %d, want the rate floor %d", rate, rateFloor)
	}
	// The original adds the expiry to the cleared modifier: 0 + 15.
	if landed, ok := w.applyEffectDelta(i, EffectSpeed, 15); !ok || landed != 15 {
		t.Fatal("expiry did not land", landed, ok)
	}
	assertNativeSpeed(t, w.entities[i], 35, 15, 21, 21)
}

func TestLoadChangeRederivesTheNativeWord(t *testing.T) {
	w, i := overloadHuman(t, 0)
	rearmSpeed(t, w, 25, 5)
	before := w.beginLoadMutation(i)
	w.entities[i].ActorLoad = ActorLoad{Present: true, OwnWeight: 6020}
	if !w.finishLoadMutation(i, before) {
		t.Fatal("load mutation refused")
	}
	assertNativeSpeed(t, w.entities[i], 25, 5, 11, 11)
}

func TestSpeedModifierByteForm(t *testing.T) {
	w, _ := overloadHuman(t, 6020)
	plain, err := w.MarshalBinary()
	if err != nil || plain[0] == speedModifierFormVersion {
		t.Fatal("a world without a modifier carries the section", err)
	}
	rearmSpeed(t, w, 25, -3)
	form, err := w.MarshalBinary()
	if err != nil || form[0] != speedModifierFormVersion || CheckSaveForm(form) != nil {
		t.Fatal("modifier section", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() || cold.entities[0].SpeedModifier != -3 {
		t.Fatal("modifier round trip", err)
	}
	start := len(form) - 9 - int(binary.LittleEndian.Uint32(form[len(form)-9:]))
	for _, mutate := range []func([]byte){
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 77) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+8:], 0) },
	} {
		bad := bytes.Clone(form)
		mutate(bad)
		var c World
		if c.UnmarshalBinary(bad) == nil {
			t.Fatal("corrupt speed modifier accepted")
		}
	}
	unit, err := NewWorld(1, fmBounds, ModeCanonical, nil, []Entity{{ID: 1, X: 4, Y: 4, HP: 30, MaxHP: 30, Speed: 20, SpeedModifier: 4}})
	if err == nil {
		_, err = unit.MarshalBinary()
	}
	if err == nil {
		t.Fatal("a Unit's speed modifier marshalled")
	}
}

// A native Human loaded from a SAV reads its modifier from the ordinary
// modifier speed word. Without an operand (a SAV written before the split)
// the speed word is the whole Speed; an operand applies while its wires match.
func TestNativeHumanSpeedSplitOnLOAD(t *testing.T) {
	loaded := func(speed int32, modifier uint16) Entity {
		e := Entity{Humanoid: true, Speed: speed}
		e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 2}}
		binary.LittleEndian.PutUint16(e.ActorLoad.Source.Modifier[4:], modifier)
		return e
	}
	split := &ActorSpeedSplit{SpeedWire: 11, ModifierWire: 5, Speed: 22, Modifier: 5}
	for _, tc := range []struct {
		name            string
		e               Entity
		split           *ActorSpeedSplit
		speed, modifier int32
	}{
		{"an old SAV splits by the modifier word", loaded(22, 5), nil, 22, 5},
		{"a negative modifier word is signed", loaded(5, 0xfff1), nil, 5, -15},
		{"the operand restores Speed", loaded(11, 5), split, 22, 5},
		{"an edited speed word wins", loaded(14, 5), split, 14, 5},
		{"an edited modifier word wins", loaded(11, 2), split, 22, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.e
			if err := e.restoreValues(ActorValues{LoadPresent: true, SpeedSplit: tc.split}); err != nil {
				t.Fatal(err)
			}
			if e.Speed != tc.speed || e.SpeedModifier != tc.modifier || e.ActorLoad.Source.Class != 0 {
				t.Fatalf("restored speed %d modifier %d source %d; want %d, %d, 0", e.Speed, e.SpeedModifier, e.ActorLoad.Source.Class, tc.speed, tc.modifier)
			}
		})
	}
	// A Human restored as source-backed holds its modifier in the source
	// record; a modifier left from its spawn is dropped.
	stale := loaded(22, 5)
	stale.SpeedModifier = 4
	if err := stale.restoreValues(ActorValues{LoadPresent: true, SourceClass: 2}); err != nil || stale.SpeedModifier != 0 {
		t.Fatal("a source-backed restore kept a native modifier", err, stale.SpeedModifier)
	}
	unit := Entity{Speed: 10, ActorLoad: ActorLoad{Present: true, Source: SourceActor{Class: 1}}}
	if unit.restoreValues(ActorValues{LoadPresent: true, SourceClass: 1, SpeedSplit: split}) == nil {
		t.Fatal("a speed split on a source Unit was accepted")
	}
}
