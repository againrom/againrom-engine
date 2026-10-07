package textinput

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestEncodeRune(t *testing.T) {
	for _, tc := range []struct {
		r        rune
		selector int
		want     byte
		ok       bool
	}{
		{'A', 0, 'A', true}, {'A', 1, 'A', true},
		{'А', 0, 0xc0, true}, {'А', 1, 0x80, true},
		{'я', 0, 0xff, true}, {'я', 1, 0xef, true},
		{'\n', 0, 0, false}, {'😀', 0, 0, false},
	} {
		got, ok := EncodeRune(tc.r, tc.selector)
		if got != tc.want || ok != tc.ok {
			t.Errorf("EncodeRune(%q, %d) = %#x, %v; want %#x, %v", tc.r, tc.selector, got, ok, tc.want, tc.ok)
		}
	}
}

// TestEncodeRuneCoversEveryRepresentableStoredByte checks every Windows-1251
// byte round-trips at selector 0, and at selector 1 either lands where
// render/text.Convert's own shift would put it or \u2014 for the one range where
// that shift would silently collide with an already-shifted letter,
// 0x80..0xAF \u2014 is refused instead. See
// TestEncodeRuneRefusesTheSelectorOneCollisionSet for that range on its own.
func TestEncodeRuneCoversEveryRepresentableStoredByte(t *testing.T) {
	for n := 0x20; n <= 0xff; n++ {
		if n == 0x7f {
			continue
		}
		decoded, err := charmap.Windows1251.NewDecoder().Bytes([]byte{byte(n)})
		if err != nil {
			continue
		}
		runes := []rune(string(decoded))
		if len(runes) != 1 || runes[0] == '\ufffd' {
			continue
		}
		got, ok := EncodeRune(runes[0], 0)
		if !ok || got != byte(n) {
			t.Errorf("byte %#x decoded as %q re-encodes as %#x, %v", n, runes[0], got, ok)
		}
		selected, ok := EncodeRune(runes[0], 1)
		switch nb := byte(n); {
		case nb >= 0x80 && nb <= 0xaf:
			if ok {
				t.Errorf("selector byte %#x decoded as %q re-encodes as %#x, true; want refused, it collides with a shifted letter's own slot", n, runes[0], selected)
			}
		case nb >= 0xc0 && nb <= 0xef:
			want := nb - 0x40
			if !ok || selected != want {
				t.Errorf("selector byte %#x decoded as %q maps as %#x, %v; want %#x", n, runes[0], selected, ok, want)
			}
		case nb >= 0xf0:
			want := nb - 0x10
			if !ok || selected != want {
				t.Errorf("selector byte %#x decoded as %q maps as %#x, %v; want %#x", n, runes[0], selected, ok, want)
			}
		default:
			if !ok || selected != nb {
				t.Errorf("selector byte %#x decoded as %q maps as %#x, %v; want %#x", n, runes[0], selected, ok, nb)
			}
		}
	}
}

// TestDecodeByteInvertsEncodeRuneAtSelectorZero is asciiLabel's own reason for
// calling this function: a SAVE label byte this build wrote must read back as
// the rune it was typed from. At selector 0 EncodeRune never shifts, so the
// round trip holds over every representable rune with no exception — checked
// here over the full stored-byte range, not one hand-picked pair.
func TestDecodeByteInvertsEncodeRuneAtSelectorZero(t *testing.T) {
	for r := rune(0x20); r <= 0x2ff; r++ {
		b, ok := EncodeRune(r, 0)
		if !ok {
			continue
		}
		got, gotOK := DecodeByte(b, 0)
		if !gotOK || got != r {
			t.Errorf("DecodeByte(EncodeRune(%q, 0)=%#x, 0) = %q, %v; want %q, true", r, b, got, gotOK, r)
		}
	}
}

// TestDecodeByteInvertsEncodeRuneAtSelectorOneOverTheShippedAlphabet is the
// same round trip at selector 1, restricted to ASCII plus the standard
// Cyrillic letter block (U+0410..U+044F) plus 'ё' (U+0451) — the domain a
// typed save name actually occupies. Ё (U+0401, uppercase) is deliberately
// excluded: its own unshifted Windows-1251 byte, 0xA8, is one of the 47
// EncodeRune now refuses at selector 1 rather than silently storing over a
// shifted letter's slot (TestEncodeRuneRefusesTheSelectorOneCollisionSet).
// Lowercase ё (0xB8) sits outside 0x80..0xAF and is not affected, which is
// why it round-trips here and Ё does not.
func TestDecodeByteInvertsEncodeRuneAtSelectorOneOverTheShippedAlphabet(t *testing.T) {
	domain := make([]rune, 0, 160)
	for r := rune(0x20); r <= 0x7e; r++ {
		domain = append(domain, r)
	}
	for r := rune(0x410); r <= 0x44f; r++ {
		domain = append(domain, r)
	}
	domain = append(domain, 'ё')
	for _, r := range domain {
		b, ok := EncodeRune(r, 1)
		if !ok {
			t.Fatalf("EncodeRune(%q, 1) refused a shipped-alphabet rune", r)
		}
		got, gotOK := DecodeByte(b, 1)
		if !gotOK || got != r {
			t.Errorf("DecodeByte(EncodeRune(%q, 1)=%#x, 1) = %q, %v; want %q, true", r, b, got, gotOK, r)
		}
	}
}

