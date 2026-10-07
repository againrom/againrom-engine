package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// AC-1: a fresh character has twelve slots, numbered 1 to 12, every one
// empty; filling one and reading it back answers what was put in, and every
// other slot still answers empty.
func TestAFreshEquipmentHasTwelveNumberedEmptySlots(t *testing.T) {
	var e data.Equipment
	for n := 1; n <= data.EquipSlots; n++ {
		code, ok := e.Code(n)
		if !ok {
			t.Fatalf("Code(%d): refused, want a valid slot", n)
		}
		if code != 0 {
			t.Errorf("Code(%d) = 0x%04x, want 0 (empty) on a fresh character", n, uint16(code))
		}
		occ, ok := e.Occupied(n)
		if !ok || occ {
			t.Errorf("Occupied(%d) = %v, %v; want false, true", n, occ, ok)
		}
	}
	if data.EquipSlots != 12 {
		t.Fatalf("EquipSlots = %d, want 12", data.EquipSlots)
	}
}

// 0110 AC-4: a code is written into one slot and every slot is read back —
// that slot answers the WHOLE CODE that was written, fields A, B and C
// included and not only the field D that drives Occupied and HeroBodyFor —
// and every other slot still answers empty.
func TestFillingOneSlotAnswersTheWholeCodeThatWasWritten(t *testing.T) {
	var e data.Equipment
	const filled = 5
	// A=4, B=2, C=3, D=7 — all four fields nonzero and distinct, so a slot
	// that only carried field D through would already be caught here.
	const want data.ItemCode = 4<<12 | 2<<8 | 3<<5 | 7
	if !e.SetCode(filled, want) {
		t.Fatalf("SetCode(%d, 0x%04x): refused, want accepted", filled, uint16(want))
	}
	for n := 1; n <= data.EquipSlots; n++ {
		code, ok := e.Code(n)
		if !ok {
			t.Fatalf("Code(%d): refused, want a valid slot", n)
		}
		wantN := data.ItemCode(0)
		if n == filled {
			wantN = want
		}
		if code != wantN {
			t.Errorf("Code(%d) = 0x%04x, want 0x%04x", n, uint16(code), uint16(wantN))
		}
		occ, ok := e.Occupied(n)
		if !ok {
			t.Fatalf("Occupied(%d): refused, want a valid slot", n)
		}
		if occ != (n == filled) {
			t.Errorf("Occupied(%d) = %v, want %v", n, occ, n == filled)
		}
	}
}

func TestASlotOutsideOneToTwelveIsRefusedByAllThree(t *testing.T) {
	var e data.Equipment
	for _, n := range []int{0, -1, 13, 100} {
		if code, ok := e.Code(n); ok || code != 0 {
			t.Errorf("Code(%d) = 0x%04x, %v; want 0, false", n, uint16(code), ok)
		}
		if e.SetCode(n, 9) {
			t.Errorf("SetCode(%d, 9): accepted, want refused", n)
		}
		if occ, ok := e.Occupied(n); ok || occ {
			t.Errorf("Occupied(%d) = %v, %v; want false, false", n, occ, ok)
		}
	}
	// SetCode's refusal must leave every real slot untouched.
	for n := 1; n <= data.EquipSlots; n++ {
		if code, _ := e.Code(n); code != 0 {
			t.Errorf("a refused SetCode wrote into slot %d: 0x%04x", n, uint16(code))
		}
	}
}

func TestEquipSlotForAnswersFieldBAlone(t *testing.T) {
	const a, c, d = 5, 3, 17 // nonzero A, C, D; irrelevant to the answer.
	code := func(b int) data.ItemCode {
		return data.ItemCode(a<<12 | b<<8 | c<<5 | d)
	}
	for b := 0; b <= 15; b++ {
		n, ok := data.EquipSlotFor(code(b))
		wantOK := b >= 1 && b <= 12
		if ok != wantOK {
			t.Errorf("EquipSlotFor(B=%d) ok = %v, want %v", b, ok, wantOK)
		}
		if n != b {
			t.Errorf("EquipSlotFor(B=%d) = %d, want %d (c.B() itself, unclamped)", b, n, b)
		}
	}
	// The two non-slots AC-1 names explicitly, checked by name rather than
	// only by number.
	if _, ok := data.EquipSlotFor(code(0)); ok {
		t.Errorf("EquipSlotFor(B=0): accepted, want no slot")
	}
	if _, ok := data.EquipSlotFor(code(int(data.ItemClassCarried))); ok {
		t.Errorf("EquipSlotFor(B=ItemClassCarried): accepted, want no slot")
	}
	if n, ok := data.EquipSlotFor(code(1)); !ok || n != 1 {
		t.Errorf("EquipSlotFor(B=1) = %d, %v; want 1, true", n, ok)
	}
}

