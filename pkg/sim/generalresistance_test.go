package sim

import (
	"bytes"
	"encoding/binary"
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
