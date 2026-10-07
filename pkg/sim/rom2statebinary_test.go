package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func rom2FormWorld(t *testing.T, empty bool) *World {
	t.Helper()
	var checks []ScriptCheck
	if !empty {
		checks = []ScriptCheck{constCheck(0, 37)}
	}
	s, err := newScript(ScriptROM2, checks, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := scriptWorld(t, s, nil)
	if w.rom2 == nil {
		t.Fatal("ROM2 world has no script state")
	}
	w.rom2.Scenario[0], w.rom2.Scenario[17], w.rom2.Scenario[1023] = -1, 0x12345678, -2147483648
	w.rom2.Groups = []rom2GroupActivity{{Owner: 0, Group: 3}, {Owner: 2, Group: 0, Forced: true}, {Owner: 2, Group: 9}}
	return w
}

func TestROM2StateFormRoundTrip(t *testing.T) {
	for _, empty := range []bool{false, true} {
		w := rom2FormWorld(t, empty)
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if raw[0] != 106 || string(raw[len(raw)-4:]) != "R2S1" {
			t.Fatal("ROM2 did not write its optional form")
		}
		if err := CheckSaveForm(raw); err != nil {
			t.Fatal(err)
		}
		span := binary.LittleEndian.Uint32(raw[len(raw)-9:])
		if span != 4100+3*9 {
			t.Fatalf("ROM2 span %d", span)
		}
		start := len(raw) - 9 - int(span)
		if binary.LittleEndian.Uint32(raw[start:]) != ^uint32(0) || binary.LittleEndian.Uint32(raw[start+17*4:]) != 0x12345678 || binary.LittleEndian.Uint32(raw[start+1023*4:]) != 0x80000000 || binary.LittleEndian.Uint32(raw[start+4096:]) != 3 {
			t.Fatal("scenario bank or group count changed wire positions")
		}
		wantRows := []byte{0, 0, 0, 0, 3, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1, 2, 0, 0, 0, 9, 0, 0, 0, 0}
		if !bytes.Equal(raw[start+4100:len(raw)-9], wantRows) {
			t.Fatal("group records differ from independent wire contract")
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		if cold.Script() == nil || cold.Script().Dialect() != ScriptROM2 || !reflect.DeepEqual(cold.rom2, w.rom2) || cold.Hash() != w.Hash() {
			t.Fatal("ROM2 state or dialect lost after restore")
		}
		cold.rom2.Scenario[17]++
		cold.rom2.Groups[0].Forced = true
		if w.rom2.Scenario[17] != 0x12345678 || w.rom2.Groups[0].Forced || cold.Hash() == w.Hash() {
			t.Fatal("restored state aliases the source or is absent from the hash")
		}
	}
}

func TestROM2EmptyFormKeepsDialect(t *testing.T) {
	s, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := scriptWorld(t, s, nil)
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != 106 {
		t.Fatal("an empty ROM2 program lost its dialect", err)
	}
	if binary.LittleEndian.Uint32(raw[len(raw)-9:]) != 4100 {
		t.Fatal("empty ROM2 state does not have the fixed-width footer")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Script() == nil || cold.Script().Dialect() != ScriptROM2 || cold.rom2 == nil || cold.Hash() != w.Hash() {
		t.Fatal("empty ROM2 failed cold restore", err)
	}
}

func TestROM2FormPreservesROM1LegacyIdentity(t *testing.T) {
	w := scriptWorld(t, nil, nil)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	wantHash := w.Hash()
	for _, s := range []*Script{nil, {}, mustScript(t, nil, nil, nil)} {
		legacy := scriptWorld(t, s, nil)
		got, err := legacy.MarshalBinary()
		if err != nil || legacy.rom2 != nil || got[0] != raw[0] || !bytes.Equal(got, raw) || legacy.Hash() != wantHash {
			t.Fatal("legacy empty script literal changed its form or hash", err)
		}
	}
	s := mustScript(t, []ScriptCheck{constCheck(0, 37)}, nil, nil)
	legacy := scriptWorld(t, s, nil)
	raw, err = legacy.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.rom2 != nil || cold.Script() == nil || cold.Script().Dialect() != ScriptROM1 || cold.Hash() != legacy.Hash() {
		t.Fatal("historical program did not default to ROM1", err)
	}
}

func TestROM2FormWrapsNativeTraining(t *testing.T) {
	w := trainingWorld(t, 40, -100)
	w.DeclareStructures([]Structure{{ID: 1, Col: 6, Row: 6, Width: 1, Height: 1, Attach: 1, Blocking: 2}})
	base, err := w.MarshalBinary()
	if err != nil || base[0] != 105 || !HasStructureBlockingForm(base) {
		t.Fatal("training fixture is not form 105", err)
	}
	s, err := newScript(ScriptROM2, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.script, w.rom2 = s, &rom2ScriptState{}
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != 106 || raw[len(raw)-5] != 105 {
		t.Fatal("ROM2 did not wrap native training", err)
	}
	if !HasStructureBlockingForm(raw) {
		t.Fatal("ROM2 outer form hid the structure blocking continuation")
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	peeled := bytes.Clone(raw[:start])
	peeled[0] = raw[len(raw)-5]
	if !bytes.Equal(peeled, base) {
		t.Fatal("independent ROM2 peel changed predecessor bytes")
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || cold.entities[0].NativeTraining != w.entities[0].NativeTraining || cold.structures[0].Blocking != 2 {
		t.Fatal("combined native training and ROM2 cold restore failed", err)
	}
}

func TestROM2FormRestoresDialectExecution(t *testing.T) {
	for _, dialect := range []ScriptDialect{ScriptROM1, ScriptROM2} {
		s, err := newScript(dialect,
			[]ScriptCheck{{Op: 23, Register: 0, Args: [scriptParams]int32{17}}, constCheck(1, 1)},
			[]ScriptInstant{{Op: 35, Args: [scriptParams]int32{2, 88}}},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true, Latch: 0}})
		if err != nil {
			t.Fatal(err)
		}
		w := scriptWorld(t, s, nil)
		if dialect == ScriptROM2 {
			w.rom2.Scenario[17] = 1
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Script().Dialect() != dialect {
			t.Fatal("script dialect lost before execution", err)
		}
		if got := cold.Script().Triggers()[0].Inert; got != (dialect == ScriptROM1) {
			t.Fatalf("dialect %d restored trigger inert=%v", dialect, got)
		}
		scriptTicks(w, 7, nil)
		scriptTicks(&cold, 7, nil)
		if cold.Hash() != w.Hash() {
			t.Fatalf("dialect %d changed post-load execution", dialect)
		}
		if dialect == ScriptROM2 {
			if cold.rom2.Scenario[2] != 88 || cold.ScriptRegister(0) != 1 || !cold.ScriptLatched(0) {
				t.Fatal("restored ROM2-only check or instant stayed inert")
			}
		} else if cold.rom2 != nil || cold.ScriptRegister(0) != 0 || cold.ScriptLatched(0) {
			t.Fatal("ROM1 executed a ROM2-only arm")
		}
	}
}

func TestROM2FormRejectsInvalidWorldState(t *testing.T) {
	cases := map[string]func(*World){
		"missing state":   func(w *World) { w.rom2 = nil },
		"missing script":  func(w *World) { w.script = nil },
		"ROM1 script":     func(w *World) { w.script = &Script{} },
		"invalid dialect": func(w *World) { w.script = &Script{dialect: 99} },
		"count limit":     func(w *World) { w.rom2.Groups = make([]rom2GroupActivity, 65536) },
		"duplicate":       func(w *World) { w.rom2.Groups[1] = w.rom2.Groups[0] },
		"owner order":     func(w *World) { w.rom2.Groups[0].Owner = 3 },
		"group order":     func(w *World) { w.rom2.Groups[1].Group = 10 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			w := rom2FormWorld(t, false)
			mutate(w)
			if _, err := w.MarshalBinary(); err == nil {
				t.Fatal("invalid dialect or group population was encoded")
			}
		})
	}
}

func TestROM2FormCorruptionIsAtomic(t *testing.T) {
	w := rom2FormWorld(t, false)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if raw[0] != 106 || len(raw) < 4109 {
		t.Fatal("ROM2 footer absent from corruption fixture")
	}
	start := len(raw) - 9 - int(binary.LittleEndian.Uint32(raw[len(raw)-9:]))
	cases := map[string]func([]byte) []byte{
		"truncated":   func(b []byte) []byte { return b[:len(b)-1] },
		"magic":       func(b []byte) []byte { b[len(b)-1] = 0; return b },
		"same base":   func(b []byte) []byte { b[len(b)-5] = 106; return b },
		"future base": func(b []byte) []byte { b[len(b)-5] = 107; return b },
		"short span":  func(b []byte) []byte { binary.LittleEndian.PutUint32(b[len(b)-9:], 4099); return b },
		"long span":   func(b []byte) []byte { binary.LittleEndian.PutUint32(b[len(b)-9:], ^uint32(0)); return b },
		"count":       func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+4096:], ^uint32(0)); return b },
		"small count": func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+4096:], 2); return b },
		"flag":        func(b []byte) []byte { b[start+4108] = 2; return b },
		"duplicate":   func(b []byte) []byte { copy(b[start+4109:start+4118], b[start+4100:start+4109]); return b },
		"reverse":     func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+4100:], 3); return b },
		"group order": func(b []byte) []byte { binary.LittleEndian.PutUint32(b[start+4113:], 10); return b },
		"bad base":    func(b []byte) []byte { b[29] = 255; return b },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			bad := mutate(bytes.Clone(raw))
			receiver := rom2FormWorld(t, true)
			before, err := receiver.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if err := receiver.UnmarshalBinary(bad); err == nil {
				t.Fatal("malformed ROM2 state was accepted")
			}
			after, err := receiver.MarshalBinary()
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("failed restore changed the receiver", err)
			}
			if err := CheckSaveForm(bad); err == nil {
				t.Fatal("load eligibility accepted malformed ROM2 state")
			}
		})
	}
}
