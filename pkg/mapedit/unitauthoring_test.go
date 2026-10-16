package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// typeSixPayload walks the physical record headers itself and returns the
// absolute offset of the last type-6 payload, independent of the model.
func typeSixPayload(t *testing.T, b []byte) int {
	t.Helper()
	at, off := -1, 20
	for n := uint32(0); n < binary.LittleEndian.Uint32(b[12:]); n++ {
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		if binary.LittleEndian.Uint32(b[off+12:]) == 6 {
			at = off + 20
		}
		off += 20 + size
	}
	if at < 0 {
		t.Fatal("fixture has no type-6 record")
	}
	return at
}

func authoringFixture(t *testing.T) ([]byte, int) {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: make([]byte, 12), Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40, ClassSubID: 2, DefID: 0},
		{X: 4*256 + 7, Y: 5*256 + 9, ClassID: 5, Flags: 0, DefID: 1234, CurrentHP: 70, HasCurrentHP: true},
	}})
	p := typeSixPayload(t, b)
	binary.LittleEndian.PutUint16(b[p+0x40:], 7)
	binary.LittleEndian.PutUint16(b[p+70+0x40:], 3)
	binary.LittleEndian.PutUint32(b[p+70+0x42:], 9)
	binary.LittleEndian.PutUint32(b[p+70+0x14:], 2)
	for k := 0x2c; k < 0x40; k++ {
		b[p+70+k] = byte(k) // carried bytes no field names
	}
	return b, p
}

func TestCloneUnitCopiesTheRecordToTheCellWithTheNextUnitID(t *testing.T) {
	b, p := authoringFixture(t)
	e := newEditor(t, b)
	got, err := e.CloneUnit(1, 9*256+7, 11*256+9)
	if err != nil || got != 2 {
		t.Fatalf("clone answered %d, %v", got, err)
	}
	after := e.Bytes()
	if len(after) != len(b)+70 {
		t.Fatalf("length %d, want %d", len(after), len(b)+70)
	}
	want := append([]byte(nil), b[p+70:p+140]...)
	binary.LittleEndian.PutUint32(want[0:], 9*256+7)
	binary.LittleEndian.PutUint32(want[4:], 11*256+9)
	binary.LittleEndian.PutUint16(want[0x40:], 8)
	if !bytes.Equal(after[p+140:p+210], want) {
		t.Fatalf("clone record % x\nwant % x", after[p+140:p+210], want)
	}
	head := append([]byte(nil), b[:p]...)
	binary.LittleEndian.PutUint32(head[p-12:], binary.LittleEndian.Uint32(b[p-12:])+70)
	binary.LittleEndian.PutUint32(head[40+0x24:], binary.LittleEndian.Uint32(b[40+0x24:])+1)
	if !bytes.Equal(after[:p], head) {
		t.Fatal("the clone changed more than the type-6 size word and the unit count")
	}
	m, err := alm.Open(after)
	if err != nil || len(m.Units) != 3 || m.Units[2].UnitID != 8 || m.Units[2].GroupID != 9 || m.Units[2].DefID != 1234 || m.Units[2].ClassID != 5 {
		t.Fatalf("decoded clone wrong: %v %+v", err, m.Units)
	}
	if !bytes.Equal(after[p+210:], b[p+140:]) || !bytes.Equal(after[p:p+140], b[p:p+140]) {
		t.Fatal("bytes outside the inserted record moved")
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) {
		t.Fatal("undo did not restore the exact bytes")
	}
	if !e.Redo() || !bytes.Equal(e.Bytes(), after) {
		t.Fatal("redo did not restore the exact clone")
	}
}

func TestCloneUnitRefusalWritesNothing(t *testing.T) {
	b, p := authoringFixture(t)
	binary.LittleEndian.PutUint16(b[p+0x40:], 0xFFFF)
	e := newEditor(t, b)
	for _, i := range []int{-1, 2} {
		if _, err := e.CloneUnit(i, 0, 0); err == nil {
			t.Fatalf("clone of index %d accepted", i)
		}
	}
	if _, err := e.CloneUnit(0, 0, 0); err == nil {
		t.Fatal("clone accepted with unit id 65535 taken")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused clone wrote or recorded")
	}
}

