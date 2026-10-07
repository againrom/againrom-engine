package sim

import "testing"

const (
	xpSkillAt = 149
	xpMindAt  = 173
	xpValueAt = 177
	xpSlotAt  = 181
	xpGainsAt = 182
)

// experienceEntity is an entity with every one of this task's five fields
// set to a distinct, nonzero value: SkillXP ascending by ten so a decode
// that dropped or transposed a slot fails on more than one field, a Mind
// and an experience value that share no digit with a slot, a credited slot
// inside 1..5 — not 0, so a decode that dropped the field cannot hide
// behind General's own zero — and a gains flag of true.
func experienceEntity(id EntityID) Entity {
	return Entity{ID: id, X: 1, Y: 1,
		SkillXP: [skillSlots]int32{10, 20, 30, 40, 50, 60},
		Mind:    77, XPValue: 88, XPSlot: 3, GainsXP: true}
}

// ---------------------------------------------------------------- AC-2

// TestAnEntitysExperienceRoundTripsByteIdentically is AC-2's first half: an
// entity holding every one of this task's five fields survives a marshal,
// an unmarshal and a second marshal producing byte-identical bytes — not
// merely an equal world, the same encoding — and each field decodes back
// to exactly the value it was given.
func TestAnEntitysExperienceRoundTripsByteIdentically(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{experienceEntity(1)})
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	second, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("a world holding an entity's experience does not round-trip through its byte form:\n"+
			" % x\n % x", first, second)
	}

	got, want := back.Entities()[0], experienceEntity(1)
	if got.SkillXP != want.SkillXP {
		t.Errorf("SkillXP decoded as %v, want %v", got.SkillXP, want.SkillXP)
	}
	if got.Mind != want.Mind {
		t.Errorf("Mind decoded as %d, want %d", got.Mind, want.Mind)
	}
	if got.XPValue != want.XPValue {
		t.Errorf("XPValue decoded as %d, want %d", got.XPValue, want.XPValue)
	}
	if got.XPSlot != want.XPSlot {
		t.Errorf("XPSlot decoded as %d, want %d", got.XPSlot, want.XPSlot)
	}
	if got.GainsXP != want.GainsXP {
		t.Errorf("GainsXP decoded as %v, want %v", got.GainsXP, want.GainsXP)
	}
}

// TestEachSkillSlotMovesTheDigestAndNoneCollide is AC-2's second half: the
// digest changes when any ONE of the six slots does, and the six changes
// are pairwise distinct — the digest says which slot moved, not merely
// that one did (TestEveryFieldChangesTheDigest's own rule, hash_test.go).
func TestEachSkillSlotMovesTheDigestAndNoneCollide(t *testing.T) {
	t.Parallel()

	base := func() *World {
		return mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{experienceEntity(1)})
	}
	unchanged := base().Hash()

	got := make([]uint64, skillSlots)
	for i := 0; i < skillSlots; i++ {
		w := base()
		w.entities[0].SkillXP[i]++
		got[i] = w.Hash()
		if got[i] == unchanged {
			t.Errorf("changing slot %d left the digest at %#016x", i, unchanged)
		}
	}
	for i := range got {
		for j := i + 1; j < len(got); j++ {
			if got[i] == got[j] {
				t.Errorf("slot %d and slot %d both hash %#016x", i, j, got[i])
			}
		}
	}
}

func TestMindXPValueXPSlotAndGainsXPEachMoveTheDigest(t *testing.T) {
	t.Parallel()

	base := func() *World {
		return mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{experienceEntity(1)})
	}
	unchanged := base().Hash()

	cases := []struct {
		name   string
		change func(*Entity)
	}{
		{"Mind", func(e *Entity) { e.Mind++ }},
		{"XPValue", func(e *Entity) { e.XPValue++ }},
		{"XPSlot", func(e *Entity) { e.XPSlot = 4 }},
		{"GainsXP", func(e *Entity) { e.GainsXP = false }},
	}
	for _, tc := range cases {
		w := base()
		tc.change(&w.entities[0])
		if got := w.Hash(); got == unchanged {
			t.Errorf("changing %s left the digest at %#016x", tc.name, unchanged)
		}
	}
}

