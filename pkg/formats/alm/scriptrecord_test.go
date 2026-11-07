package alm_test

// The extended script record. Every fixture is assembled here from the
// documented layout; nothing reads a file.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
)

// recordFixture is two triggers over the record's whole grammar: two groups
// of three conditions and six actions, every comparison code, then a trigger
// with neither conditions nor actions.
func recordFixture() alm.ScriptRecord {
	return alm.ScriptRecord{Variables: 150, Triggers: []alm.ScriptTrigger{
		{Name: "Either", Once: 1, Group: []uint32{0, 0, 0, 1, 1, 1},
			Left: []uint32{1, 2, 3, 4, 5, 6}, Right: []uint32{6, 5, 4, 3, 2, 1}, Cmp: []uint32{0, 1, 2, 3, 4, 5},
			Acts: []uint32{1, 2, 3, 4, 5, 5000}},
		{Name: "Empty"},
	}}
}

// sameRecordTriggers compares triggers field by field, an empty list equal
// to a nil one.
func sameRecordTriggers(a, b []alm.ScriptTrigger) bool {
	return slices.EqualFunc(a, b, func(x, y alm.ScriptTrigger) bool {
		return x.Name == y.Name && x.Once == y.Once && slices.Equal(x.Group, y.Group) && slices.Equal(x.Left, y.Left) &&
			slices.Equal(x.Right, y.Right) && slices.Equal(x.Cmp, y.Cmp) && slices.Equal(x.Acts, y.Acts)
	})
}

