package data_test

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/data"
)

// TestFieldsAtEveryBoundaryValue — AC-1's Given: codes putting every field
// at a boundary, all bits clear, and all bits set.
func TestFieldsAtEveryBoundaryValue(t *testing.T) {
	for _, tc := range []struct {
		name       string
		code       data.ItemCode
		a, b, c, d int
	}{
		{"all bits clear", 0x0000, 0, 0, 0, 0},
		{"all bits set", 0xffff, 15, 15, 7, 31},
		{"only A at its own max", 0xf000, 15, 0, 0, 0},
		{"only B at its own max", 0x0f00, 0, 15, 0, 0},
		{"only C at its own max", 0x00e0, 0, 0, 7, 0},
		{"only D at its own max", 0x001f, 0, 0, 0, 31},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.code.A(); got != tc.a {
				t.Errorf("A() = %d, want %d", got, tc.a)
			}
			if got := tc.code.B(); got != tc.b {
				t.Errorf("B() = %d, want %d", got, tc.b)
			}
			if got := tc.code.C(); got != tc.c {
				t.Errorf("C() = %d, want %d", got, tc.c)
			}
			if got := tc.code.D(); got != tc.d {
				t.Errorf("D() = %d, want %d", got, tc.d)
			}
		})
	}
}

// TestEveryCodeInTheDomainAnswersItsFourFieldsAndRecomposes — AC-1's third
// Given, a walk of the whole sixteen-bit domain: every one of the 65536
// codes answers four fields, each within its own bit range, and the four
// recompose the code they came from. The expectation is computed from the
// raw loop integer by the same bit split the spec states, never through
// ItemCode's own methods, so a bug shared by A/B/C/D and this check would
// still have to be a bug in the SAME arithmetic written twice rather than
// read once.
func TestEveryCodeInTheDomainAnswersItsFourFieldsAndRecomposes(t *testing.T) {
	for code := 0; code <= 0xffff; code++ {
		c := data.ItemCode(code)
		a, b, cc, d := c.A(), c.B(), c.C(), c.D()

		if a < 0 || a > 15 {
			t.Fatalf("code 0x%04x: A() = %d, outside [0,15]", code, a)
		}
		if b < 0 || b > 15 {
			t.Fatalf("code 0x%04x: B() = %d, outside [0,15]", code, b)
		}
		if cc < 0 || cc > 7 {
			t.Fatalf("code 0x%04x: C() = %d, outside [0,7]", code, cc)
		}
		if d < 0 || d > 31 {
			t.Fatalf("code 0x%04x: D() = %d, outside [0,31]", code, d)
		}

		if want := (code >> 12) & 0xf; a != want {
			t.Fatalf("code 0x%04x: A() = %d, want %d", code, a, want)
		}
		if want := (code >> 8) & 0xf; b != want {
			t.Fatalf("code 0x%04x: B() = %d, want %d", code, b, want)
		}
		if want := (code >> 5) & 0x7; cc != want {
			t.Fatalf("code 0x%04x: C() = %d, want %d", code, cc, want)
		}
		if want := code & 0x1f; d != want {
			t.Fatalf("code 0x%04x: D() = %d, want %d", code, d, want)
		}

		if recomposed := a<<12 | b<<8 | cc<<5 | d; recomposed != code {
			t.Fatalf("code 0x%04x: fields recompose to 0x%04x, want the original", code, recomposed)
		}
	}
}

// TestNameSpellsBothFormsAtTheirBoundaryValues — AC-2's Given, exercised at
// each form's own boundary: B == ItemClassCarried spells the whole low byte
// as three digits at its own min and max, and B != ItemClassCarried spells
// C and D separately, including the two values immediately beside the
// carried class. Every want string is written out; none is built by calling
// Name or any of the field readers.
func TestNameSpellsBothFormsAtTheirBoundaryValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		code data.ItemCode
		want string
	}{
		// B != ItemClassCarried: C and D spelled separately.
		{"all bits clear", 0x0000, "0000000"},
		{"all bits set (B=15, not the carried class)", 0xffff, "1515731"},
		{"B one below the carried class", 0x0d00, "0013000"},
		{"B one above the carried class", 0x0f00, "0015000"},

		// B == ItemClassCarried: the whole low byte as three digits.
		{"carried class, low byte at its minimum", 0x0e00, "0014000"},
		{"carried class, low byte at its maximum", 0xfeff, "1514255"},
		{"carried class, A at its own max, low byte 0", 0xfe00, "1514000"},
		{"carried class, low byte from D alone", 0x0e1f, "0014031"},
		{"carried class, low byte from C alone", 0x0ee0, "0014224"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.code.Name(); got != tc.want {
				t.Errorf("Name() = %q, want %q (A=%d B=%d C=%d D=%d)",
					got, tc.want, tc.code.A(), tc.code.B(), tc.code.C(), tc.code.D())
			}
		})
	}
}

