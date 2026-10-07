package data

import (
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The measured key inventories, written out here rather than derived from the
// tables they check: a table that has silently lost or gained a row is exactly
// what this number catches, and a length taken from the table itself would move
// with it.
const (
	unitKeyCount      = 37
	objectKeyCount    = 16
	structureKeyCount = 23
)

// goTypeOf is the Go type each kind is stored at, stated here rather than read
// off the loader so the test asserts the mapping instead of agreeing with it.
func goTypeOf(t *testing.T, k keyKind) reflect.Type {
	t.Helper()
	switch k {
	case kindInt:
		return reflect.TypeOf(int32(0))
	case kindStr:
		return reflect.TypeOf("")
	case kindArray:
		return reflect.TypeOf([]int32(nil))
	}
	t.Fatalf("unknown key kind %d", int(k))
	return nil
}

// checkTable pins one key table against its struct: every exported field is one
// row and every row one field, at the field's own type, so no key can be
// validated but never stored, or stored but never kind-checked.
//
// What it deliberately cannot see is a row whose ptr yields ANOTHER field of the
// same kind — the pointer is checked to be a field of this struct and to carry
// the kind's type, not to be the field the row names. That pairing is pinned by
// the written-out inventory round-trip instead.
func checkTable[T any](t *testing.T, label string, rows []keyRow[T], wantKeys int) {
	t.Helper()

	if len(rows) != wantKeys {
		t.Errorf("%s: key table has %d rows, want %d — the measured inventory", label, len(rows), wantKeys)
	}

	var c T
	v := reflect.ValueOf(&c).Elem()
	typ := v.Type()

	// Every exported field by name, and the address of each, so a row's closure
	// can be shown to yield a field of this struct rather than some other
	// variable it happens to have captured.
	fieldIndex := make(map[string]int, typ.NumField())
	fieldAddr := make(map[uintptr]bool, typ.NumField())
	exported := 0
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		exported++
		fieldIndex[f.Name] = i
		fieldAddr[v.Field(i).Addr().Pointer()] = true
	}
	if exported != wantKeys {
		t.Errorf("%s: %s has %d exported fields, want %d — every exported field is an inventory key",
			label, typ.Name(), exported, wantKeys)
	}

	rowOf := make(map[string]bool, len(rows))
	for _, row := range rows {
		if rowOf[row.name] {
			t.Errorf("%s: key %q has more than one row", label, row.name)
			continue
		}
		rowOf[row.name] = true

		i, ok := fieldIndex[row.name]
		if !ok {
			t.Errorf("%s: key %q has no field on %s — it would be validated and then dropped",
				label, row.name, typ.Name())
			continue
		}

		want := goTypeOf(t, row.kind)
		if got := typ.Field(i).Type; got != want {
			t.Errorf("%s: key %q is read at kind %v (%s) but field %s is %s",
				label, row.name, row.kind, want, row.name, got)
		}

		p := row.ptr(&c)
		pv := reflect.ValueOf(p)
		if !pv.IsValid() || pv.Kind() != reflect.Pointer || pv.IsNil() {
			t.Errorf("%s: key %q yields %v, want a pointer to its field", label, row.name, p)
			continue
		}
		if got := pv.Type().Elem(); got != want {
			t.Errorf("%s: key %q is read at kind %v but its pointer is *%s, want *%s",
				label, row.name, row.kind, got, want)
		}
		if !fieldAddr[pv.Pointer()] {
			t.Errorf("%s: key %q points outside %s — a value stored there is not on the class",
				label, row.name, typ.Name())
		}
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.IsExported() && !rowOf[f.Name] {
			t.Errorf("%s: field %s.%s has no key table row — nothing would ever load it",
				label, typ.Name(), f.Name)
		}
	}
}

func TestKeyTablesPinnedToTheirStructs(t *testing.T) {
	checkTable(t, "units", unitKeys, unitKeyCount)
	checkTable(t, "objects", objectKeys, objectKeyCount)
	checkTable(t, "structures", structureKeys, structureKeyCount)
}

// The descriptors are transcription too: a wrong count key or section prefix
// makes a whole registry read as missing, at the first lookup.
func TestDescriptors(t *testing.T) {
	for _, tc := range []struct {
		label     string
		got, want descriptor
	}{
		{"units", unitDesc, descriptor{"UnitCount", "Unit", "units/", true, true}},
		{"objects", objectDesc, descriptor{"ObjectCount", "Object", "objects/", true, true}},
		{"structures", structureDesc, descriptor{"Count", "Structure", "structures/", false, false}},
	} {
		if tc.got != tc.want {
			t.Errorf("%s descriptor = %+v, want %+v", tc.label, tc.got, tc.want)
		}
	}
}

// readerReg is the synthetic registry the node reader runs against: one class
// section holding a key of each kind, the array sentinel, and the three shapes
// that must read as wrong kind rather than as absent.
func readerReg(t *testing.T) *reg.Reg {
	t.Helper()
	r, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{
		{Name: "Unit0", Kind: 0x01, Children: []synth.RegNode{
			{Name: "ID", Kind: 0x02, Int: 7},
			{Name: "DescText", Kind: 0x00, Str: "alpha"},
			{Name: "Sound", Kind: 0x06, Ints: []int32{1, 2, 3, 4, 5}},
			// The editor's "none" marker: an array key stored as a
			// zero-length string.
			{Name: "AttackAnimTime", Kind: 0x00, Str: ""},
			// A non-empty string where an array is expected.
			{Name: "MoveAnimTime", Kind: 0x00, Str: "x"},
			// A scalar at the wrong type.
			{Name: "Palette", Kind: 0x00, Str: "x"},
			// A directory where a key is expected.
			{Name: "Inner", Kind: 0x01, Children: []synth.RegNode{
				{Name: "Depth", Kind: 0x02, Int: 1},
			}},
		}},
		{Name: "Version", Kind: 0x02, Int: 3}, // a value node directly under root
	}))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return r
}

