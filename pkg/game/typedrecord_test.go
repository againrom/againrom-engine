package game

import (
	"testing"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// recordMarkedFont: record k carries Advance k.
func recordMarkedFont(selector int) *text.Font {
	f := &text.Font{Selector: selector}
	for k := 0; k < 224; k++ {
		f.Glyphs = append(f.Glyphs, text.Glyph{Width: 1, Height: 1, Advance: k})
	}
	return f
}

func typedRune(t *testing.T, b byte) rune {
	t.Helper()
	d, err := charmap.Windows1251.NewDecoder().Bytes([]byte{b})
	if err != nil {
		t.Fatalf("byte %#02x: %v", b, err)
	}
	for _, r := range string(d) {
		return r
	}
	t.Fatalf("byte %#02x decodes to nothing", b)
	return 0
}

// TEXT-108: typed record is byte-0x30 over 0xC0..0xEF, byte-0x20 over 0xF0..0xFF.
func TestTypedByteReachesTheRecordTheRussianRuleStates(t *testing.T) {
	font := recordMarkedFont(1)
	for b := 0xc0; b <= 0xff; b++ {
		wantStored, wantRecord := b-0x10, b-0x20
		if b <= 0xef {
			wantStored, wantRecord = b-0x40, b-0x30
		}
		stored, ok := textinput.EncodeRune(typedRune(t, byte(b)), 1)
		if !ok || int(stored) != wantStored {
			t.Fatalf("typed %#02x: stored %#02x ok=%v, want %#02x", b, stored, ok, wantStored)
		}
		if got := font.GlyphFor(stored).Advance; got != wantRecord {
			t.Fatalf("typed %#02x: reaches record %d, want %d", b, got, wantRecord)
		}
		if wantRecord < 144 || wantRecord > 223 {
			t.Fatalf("typed %#02x: rule record %d outside the Cyrillic block 144..223", b, wantRecord)
		}
	}
}

// TEXT-110.
func TestTypedByteReachesTheRecordTheEnglishRuleStates(t *testing.T) {
	font := recordMarkedFont(0)
	for b := 0xc0; b <= 0xff; b++ {
		stored, ok := textinput.EncodeRune(typedRune(t, byte(b)), 0)
		if !ok || int(stored) != b {
			t.Fatalf("typed %#02x: stored %#02x ok=%v", b, stored, ok)
		}
		if got := font.GlyphFor(stored).Advance; got != b-0x20 {
			t.Fatalf("typed %#02x: reaches record %d, want %d", b, got, b-0x20)
		}
	}
}

// TEXT-108: a shipped byte passes the display conversion alone.
func TestShippedStringByteReachesTheDisplayConversionRecord(t *testing.T) {
	font := recordMarkedFont(1)
	for b := 0x20; b <= 0xff; b++ {
		want := b - 0x20
		switch {
		case b >= 0x80 && b <= 0xaf:
			want = b + 0x10
		case b >= 0xe0 && b <= 0xef:
			want = b - 0x10
		}
		if got := font.GlyphFor(byte(b)).Advance; got != want {
			t.Fatalf("shipped byte %#02x: record %d, want %d", b, got, want)
		}
	}
}

// TEXT-COLL-025; the refusal of 0x80..0xAF is DIV-2330.
func TestTypedByteBelowTheCyrillicBlockOnTheRussianSelector(t *testing.T) {
	for b := 0x80; b <= 0xbf; b++ {
		r := typedRune(t, byte(b))
		if r == 0xfffd {
			continue
		}
		stored, ok := textinput.EncodeRune(r, 1)
		if b >= 0xb0 {
			if !ok || int(stored) != b {
				t.Fatalf("typed %#02x: stored %#02x ok=%v, want unchanged", b, stored, ok)
			}
		} else if ok {
			t.Fatalf("typed %#02x: stored %#02x, want refusal", b, stored)
		}
	}
}