func TestNameProducesExactlySevenDigitsOverTheWholeDomain(t *testing.T) {
	for code := 0; code <= 0xffff; code++ {
		c := data.ItemCode(code)
		got := c.Name()

		if len(got) != 7 {
			t.Fatalf("code 0x%04x: Name() = %q, %d bytes, want 7", code, got, len(got))
		}
		for i := 0; i < len(got); i++ {
			if got[i] < '0' || got[i] > '9' {
				t.Fatalf("code 0x%04x: Name() = %q, byte %d is %q, want a decimal digit",
					code, got, i, got[i])
			}
		}

		a, b := (code>>12)&0xf, (code>>8)&0xf
		var want string
		if b == data.ItemClassCarried {
			want = fmt.Sprintf("%02d%02d%03d", a, b, code&0xff)
		} else {
			want = fmt.Sprintf("%02d%02d%1d%02d", a, b, (code>>5)&0x7, code&0x1f)
		}
		if got != want {
			t.Fatalf("code 0x%04x: Name() = %q, want %q", code, got, want)
		}
	}
}

func TestFigureDirForIsTotalOverTheFourCombinations(t *testing.T) {
	for _, tc := range []struct {
		mage, female bool
		want         data.FigureDir
	}{
		{false, false, data.FigureDirManFighter},
		{true, false, data.FigureDirManMage},
		{false, true, data.FigureDirWomanFighter},
		{true, true, data.FigureDirWomanMage},
	} {
		if got := data.FigureDirFor(tc.mage, tc.female); got != tc.want {
			t.Errorf("FigureDirFor(mage=%v, female=%v) = %q, want %q",
				tc.mage, tc.female, got, tc.want)
		}
	}
}

// TestFigureDirConstantsSpellTheArchiveDirectoriesExactly pins the four
// constants' underlying strings against the archive's own spelling
// (HERO-APPEAR-051), so a typo in one drifts loudly rather than only
// changing which of four opaque values a test compares against another.
func TestFigureDirConstantsSpellTheArchiveDirectoriesExactly(t *testing.T) {
	for _, tc := range []struct {
		dir  data.FigureDir
		want string
	}{
		{data.FigureDirManFighter, "mfighter"},
		{data.FigureDirManMage, "mmage"},
		{data.FigureDirWomanFighter, "ffighter"},
		{data.FigureDirWomanMage, "fmage"},
	} {
		if string(tc.dir) != tc.want {
			t.Errorf("%v = %q, want %q", tc.dir, string(tc.dir), tc.want)
		}
	}
}

func TestItemIconPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		code data.ItemCode
		want string
	}{
		{"all bits clear", 0x0000, "inventory/0000000.16a"},
		{"carried class", 0x0e00, "inventory/0014000.16a"},
		{"all bits set", 0xffff, "inventory/1515731.16a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := data.ItemIconPath(tc.code); got != tc.want {
				t.Errorf("ItemIconPath(0x%04x) = %q, want %q", uint16(tc.code), got, tc.want)
			}
		})
	}
}

func TestItemFigureLayerPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  data.FigureDir
		code data.ItemCode
		want string
	}{
		{"man fighter, all bits clear", data.FigureDirManFighter, 0x0000,
			"equipment/mfighter/primary/0000000.256"},
		{"woman mage, carried class", data.FigureDirWomanMage, 0x0e00,
			"equipment/fmage/primary/0014000.256"},
		{"man mage, all bits set", data.FigureDirManMage, 0xffff,
			"equipment/mmage/primary/1515731.256"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := data.ItemFigureLayerPath(tc.dir, tc.code); got != tc.want {
				t.Errorf("ItemFigureLayerPath(%q, 0x%04x) = %q, want %q",
					tc.dir, uint16(tc.code), got, tc.want)
			}
		})
	}
}

func TestItemFigureBasePath(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  data.FigureDir
		face int
		want string
	}{
		{"man fighter, the first face", data.FigureDirManFighter, 1, "equipment/mfighter/1.256"},
		{"man fighter, a later face", data.FigureDirManFighter, 7, "equipment/mfighter/7.256"},
		{"woman mage, the first face", data.FigureDirWomanMage, 1, "equipment/fmage/1.256"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := data.ItemFigureBasePath(tc.dir, tc.face); got != tc.want {
				t.Errorf("ItemFigureBasePath(%q, %d) = %q, want %q", tc.dir, tc.face, got, tc.want)
			}
		})
	}
}

