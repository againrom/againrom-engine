package mapedit_test

// The type-0 setters: the angle, the five stored scalars, the low-bit word and
// the two string fields. Every diff is taken through fixture_test.go's
// comparator with the field located by the frame walk, so no case asks the model
// where it wrote; every expected image is spelled out here rather than obtained
// from the code under test. Code-page text appears as code points with the bytes
// it must become declared beside it, never as literal non-ASCII source.

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The values these cases write. Each is outside the rich fixture's declared
// table and distinct from the others, so a setter that landed on the wrong field
// cannot pass by writing a value that was already sitting there.
const (
	// A finite float32 (1.5), and a second signalling NaN — exponent all ones,
	// quiet bit clear, mantissa non-zero — distinct from the fixture's own.
	metaNewAngleBits = 0x3fc00000
	metaNewSNaNBits  = 0x7f800001

	metaNewWord0C  = 0x1f00000c
	metaNewWord10  = 0x1f000010
	metaNewWord14  = 0x1f000014
	metaNewWord70  = 0x1f000070
	metaNewWord74  = 0x1f000074
	metaNewBitmask = 0x00001f18

	// Shorter than the fixture's "Rich", so the replaced field cannot keep a
	// tail of the old text and pass.
	metaNewName = "Ok"

	// Three Cyrillic code points, written as code points. Windows-1251 has one
	// byte for each, and those bytes are declared below rather than left to a
	// codec to decide.
	metaNewDesc = "\u0414\u043e\u043c"

	// One Cyrillic code point, used to fill a field and to overflow it, and the
	// two code points no field can hold: a rune outside Windows-1251's
	// repertoire, and the replacement character the reader yields for that code
	// page's one undefined byte.
	metaCyrillicA   = "\u0430"
	metaNoCP1251    = "\u2603"
	metaReplacement = "\ufffd"
)

var metaNewDescBytes = []byte{0xc4, 0xee, 0xec}

