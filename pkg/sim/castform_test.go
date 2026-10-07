package sim

// The CASTING SECTION of the byte form (AC-9, AC-10): a world holding
// pending casts and standing area effects round-trips unchanged, and each of
// the six states no tick can leave is refused by name.
//
// EVERY REFUSAL HERE IS WITNESSED BY SPOILING A GOOD FORM, never by building a
// bad one: a case that assembled its own bytes could differ from a well-formed
// one in more than the one way its name says, and then the refusal it saw would
// not be the refusal it named.

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

// cfWorld is a world holding two pending casts and four area effects, over the
// same fixture spell table the arms are tested against.
//
// The effects span TWO cells and one of them holds three, so the round trip
// covers the ordering the encoder relies on and the decoder checks.
func cfWorld(t *testing.T) *World {
	t.Helper()
	w := scWorld(t, mustScript(t, nil, nil, nil))
	w.casts = []scriptCast{
		{FromX: 42, FromY: 33, ToX: 38, ToY: 33, Spell: uint8(scAreaSpell), Power: 99},
		{FromX: 20, FromY: 15, Spell: uint8(scInertSpell), Power: 100, Target: scTarget, AtUnit: true},
	}
	w.placeCellEffect(cellKey(38, 33), scAreaSpell, 398)
	w.placeCellEffect(cellKey(38, 33), scDamageSpell, 12)
	w.placeCellEffect(cellKey(38, 33), scHealSpell, 1)
	w.placeCellEffect(cellKey(130, 94), scAreaSpell, 60000)
	return w
}

// castingSectionAt is where cfWorld's casting section begins: the whole form
// minus the relation, the script section a world running none writes, the
// STRUCTURE SECTION (1033 B3) between it and the script-state section, and
// the section's own two counts and records, and the SCRIPT-STATE SECTION
// 0166 put between it and the structure section. It is derived from the END
// on strippedOfTheCasting's own reason (binary_test.go) — every section in
// front of it is sized by a count, and the three behind it are fixed on
// this fixture.
func castingSectionAt(t *testing.T, w *World, form []byte) int {
	t.Helper()
	at := len(form) - entityIDFloorLen - spellDeliverySpanLen - 65 - relationLen - w.originalDeadSectionLen() - w.instanceWeightSectionLen() - w.actorLoadSectionLen() - (scriptStateLen + scriptCountsLen) -
		w.scrollSectionLen() - w.itemStateSectionLen() -
		w.structureSectionLen() -
		(relationSlots + tailCountLen) -
		w.castingSectionLen()
	if at <= 0 || at >= len(form) {
		t.Fatalf("the casting section computed to offset %d of a %d-byte form", at, len(form))
	}
	return at
}

// ---------------------------------------------------------------- AC-9

// TestAWorldHoldingCastsAndEffectsRoundTrips is AC-9's first clause: the two
// new kinds of state survive the byte form exactly, and the digest follows.
func TestAWorldHoldingCastsAndEffectsRoundTrips(t *testing.T) {
	t.Parallel()

	w := cfWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if !bytes.Equal(form, again) {
		t.Error("a world holding pending casts and area effects does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the decoded world hashes %#016x and the original %#016x", back.Hash(), w.Hash())
	}
	if got, want := len(back.ScriptCasts()), len(w.ScriptCasts()); got != want {
		t.Errorf("the decoded world holds %d pending cast(s), want %d", got, want)
	}
	for i, e := range back.CellEffects() {
		if !reflect.DeepEqual(e, w.CellEffects()[i]) {
			t.Errorf("decoded effect %d is %+v, want %+v", i, e, w.CellEffects()[i])
		}
	}
}

// TestTwoWorldsDifferingOnlyInACastHashDifferently is AC-9's own discriminating
// half: without it the round trip above would pass over a section the encoder
// never wrote.
func TestTwoWorldsDifferingOnlyInACastHashDifferently(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what   string
		change func(*World)
	}{
		{"one pending cast's spell", func(w *World) { w.casts[0].Spell = uint8(scDamageSpell) }},
		{"one pending cast's power", func(w *World) { w.casts[0].Power = 98 }},
		{"one pending cast's destination", func(w *World) { w.casts[0].ToX = 39 }},
		{"one pending cast's aim", func(w *World) { w.casts[0].AtUnit = true }},
		{"one area effect's remaining lifetime", func(w *World) { w.effects[0].Remaining = 397 }},
		{"one area effect's cell", func(w *World) { w.effects[3].Key = cellKey(131, 94) }},
	} {
		t.Run(tc.what, func(t *testing.T) {
			a, b := cfWorld(t), cfWorld(t)
			tc.change(b)
			if a.Hash() == b.Hash() {
				t.Errorf("two worlds differing in %s hash the same %#016x", tc.what, a.Hash())
			}
		})
	}
}

