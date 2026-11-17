package textinput

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// A rune's byte is the code page's own, asked twice: the first answer is
// made by the encoder and the second is the one held.
func TestCodePageByteHeldAnswerIsTheEncoders(t *testing.T) {
	for pass := 0; pass < 2; pass++ {
		for r := rune(0); r < 0x600; r++ {
			want, wantOK := byte(0), false
			if r >= 0x20 && r != 0x7f {
				if b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(string(r))); err == nil && len(b) == 1 && b[0] >= 0x20 && b[0] != 0x7f {
					want, wantOK = b[0], true
				}
			}
			if got, ok := codePageByte(r); got != want || ok != wantOK {
				t.Fatalf("pass %d rune %U: got %#x %v, want %#x %v", pass, r, got, ok, want, wantOK)
			}
		}
	}
}