// mustMarshalExperienceWorld is the one-entity, one-record fixture the
// decoder refusals below spoil: Bounds{4,4} so the record's own base is
// headerLen + 3*16, written out rather than read back off the encoder.
func mustMarshalExperienceWorld(t *testing.T) ([]byte, int) {
	t.Helper()
	valid := mustMarshal(t, mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{experienceEntity(1)}))
	return valid, headerLen + 3*16
}

func TestNewWorldRefusesAnExperienceSlotOutsideZeroToFive(t *testing.T) {
	t.Parallel()

	for _, slot := range []uint8{6, 255} {
		e := Entity{ID: 1, X: 1, Y: 1, XPSlot: slot}
		if _, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e}); err == nil {
			t.Errorf("NewWorld accepted a credited slot of %d", slot)
		}
	}
	// The top of the legal range is not refused, so the case above measures
	// the boundary and not merely "every slot is refused".
	e := Entity{ID: 1, X: 1, Y: 1, XPSlot: skillSlots - 1}
	if _, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e}); err != nil {
		t.Errorf("a credited slot of %d (the top of the six) was refused: %v", skillSlots-1, err)
	}
}

// TestNewWorldCarriesSignedSlotExperience is the item-Poison boundary: the
// original sink accepts signed awards, so every slot must preserve a negative
// bank without lowering its separately stored level.
func TestNewWorldCarriesSignedSlotExperience(t *testing.T) {
	t.Parallel()

	for i := 0; i < skillSlots; i++ {
		e := Entity{ID: 1, X: 1, Y: 1}
		e.SkillXP[i] = -1
		w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e})
		if err != nil || w.entities[0].SkillXP[i] != -1 {
			t.Errorf("NewWorld signed slot %d = %d, %v; want -1, nil", i, w.entities[0].SkillXP[i], err)
		}
	}
	if _, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 1, Y: 1, SkillXP: [skillSlots]int32{0, 0, 0, 0, 0, 0}}}); err != nil {
		t.Errorf("six zero experiences were refused: %v", err)
	}
}

func TestUnmarshalRefusesAnExperienceSlotOutsideZeroToFive(t *testing.T) {
	t.Parallel()

	valid, base := mustMarshalExperienceWorld(t)
	for _, v := range []byte{6, 0xff} {
		var w World
		if err := w.UnmarshalBinary(withByte(valid, base+xpSlotAt, v)); err == nil {
			t.Errorf("a credited slot of %d was accepted", v)
		}
	}
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

// TestUnmarshalCarriesSignedSlotExperience is the decoder half of the same
// signed persistence rule.
func TestUnmarshalCarriesSignedSlotExperience(t *testing.T) {
	t.Parallel()

	valid, base := mustMarshalExperienceWorld(t)
	for i := 0; i < skillSlots; i++ {
		var w World
		spoiled := withU32(valid, base+xpSkillAt+4*i, 0xffffffff) // -1
		if err := w.UnmarshalBinary(spoiled); err != nil || w.entities[0].SkillXP[i] != -1 {
			t.Errorf("decoded signed slot %d = %d, %v; want -1, nil", i, w.entities[0].SkillXP[i], err)
		}
	}
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

func TestUnmarshalRefusesAGainsXPByteOtherThanZeroOrOne(t *testing.T) {
	t.Parallel()

	valid, base := mustMarshalExperienceWorld(t)
	for _, v := range []byte{2, 0xff} {
		var w World
		if err := w.UnmarshalBinary(withByte(valid, base+xpGainsAt, v)); err == nil {
			t.Errorf("a gains-xp byte of %d was accepted", v)
		}
	}
	if err := (&World{}).UnmarshalBinary(valid); err != nil {
		t.Errorf("the unspoiled form was refused: %v", err)
	}
}

func TestMindAndXPValueAreCarriedWholeAndRefusedNowhere(t *testing.T) {
	t.Parallel()

	for _, v := range []int32{-2147483648, -1, 0, 2147483647} {
		e := Entity{ID: 1, X: 1, Y: 1, Mind: v, XPValue: v}
		w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e})
		if err != nil {
			t.Fatalf("Mind and XPValue of %d refused by the constructor: %v", v, err)
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("Mind and XPValue of %d refused by the decoder: %v", v, err)
		}
		got := back.Entities()[0]
		if got.Mind != v || got.XPValue != v {
			t.Errorf("Mind/XPValue round-tripped as %d/%d, want %d/%d", got.Mind, got.XPValue, v, v)
		}
	}
}