// fixtureList is a small, entirely invented body list — never the shipped
// one (SC-3) — used to exercise the derivation's three arms. Row 1 (the
// bare-handed row) is entry 0.
var fixtureList = data.BodyList{"bare", "sword", "axe", "", "bow"}

// equipmentWithCode builds a fresh Equipment carrying code in slot 1 — the
// only slot HeroBodyFor reads.
func equipmentWithCode(code data.ItemCode) data.Equipment {
	var e data.Equipment
	e.SetCode(1, code)
	return e
}

// AC-3: a first slot occupied with a code whose field D is inside the list
// derives the entry one before that row, and reports a name produced. The
// last two cases also set A, B and C to show HeroBodyFor reaches past them
// to field D alone — the derived name must not move.
func TestAnOccupiedFirstSlotDerivesTheEntryOneBeforeItsRow(t *testing.T) {
	for _, tc := range []struct {
		code data.ItemCode
		want data.HeroBody
	}{
		{1, "bare"},
		{2, "sword"},
		{3, "axe"},
		{5, "bow"},
		{1<<12 | 1<<8 | 1<<5 | 1, "bare"},
		{9<<12 | 6<<8 | 7<<5 | 5, "bow"},
	} {
		got, ok := data.HeroBodyFor(fixtureList, equipmentWithCode(tc.code))
		if !ok {
			t.Errorf("code 0x%04x: refused, want %q produced", uint16(tc.code), tc.want)
			continue
		}
		if got != tc.want {
			t.Errorf("code 0x%04x: derived %q, want %q", uint16(tc.code), got, tc.want)
		}
	}
}

// AC-4: an empty first slot derives the list's first entry, the bare-handed
// name, and reports a name produced.
func TestAnEmptyFirstSlotDerivesTheBareHandedName(t *testing.T) {
	var e data.Equipment // slot 1 left at its zero-value empty
	got, ok := data.HeroBodyFor(fixtureList, e)
	if !ok {
		t.Fatal("an empty first slot: refused, want the bare-handed name produced")
	}
	if got != fixtureList[0] {
		t.Errorf("derived %q, want the list's first entry %q", got, fixtureList[0])
	}
	if got != "bare" {
		t.Errorf("derived %q, want %q", got, "bare")
	}
}

// AC-5: a row at or past the list's end, and one naming an empty entry, each
// derive no name and report that none was produced.
//
// A ROW BEFORE THE LIST'S START IS NOT A CASE HERE ANY MORE: field D is five
// bits and the empty-slot arm already claims D == 0, so a real code's row
// — after HeroBodyFor's own empty-to-1 substitution — is always in [1,
// 31]. The function still guards i < 0 for its own totality; nothing in this
// package's public surface can build a code that reaches that guard.
func TestARowNamingNoEntryProducesNoName(t *testing.T) {
	for _, tc := range []struct {
		name string
		code data.ItemCode
	}{
		{"one past the list's end", data.ItemCode(len(fixtureList)) + 1},
		{"far past the list's end, at field D's own maximum", 31},
		{"naming the list's own empty entry", 4}, // fixtureList[3] == ""
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := data.HeroBodyFor(fixtureList, equipmentWithCode(tc.code))
			if ok {
				t.Errorf("code 0x%04x (%s): produced %q, want refused", uint16(tc.code), tc.name, got)
			}
			if got != "" {
				t.Errorf("code 0x%04x (%s): refusal carried %q, want the empty name", uint16(tc.code), tc.name, got)
			}
		})
	}
}