// metaWordImage is the little-endian image a four-byte field must hold for v,
// written out byte by byte: a witness that encoded its expectation the way the
// setter does would agree with a byte-swapped setter.
func metaWordImage(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

// metaFieldImage is what a 64-byte string field must hold for encoded bytes b:
// b, then NUL to the field's width. Spelled out here because "the whole field is
// rewritten" is the claim — a setter that wrote the text and left the rest of
// the field alone must fail against this.
func metaFieldImage(b []byte) []byte {
	img := make([]byte, metaFieldLen)
	copy(img, b)
	return img
}

// metaScalars is the seven four-byte type-0 fields this story sets: the setter,
// the payload-relative offset it must write, the bits the field must then hold,
// and how the shipped decoder reads that field back. The angle is read back as
// bits, never as a float: the fixture's own angle is a signalling NaN and NaN
// equals nothing, itself included.
var metaScalars = []struct {
	name string
	rel  int
	set  func(*mapedit.Editor) error
	bits uint32
	view func(*alm.Map) uint32
}{
	{"angle", metaAngle,
		func(ed *mapedit.Editor) error { return ed.SetAngle(math.Float32frombits(metaNewAngleBits)) },
		metaNewAngleBits, func(m *alm.Map) uint32 { return math.Float32bits(m.Angle) }},
	{"Word0C", metaWord0C,
		func(ed *mapedit.Editor) error { return ed.SetWord0C(metaNewWord0C) },
		metaNewWord0C, func(m *alm.Map) uint32 { return m.Meta.Word0C }},
	{"Word10", metaWord10,
		func(ed *mapedit.Editor) error { return ed.SetWord10(metaNewWord10) },
		metaNewWord10, func(m *alm.Map) uint32 { return m.Meta.Word10 }},
	{"Word14", metaWord14,
		func(ed *mapedit.Editor) error { return ed.SetWord14(metaNewWord14) },
		metaNewWord14, func(m *alm.Map) uint32 { return m.Meta.Word14 }},
	{"Word70", metaWord70,
		func(ed *mapedit.Editor) error { return ed.SetWord70(metaNewWord70) },
		metaNewWord70, func(m *alm.Map) uint32 { return m.Meta.Word70 }},
	{"Word74", metaWord74,
		func(ed *mapedit.Editor) error { return ed.SetWord74(metaNewWord74) },
		metaNewWord74, func(m *alm.Map) uint32 { return m.Meta.Word74 }},
	{"bitmask", metaBitmask,
		func(ed *mapedit.Editor) error { return ed.SetBitmask(metaNewBitmask) },
		metaNewBitmask, func(m *alm.Map) uint32 { return m.Meta.Bitmask }},
}

// ---------------------------------------------------------------------------
// SC-3, type-0 half — each field inside its own four or sixty-four bytes
// ---------------------------------------------------------------------------

// TestMetaScalarSetterWritesItsFieldAndCarriesEveryOtherByte is AC-2's scalar
// half, each mutation on a fresh load: the diff is confined to the field's four
// bytes, the field holds the value's own little-endian image, and the shipped
// decoder reads the new value back. Everything the story requires a type-0 edit
// to carry — the 448 bytes of text slots, both strings' post-NUL residue, the
// file header's dataSize, the per-map constants, the counts and, for every
// setter but the angle's, the signalling-NaN angle bits — is carried by the
// comparator's own clause rather than by a list of assertions here.
func TestMetaScalarSetterWritesItsFieldAndCarriesEveryOtherByte(t *testing.T) {
	for _, v := range fixtureVariants() {
		for _, s := range metaScalars {
			t.Run(v.name+"/"+s.name, func(t *testing.T) {
				before := richFixture(v.order, v.nUnits)
				ed := newEditor(t, before)

				if err := s.set(ed); err != nil {
					t.Fatalf("set %s: %v", s.name, err)
				}
				after := ed.Bytes()
				if len(after) != len(before) {
					t.Fatalf("a fixed-length edit changed the image size: %d -> %d", len(before), len(after))
				}

				fsBefore, fsAfter := walkFrame(t, before), walkFrame(t, after)
				target := fsBefore.payload(t, 0, s.rel, 4)
				if got := fsAfter.payload(t, 0, s.rel, 4); got != target {
					t.Fatalf("the field moved: %+v -> %+v", target, got)
				}

				if ok, why := carriedIdentical(before, after, []region{target}, []region{target}); !ok {
					t.Errorf("a byte outside the field changed: %s", why)
				}

				want := metaWordImage(s.bits)
				if got := fsAfter.at(t, after, target); !bytes.Equal(got, want) {
					t.Errorf("the field holds % x, want % x", got, want)
				}
				if bytes.Equal(fsBefore.at(t, before, target), want) {
					t.Fatal("the field already held the value written, so this case witnesses no write")
				}

				m, err := ed.Map()
				if err != nil {
					t.Fatalf("the view failed after the edit: %v", err)
				}
				if got := s.view(m); got != s.bits {
					t.Errorf("the view's %s = %#08x, want %#08x", s.name, got, s.bits)
				}
			})
		}
	}
}

// TestSetAngleWritesTheCallersBitsAndNotAFloatOfThem is the angle's own case,
// and the reason the setter converts rather than reads: the value written is a
// signalling NaN, so a setter that had passed it through a float comparison, a
// normalisation or a read-back of the field's current bytes would land on the
// quiet pattern instead of this one. The fixture's angle is already a different
// signalling NaN, so neither "left it alone" nor "quieted it" passes.
func TestSetAngleWritesTheCallersBitsAndNotAFloatOfThem(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			before := richFixture(order, fixUnitsMax)
			ed := newEditor(t, before)

			if err := ed.SetAngle(math.Float32frombits(metaNewSNaNBits)); err != nil {
				t.Fatalf("SetAngle: %v", err)
			}
			after := ed.Bytes()

			target := walkFrame(t, after).payload(t, 0, metaAngle, 4)
			want := metaWordImage(metaNewSNaNBits)
			if got := walkFrame(t, after).at(t, after, target); !bytes.Equal(got, want) {
				t.Errorf("the angle field holds % x, want the signalling NaN % x", got, want)
			}
			if ok, why := carriedIdentical(before, after, []region{target}, []region{target}); !ok {
				t.Errorf("a byte outside the angle field changed: %s", why)
			}

			m, err := ed.Map()
			if err != nil {
				t.Fatalf("the view failed after the edit: %v", err)
			}
			if got := math.Float32bits(m.Angle); got != metaNewSNaNBits {
				t.Errorf("the view's angle bits = %#08x, want %#08x", got, uint32(metaNewSNaNBits))
			}
		})
	}
}

