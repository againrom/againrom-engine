package res

import "golang.org/x/text/encoding/charmap"

func WindowsCyrillicToDOS(raw []byte) []byte {
	out := make([]byte, len(raw))
	for i, b := range raw {
		c, ok := charmap.CodePage866.EncodeRune(charmap.Windows1251.DecodeByte(b))
		if !ok {
			c = b
		}
		out[i] = c
	}
	return out
}

// DOSCyrillicToWindows is WindowsCyrillicToDOS's inverse: each DOS Cyrillic
// byte becomes the Windows Cyrillic byte of the same letter, and a byte with
// no such letter is kept.
func DOSCyrillicToWindows(raw []byte) []byte {
	out := make([]byte, len(raw))
	for i, b := range raw {
		c, ok := charmap.Windows1251.EncodeRune(charmap.CodePage866.DecodeByte(b))
		if !ok {
			c = b
		}
		out[i] = c
	}
	return out
}