// The two live arms of HeroBodyFor meet at row 1: an empty slot and an
// occupied slot naming row 1 must answer the identical name, because the
// bare-handed entry is both the list's first line and the entry named by the
// bare hand's own row.
func TestTheEmptyArmAndRowOneAgreeOnTheBareHandedName(t *testing.T) {
	empty, ok1 := data.HeroBodyFor(fixtureList, data.Equipment{})
	rowOne, ok2 := data.HeroBodyFor(fixtureList, equipmentWithCode(1))
	if !ok1 || !ok2 {
		t.Fatalf("both arms must produce a name: empty ok=%v, row-1 ok=%v", ok1, ok2)
	}
	if empty != rowOne {
		t.Errorf("empty slot derived %q, row 1 derived %q; want them equal", empty, rowOne)
	}
}

// ParseBodyList: every line is an entry including an empty one, split over
// CRLF or bare LF, and a trailing newline adds no final entry. None of the
// payloads below is the shipped list (SC-3) — every byte is written out
// here.
func TestParseBodyListSplitsEveryLineIncludingAnEmptyOne(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		want data.BodyList
	}{
		{"LF, no trailing newline", []byte("bare\nsword\naxe"),
			data.BodyList{"bare", "sword", "axe"}},
		{"LF, trailing newline adds nothing", []byte("bare\nsword\naxe\n"),
			data.BodyList{"bare", "sword", "axe"}},
		{"CRLF, trailing newline adds nothing", []byte("bare\r\nsword\r\naxe\r\n"),
			data.BodyList{"bare", "sword", "axe"}},
		{"an interior empty line is kept", []byte("bare\n\naxe\n"),
			data.BodyList{"bare", "", "axe"}},
		{"an interior empty CRLF line is kept", []byte("bare\r\n\r\naxe\r\n"),
			data.BodyList{"bare", "", "axe"}},
		{"a lone empty line", []byte("\n"), data.BodyList{""}},
		{"empty payload", []byte{}, data.BodyList{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := data.ParseBodyList(tc.data)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseBodyList(%q) = %#v (%d entries), want %#v (%d entries)",
					tc.data, got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("entry %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestFigureHeldLastMatchesTheSixLiteralBodyNames — 0151-layers-and-names
// T7, HERO-FIGURE-060: slot 1 paints last when its resolved body name is one
// of the six two-handed or ranged bodies; slot 2 paints last otherwise,
// including when slot 1 is empty, resolves to a one-handed body, or
// resolves to no name at all (a row past the list's end).
func TestFigureHeldLastMatchesTheSixLiteralBodyNames(t *testing.T) {
	// Row i (1-based, HeroBodyFor's own indexing) resolves to list[i-1].
	list := data.BodyList{
		data.BodyUnarmed,     // row 1 — also the empty-slot arm's own answer
		data.BodySwordsman,   // row 2 — one-handed
		data.BodySwordsman2H, // row 3
		data.BodyAxeman2H,    // row 4
		data.BodyArcher,      // row 5
		data.BodyBowman,      // row 6 — HERO-APPEAR-052's own dead arm
		data.BodyCrossbowman, // row 7
		data.BodyMageStaff,   // row 8
	}

	for _, tc := range []struct {
		name string
		row  int // 0 leaves slot 1 empty
		want int
	}{
		{"empty slot 1", 0, 2},
		{"a one-handed body", 2, 2},
		{"swordsman2h", 3, 1},
		{"axeman2h", 4, 1},
		{"archer", 5, 1},
		{"bowman, though no shipped row ever produces it", 6, 1},
		{"xbowman", 7, 1},
		{"mage_st", 8, 1},
		{"a row past the list's end resolves no name", 20, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var eq data.Equipment
			eq.SetCode(1, data.ItemCode(tc.row))
			if got := data.FigureHeldLast(list, eq); got != tc.want {
				t.Errorf("FigureHeldLast(row %d) = %d, want %d", tc.row, got, tc.want)
			}
		})
	}

	t.Run("a nil list resolves no name", func(t *testing.T) {
		var eq data.Equipment
		eq.SetCode(1, data.ItemCode(3)) // would be swordsman2h on the list above
		if got := data.FigureHeldLast(nil, eq); got != 2 {
			t.Errorf("FigureHeldLast(nil list) = %d, want 2 — nothing to resolve a name from", got)
		}
	})
}
