// Package textinput converts the one-rune input accepted by character
// generation into the byte alphabet used by an install's character field.
package textinput

import "golang.org/x/text/encoding/charmap"

// EncodeRune encodes r as one Windows-1251 byte, the byte a typed character
// stores in an install's character field, for every selector. Control bytes and
// runes that the code page cannot represent are refused.
//
// Selector 1 applies the original's input conversion (TEXT-108): a typed byte
// 0xC0..0xEF is stored 0x40 lower and 0xF0..0xFF 0x10 lower. The draw then runs
// render/text.Convert on the stored byte, so a typed byte t reaches record t-0x30
// over 0xC0..0xEF (records 144..191) and t-0x20 over 0xF0..0xFF (records
// 208..223), the records that hold the typed letter in the 224-record fonts
// (TEXT-109). The two shifts do not compose to one constant and are not meant to.
// Selector 0 stores the typed byte and draws record t-0x20 (TEXT-110).
//
// A code point whose own Windows-1251 byte lies in 0x80..0xAF is refused at
// selector 1. The original stores such a byte unchanged and draws another
// letter for it; here the stored byte is Convert's shifted destination for a
// letter in 0xC0..0xEF, so DecodeByte would read it back as that other letter.
// 47 code points are refused (0x98 is unassigned), Ё among them; DIV-2330
// records the difference.
func EncodeRune(r rune, selector int) (byte, bool) {
	if r < 0x20 || r == 0x7f {
		return 0, false
	}
	b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(string(r)))
	if err != nil || len(b) != 1 || b[0] < 0x20 || b[0] == 0x7f {
		return 0, false
	}
	v := b[0]
	if selector == 1 {
		switch {
		case v >= 0xc0 && v <= 0xef:
			v -= 0x40
		case v >= 0xf0:
			v -= 0x10
		case v >= 0x80 && v <= 0xaf:
			return 0, false
		}
	}
	return v, true
}

// DecodeByte is EncodeRune's own inverse: given a byte this install actually
// stored under selector — a SAVE label byte or an installed character-field
// byte — it answers the rune EncodeRune produced that byte from.
//
// IT REFUSES ONLY A GENUINELY UNINTERPRETABLE BYTE: a control byte or 0x7f at
// either selector, and — at selector 0 only, which applies no shift — 0x98,
// unassigned in Windows-1251 itself. At selector 1 it never refuses a
// non-control byte: the shift routes that same 0x98 to 0xD8 ('Ш') before the
// codec ever sees it, so every byte in 0x20..0xFF except the two control
// exclusions decodes to some letter.
//
// THIS IS EncodeRune's EXACT INVERSE for every byte EncodeRune can now
// produce: EncodeRune refuses the 47-value collision this comment used to
// describe (see EncodeRune's own doc), so a stored byte in 0x80..0xAF this
// build wrote itself can only have arrived via the shift, and undoing the
// shift recovers the letter EncodeRune actually stored, exactly, not a
// guess. For a byte this build did NOT write — a genuinely original .sav, or
// data from before this fix — DecodeByte still cannot tell the difference
// between "arrived via the shift" and "some other writer stored this
// Windows-1251 byte unshifted", and always chooses the shift reading. That
// choice is pinned, not proven: see DIV-1339.
func DecodeByte(b byte, selector int) (rune, bool) {
	if b < 0x20 || b == 0x7f {
		return 0, false
	}
	v := b
	if selector == 1 {
		switch {
		case v >= 0x80 && v <= 0xaf:
			v += 0x40
		case v >= 0xe0 && v <= 0xef:
			v += 0x10
		}
	}
	decoded, err := charmap.Windows1251.NewDecoder().Bytes([]byte{v})
	if err != nil {
		return 0, false
	}
	runes := []rune(string(decoded))
	if len(runes) != 1 || runes[0] == '�' {
		return 0, false
	}
	return runes[0], true
}