func encodeRecord(t *testing.T, r alm.ScriptRecord) []byte {
	t.Helper()
	b, err := alm.EncodeScriptRecord(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestScriptRecordLayoutAndRoundTrip(t *testing.T) {
	r := recordFixture()
	b := encodeRecord(t, r)
	le := binary.LittleEndian
	if len(b) != 12+2*76+6*16+6*4 {
		t.Fatalf("payload is %d bytes", len(b))
	}
	if le.Uint32(b) != 1 || le.Uint32(b[4:]) != 150 || le.Uint32(b[8:]) != 2 {
		t.Fatal("head words", b[:12])
	}
	first := b[12:]
	if !bytes.HasPrefix(first, []byte("Either\x00")) || le.Uint32(first[0x40:]) != 1 || le.Uint32(first[0x44:]) != 6 || le.Uint32(first[0x48:]) != 6 {
		t.Fatal("first trigger head", first[:76])
	}
	// The fourth condition opens group 1: group, left, right, comparison code.
	cond := first[76+3*16:]
	if le.Uint32(cond) != 1 || le.Uint32(cond[4:]) != 4 || le.Uint32(cond[8:]) != 3 || le.Uint32(cond[12:]) != 3 {
		t.Fatal("fourth condition", cond[:16])
	}
	if acts := first[76+6*16:]; le.Uint32(acts) != 1 || le.Uint32(acts[20:]) != 5000 {
		t.Fatal("action ids", acts[:24])
	}
	got, err := alm.DecodeScriptRecord(b)
	if err != nil || got.Variables != 150 || !sameRecordTriggers(got.Triggers, r.Triggers) {
		t.Fatalf("decode: %v %+v", err, got)
	}
	if again := encodeRecord(t, got); !bytes.Equal(again, b) {
		t.Fatal("re-encode differs")
	}
}

// recordMap is a 2x2 map whose type-7 record holds two condition nodes and
// one action node and no trigger.
func recordMap(trgs ...[]byte) []byte {
	sec := baseSections(2, 2, 0, 0, 0)
	sec[7] = scriptPayload([][]byte{scriptNode("Act", 3, 1, [10]uint32{120, 1}, [10]uint32{})},
		[][]byte{scriptNode("A", 19, 1, [10]uint32{120}, [10]uint32{}), scriptNode("B", 0x10002, 2, [10]uint32{1}, [10]uint32{})}, trgs)
	return buildMap(sec)
}

func TestMapTakesItsTriggersFromTheScriptRecord(t *testing.T) {
	plain := recordMap()
	m := openOK(t, plain)
	s, err := m.Script()
	if err != nil || s.Extended || m.HasScriptRecord() || len(s.Conditions) != 2 {
		t.Fatal("a map without the record", err, s.Extended)
	}
	stream, err := alm.AppendScriptRecord(plain, recordFixture())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stream[20:len(plain)], plain[20:]) {
		t.Fatal("the original records moved")
	}
	m = openOK(t, stream)
	if m.RecordCount != 11 || len(m.Records) != 11 || m.Records[10].TypeID != alm.ScriptRecordTypeID || !m.HasScriptRecord() {
		t.Fatalf("record frame: count %d, %d records", m.RecordCount, len(m.Records))
	}
	s, err = m.Script()
	if err != nil || !s.Extended || s.Variables != 150 || len(s.Actions) != 1 || len(s.Conditions) != 2 ||
		!sameRecordTriggers(s.Triggers, recordFixture().Triggers) {
		t.Fatalf("extended script: %v %+v", err, s)
	}
	if _, err := alm.AppendScriptRecord(stream, recordFixture()); err == nil {
		t.Fatal("a second record was appended")
	}
}

func TestScriptRecordBesideTypeSevenTriggersOrTwiceIsRefused(t *testing.T) {
	payload := encodeRecord(t, recordFixture())
	beside, err := alm.AppendScriptRecord(recordMap(scriptTrigger("T", [3]uint32{1}, [3]uint32{2}, [3]uint32{}, [4]uint32{1}, 0)), recordFixture())
	if err != nil {
		t.Fatal(err)
	}
	twice := slices.Concat(beside, rec(alm.ScriptRecordTypeID, payload))
	binary.LittleEndian.PutUint32(twice[12:], 12)
	for name, stream := range map[string][]byte{"type-7 trigger count": beside, "records": twice} {
		_, err := openOK(t, stream).Script()
		var re *alm.ScriptRecordError
		if !errors.As(err, &re) || re.Field != name {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestScriptRecordRefusesWhatTheMachineCannotHold(t *testing.T) {
	put := func(off int, v uint32) func([]byte) []byte {
		return func(b []byte) []byte { binary.LittleEndian.PutUint32(b[off:], v); return b }
	}
	const trig, cond = 12, 12 + 76
	acts := cond + 6*16
	for _, tc := range []struct {
		field string
		edit  func([]byte) []byte
	}{
		{"version", put(0, 2)},
		{"variables", put(4, 4097)},
		{"triggers", put(8, 10001)},
		{"fire-once word", put(trig+0x40, 2)},
		{"conditions", put(trig+0x44, 65)},
		{"actions", put(trig+0x48, 65)},
		{"group", put(cond, 1)},
		{"group", put(cond+3*16, 2)},
		{"group", put(cond+5*16, 0)},
		{"condition id", put(cond+4, 0)},
		{"condition id", put(cond+8, 5001)},
		{"comparison code", put(cond+12, 6)},
		{"action id", put(acts, 0)},
		{"action id", put(acts+4, 5001)},
		{"trailing", func(b []byte) []byte { return append(b, 0) }},
		{"length", func(b []byte) []byte { return b[:len(b)-1] }},
	} {
		b := tc.edit(encodeRecord(t, recordFixture()))
		_, err := alm.DecodeScriptRecord(b)
		var re *alm.ScriptRecordError
		if !errors.As(err, &re) || re.Field != tc.field {
			t.Fatalf("%s: %v", tc.field, err)
		}
	}
	// The encoder refuses the same values.
	for _, edit := range []func(*alm.ScriptRecord){
		func(r *alm.ScriptRecord) { r.Variables = 4097 },
		func(r *alm.ScriptRecord) { r.Triggers[0].Group[0] = 1 },
		func(r *alm.ScriptRecord) { r.Triggers[0].Cmp[0] = 6 },
		func(r *alm.ScriptRecord) { r.Triggers[0].Acts[0] = 0 },
		func(r *alm.ScriptRecord) { r.Triggers[0].Once = 2 },
		func(r *alm.ScriptRecord) { r.Triggers[0].Left = make([]uint32, 65) },
	} {
		r := recordFixture()
		edit(&r)
		var re *alm.ScriptRecordError
		if _, err := alm.EncodeScriptRecord(r); !errors.As(err, &re) {
			t.Fatalf("encoder accepted %+v: %v", r, err)
		}
	}
}

// Every cut of a valid payload and every byte forced high decodes to a typed
// refusal or a record, never a panic; a count past the bytes is refused
// before anything is allocated for it.
func TestScriptRecordTruncatedAndOversizedCounts(t *testing.T) {
	b := encodeRecord(t, recordFixture())
	for n := range len(b) {
		_, err := alm.DecodeScriptRecord(b[:n])
		var re *alm.ScriptRecordError
		if !errors.As(err, &re) {
			t.Fatalf("cut at %d: %v", n, err)
		}
	}
	for i := range b {
		for _, v := range []byte{0xff, 0x80, 0x01} {
			c := slices.Clone(b)
			c[i] = v
			if _, err := alm.DecodeScriptRecord(c); err != nil {
				var re *alm.ScriptRecordError
				if !errors.As(err, &re) {
					t.Fatalf("byte %d = %#x: untyped %v", i, v, err)
				}
			}
		}
	}
	le := binary.LittleEndian
	head := func(vars, n uint32) []byte {
		return slices.Concat(le.AppendUint32(nil, 1), le.AppendUint32(nil, vars), le.AppendUint32(nil, n))
	}
	trigger := func(conds, acts uint32) []byte {
		h := make([]byte, 76)
		le.PutUint32(h[0x44:], conds)
		le.PutUint32(h[0x48:], acts)
		return h
	}
	for _, tc := range []struct {
		field   string
		payload []byte
	}{
		{"variables", head(0xffffffff, 0)},
		{"triggers", head(0, 0xffffffff)},
		{"length", head(0, 10000)},
		{"conditions", slices.Concat(head(0, 1), trigger(0xffffffff, 0))},
		{"actions", slices.Concat(head(0, 1), trigger(0, 0xffffffff))},
		{"length", slices.Concat(head(0, 1), trigger(64, 64))},
	} {
		_, err := alm.DecodeScriptRecord(tc.payload)
		var re *alm.ScriptRecordError
		if !errors.As(err, &re) || re.Field != tc.field {
			t.Fatalf("%s: %v", tc.field, err)
		}
	}
	if r, err := alm.DecodeScriptRecord(head(4096, 0)); err != nil || r.Variables != 4096 || len(r.Triggers) != 0 {
		t.Fatal("the variable limit itself", err)
	}
}
