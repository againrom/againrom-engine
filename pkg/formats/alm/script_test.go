package alm_test

// The type-7 leaf grammar. Every fixture is a synthetic byte stream assembled
// here from the documented record layout; nothing reads a file.

import (
	"testing"

	"againrom/pkg/formats/alm"
)

// scriptNode builds one 796-byte type-7 node record.
func scriptNode(label string, opcode, id uint32, vals, types [10]uint32) []byte {
	r := make([]byte, 796)
	copy(r[0x000:], label)
	copy(r[0x040:], le32(opcode))
	copy(r[0x044:], le32(id))
	for i := 0; i < 10; i++ {
		copy(r[0x04c+4*i:], le32(vals[i]))
		copy(r[0x074+4*i:], le32(types[i]))
		// The ten 64-byte parameter names at +0x09c: filled with something this
		// decoder must not read, so a decoder that read them would be caught.
		copy(r[0x09c+64*i:], "Par")
	}
	return r
}

// scriptTrigger builds one 184-byte type-7 trigger record.
func scriptTrigger(name string, left, right, cmp [3]uint32, acts [4]uint32, once uint32) []byte {
	r := make([]byte, 184)
	copy(r[0x00:], name)
	// The 64 bytes at +0x40 are editor heap addresses and are read by nothing.
	for i := 0x40; i < 0x80; i++ {
		r[i] = 0xEE
	}
	for i := 0; i < 3; i++ {
		copy(r[0x80+8*i:], le32(left[i]))
		copy(r[0x84+8*i:], le32(right[i]))
		copy(r[0xa8+4*i:], le32(cmp[i]))
	}
	for i := 0; i < 4; i++ {
		copy(r[0x98+4*i:], le32(acts[i]))
	}
	copy(r[0xb4:], le32(once))
	return r
}

// scriptPayload assembles a whole type-7 payload — the three counted arrays —
// including the leading count word alm.go reads as Triggers.EntryCount.
func scriptPayload(acts, conds, trgs [][]byte) []byte {
	out := le32(uint32(len(acts)))
	for _, a := range acts {
		out = concat(out, a)
	}
	out = concat(out, le32(uint32(len(conds))))
	for _, c := range conds {
		out = concat(out, c)
	}
	out = concat(out, le32(uint32(len(trgs))))
	for _, t := range trgs {
		out = concat(out, t)
	}
	return out
}

// mapWithScript opens a 2x2 map whose type-7 payload is p.
func mapWithScript(t *testing.T, p []byte) *alm.Map {
	t.Helper()
	sec := baseSections(2, 2, 0, 0, 0)
	sec[7] = p
	return openOK(t, buildMap(sec))
}

// ---------------------------------------------------------------------------
// AC-1 — the three arrays decode, field for field, and the walk is exact.
// ---------------------------------------------------------------------------

func TestScriptDecodesThreeCountedArrays(t *testing.T) {
	vals := [10]uint32{11, 22, 33, 0, 0, 0, 0, 0, 0, 99}
	types := [10]uint32{1, 5, 6, 0, 0, 0, 0, 0, 0, 2}

	acts := [][]byte{
		scriptNode("Send message", 2, 7, [10]uint32{15}, [10]uint32{1}),
		scriptNode("Patrol", 6, 9, vals, types),
	}
	conds := [][]byte{
		scriptNode("near hut", 7, 4, [10]uint32{36, 51}, [10]uint32{5, 6}),
	}
	trgs := [][]byte{
		scriptTrigger("Snoot",
			[3]uint32{4, 0, 0}, [3]uint32{12, 0, 0}, [3]uint32{5, 0, 0},
			[4]uint32{7, 9, 0, 0}, 1),
	}

	m := mapWithScript(t, scriptPayload(acts, conds, trgs))
	s, err := m.Script()
	if err != nil {
		t.Fatalf("Script() = %v, want a decoded script", err)
	}
	if s.Empty() {
		t.Fatalf("Script() reports empty on a payload carrying 2/1/1")
	}
	if len(s.Actions) != 2 || len(s.Conditions) != 1 || len(s.Triggers) != 1 {
		t.Fatalf("arrays are %d/%d/%d, want 2/1/1",
			len(s.Actions), len(s.Conditions), len(s.Triggers))
	}

	a := s.Actions[1]
	if a.Label != "Patrol" || a.Opcode != 6 || a.ID != 9 {
		t.Errorf("action[1] = {%q, op %d, id %d}, want {\"Patrol\", op 6, id 9}", a.Label, a.Opcode, a.ID)
	}
	if a.Value != vals || a.Type != types {
		t.Errorf("action[1] value/type = %v/%v, want %v/%v", a.Value, a.Type, vals, types)
	}
	// The parameter slots are read BY SLOT: value[9] must be 99 and not
	// compacted into an earlier slot, which is the shape a packed reading gets
	// wrong on 804 shipped slots.
	if a.Value[9] != 99 || a.Value[3] != 0 {
		t.Errorf("action[1] parameters were packed: value = %v", a.Value)
	}

	c := s.Conditions[0]
	if c.Label != "near hut" || c.Opcode != 7 || c.ID != 4 ||
		c.Value[0] != 36 || c.Value[1] != 51 || c.Type[0] != 5 || c.Type[1] != 6 {
		t.Errorf("condition[0] = %+v, want the (7, id 4, 36/51, types 5/6) record", c)
	}

	g := s.Triggers[0]
	if g.Name != "Snoot" || g.Once != 1 {
		t.Errorf("trigger[0] = {%q, once %d}, want {\"Snoot\", once 1}", g.Name, g.Once)
	}
	if g.Left != ([3]uint32{4, 0, 0}) || g.Right != ([3]uint32{12, 0, 0}) ||
		g.Cmp != ([3]uint32{5, 0, 0}) || g.Acts != ([4]uint32{7, 9, 0, 0}) {
		t.Errorf("trigger[0] bindings = %v %v %v %v, want {4,0,0} {12,0,0} {5,0,0} {7,9,0,0}",
			g.Left, g.Right, g.Cmp, g.Acts)
	}
}