// metaStrings is the two string fields: the setter, the offset of the 64 bytes
// it must rewrite, the whole image the field must then hold, the string the
// shipped decoder must give back, the residue of the text being replaced, and
// the other field's decoded value, which must not move.
var metaStrings = []struct {
	name      string
	rel       int
	set       func(*mapedit.Editor) error
	wantField []byte
	wantView  string
	view      func(*alm.Map) string
	residue   []byte
	otherView func(*alm.Map) string
}{
	{
		"name", metaName,
		func(ed *mapedit.Editor) error { return ed.SetName(metaNewName) },
		metaFieldImage([]byte(metaNewName)), metaNewName,
		func(m *alm.Map) string { return m.Name },
		fixNameResidue,
		func(m *alm.Map) string { return m.Description },
	},
	{
		"description", metaDesc,
		func(ed *mapedit.Editor) error { return ed.SetDescription(metaNewDesc) },
		metaFieldImage(metaNewDescBytes), metaNewDesc,
		func(m *alm.Map) string { return m.Description },
		fixDescResidue,
		func(m *alm.Map) string { return m.Name },
	},
}

// TestStringSetterRewritesItsWholeFieldAndCarriesEveryOtherByte is AC-2's string
// half and SC-3's residue clause in one: the diff is confined to the field's own
// sixty-four bytes, those bytes are the encoded text then NUL fill, the bytes
// past the replaced text's terminator are gone from the field, and the other
// field — whose own post-NUL residue is carried — still decodes to what it did.
func TestStringSetterRewritesItsWholeFieldAndCarriesEveryOtherByte(t *testing.T) {
	for _, v := range fixtureVariants() {
		for _, s := range metaStrings {
			t.Run(v.name+"/"+s.name, func(t *testing.T) {
				before := richFixture(v.order, v.nUnits)
				ed := newEditor(t, before)

				otherBefore, err := alm.Open(before)
				if err != nil {
					t.Fatalf("alm.Open of the fixture: %v", err)
				}

				if err := s.set(ed); err != nil {
					t.Fatalf("set the %s: %v", s.name, err)
				}
				after := ed.Bytes()
				if len(after) != len(before) {
					t.Fatalf("a fixed-length edit changed the image size: %d -> %d", len(before), len(after))
				}

				fsBefore, fsAfter := walkFrame(t, before), walkFrame(t, after)
				target := fsBefore.payload(t, 0, s.rel, metaFieldLen)
				if got := fsAfter.payload(t, 0, s.rel, metaFieldLen); got != target {
					t.Fatalf("the field moved: %+v -> %+v", target, got)
				}

				if ok, why := carriedIdentical(before, after, []region{target}, []region{target}); !ok {
					t.Errorf("a byte outside the field changed: %s", why)
				}

				field := fsAfter.at(t, after, target)
				if !bytes.Equal(field, s.wantField) {
					t.Errorf("the %s field holds\n % x\nwant\n % x", s.name, field, s.wantField)
				}
				if bytes.Equal(fsBefore.at(t, before, target), s.wantField) {
					t.Fatal("the field already held the image written, so this case witnesses no write")
				}
				// The residue clause, stated on its own: the replaced text's
				// post-NUL bytes are not somewhere else in the field.
				if i := bytes.Index(field, s.residue); i >= 0 {
					t.Errorf("the rewritten %s field still holds the replaced text's residue % x at byte %d",
						s.name, s.residue, i)
				}

				m, err := ed.Map()
				if err != nil {
					t.Fatalf("the view failed after the edit: %v", err)
				}
				if got := s.view(m); got != s.wantView {
					t.Errorf("the view's %s = %+q, want %+q", s.name, got, s.wantView)
				}
				if got, want := s.otherView(m), s.otherView(otherBefore); got != want {
					t.Errorf("setting the %s changed the other string field: %+q -> %+q", s.name, want, got)
				}
			})
		}
	}
}

