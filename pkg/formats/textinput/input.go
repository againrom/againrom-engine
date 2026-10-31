// Package textinput is the install text codec: it converts a typed rune or a
// UTF-8 text into the byte alphabet an install's fonts draw and its character
// field and save label store.
package textinput

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/locale"
)

// EncodeRune encodes r as one Windows-1251 byte, the byte a typed character
// stores in an install's character field, for every selector. Control bytes and
// runes that the code page cannot represent are refused.
//
// Selector 1 applies the original's input conversion (TEXT-108): a typed byte
// 0xC0..0xEF is stored 0x40 lower and 0xF0..0xFF 0x10 lower. The draw then runs
// render/text.Convert on the stored byte, so a typed byte t reaches record t-0x30
// over 0xC0..0xEF and t-0x20 over 0xF0..0xFF, the records that hold the typed
// letter in the 224-record fonts (TEXT-109). Selector 0 stores the typed byte and draws record t-0x20 (TEXT-110).
//
// A code point whose own Windows-1251 byte lies in 0x80..0xAF is refused at
// selector 1: the original stores it unchanged and draws another letter, while
// here that byte is the shifted form of a letter in 0xC0..0xEF. 47 code points
// are refused (0x98 is unassigned), Ё among them; DIV-2330.
func EncodeRune(r rune, selector int) (byte, bool) {
	if r < 0x20 || r == 0x7f {
		return 0, false
	}
	b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(string(r)))
	if err != nil || len(b) != 1 || b[0] < 0x20 || b[0] == 0x7f {
		return 0, false
	}
	v := b[0]
	if locale.FontRemaps(selector) {
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

// DecodeByte is EncodeRune's exact inverse: given a byte this install stored
// under selector (a SAVE label or character-field byte), it answers the rune
// EncodeRune made it from.
//
// It refuses only a control byte or 0x7f at either selector and, at selector 0
// (no shift), 0x98, which Windows-1251 leaves unassigned. At selector 1 the
// shift routes 0x98 to 0xD8, so every other byte in 0x20..0xFF decodes.
//
// A byte this build did not write (an original .sav) cannot be told from an
// unshifted Windows-1251 byte; it always takes the shift reading (DIV-1339).
func DecodeByte(b byte, selector int) (rune, bool) {
	if b < 0x20 || b == 0x7f {
		return 0, false
	}
	v := b
	if locale.FontRemaps(selector) {
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

// Unrepresentable is what EncodeText does with a rune EncodeRune refuses.
type Unrepresentable int

const (
	// Refuse refuses the whole text: a save label, which is stored.
	Refuse Unrepresentable = iota
	// Drop leaves the rune out: install and engine words, which are drawn.
	Drop
	// Substitute draws '?' for the rune: the map editor's host text, so a
	// path or message keeps the place of every character.
	Substitute
)

// RefusedError names the first rune a Refuse encode could not store.
type RefusedError struct{ Rune rune }

func (e RefusedError) Error() string { return fmt.Sprintf("textinput: %U has no byte", e.Rune) }

// EncodeText is the install text codec: s rune by rune through EncodeRune
// under selector, a refused rune handled by policy. Only Refuse returns an
// error, a RefusedError.
func EncodeText(s string, selector int, policy Unrepresentable) (string, error) {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		c, ok := EncodeRune(r, selector)
		switch {
		case ok:
			b = append(b, c)
		case policy == Refuse:
			return "", RefusedError{r}
		case policy == Substitute:
			b = append(b, '?')
		}
	}
	return string(b), nil
}