// TestEncodeRuneRefusesTheSelectorOneCollisionSet enumerates the full
// collision set EncodeRune's own doc names: every Windows-1251 code point in
// 0x80..0xAF except the one Windows-1251 leaves unassigned (0x98) — 47 code
// points, among them Cyrillic Ё (0xA8) — is refused at selector 1, because
// that byte range is exactly where render/text.Convert's own shift lands a
// letter from 0xC0..0xEF (EncodeRune's own first selector-1 shift, above).
// Storing one of these 47 unshifted would silently take over a real letter's
// slot; every one is still encodable at selector 0, which applies no shift
// and has no such collision.
func TestEncodeRuneRefusesTheSelectorOneCollisionSet(t *testing.T) {
	count := 0
	for n := 0x80; n <= 0xaf; n++ {
		if n == 0x98 {
			continue // unassigned in Windows-1251 itself, not this collision
		}
		decoded, err := charmap.Windows1251.NewDecoder().Bytes([]byte{byte(n)})
		if err != nil {
			t.Fatalf("Windows-1251 byte %#x has no rune to test", n)
		}
		runes := []rune(string(decoded))
		if len(runes) != 1 {
			t.Fatalf("Windows-1251 byte %#x decoded to %d runes", n, len(runes))
		}
		count++
		if b, ok := EncodeRune(runes[0], 1); ok {
			t.Errorf("EncodeRune(%q, 1) = %#x, true; want refused, it collides with the letter at %#x", runes[0], b, n+0x40)
		}
		if _, ok := EncodeRune(runes[0], 0); !ok {
			t.Errorf("EncodeRune(%q, 0) refused; selector 0 applies no shift and has no collision here", runes[0])
		}
	}
	if count != 47 {
		t.Fatalf("expected 47 colliding Windows-1251 code points in 0x80..0xaf, counted %d", count)
	}
}

// TestDecodeByteStillPrefersTheShiftReadingForAForeignByte pins DecodeByte's
// remaining, deliberate choice for a byte this build's own EncodeRune can no
// longer produce at all: a genuinely original .sav, or data from before this
// fix, could still carry a Windows-1251 byte stored unshifted in 0x80..0xAF.
// DecodeByte cannot tell that case apart from EncodeRune's own shifted
// letter, and always reads it as the shifted letter — here, '§' (U+00A7,
// Windows-1251 0xA7, unshifted) reads back as 'з' (U+0437), the letter whose
// own selector-1 encoding shifts onto that same byte. This is no longer a
// round-trip defect (EncodeRune never emits 0xA7 at selector 1 for any rune
// any more), only DIV-1339's already-pinned choice for a byte this build did
// not itself write.
func TestDecodeByteStillPrefersTheShiftReadingForAForeignByte(t *testing.T) {
	if _, ok := EncodeRune('§', 1); ok {
		t.Fatal("EncodeRune('§', 1) should refuse, it is now part of the collision set")
	}
	got, ok := DecodeByte(0xa7, 1)
	if !ok || got != 'з' {
		t.Fatalf("DecodeByte(0xa7, 1) = %q, %v; want 'з', true", got, ok)
	}
}

func TestDecodeByteRefusesWhatEncodeRuneNeverProduces(t *testing.T) {
	for _, c := range []struct {
		b        byte
		selector int
	}{
		{0x00, 0}, {0x1f, 0}, {0x7f, 0}, {0x7f, 1},
		// 0x98 is unassigned in Windows-1251 itself, so it is undefined at
		// selector 0, which applies no shift. At selector 1 it is not: the
		// shift moves it to 0xD8 ('Ш') first, so it decodes there instead —
		// covered by TestDecodeByteInvertsEncodeRuneAtSelectorOne... above.
		{0x98, 0},
	} {
		if r, ok := DecodeByte(c.b, c.selector); ok {
			t.Errorf("DecodeByte(%#x, %d) = %q, true; want false", c.b, c.selector, r)
		}
	}
}