func TestFindSection(t *testing.T) {
	r := readerReg(t)

	if findSection(r, "Unit0") == nil {
		t.Error("findSection(Unit0) = nil, want the section node")
	}
	if findSection(r, "uNiT0") == nil {
		t.Error("findSection(uNiT0) = nil, want the section node — names fold over ASCII")
	}
	if got := findSection(r, "Unit1"); got != nil {
		t.Errorf("findSection(Unit1) = %v, want nil — no such section", got)
	}
	if got := findSection(r, "Version"); got != nil {
		t.Errorf("findSection(Version) = %v, want nil — a value node is not a section", got)
	}
	if got := findSection(nil, "Unit0"); got != nil {
		t.Errorf("findSection(nil, ...) = %v, want nil", got)
	}
}

func TestReadKeySeparatesAbsentPresentAndWrongKind(t *testing.T) {
	sec := findSection(readerReg(t), "Unit0")
	if sec == nil {
		t.Fatal("findSection(Unit0) = nil")
	}

	for _, tc := range []struct {
		name string
		key  string
		kind keyKind
		want keyState
	}{
		{"int present", "ID", kindInt, keyPresent},
		{"int present, folded name", "iD", kindInt, keyPresent},
		{"str present", "DescText", kindStr, keyPresent},
		{"array present", "Sound", kindArray, keyPresent},
		{"the empty-string sentinel stands as an array", "AttackAnimTime", kindArray, keyPresent},

		{"int absent", "AttackDelay", kindInt, keyAbsent},
		{"str absent", "InfoPicture", kindStr, keyAbsent},
		{"array absent", "ShootOffset", kindArray, keyAbsent},

		{"str where an int is expected", "Palette", kindInt, keyWrongKind},
		{"int where a str is expected", "ID", kindStr, keyWrongKind},
		{"int where an array is expected", "ID", kindArray, keyWrongKind},
		{"a non-empty string where an array is expected", "MoveAnimTime", kindArray, keyWrongKind},
		{"an array where an int is expected", "Sound", kindInt, keyWrongKind},
		{"a directory where a key is expected", "Inner", kindInt, keyWrongKind},
		{"a directory where a str is expected", "Inner", kindStr, keyWrongKind},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, st := readKey(sec, tc.key, tc.kind)
			if st != tc.want {
				t.Fatalf("readKey(%q, %v) state = %d, want %d", tc.key, tc.kind, st, tc.want)
			}
			if (n != nil) != (st == keyPresent) {
				t.Errorf("readKey(%q, %v) = %v at state %d; a node comes back only when present",
					tc.key, tc.kind, n, st)
			}
		})
	}

	t.Run("a present node carries its value", func(t *testing.T) {
		n, _ := readKey(sec, "ID", kindInt)
		if n == nil || n.Int != 7 {
			t.Errorf("readKey(ID) node = %v, want Int 7", n)
		}
		n, _ = readKey(sec, "DescText", kindStr)
		if n == nil || n.Str != "alpha" {
			t.Errorf("readKey(DescText) node = %v, want Str \"alpha\"", n)
		}
		n, _ = readKey(sec, "Sound", kindArray)
		if n == nil || len(n.Ints) != 5 {
			t.Errorf("readKey(Sound) node = %v, want 5 elements", n)
		}
		// The sentinel is present and length 0 — which is what makes it
		// inherit like an absent key rather than clear anything.
		n, _ = readKey(sec, "AttackAnimTime", kindArray)
		if n == nil || len(n.Ints) != 0 {
			t.Errorf("readKey(AttackAnimTime) node = %v, want a present node of length 0", n)
		}
	})

	t.Run("a nil section reads as absent, not a panic", func(t *testing.T) {
		if n, st := readKey(nil, "ID", kindInt); n != nil || st != keyAbsent {
			t.Errorf("readKey(nil section) = %v, %d; want nil, keyAbsent", n, st)
		}
	})
}

// The fold is this package's own copy, so its boundary is pinned here too: A-Z
// folds and a byte >= 0x80 compares as itself. Names are built from bytes, never
// written as literal non-ASCII text.
func TestFoldBoundary(t *testing.T) {
	hi := string([]byte{0x53, 0x80})       // "S" + 0x80
	hiFolded := string([]byte{0x73, 0x80}) // "s" + 0x80
	hiOther := string([]byte{0x53, 0x81})  // "S" + 0x81

	r, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{
		{Name: hi, Kind: 0x01, Children: []synth.RegNode{
			{Name: "ID", Kind: 0x02, Int: 5},
		}},
	}))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if findSection(r, hi) == nil || findSection(r, hiFolded) == nil {
		t.Error("the ASCII byte of a name must fold even beside a byte >= 0x80")
	}
	if got := findSection(r, hiOther); got != nil {
		t.Errorf("findSection(0x53 0x81) = %v, want nil — 0x80 and 0x81 are not ASCII case variants", got)
	}
}
