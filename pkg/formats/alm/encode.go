package alm

// The authored text fields: one TextField per field, the reader and the writer
// of its bytes. A field is a fixed-width NUL-terminated slot, so writing one is
// not "encode the text": it is building the whole field image — encoded
// bytes, then NUL fill — and it is allowed to fail. Failing is what lets a
// caller be promised that a string this package decoded from a field is
// written back as that field's own bytes or not written at all.

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
	ErrFieldTooLong     encodeError = "encoded text does not fit the field"
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
	return mapNameASCII.Image(s)
}

// EncodeDescription builds the 64-byte image of the type-0 description field
// (+0x78) for s, on the same terms as EncodeName but through Windows-1251:
// the code page's bytes for s, then NUL fill. A rune the code page has no byte
// for is rejected with the codec's own error, so the reject set is the code
// page's, not an approximation of it.
func EncodeDescription(s string) ([]byte, error) {
	return DescriptionText.Image(s)
}

// encodeASCII is the write half of decodeASCII, and it is the half that holds
// the field to ASCII on its own. decodeASCII is string(cstr(b)) — no code-page
// mapping and no validation — so it reads back every NUL-free byte string,
// above 0x7f included, and the round-trip check in TextField.Encode therefore
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

// TextField is one authored text field of a map record: its width, its code
// page and whether the text must leave room for a NUL terminator. Every field
// refuses a rune its code page has no byte for, an embedded NUL and a text
// longer than the field; nothing is substituted, dropped or truncated. Decode
// reads the field through its first NUL.
type TextField struct {
	Name       string // what an error names
	Width      int    // bytes in the record
	Terminated bool   // the text leaves at least one NUL inside Width
	ascii      bool   // ASCII only; otherwise Windows-1251
}

// The authored text fields. Windows-1251 is ASCII's superset, so every shipped
// name, all of them ASCII, reads and writes the same bytes under either.
var (
	// MapNameText is the type-0 name (+0x30) as the editor authors it
	// (DIV-3386). EncodeName keeps the format's own ASCII rule (ALM-META-010).
	MapNameText = TextField{Name: "name", Width: MetaNameLen, Terminated: true}
	// DescriptionText is the type-0 description (+0x78) (ALM-META-010).
	DescriptionText = TextField{Name: "description", Width: MetaDescLen, Terminated: true}
	// PlayerNameText is the type-5 player name (+0x0c). Map.Groups keeps the
	// field's bytes, which the world carries; this reads them as text.
	PlayerNameText = TextField{Name: "player name", Width: GroupNameLen}
	// ScriptLabelText is a node's label (+0x000) and a trigger's name (+0x00).
	// The game reads neither as text (ALM-NODEREAD-232, ALM-TRIGREAD-233).
	ScriptLabelText = TextField{Name: "script label", Width: ScriptLabelLen}
	// mapNameASCII is the format's type-0 name rule, EncodeName's codec.
	mapNameASCII = TextField{Name: "name", Width: MetaNameLen, Terminated: true, ascii: true}
)

// Decode is the field's text through its first NUL, as UTF-8.
func (f TextField) Decode(b []byte) string {
	if f.ascii {
		return decodeASCII(b)
	}
	return decodeCP1251(b)
}

// Encode is the field's bytes for s, without the NUL fill: at most Width
// bytes, Width-1 for a terminated field. It refuses what Image refuses.
func (f TextField) Encode(s string) ([]byte, error) {
	encode := encodeCP1251
	if f.ascii {
		encode = encodeASCII
	}
	text, err := encode(s)
	if err != nil {
		return nil, fmt.Errorf("alm: %s field: %w", f.Name, err)
	}
	limit := f.Width
	if f.Terminated {
		limit--
	}
	if len(text) > limit {
		return nil, fmt.Errorf("alm: %s field: %d encoded bytes, at most %d: %w", f.Name, len(text), limit, ErrFieldTooLong)
	}
	if back := f.Decode(text); back != s {
		return nil, fmt.Errorf("alm: %s field: the bytes read back as %q, not %q: %w", f.Name, back, s, ErrFieldRoundTrip)
	}
	return text, nil
}

// Image is the whole Width-byte field for s: its bytes, then NUL fill, in a
// fresh buffer.
func (f TextField) Image(s string) ([]byte, error) {
	text, err := f.Encode(s)
	if err != nil {
		return nil, err
	}
	img := make([]byte, f.Width)
	copy(img, text)
	return img, nil
}
