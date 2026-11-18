package textinput

import (
	"errors"
	"testing"
)

// Each policy answers the one rune the selector cannot store its own way, and
// the representable runes encode as EncodeRune encodes them one by one.
func TestEncodeTextStatesItsPolicy(t *testing.T) {
	const s = "Ёж ok" // Ё has no byte at selector 1
	var want []byte
	for _, r := range "ж ok" {
		c, _ := EncodeRune(r, 1)
		want = append(want, c)
	}
	if got, err := EncodeText(s, 1, Drop); err != nil || got != string(want) {
		t.Errorf("Drop = % x %v, want % x", got, err, want)
	}
	if got, err := EncodeText(s, 1, Substitute); err != nil || got != "?"+string(want) {
		t.Errorf("Substitute = % x %v", got, err)
	}
	got, err := EncodeText(s, 1, Refuse)
	var refused RefusedError
	if got != "" || !errors.As(err, &refused) || refused.Rune != 'Ё' {
		t.Errorf("Refuse = %q %v", got, err)
	}
	if got, err := EncodeText(s, 0, Refuse); err != nil || len(got) != len([]rune(s)) {
		t.Errorf("selector 0 stores Ё: % x %v", got, err)
	}
}

// No rune of the Windows-1251 range 0x80..0xBF reaches a byte in that range
// under Substitute on the remapping selector: those bytes draw other letters.
func TestSubstituteNeverDrawsAnotherLetter(t *testing.T) {
	for b := 0x80; b <= 0xbf; b++ {
		r, ok := DecodeByte(byte(b), 0)
		if !ok {
			continue
		}
		got, err := EncodeText(string(r), 1, Substitute)
		if err != nil || len(got) != 1 || got[0] != '?' {
			t.Errorf("%U (byte %#x) encodes to % x, want a placeholder", r, b, got)
		}
		if got, _ := EncodeText(string(r), 0, Substitute); len(got) != 1 || got[0] != byte(b) {
			t.Errorf("%U at selector 0 encodes to % x, want %#x", r, got, b)
		}
	}
}
