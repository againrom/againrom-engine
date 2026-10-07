package alm

// The write half of the two type-0 string-field rules whose read half is
// decodeASCII and decodeCP1251. A field is a fixed-width NUL-terminated slot,
// so writing one is not "encode the text": it is building the whole field
// image — encoded bytes, then NUL fill — and it is allowed to fail. Failing is
// what lets a caller be promised that a string this package decoded from a
// field is written back as that field's own bytes or not written at all: the
// two directions are one rule, and it lives in one package.

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"
)

// encodeError is the type of this file's three rejection causes. It is a
// constant string type rather than a set of errors.New values so that the
// encoders add no import to a leaf whose import set is itself part of its tier
// contract; being comparable, each constant works with errors.Is.
type encodeError string

func (e encodeError) Error() string { return string(e) }

// The three causes a field encode is rejected for, kept distinct because they
// are three different failures and a caller (or a witness) that cannot tell
// them apart cannot tell a correct encoder from one that refuses everything.
// ErrFieldRoundTrip is ours rather than either codec's: Windows-1251 encodes
// U+0000 to a plain NUL byte without complaint and the reader stops at the
// first NUL, so an embedded NUL is caught by the round trip alone.
const (
	ErrFieldTooLong     encodeError = "encoded text leaves no room for the field's NUL terminator"
	ErrFieldUnencodable encodeError = "the field's codec cannot represent the string"
	ErrFieldRoundTrip   encodeError = "the field image does not decode back to the string given"
)

// EncodeName builds the 64-byte image of the type-0 name field (+0x30) for s:
// the ASCII bytes of s followed by NUL fill to the field's width, in a fresh
// buffer the package does not retain. It rejects a string that does not fit
// the field with room for its terminator, one holding a rune outside ASCII
// (any byte >= 0x80, an invalid UTF-8 byte included), and one whose image does
// not decode back through this package's own reader to s — an embedded NUL,
// for one. Nothing is truncated and nothing is substituted: the image is
// byte-exact or there is an error and no image.
func EncodeName(s string) ([]byte, error) {
	return encodeField("name", metaNameLen, s, encodeASCII, decodeASCII)
}

// EncodeDescription builds the 64-byte image of the type-0 description field
// (+0x78) for s, on the same terms as EncodeName but through Windows-1251:
// the code page's bytes for s, then NUL fill. A rune the code page has no byte
// for is rejected with the codec's own error, so the reject set is the code
// page's, not an approximation of it.
func EncodeDescription(s string) ([]byte, error) {
	return encodeField("description", metaDescLen, s, encodeCP1251, decodeCP1251)
}

// encodeField is the field rule both encoders are: encode, check the text
// leaves room for the terminator, lay it into a NUL-filled image of the
// field's own width, and require that image to read back through the field's
// own decoder as the string given. The last step makes byte-identical-or-
// rejected a property of this package rather than of a codec's default error
// policy, since a codec may encode a string it cannot get back.
func encodeField(field string, width int, s string,
	encode func(string) ([]byte, error), decode func([]byte) string) ([]byte, error) {

	text, err := encode(s)
	if err != nil {
		return nil, fmt.Errorf("alm: %s field: %w", field, err)
	}
	if len(text) > width-1 {
		return nil, fmt.Errorf("alm: %s field: %d encoded bytes do not fit %d with a terminator: %w",
			field, len(text), width, ErrFieldTooLong)
	}
	img := make([]byte, width)
	copy(img, text)
	if back := decode(img); back != s {
		return nil, fmt.Errorf("alm: %s field: image reads back as %q, not %q: %w",
			field, back, s, ErrFieldRoundTrip)
	}
	return img, nil
}

// encodeASCII is the write half of decodeASCII, and it is the half that holds
// the field to ASCII on its own. decodeASCII is string(cstr(b)) — no code-page
// mapping and no validation — so it reads back every NUL-free byte string,
// above 0x7f included, and the round-trip check in encodeField therefore
// recovers nothing about ASCII-ness. The two guards are orthogonal rather than
// one being a weaker form of the other: this scan owns the non-ASCII case, the
// round trip owns the NUL case. Any rune above 0x7f is rejected here; a byte
// sequence that is not valid UTF-8 ranges as U+FFFD and is rejected with it.
func encodeASCII(s string) ([]byte, error) {
	for i, r := range s {
		if r > 0x7f {
			return nil, fmt.Errorf("at byte %d, %#U is not ASCII: %w", i, r, ErrFieldUnencodable)
		}
	}
	return []byte(s), nil
}

// encodeCP1251 is the write half of decodeCP1251.
func encodeCP1251(s string) ([]byte, error) {
	b, err := charmap.Windows1251.NewEncoder().Bytes([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFieldUnencodable, err)
	}
	return b, nil
}
