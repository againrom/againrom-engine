package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestGeneralResistanceUsesObservedSourceByte(t *testing.T) {
	a, target := cbEnt(1, 0, 0), cbEnt(2, 1, 0)
	a.DamageBase, a.ToHit, a.XPSlot = 100, 2147483647, 0
	target.HP, target.MaxHP = 1000, 1000
	w := cbWorld(t, 17, a, target)
	w.entities[1].ActorLoad.Source.Class = 1
	w.entities[1].ActorLoad.Source.Defence[16] = 50
	w.resolveBlow(0, 1, nil)
	if got := 1000 - cbAt(t, w, 2).HP; got != 50 {
		t.Fatal("General ignored independently observed slot0 resistance", got)
	}
}

func TestNativeGeneralResistanceAvailabilityColdHashAndTypedSlot(t *testing.T) {
	for _, tc := range []struct {
		name           string
		present, known bool
		value, slot    uint8
		want           int32
	}{
		{"known50", true, true, 50, 0, 50}, {"knownzero", true, true, 0, 0, 100},
		{"unknown", true, false, 50, 0, 100}, {"absent", false, false, 0, 0, 100},
		{"typedBlade", true, true, 50, 1, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, target := cbEnt(0, 0, 0), cbEnt(3, 1, 0)
			a.DamageBase, a.ToHit, a.XPSlot = 100, 2147483647, tc.slot
			target.HP, target.MaxHP = 1000, 1000
			target.Resistance[0] = 60
			target.NativeBasis.DefencePresent = tc.present
			target.NativeBasis.Defence[16] = tc.value
			if tc.known {
				target.NativeBasis.DefenceKnown = 1 << 16
			}
			w := cbWorld(t, 17, a, target)
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
				t.Fatal("cold current resistance lost", err)
			}
			if tc.present {
				before := w.Hash()
				w.entities[1].NativeBasis.Defence[16] ^= 1
				if w.Hash() == before {
					t.Fatal("represented resistance/residue excluded from hash")
				}
				w.entities[1].NativeBasis.Defence[16] ^= 1
			}
			w.resolveBlow(0, 1, nil)
			cold.resolveBlow(0, 1, nil)
			if got := 1000 - cbAt(t, &cold, 3).HP; got != tc.want || cold.Hash() != w.Hash() {
				t.Fatal("real General/typed hit lost current availability", got, tc.want)
			}
		})
	}
}

