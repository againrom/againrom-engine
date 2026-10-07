package alm_test

// Tests for the write half of the type-0 string-field codecs
// (docs/0025-mapedit-model). Every fixture is a synthetic byte stream built by
// the 0003 builders in alm_test.go; no test reads a file from disk. Every
// non-ASCII value appears as a code point escape or as raw bytes, never as
// literal text in this source.

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/formats/alm"
)

// fieldWidth is both string fields' width; the test states it independently of
// the package so an encoder that changed it could not agree with this witness
// by construction.
const fieldWidth = 64

// fieldImage builds the field image whose leading bytes are b and whose
// remainder is NUL: what the field holds, not what the text is.
func fieldImage(b ...byte) []byte {
	img := make([]byte, fieldWidth)
	copy(img, b)
	return img
}

// readFields opens a map whose name field holds nameField and whose
// description field holds descField, and returns the two strings the shipped
// reader decoded out of them. The strings come from the reader, never from a
// literal here: that is what makes the round trip below a claim about this
// package rather than about the test's idea of the codecs.
func readFields(t *testing.T, nameField, descField []byte) (name, desc string) {
	t.Helper()
	p := baseSections(2, 2, 0, 0, 0)
	p[0] = buildMeta(2, 2, 0, 0, 0, string(nameField), descField)
	m := openOK(t, buildMap(p))
	return m.Name, m.Description
}

// assertRejected requires a rejection with a named cause and no image: "it was
// rejected" is not the claim being made anywhere below, "it was rejected for
// this reason and not for either of the others" is.
func assertRejected(t *testing.T, what string, img []byte, err error, want error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: want rejection caused by %q, got a %d-byte image and no error", what, want, len(img))
	}
	if img != nil {
		t.Errorf("%s: rejected but returned a %d-byte image, want nil", what, len(img))
	}
	if !errors.Is(err, want) {
		t.Errorf("%s: error %v does not carry the cause %q", what, err, want)
	}
	for _, other := range []error{alm.ErrFieldTooLong, alm.ErrFieldUnencodable, alm.ErrFieldRoundTrip} {
		if other != want && errors.Is(err, other) {
			t.Errorf("%s: error %v also carries the unrelated cause %q", what, err, other)
		}
	}
}

// ---------------------------------------------------------------------------
// SC-4: the byte-identical-or-rejected property, over every field byte
// ---------------------------------------------------------------------------

