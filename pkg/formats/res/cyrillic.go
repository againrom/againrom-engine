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