// TestAVersionFortyEightFormIsRefused is AC-9's last clause. Version 48 wrote
// every byte in front of the spell table identically, so nothing but the version
// check stands between that stream and a decode that would read the script
// section starting eight bytes into a neighbour.
func TestAVersionFortyEightFormIsRefused(t *testing.T) {
	t.Parallel()

	w := cfWorld(t)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	err = back.UnmarshalBinary(withByte(form, 0, 48))
	if err == nil {
		t.Fatal("a version-48 form was accepted")
	}
	if !strings.Contains(err.Error(), "48") {
		t.Errorf("the refusal is %q and does not name the version it refused", err)
	}
}

// ---------------------------------------------------------------- AC-10

// TestTheCastingSectionsSixRefusals is AC-10. Each case spoils exactly one field
// of a well-formed form and names the state it made, and each is checked to
// carry a message a reader can act on.
func TestTheCastingSectionsSixRefusals(t *testing.T) {
	t.Parallel()

	w := cfWorld(t)
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	base := castingSectionAt(t, w, valid)
	cast0 := base + castingCountLen
	effects := cast0 + castRecordLen*len(w.casts) + castingCountLen

	// The section is where this test thinks it is, checked before a single
	// case spoils a byte: both counts must read back as the world's own.
	if got := binary.LittleEndian.Uint32(valid[base : base+4]); int(got) != len(w.casts) {
		t.Fatalf("the pending-cast count at %d reads %d, want %d", base, got, len(w.casts))
	}
	if got := binary.LittleEndian.Uint32(valid[effects-castingCountLen : effects]); int(got) != len(w.effects) {
		t.Fatalf("the area-effect count reads %d, want %d", got, len(w.effects))
	}

	for _, tc := range []struct {
		what string
		form []byte
		says string
	}{
		{"a pending cast naming spell id 0", withByte(valid, cast0+4, 0), "spell id 0"},
		{"a pending cast whose target byte is neither 0 nor 1",
			withByte(valid, cast0+castTargetOffset, 2), "want 0 or 1"},
		{"an area effect naming spell id 0", withU16(valid, effects+3, 0), "spell id 0"},
		{"area effects out of ascending key order",
			withU16(valid, effects+1, cellKey(200, 200)), "ascending key order"},
		{"a seventh area effect on one cell", sevenOnOneCell(t), "a cell holds"},
	} {
		t.Run(tc.what, func(t *testing.T) {
			var back World
			err := back.UnmarshalBinary(tc.form)
			if err == nil {
				t.Fatalf("%s was accepted", tc.what)
			}
			if !strings.Contains(err.Error(), tc.says) {
				t.Errorf("the refusal is %q, which does not say %q", err, tc.says)
			}
		})
	}
}

// sevenOnOneCell builds a form declaring seven effects on one key. It cannot be
// made by spoiling one byte of cfWorld's form — the world's own placement
// refuses the seventh — so it is assembled by rewriting the effect list of a
// form the encoder wrote, which is the smallest departure that reaches the state.
func sevenOnOneCell(t *testing.T) []byte {
	t.Helper()
	w := scWorld(t, mustScript(t, nil, nil, nil))
	for i := 0; i < cellEffectSlots; i++ {
		w.placeCellEffect(cellKey(3, 4), uint16(i+1), 100)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	at := castingSectionAt(t, w, form)
	countAt := at + castingCountLen + castRecordLen*len(w.casts)
	first := countAt + castingCountLen

	out := append([]byte(nil), form[:countAt]...)
	out = binary.LittleEndian.AppendUint32(out, uint32(cellEffectSlots+1))
	out = append(out, form[first:first+effectRecordLen*cellEffectSlots]...)
	seventh := append([]byte(nil), form[first:first+effectRecordLen]...)
	binary.LittleEndian.PutUint16(seventh[3:5], 99)
	out = append(out, seventh...)
	return append(out, form[first+effectRecordLen*cellEffectSlots:]...)
}

// TestTheCastingSectionsCountsAreBoundedAgainstTheBuffer is the two declared
// counts' own guard: a count the payload cannot supply is refused before a
// single record is allocated.
func TestTheCastingSectionsCountsAreBoundedAgainstTheBuffer(t *testing.T) {
	t.Parallel()

	w := cfWorld(t)
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	base := castingSectionAt(t, w, valid)
	effectsCountAt := base + castingCountLen + castRecordLen*len(w.casts)

	for _, tc := range []struct {
		what string
		form []byte
	}{
		{"a pending-cast count the payload cannot supply", withU32(valid, base, 0xffffff)},
		{"an area-effect count the payload cannot supply", withU32(valid, effectsCountAt, 0xffffff)},
	} {
		t.Run(tc.what, func(t *testing.T) {
			var back World
			if err := back.UnmarshalBinary(tc.form); err == nil {
				t.Errorf("%s was accepted", tc.what)
			}
		})
	}
}