// ---------------------------------------------------------------------------
// AC-1 — an absent record is an empty script and not an error; a present but
// empty one likewise, and Present tells the two apart.
// ---------------------------------------------------------------------------

func TestScriptAbsentAndEmptyRecords(t *testing.T) {
	t.Run("present but empty", func(t *testing.T) {
		m := mapWithScript(t, scriptPayload(nil, nil, nil))
		s, err := m.Script()
		if err != nil || !s.Empty() {
			t.Fatalf("Script() = %v, %v; want an empty script and no error", s, err)
		}
		if !m.Present(7) {
			t.Errorf("Present(7) is false on a map that carried an empty type7 record")
		}
	})

	t.Run("absent", func(t *testing.T) {
		sec := baseSections(2, 2, 0, 0, 0)
		delete(sec, 7)
		m := openOK(t, buildPartial(sec, 0, 1, 2, 3, 5, 4, 9, 8, 6))
		s, err := m.Script()
		if err != nil || !s.Empty() {
			t.Fatalf("Script() = %v, %v; want an empty script and no error", s, err)
		}
		if m.Present(7) {
			t.Errorf("Present(7) is true on a map with no type7 record")
		}
	})
}

// ---------------------------------------------------------------------------
// AC-1 — the walk must consume the payload EXACTLY. A short walk means the
// model is wrong, not that the map has a tail.
// ---------------------------------------------------------------------------

func TestScriptWalkIsExact(t *testing.T) {
	one := scriptNode("a", 1, 1, [10]uint32{}, [10]uint32{})

	tests := []struct {
		name    string
		payload []byte
	}{
		{"trailing byte", concat(scriptPayload([][]byte{one}, nil, nil), []byte{0x00})},
		{"node count one too high", concat(le32(2), one, le32(0), le32(0))},
		{"missing the condition count word", concat(le32(1), one)},
		{"trigger record truncated", concat(le32(0), le32(0), le32(1),
			make([]byte, 183))},
		{"count near the top of its width", concat(le32(0xFFFFFFF0), le32(0), le32(0))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mapWithScript(t, tc.payload)
			if s, err := m.Script(); err == nil {
				t.Errorf("Script() accepted %s: %d/%d/%d nodes",
					tc.name, len(s.Actions), len(s.Conditions), len(s.Triggers))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-1 — decoding the leaf grammar does not disturb the raw body, so a map that
// round-trips through Write is unaffected by anything in script.go.
// ---------------------------------------------------------------------------

func TestScriptLeavesTheRawBodyAlone(t *testing.T) {
	p := scriptPayload(
		[][]byte{scriptNode("a", 1, 1, [10]uint32{}, [10]uint32{})}, nil, nil)
	m := mapWithScript(t, p)

	before := append([]byte(nil), m.Triggers.Body...)
	count := m.Triggers.EntryCount
	if _, err := m.Script(); err != nil {
		t.Fatalf("Script() = %v", err)
	}
	if m.Triggers.EntryCount != count {
		t.Errorf("Triggers.EntryCount moved from %d to %d", count, m.Triggers.EntryCount)
	}
	if string(m.Triggers.Body) != string(before) {
		t.Errorf("Triggers.Body changed under Script()")
	}
}

// ---------------------------------------------------------------------------
// AC-2 — the type-6 record's two identifier words are exposed, at the two
// offsets and the two widths the corrected reading names.
// ---------------------------------------------------------------------------

func TestUnitIdentifierWords(t *testing.T) {
	r := type6Record(0x0180, 0x0280, 0, 0, 0, 0)
	copy(r[0x40:], le16(21))
	copy(r[0x42:], le32(0x00010002))

	sec := baseSections(2, 2, 0, 0, 1)
	sec[6] = r
	m := openOK(t, buildMap(sec))

	if len(m.Units) != 1 {
		t.Fatalf("decoded %d units, want 1", len(m.Units))
	}
	if got := m.Units[0].UnitID; got != 21 {
		t.Errorf("UnitID = %d, want 21 (+0x40, u16)", got)
	}
	if got := m.Units[0].GroupID; got != 0x00010002 {
		t.Errorf("GroupID = %#x, want 0x00010002 (+0x42, u32)", got)
	}
}