// metaFixtureWithFieldByte returns the rich fixture with both string fields
// holding the single byte b and NUL fill. It edits the image through the frame
// walk, so the fields are found the same way every other case here finds them.
func metaFixtureWithFieldByte(t *testing.T, b byte) []byte {
	t.Helper()
	data := richFixture(orderType0First, fixUnitsMax)
	fs := walkFrame(t, data)
	for _, rel := range []int{metaName, metaDesc} {
		field := fs.at(t, data, fs.payload(t, 0, rel, metaFieldLen))
		for i := range field {
			field[i] = 0
		}
		field[0] = b
	}
	return data
}

// TestSettingAFieldToTheStringTheReaderGaveBackRestoresItsBytes is SC-4 driven
// through the setters instead of through the encoders: for every byte a field
// can hold, the model is asked to write back the string the model's own view
// decoded out of that field, and the whole image must come out unchanged — the
// field restored byte for byte and everything around it carried — or the call
// must be rejected with nothing touched.
//
// The reject sets are asserted exactly and in both directions, which is what
// makes the case discriminating: a setter that refused every string would report
// a reject set of 1..255 and fail, and one that accepted a byte it cannot
// reproduce would fail on the image.
func TestSettingAFieldToTheStringTheReaderGaveBackRestoresItsBytes(t *testing.T) {
	var nameRejected, descRejected []byte

	for v := 1; v <= 255; v++ {
		b := byte(v)
		data := metaFixtureWithFieldByte(t, b)

		for _, s := range []struct {
			what     string
			rejected *[]byte
			read     func(*alm.Map) string
			set      func(*mapedit.Editor, string) error
		}{
			{"name", &nameRejected, func(m *alm.Map) string { return m.Name },
				func(ed *mapedit.Editor, s string) error { return ed.SetName(s) }},
			{"description", &descRejected, func(m *alm.Map) string { return m.Description },
				func(ed *mapedit.Editor, s string) error { return ed.SetDescription(s) }},
		} {
			ed := newEditor(t, data)
			m, err := ed.Map()
			if err != nil {
				t.Fatalf("the view of a map whose %s field holds %#02x failed: %v", s.what, b, err)
			}
			before := ed.Bytes()

			if err := s.set(ed, s.read(m)); err != nil {
				*s.rejected = append(*s.rejected, b)
				if got := ed.Bytes(); !bytes.Equal(got, before) {
					t.Errorf("a rejected %s write for field byte %#02x changed the image", s.what, b)
				}
				if ed.CanUndo() {
					t.Errorf("a rejected %s write for field byte %#02x entered the history", s.what, b)
				}
				continue
			}
			if got := ed.Bytes(); !bytes.Equal(got, before) {
				t.Errorf("setting the %s to the string the reader gave back for field byte %#02x changed the image",
					s.what, b)
			}
		}
	}

	// The name field is ASCII, so every high byte is out; the description is
	// Windows-1251, whose one undefined byte is 0x98.
	var wantName []byte
	for v := 0x80; v <= 0xff; v++ {
		wantName = append(wantName, byte(v))
	}
	if got, want := fmt.Sprintf("% x", nameRejected), fmt.Sprintf("% x", wantName); got != want {
		t.Errorf("SetName reject set:\n got %s\nwant %s", got, want)
	}
	if got, want := fmt.Sprintf("% x", descRejected), fmt.Sprintf("% x", []byte{0x98}); got != want {
		t.Errorf("SetDescription reject set:\n got %s\nwant %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// SC-6 — the string rejections
// ---------------------------------------------------------------------------

// TestStringSetterAcceptsAFieldFullOfText is the boundary the rejections below
// are measured against: the field is 64 bytes and must stay terminated, so 63
// encoded bytes fit. Sixty-three Cyrillic code points are 126 bytes of UTF-8 and
// 63 of Windows-1251, which pins what the limit counts.
func TestStringSetterAcceptsAFieldFullOfText(t *testing.T) {
	ed := newEditor(t, richFixture(orderType6First, fixUnitsMax))

	if err := ed.SetName(strings.Repeat("a", metaFieldLen-1)); err != nil {
		t.Errorf("SetName(%d ASCII bytes): %v", metaFieldLen-1, err)
	}
	full := strings.Repeat(metaCyrillicA, metaFieldLen-1)
	if err := ed.SetDescription(full); err != nil {
		t.Errorf("SetDescription(%d Cyrillic code points, %d UTF-8 bytes): %v", metaFieldLen-1, len(full), err)
	}

	m, err := ed.Map()
	if err != nil {
		t.Fatalf("the view failed after the edits: %v", err)
	}
	if got := len(m.Name); got != metaFieldLen-1 {
		t.Errorf("the view's name is %d bytes, want %d", got, metaFieldLen-1)
	}
	if got := len([]rune(m.Description)); got != metaFieldLen-1 {
		t.Errorf("the view's description is %d code points, want %d", got, metaFieldLen-1)
	}
}

// TestStringSetterRejectsWhatItsFieldCannotHold is AC-4's string half. Each case
// runs against three history states rather than only a fresh model: the undone
// one is the discriminating state, because a setter that built and applied its
// edit before validating would truncate the redo history on its way to returning
// an error, and only a model with something to redo can see that.
//
// The name field's rejection is a rune-level rule, and the last two cases are
// the pair that says so. A raw 0x80 is not valid UTF-8; U+00E9 is valid UTF-8
// and still not ASCII. Anything that tested validity rather than the code point
// would accept the second, so the two together pin the rule as "not ASCII"
// rather than "not decodable".
func TestStringSetterRejectsWhatItsFieldCannotHold(t *testing.T) {
	cases := []struct {
		name string
		call func(*mapedit.Editor) error
	}{
		{"a description of 64 Cyrillic code points", func(ed *mapedit.Editor) error {
			return ed.SetDescription(strings.Repeat(metaCyrillicA, metaFieldLen))
		}},
		{"a description holding a rune Windows-1251 has no byte for", func(ed *mapedit.Editor) error {
			return ed.SetDescription(metaNoCP1251)
		}},
		{"a description holding U+FFFD, which the reader yields for the undefined byte", func(ed *mapedit.Editor) error {
			return ed.SetDescription(metaReplacement)
		}},
		{"a description with an embedded NUL", func(ed *mapedit.Editor) error {
			return ed.SetDescription("ab\x00cd")
		}},
		{"a name of 64 ASCII bytes", func(ed *mapedit.Editor) error {
			return ed.SetName(strings.Repeat("a", metaFieldLen))
		}},
		{"a name holding a Cyrillic code point", func(ed *mapedit.Editor) error {
			return ed.SetName(metaCyrillicA)
		}},
		{"a name holding a raw byte that is not valid UTF-8", func(ed *mapedit.Editor) error {
			return ed.SetName(string([]byte{'a', 0x80}))
		}},
		{"a name holding U+00E9, valid UTF-8 and still not ASCII", func(ed *mapedit.Editor) error {
			return ed.SetName(string(rune(0x00e9)))
		}},
		{"a name with a trailing NUL", func(ed *mapedit.Editor) error {
			return ed.SetName("ab\x00")
		}},
	}

	states := []struct {
		name  string
		setUp func(t *testing.T, ed *mapedit.Editor)
	}{
		{"a fresh model", func(*testing.T, *mapedit.Editor) {}},
		{"one accepted edit", func(t *testing.T, ed *mapedit.Editor) {
			if err := ed.SetBitmask(metaNewBitmask); err != nil {
				t.Fatalf("the set-up edit failed: %v", err)
			}
		}},
		{"one edit, undone", func(t *testing.T, ed *mapedit.Editor) {
			if err := ed.SetBitmask(metaNewBitmask); err != nil {
				t.Fatalf("the set-up edit failed: %v", err)
			}
			if !ed.Undo() {
				t.Fatal("the set-up Undo reported nothing to undo")
			}
		}},
	}

	for _, state := range states {
		for _, c := range cases {
			t.Run(state.name+"/"+c.name, func(t *testing.T) {
				ed := newEditor(t, richFixture(orderType6First, fixUnitsMax))
				state.setUp(t, ed)
				rejectionChangesNothing(t, ed, func() error { return c.call(ed) })
			})
		}
	}
}