func TestSetUnitFieldWritesOnlyTheFieldBytes(t *testing.T) {
	cases := []struct {
		key  string
		v    int64
		off  int
		img  []byte
		view func(alm.Unit) bool
	}{
		{"owner", 3, 0x14, []byte{3, 0, 0, 0}, func(u alm.Unit) bool { return u.Owner == 3 }},
		{"class", -2, 0x08, []byte{0xfe, 0xff}, func(u alm.Unit) bool { return u.ClassID == -2 }},
		{"definition", 0x01020304, 0x10, []byte{4, 3, 2, 1}, func(u alm.Unit) bool { return u.DefID == 0x01020304 }},
		{"hp", -1, 0x20, []byte{0xff, 0xff}, func(u alm.Unit) bool { return !u.HasCurrentHP }},
		{"hp", 250, 0x20, []byte{250, 0}, func(u alm.Unit) bool { return u.HasCurrentHP && u.CurrentHP == 250 }},
	}
	for _, c := range cases {
		b, p := authoringFixture(t)
		e := newEditor(t, b)
		if err := e.SetUnitField(1, c.key, c.v); err != nil {
			t.Fatalf("%s: %v", c.key, err)
		}
		want := append([]byte(nil), b...)
		copy(want[p+70+c.off:], c.img)
		after := e.Bytes()
		if !bytes.Equal(after, want) {
			t.Fatalf("%s wrote outside its %d bytes", c.key, len(c.img))
		}
		if v, err := e.UnitFieldValue(1, c.key); err != nil || v != c.v {
			t.Fatalf("%s reads back %d, %v", c.key, v, err)
		}
		m, err := alm.Open(after)
		if err != nil || !c.view(m.Units[1]) {
			t.Fatalf("%s: decoded view %+v, %v", c.key, m.Units[1], err)
		}
		if !e.Undo() || !bytes.Equal(e.Bytes(), b) || !e.Redo() || !bytes.Equal(e.Bytes(), after) {
			t.Fatalf("%s: undo/redo not byte exact", c.key)
		}
	}
}

func TestSetUnitFieldRefusalWritesNothing(t *testing.T) {
	b, _ := authoringFixture(t)
	e := newEditor(t, b)
	for _, c := range []struct {
		i   int
		key string
		v   int64
	}{{1, "owner", 0}, {1, "owner", 1 << 32}, {1, "class", 32768}, {1, "hp", -32769}, {1, "definition", -1}, {1, "group", -1}, {1, "group", 1 << 32}, {2, "hp", 1}} {
		if err := e.SetUnitField(c.i, c.key, c.v); err == nil {
			t.Fatalf("unit %d %s=%d accepted", c.i, c.key, c.v)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused field write wrote or recorded")
	}
}

func TestUnitFieldTableNamesFieldsInsideTheRecord(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range mapedit.UnitFields() {
		if seen[f.Key] || f.Width != 2 && f.Width != 4 || f.Offset < 8 || f.Offset+f.Width > 70 || f.Min > f.Max {
			t.Fatalf("bad field row %+v", f)
		}
		seen[f.Key] = true
	}
	if len(seen) != 5 || !seen["group"] {
		t.Fatalf("table has %d fields", len(seen))
	}
}

// scriptedFixture is a three-unit map with ids 1, 2 and 3 whose type-7 script
// names unit 3 as a Target_Unit, a hero-band value 10001 as a Target_Unit and
// the value 50 as a plain integer.
func scriptedFixture(t *testing.T) []byte {
	t.Helper()
	node := make([]byte, 796)
	put := func(k int, v, ty uint32) {
		binary.LittleEndian.PutUint32(node[0x4c+4*k:], v)
		binary.LittleEndian.PutUint32(node[0x74+4*k:], ty)
	}
	put(0, 3, 4)
	put(1, 10001, 4)
	put(2, 50, 1)
	payload := binary.LittleEndian.AppendUint32(nil, 1)
	payload = append(payload, node...)
	payload = binary.LittleEndian.AppendUint32(payload, 0)
	payload = binary.LittleEndian.AppendUint32(payload, 0)
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40}, {X: 4*256 + 128, Y: 5*256 + 128, ClassID: 40}, {X: 6*256 + 128, Y: 7*256 + 128, ClassID: 40},
	}})
	p := typeSixPayload(t, b)
	for k := 0; k < 3; k++ {
		binary.LittleEndian.PutUint16(b[p+70*k+0x40:], uint16(k+1))
	}
	return b
}

func TestCloneUnitNeverTakesAnIDAScriptTargetUnitNames(t *testing.T) {
	e := newEditor(t, scriptedFixture(t))
	if err := e.DeleteUnit(2); err != nil {
		t.Fatal(err)
	}
	oldRule := uint16(3) // one above the highest type-6 id left, 2
	i, err := e.CloneUnit(0, 8*256, 9*256)
	if err != nil {
		t.Fatal(err)
	}
	m, err := e.Map()
	if err != nil || len(m.Units) != 3 {
		t.Fatalf("map after clone: %v, %d units", err, len(m.Units))
	}
	got := m.Units[i].UnitID
	if got == oldRule {
		t.Fatalf("clone took the deleted scripted unit's id %d", got)
	}
	if got != 4 {
		t.Fatalf("clone id %d, want 4 (one above the scripted id 3; the hero-band 10001 and the integer 50 do not count)", got)
	}
	for _, u := range m.Units[:2] {
		if u.UnitID == got {
			t.Fatalf("clone id %d equals a type-6 id", got)
		}
	}
}

func TestCloneUnitOnAScriptedMapWithNothingDeletedTakesTheNextID(t *testing.T) {
	b := scriptedFixture(t)
	e := newEditor(t, b)
	if i, err := e.CloneUnit(0, 8*256, 9*256); err != nil {
		t.Fatal(err)
	} else if m, _ := e.Map(); m.Units[i].UnitID != 4 {
		t.Fatalf("clone id %d, want 4", m.Units[i].UnitID)
	}
}