func TestNativeLiveBlocksCanonicalAvailabilityAndAtomicDecode(t *testing.T) {
	w := nativeBasisWorld(t)
	old, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b := NativeActorBasis{AttackPresent: true, AttackKnown: 3 << 22, Attack: [24]byte{22: 0xa5, 23: 0x5a}, DefencePresent: true, DefenceKnown: 1 << 16, Defence: [22]byte{4: 0xc7, 16: 50}}
	w.entities[0].NativeBasis = b
	w.entities[1].NativeBasis = NativeActorBasis{DefencePresent: true, Defence: [22]byte{5: 0xcd}}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckSaveForm(raw); err != nil {
		t.Fatal("current actor history form refused", err)
	}
	if raw[0] != nativeLiveFormVersion || !bytes.Equal(raw[len(raw)-4:], []byte("NLB1")) {
		t.Fatal("native live envelope absent")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != b || cold.Hash() != w.Hash() {
		t.Fatal("known bytes or unknown mask residue lost", err)
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	for _, at := range []int{start, start + 8, start + 12, start + 40} {
		bad := bytes.Clone(raw)
		bad[at] = 255
		if CheckSaveForm(bad) == nil {
			t.Fatal("malformed actor history form admitted")
		}
		before := cold.Hash()
		if cold.UnmarshalBinary(bad) == nil || cold.Hash() != before {
			t.Fatal("malformed native live decode accepted or changed state", at)
		}
	}
	if err := cold.UnmarshalBinary(old); err != nil || cold.entities[0].NativeBasis.HasValues() {
		t.Fatal("old byte form invented current raw history", err)
	}
}

// TestActiveSlotZeroStrikeReadsTheDamageKindByteAtCE is HERO-DKIDX-162 for an
// attacker whose active slot is 0: bare hands, and an elemental weapon such as
// the Flame Thrower, whose ranged fold clears the slot and adds a Fire
// secondary pair. The physical part is reduced by the target's byte at actor
// +0xce with ftol(damage*(100-v)/100+0.75); zero skips the step. The Fire part
// is reduced by Fire protection alone. Slots 1..5 keep reading their own byte.
func TestActiveSlotZeroStrikeReadsTheDamageKindByteAtCE(t *testing.T) {
	const physical, fire = 97, 13
	attackers := []struct {
		name      string
		secondary SecondaryDamage
		extra     int32
	}{
		{"bareHands", SecondaryDamage{}, 0},
		{"flameThrower", SecondaryDamage{Base: fire, Selector: 0}, fire},
	}
	for _, at := range attackers {
		for _, v := range []uint8{0, 50, 100} {
			want := int32(resistPhysicalDamage(physical, v)) + at.extra
			for _, sourced := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/sourced=%t", at.name, v, sourced), func(t *testing.T) {
					a, target := cbEnt(1, 0, 0), cbEnt(2, 1, 0)
					a.DamageBase, a.ToHit, a.XPSlot, a.SecondaryDamage = physical, 2147483647, 0, at.secondary
					target.HP, target.MaxHP = 1000, 1000
					target.Resistance = [5]uint8{100, 100, 100, 100, 100}
					w := cbWorld(t, 23, a, target)
					if sourced {
						w.entities[1].ActorLoad.Source.Class = 1
						w.entities[1].ActorLoad.Source.Defence[16] = v
					} else {
						w.entities[1].NativeBasis.DefencePresent = true
						w.entities[1].NativeBasis.DefenceKnown = 1 << 16
						w.entities[1].NativeBasis.Defence[16] = v
					}
					w.resolveBlow(0, 1, nil)
					if got := 1000 - cbAt(t, w, 2).HP; got != want {
						t.Fatalf("slot-0 strike against +0xce=%d dealt %d, want %d", v, got, want)
					}
				})
			}
		}
	}
	if resistPhysicalDamage(physical, 50) != 49 || resistPhysicalDamage(physical, 100) != 0 || resistPhysicalDamage(physical, 0) != physical {
		t.Fatal("ftol(97*(100-v)/100+0.75) is 97, 49 and 0 for v 0, 50 and 100")
	}
	for slot := uint8(1); slot <= 5; slot++ {
		for _, ce := range []uint8{0, 100} {
			a, target := cbEnt(1, 0, 0), cbEnt(2, 1, 0)
			a.DamageBase, a.ToHit, a.XPSlot = physical, 2147483647, slot
			target.HP, target.MaxHP = 1000, 1000
			target.Resistance[slot-1] = 20
			target.NativeBasis.DefencePresent, target.NativeBasis.DefenceKnown = true, 1<<16
			target.NativeBasis.Defence[16] = ce
			w := cbWorld(t, 23, a, target)
			w.resolveBlow(0, 1, nil)
			if got, want := 1000-cbAt(t, w, 2).HP, int32(resistPhysicalDamage(physical, 20)); got != want {
				t.Fatalf("slot %d with +0xce=%d dealt %d, want its own byte's %d", slot, ce, got, want)
			}
		}
	}
}

// TestGodOrderLeavesASlotZeroStrikeNoDamage: the `+god` writer stores 100 in
// every damage-kind byte (HERO-MODDK-161), so a slot-0 strike on its target
// deals ftol(damage*0/100+0.75) = 0, before and after a cold load.
func TestGodOrderLeavesASlotZeroStrikeNoDamage(t *testing.T) {
	a, target := cbEnt(1, 0, 0), cbEnt(2, 1, 0)
	a.DamageBase, a.ToHit, a.XPSlot = 97, 2147483647, 0
	target.HP, target.MaxHP = 1000, 1000
	w := cbWorld(t, 23, a, target)
	if !w.CheatGod(2) {
		t.Fatal("god order refused")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("cold load of the god target", err)
	}
	for _, world := range []*World{w, &cold} {
		world.resolveBlow(0, 1, nil)
		if got := 1000 - cbAt(t, world, 2).HP; got != 0 {
			t.Fatalf("slot-0 strike on a god target dealt %d, want 0", got)
		}
	}
}
