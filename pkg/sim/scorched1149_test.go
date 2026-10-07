package sim

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"
)

func widenedScorchedPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0, 0, 0, 0, 0)
	out[0] = 89
	return widenedSavedFormationPin(out)
}

func strippedScorchedPin(form []byte) []byte {
	out := strippedSavedFormationPin(form)
	if len(out) > 0 && out[0] >= 89 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 88
	}
	if len(out) > 0 && out[0] >= 88 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 87
	}
	return out
}

func TestScorched1149FirePlacementExpiryAndContinuation(t *testing.T) {
	w := fireWallHotfixWorld(t)
	rule := w.spells[0]
	if !w.landArea(rule, 0, 0, false, 20, 20, 24, 20, nil) {
		t.Fatal("wall did not land")
	}
	want := []uint16{18*256 + 24, 18*256 + 25, 19*256 + 24, 19*256 + 25, 20*256 + 24, 20*256 + 25, 21*256 + 24, 21*256 + 25, 22*256 + 24, 22*256 + 25}
	if !slices.Equal(w.ScorchedCells(), want) {
		t.Fatalf("wall footprint %v want %v", w.ScorchedCells(), want)
	}
	clone := w.ScorchedCells()
	clone[0] = 0
	if !slices.Equal(w.ScorchedCells(), want) {
		t.Fatal("read aliases world")
	}
	for range 40 {
		cold := retreatRoundTrip1089(t, w)
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() || !slices.Equal(cold.ScorchedCells(), want) {
			t.Fatal("cold continuation lost history")
		}
	}
	if len(w.CellEffects()) != 0 || !slices.Equal(w.ScorchedCells(), want) {
		t.Fatal("expiry removed scorch")
	}
	before, rng := slices.Clone(w.grid), w.rng
	w.scorchCells(2, []uint16{65535, 257, 257, 0})
	w.scorchCells(7, []uint16{258})
	if slices.Contains(w.scorchedCells, 65535) || slices.Contains(w.scorchedCells, 258) || !slices.Contains(w.scorchedCells, 257) || !slices.Equal(before, w.grid) || rng != w.rng {
		t.Fatal("bounds, spell admission, collision or RNG changed")
	}
}

func TestScorched1149BlastAndRefusedWall(t *testing.T) {
	w := fireWallHotfixWorld(t)
	if !w.landArea(SpellRule{ID: 2, Area: true, Radius: 1}, 0, 0, false, 1, 1, 0, 0, nil) {
		t.Fatal("blast refused")
	}
	// The blast clips against the packed-cell boundary before the history is
	// bounded against the world. It leaves no continuing effect record.
	if !slices.Equal(w.ScorchedCells(), []uint16{0, 1, 256, 257}) || len(w.CellEffects()) != 0 {
		t.Fatal("blast footprint or lifetime changed", w.ScorchedCells())
	}
	before := w.Hash()
	if w.landArea(SpellRule{ID: 3, Area: true, Distribution: distributionStaged, Radius: 2, AreaDuration: 1}, 0, 0, false, 20, 20, 24, 20, nil) || w.Hash() != before {
		t.Fatal("refused wall changed scenery or another field")
	}
}

func TestScorched1149LiteralFooterAndAtomicValidation(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 32, Height: 32}, ModeCanonical, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.scorchCells(2, []uint16{0x0204, 0x0103, 0x0103})
	b := mustMarshal(t, w)
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(b) - entityIDFloorLen - spellDeliverySpanLen
	if b[0] != formatVersion || !bytes.Equal(b[end-24:end-16], []byte{3, 1, 4, 2, 4, 0, 0, 0}) {
		t.Fatal("literal footer")
	}
	for _, mutate := range []func([]byte){
		func(b []byte) { b[len(b)-8] = 255 },
		func(b []byte) { b[len(b)-6], b[len(b)-5] = 3, 1 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-4:], 3) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-4:], 0xffffffff) },
	} {
		bad := bytes.Clone(b)
		mutate(bad[:len(bad)-entityIDFloorLen-spellDeliverySpanLen-16])
		hash := w.Hash()
		if w.UnmarshalBinary(bad) == nil || w.Hash() != hash {
			t.Fatal("accepted corrupt footer or partly adopted")
		}
	}
}

func TestScorched1149Historical87Pins(t *testing.T) {
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, 0x3fae55f3e842e5ba}, {rtfBytes, 0xead9319c3c62159b}} {
		old := strippedScorchedPin(tc.form)
		// pinBytes and rtfBytes both close on entityIDFloorLen=10 (see their own
		// var declarations); the widen chain up to widenedAttackNoticePin stops
		// at form93 to keep areaHeaderDigest1164's frozen digest untouched, so
		// form94's own outermost floor wrap is applied here explicitly.
		if fnv1a(old) != tc.hash || !bytes.Equal(widenedEntityIDFloorPin(widenedScorchedPin(old), 10), tc.form) {
			t.Fatal("historical87 changed outside tag/footer")
		}
	}
}