// TestFieldImageRestoresEveryByteTheReaderDecoded walks all 255 non-NUL bytes
// a field can hold, decodes each through the shipped reader and asks the
// encoder for the field back. The reject sets are asserted exactly, in both
// directions: an encoder that refuses everything fails here, and so does one
// that accepts a byte it cannot reproduce.
func TestFieldImageRestoresEveryByteTheReaderDecoded(t *testing.T) {
	var nameRejected, descRejected []byte

	for v := 1; v <= 255; v++ {
		b := byte(v)
		want := fieldImage(b)
		name, desc := readFields(t, []byte{b}, []byte{b})

		img, err := alm.EncodeName(name)
		switch {
		case err != nil:
			nameRejected = append(nameRejected, b)
		case !bytes.Equal(img, want):
			t.Errorf("EncodeName(reader's string for name byte %#02x) = % x, want % x", b, img, want)
		}

		img, err = alm.EncodeDescription(desc)
		switch {
		case err != nil:
			descRejected = append(descRejected, b)
		case !bytes.Equal(img, want):
			t.Errorf("EncodeDescription(reader's string for description byte %#02x) = % x, want % x", b, img, want)
		}
	}

	// The name field is ASCII, so every high byte is out; the description is
	// Windows-1251, whose one undefined byte is 0x98 (it decodes to U+FFFD,
	// which no byte encodes back).
	var wantName []byte
	for v := 0x80; v <= 0xff; v++ {
		wantName = append(wantName, byte(v))
	}
	if got, want := fmt.Sprintf("% x", nameRejected), fmt.Sprintf("% x", wantName); got != want {
		t.Errorf("EncodeName reject set:\n got %s\nwant %s", got, want)
	}
	if got, want := fmt.Sprintf("% x", descRejected), fmt.Sprintf("% x", []byte{0x98}); got != want {
		t.Errorf("EncodeDescription reject set:\n got %s\nwant %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// The images carry each field's own codec, read back by the shipped reader
// ---------------------------------------------------------------------------

func TestFieldImagesCarryTheirCodecsBytes(t *testing.T) {
	// Six Cyrillic code points, written as code points; Windows-1251 has one
	// byte for each.
	const cyrillic = "\u041f\u0440\u0438\u0432\u0435\u0442"
	wantDesc := fieldImage(0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2)
	descImg, err := alm.EncodeDescription(cyrillic)
	if err != nil {
		t.Fatalf("EncodeDescription(6 Cyrillic code points): %v", err)
	}
	if !bytes.Equal(descImg, wantDesc) {
		t.Errorf("EncodeDescription image = % x, want % x", descImg, wantDesc)
	}

	const ascii = "Ruins"
	wantName := fieldImage('R', 'u', 'i', 'n', 's')
	nameImg, err := alm.EncodeName(ascii)
	if err != nil {
		t.Fatalf("EncodeName(%q): %v", ascii, err)
	}
	if !bytes.Equal(nameImg, wantName) {
		t.Errorf("EncodeName image = % x, want % x", nameImg, wantName)
	}

	// The images are fields: written into a map at their offsets, the reader
	// gives both strings back.
	name, desc := readFields(t, nameImg, descImg)
	if name != ascii {
		t.Errorf("reader read the name image back as %q, want %q", name, ascii)
	}
	if desc != cyrillic {
		t.Errorf("reader read the description image back as %q, want %q", desc, cyrillic)
	}
}

// ---------------------------------------------------------------------------
// The three rejections, each under its own case and its own cause
// ---------------------------------------------------------------------------

// TestEncodeRejectsTextThatLeavesNoRoomForTheTerminator also pins what the
// limit counts: encoded bytes. Sixty-three Cyrillic code points are 126 bytes
// of UTF-8 and 63 bytes of Windows-1251, and they fit.
func TestEncodeRejectsTextThatLeavesNoRoomForTheTerminator(t *testing.T) {
	fits := strings.Repeat("\u0430", fieldWidth-1)
	img, err := alm.EncodeDescription(fits)
	if err != nil {
		t.Fatalf("EncodeDescription(%d Cyrillic code points, %d UTF-8 bytes): %v", fieldWidth-1, len(fits), err)
	}
	if !bytes.Equal(img, fieldImage(bytes.Repeat([]byte{0xe0}, fieldWidth-1)...)) {
		t.Errorf("EncodeDescription image = % x, want %d x 0xe0 then one NUL", img, fieldWidth-1)
	}

	img, err = alm.EncodeDescription(strings.Repeat("\u0430", fieldWidth))
	assertRejected(t, "EncodeDescription(64 Cyrillic code points)", img, err, alm.ErrFieldTooLong)

	if _, err := alm.EncodeName(strings.Repeat("a", fieldWidth-1)); err != nil {
		t.Fatalf("EncodeName(%d ASCII bytes): %v", fieldWidth-1, err)
	}
	img, err = alm.EncodeName(strings.Repeat("a", fieldWidth))
	assertRejected(t, "EncodeName(64 ASCII bytes)", img, err, alm.ErrFieldTooLong)
}

// TestEncodeRejectsRunesTheCodecCannotRepresent asks the codec itself for the
// verdict the encoder must be carrying, so the case cannot pass against an
// encoder that invented a reject set of its own.
func TestEncodeRejectsRunesTheCodecCannotRepresent(t *testing.T) {
	const noByte = "\u2603" // outside Windows-1251's repertoire
	_, codecErr := charmap.Windows1251.NewEncoder().Bytes([]byte(noByte))
	if codecErr == nil {
		t.Fatalf("baseline: the Windows-1251 encoder accepted %+q; this case no longer witnesses a codec rejection", noByte)
	}

	img, err := alm.EncodeDescription(noByte)
	assertRejected(t, "EncodeDescription(U+2603)", img, err, alm.ErrFieldUnencodable)
	if !errors.Is(err, codecErr) {
		t.Errorf("EncodeDescription(U+2603): error %v does not carry the codec's own error %v", err, codecErr)
	}

	// U+FFFD is what the reader yields for the one undefined byte, so it must
	// not encode back to some other byte.
	img, err = alm.EncodeDescription("\ufffd")
	assertRejected(t, "EncodeDescription(U+FFFD)", img, err, alm.ErrFieldUnencodable)

	// The name field is ASCII: a Cyrillic code point and a raw high byte are
	// equally out, the second reaching the scan as U+FFFD.
	img, err = alm.EncodeName("\u0430")
	assertRejected(t, "EncodeName(U+0430)", img, err, alm.ErrFieldUnencodable)
	img, err = alm.EncodeName(string([]byte{'a', 0x80}))
	assertRejected(t, "EncodeName(raw 0x80)", img, err, alm.ErrFieldUnencodable)

	// A code point in U+0080..U+00FF is the one input that separates "not
	// ASCII" from "not valid UTF-8": it is valid UTF-8 and still not ASCII, so
	// a check written as a UTF-8 validity test accepts it. The scan is what
	// rejects it — and not because the round trip happens to be weak on this
	// input. decodeASCII is string(cstr(b)) and validates nothing, so the round
	// trip reads back every NUL-free image, a raw 0x80 and Cyrillic included:
	// it is no kind of non-ASCII check. The two guards are orthogonal — this
	// field's non-ASCII case is the scan's, its NUL case the round trip's.
	img, err = alm.EncodeName(string(rune(0x00e9)))
	assertRejected(t, "EncodeName(U+00E9)", img, err, alm.ErrFieldUnencodable)
}

// TestEncodeRejectsEmbeddedNULAsARoundTripFailure is the case that separates
// our contract from the codecs': both codecs encode U+0000 to a plain NUL byte
// and report nothing, and the reader stops at the first NUL, so the image
// would silently hold a different string than the caller asked for.
func TestEncodeRejectsEmbeddedNULAsARoundTripFailure(t *testing.T) {
	const embedded = "ab\x00cd"
	got, codecErr := charmap.Windows1251.NewEncoder().Bytes([]byte(embedded))
	if codecErr != nil || !bytes.Equal(got, []byte(embedded)) {
		t.Fatalf("baseline: the Windows-1251 encoder returned (% x, %v) for an embedded NUL; this case would then witness the codec, not the round trip", got, codecErr)
	}

	img, err := alm.EncodeDescription(embedded)
	assertRejected(t, "EncodeDescription(embedded NUL)", img, err, alm.ErrFieldRoundTrip)
	img, err = alm.EncodeName(embedded)
	assertRejected(t, "EncodeName(embedded NUL)", img, err, alm.ErrFieldRoundTrip)

	// A trailing NUL is the same failure, and the one a caller is likeliest to
	// arrive at: the image is NUL-filled regardless, so the reader gives back
	// the text without it.
	img, err = alm.EncodeName("ab\x00")
	assertRejected(t, "EncodeName(trailing NUL)", img, err, alm.ErrFieldRoundTrip)
}

// ---------------------------------------------------------------------------
// What a returned image is: field-width, NUL-filled, and the caller's alone
// ---------------------------------------------------------------------------

func TestEncodeImagesAreFieldWidthAndOwnedByTheCaller(t *testing.T) {
	empty, err := alm.EncodeDescription("")
	if err != nil {
		t.Fatalf("EncodeDescription(\"\"): %v", err)
	}
	if !bytes.Equal(empty, make([]byte, fieldWidth)) {
		t.Errorf("EncodeDescription(\"\") = % x, want %d NUL bytes", empty, fieldWidth)
	}

	first, err := alm.EncodeName("map")
	if err != nil {
		t.Fatalf("EncodeName: %v", err)
	}
	second, err := alm.EncodeName("map")
	if err != nil {
		t.Fatalf("EncodeName: %v", err)
	}
	if len(first) != fieldWidth {
		t.Errorf("image is %d bytes, want %d", len(first), fieldWidth)
	}

	first[0] = 'X'
	if second[0] != 'm' {
		t.Errorf("mutating one image changed another: second = % x", second)
	}
	third, err := alm.EncodeName("map")
	if err != nil {
		t.Fatalf("EncodeName: %v", err)
	}
	if !bytes.Equal(third, fieldImage('m', 'a', 'p')) {
		t.Errorf("after a caller mutated an earlier image, EncodeName returned % x", third)
	}
}
