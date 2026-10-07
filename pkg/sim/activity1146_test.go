package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// Independent transcription: form86 adds one empty four-byte suffix to these
// legacy fixtures. These adapters never consult the production codec.
func widenedActionClockPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 86
	return widenedGroupRoamPin(out)
}

func strippedActionClockPin(form []byte) []byte {
	out := strippedGroupRoamPin(form)
	if len(out) > 0 && out[0] >= 86 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 85
	}
	return out
}

func TestIdleRegen1146SignedDeadlineAndBothConsumers(t *testing.T) {
	for _, tc := range []struct {
		now, end uint32
		rate     int32
	}{
		{180, 100, 1}, {181, 100, 3}, {99, 100, 1},
		{64, 0xfffffff0, 1}, {65, 0xfffffff0, 3},
		{0x80000000, 0, 1}, {0x7fffffff, 0, 3},
	} {
		for _, original := range []bool{false, true} {
			t.Run(fmt.Sprintf("%x-%x/original=%t", tc.now, tc.end, original), func(t *testing.T) {
				e := Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 100, Mana: 1, MaxMana: 100,
					HealthRegenPeriod: 100, ManaRegenPeriod: 100,
					ActionClock: ActionClock{Known: true, End: tc.end}}
				w := rgWorld(t, e)
				if original {
					w.entities[0].CurrentProfileBasis = ProfileOriginalCurrent
				}
				w.tick = uint64(tc.now)
				w.regenerateActors(true)
				got := w.entities[0]
				if got.HP != 10+2*tc.rate || got.Mana != 1+tc.rate {
					t.Fatalf("pools %d/%d want %d/%d", got.HP, got.Mana, 10+2*tc.rate, 1+tc.rate)
				}
			})
		}
	}
}

func TestIdleRegen1146NativePositiveGainDoesNotOverflowAtRate3(t *testing.T) {
	w := rgWorld(t, Entity{ID: 1, HP: 10, MaxHP: 2000000000, HealthRegenPeriod: 7,
		HealthRegeneration: 1500000000, ActionClock: ActionClock{Known: true}})
	w.tick = 100
	w.regenerateActors(true)
	// (2000000000*2*1500000100*3)/7 = 2571428742857142857.
	// Add1000; the remainder is57 and the quotient caps at the maximum.
	if e := w.entities[0]; e.HP != 2000000000 || e.HealthHundredths != 57 {
		t.Fatalf("positive rate3 gain wrapped: HP%d remainder%d", e.HP, e.HealthHundredths)
	}
}

func TestIdleRegen1146ActionProducersAndReload(t *testing.T) {
	for _, kind := range []string{"move", "turn", "attack", "unit-cast", "cell-cast", "scroll"} {
		t.Run(kind, func(t *testing.T) {
			w := manualCastWorld(t)
			if kind == "scroll" {
				w = scrollWorld1090(t, 3, 1)
			}
			w.tick = 1000
			e := &w.entities[0]
			e.HealthRegenPeriod, e.ManaRegenPeriod = 100, 100
			e.ActionClock = ActionClock{Known: true}
			if e.regenerationRate(w.tick) != 3 {
				t.Fatal("fixture is not idle")
			}
			var cmd Command
			wantEnd := uint32(1020) // wind-up8 plus base recovery12
			switch kind {
			case "move", "turn":
				cmd = Command{Entity: e.ID, X: 3, Y: 8}
				wantEnd = 1001
				if kind == "turn" {
					e.Facing, e.DesiredFacing, e.RotationSpeed = 0, 0, 16
					wantEnd = 1008 // half-turn128 at speed16
				}
			case "attack":
				cmd = Command{Kind: KindAttack, Entity: e.ID, X: 2}
			case "unit-cast":
				cmd = spCast(e.ID, 2, 1)
			case "cell-cast":
				cmd = Command{Kind: KindCastAt, Entity: e.ID, Spell: 26, X: 6, Y: 3}
			case "scroll":
				cmd = Command{Kind: KindUseScroll, Entity: e.ID, X: 2}
				wantEnd = 1011 // next-tick admission, minimum wind-up8 plus recovery2
			}
			for n := 0; n < 20 && w.entities[0].ActionClock.End == 0; n++ {
				var cmds []Command
				if n == 0 {
					cmds = []Command{cmd}
				}
				Step(w, cmds)
			}
			if w.entities[0].ActionClock.End != wantEnd || w.entities[0].regenerationRate(w.tick) != 1 {
				t.Fatalf("action clock %+v, want deadline%d", w.entities[0].ActionClock, wantEnd)
			}
			if kind == "scroll" && (len(w.scrollCasts) != 1 || !w.scrollCasts[0].Started) {
				t.Fatal("scroll did not reach its casting interval")
			}
			back := retreatRoundTrip1089(t, w)
			for n := 0; n < 192; n++ {
				Step(w, nil)
				Step(back, nil)
				if w.Hash() != back.Hash() {
					t.Fatalf("reload differs at %d", n)
				}
			}
		})
	}
}

func TestIdleRegen1146RefusedClickDoesNotResetDeadline(t *testing.T) {
	w := manualCastWorld(t)
	w.entities[0].ManaRegenPeriod = 100
	w.entities[0].ActionClock = ActionClock{Known: true, End: 20}
	w.tick = 1000
	Step(w, []Command{{Kind: KindCastAt, Entity: 1, Spell: 26, X: -1}})
	if w.entities[0].ActionClock.End != 20 || w.entities[0].regenerationRate(w.tick) != 3 {
		t.Fatal("rejected click broke the idle bonus")
	}
}

func TestIdleRegen1146LiteralSuffixAndAtomicRefusals(t *testing.T) {
	w := rgWorld(t, Entity{ID: 1, HP: 10, MaxHP: 100,
		ActionClock: ActionClock{Known: true, End: 0xfffffff0}})
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b = strippedScorchedPin(b)
	if b[0] != 87 || !bytes.Equal(b[len(b)-16:len(b)-4], []byte{1, 0, 0, 0, 0xf0, 0xff, 0xff, 0xff, 8, 0, 0, 0}) {
		t.Fatal("literal form86 action clock suffix")
	}
	for _, corrupt := range []func([]byte){
		func(v []byte) { binary.LittleEndian.PutUint32(v[len(v)-8:], 0xffffffff) },
		func(v []byte) { binary.LittleEndian.PutUint32(v[len(v)-8:], 7) },
		func(v []byte) { binary.LittleEndian.PutUint32(v[len(v)-16:], 99) },
	} {
		bad := append([]byte(nil), b...)
		corrupt(bad)
		before := w.Hash()
		if err := w.UnmarshalBinary(widenedScorchedPin(bad)); err == nil || before != w.Hash() {
			t.Fatal("non-atomic invalid clock")
		}
	}
}