// TestItemFigureSecondaryLayerPath — 0151-layers-and-names T2, HERO-APPEAR-050:
// the second sheet is the code's own name, under "secondary" rather than
// "primary", otherwise identical to ItemFigureLayerPath.
func TestItemFigureSecondaryLayerPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  data.FigureDir
		code data.ItemCode
		want string
	}{
		{"man fighter, all bits clear", data.FigureDirManFighter, 0x0000,
			"equipment/mfighter/secondary/0000000.256"},
		{"woman mage, carried class", data.FigureDirWomanMage, 0x0e00,
			"equipment/fmage/secondary/0014000.256"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := data.ItemFigureSecondaryLayerPath(tc.dir, tc.code); got != tc.want {
				t.Errorf("ItemFigureSecondaryLayerPath(%q, 0x%04x) = %q, want %q",
					tc.dir, uint16(tc.code), got, tc.want)
			}
		})
	}
}

// TestHasItemFigureSecondaryLayer — HERO-APPEAR-050's corpus set, exactly:
// slots 4, 8, 9 and 10 carry a second sheet, no other slot in 1..12 does.
func TestHasItemFigureSecondaryLayer(t *testing.T) {
	want := map[int]bool{4: true, 8: true, 9: true, 10: true}
	for n := 1; n <= data.EquipSlots; n++ {
		if got := data.HasItemFigureSecondaryLayer(n); got != want[n] {
			t.Errorf("HasItemFigureSecondaryLayer(%d) = %v, want %v", n, got, want[n])
		}
	}
}

// stepString spells a pass as slots, a trailing "s" for a second sheet and a
// "t" for a tag-only step; a slot with no painting step is left out.
func stepString(steps []data.FigureStep) string {
	var parts []string
	for _, st := range steps {
		if st.Kind == data.FigureStepNone {
			continue
		}
		p := fmt.Sprint(st.Slot)
		if st.Secondary {
			p += "s"
		}
		if st.Kind == data.FigureStepTag {
			p += "t"
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, " ")
}

// The non-mage pass is HERO-FIGURE-059's and its tail is HERO-FIGURE-060's: slot 1
// before slot 8, the second sheets of 4, 9 and 10 after it, then slot 1 or
// slot 2. Every slot keeps a step either way.
func TestFigureDrawStepsOfAFighter(t *testing.T) {
	twoHanded := data.BodyList{"unarmed", "swordsman2h"}
	oneHanded := data.BodyList{"unarmed", "swordsman2h", "swordsman"}
	for _, dir := range []data.FigureDir{data.FigureDirManFighter, data.FigureDirWomanFighter} {
		for _, tc := range []struct {
			name string
			l    data.BodyList
			row  int
			want string
		}{
			{"one-handed", oneHanded, 3, "12 11 7 4 5 9 10 1 8 4s 6 9s 10s 2"},
			{"two-handed", twoHanded, 2, "12 11 7 4 5 9 10 1 8 4s 6 9s 10s 1"},
			{"no weapon", twoHanded, 0, "12 11 7 4 5 9 10 1 8 4s 6 9s 10s 2"},
		} {
			var eq data.Equipment
			eq.SetCode(1, data.ItemCode(tc.row))
			steps := data.FigureDrawSteps(dir, tc.l, eq)
			if got := stepString(steps); got != tc.want {
				t.Errorf("%s %s steps = %s, want %s", dir, tc.name, got, tc.want)
			}
			order := data.FigureDrawOrder(dir, tc.l, eq)
			seen := map[int]bool{}
			for _, n := range order {
				if n < 1 || n > data.EquipSlots || seen[n] {
					t.Fatalf("%s order %v is not a permutation of 1..%d", dir, order, data.EquipSlots)
				}
				seen[n] = true
			}
			if len(order) != data.EquipSlots {
				t.Fatalf("%s order %v names %d slots", dir, order, len(order))
			}
		}
	}
}

// A mage's pass is the mage half of HERO-FIGURE-059 whatever it holds and
// whatever the body list says: slot 9 is tagged and not painted, and slots
// 11, 3 and 2 have no step.
func TestFigureDrawStepsOfAMage(t *testing.T) {
	const want = "8 12 10 10s 4 4s 7 5 9t 1 6 8s"
	for _, dir := range []data.FigureDir{data.FigureDirManMage, data.FigureDirWomanMage} {
		for _, l := range []data.BodyList{nil, {"unarmed", "swordsman2h"}, {"unarmed", "swordsman2h", "swordsman"}} {
			for row := 0; row < 4; row++ {
				var eq data.Equipment
				eq.SetCode(1, data.ItemCode(row))
				steps := data.FigureDrawSteps(dir, l, eq)
				if got := stepString(steps); got != want {
					t.Errorf("%s steps = %s, want %s", dir, got, want)
				}
				if got := len(data.FigureDrawOrder(dir, l, eq)); got != data.EquipSlots {
					t.Errorf("%s order names %d slots, want %d", dir, got, data.EquipSlots)
				}
			}
		}
	}
}
