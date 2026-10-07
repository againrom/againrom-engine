package sav

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestNativeActionRegistryFramingIsStrict(t *testing.T) {
	var state DocumentStateData
	for _, value := range [][]byte{[]byte("x"), []byte("typed-current-actions"), bytes.Repeat([]byte{7}, 128)} {
		if err := SetNativeActions(&state, value); err != nil {
			t.Fatal(err)
		}
		back, ok, err := NativeActions(state)
		if err != nil || !ok || !bytes.Equal(value, back) {
			t.Fatal("registry round trip", err)
		}
	}
	for _, which := range []string{"case", "duplicate", "version", "length", "padding", "kind"} {
		var s DocumentStateData
		_ = SetNativeActions(&s, []byte("x"))
		r := &s.ValueRecords[0]
		switch which {
		case "case":
			r.Path = "/CurrentState/againromActions"
		case "duplicate":
			s.ValueRecords = append(s.ValueRecords, *r)
		case "version":
			r.Value.Bytes[0] = 2
		case "length":
			r.Value.Bytes[4] = 255
		case "padding":
			r.Value.Bytes[11] = 1
		case "kind":
			r.Value.Kind = 0
		}
		if _, _, err := NativeActions(s); err == nil {
			t.Fatalf("accepted malformed %s", which)
		}
	}
}

func requireActionPayload(t *testing.T, state DocumentStateData, want string) {
	t.Helper()
	data, present, err := NativeActions(state)
	if err != nil || !present {
		t.Fatal("missing continuation", err)
	}
	var expected, actual any
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &actual); err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fatalf("current action addresses or opaque values changed: %s; %v", data, err)
	}
}

func requireCurrentActionReindex(t *testing.T, source DocumentData, oldActor, newActor uint16) {
	t.Helper()
	input := source
	payload := fmt.Sprintf(`{"Version":1,"Bindings":[{"ID":41,"Object":%d,"Structure":false,"Missing":false},{"ID":99,"Object":%d,"Structure":false,"Missing":false},{"ID":5,"Object":0,"Structure":true,"Missing":true}],"Objects":[{"ID":123,"Object":%d}],"Actions":{"Progress":17,"Target":99},"VisualNext":789}`, oldActor, newActor, oldActor+1)
	if err := SetNativeActions(&input.State, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(input)
	out, _, err := ReindexDocumentData(input)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`{"Version":1,"Bindings":[{"ID":41,"Object":3,"Structure":false,"Missing":false},{"ID":99,"Object":2,"Structure":false,"Missing":false},{"ID":5,"Object":0,"Structure":true,"Missing":true}],"Objects":[{"ID":123,"Object":%d}],"Actions":{"Progress":17,"Target":99},"VisualNext":789}`, oldActor+2)
	requireActionPayload(t, out.State, want)
	raw, err := EncodeDocumentData(out)
	if err != nil {
		t.Fatal(err)
	}
	cold, err := DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := ReindexDocumentData(cold)
	if err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, second.State, want)
	after, _ := json.Marshal(input)
	if !bytes.Equal(before, after) {
		t.Fatal("action reindex changed input")
	}
}

func requireCurrentActionRetirement(t *testing.T, source DocumentData) {
	t.Helper()
	input := source
	payload := `{"Version":1,"Bindings":[{"ID":20,"Object":2,"Structure":true,"Missing":false},{"ID":60,"Object":6,"Structure":true,"Missing":false}],"Objects":[{"ID":30,"Object":3},{"ID":40,"Object":4},{"ID":50,"Object":5}],"Actions":{"Reservations":[30,40,50],"Progress":17}}`
	if err := SetNativeActions(&input.State, []byte(payload)); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(input)
	out, _, err := RetireDocumentData(input, []uint16{2, 3})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"Version":1,"Bindings":[{"ID":20,"Object":0,"Structure":true,"Missing":true},{"ID":60,"Object":2,"Structure":true,"Missing":false}],"Objects":[{"ID":40,"Object":4},{"ID":50,"Object":5}],"Actions":{"Reservations":[30,40,50],"Progress":17}}`
	requireActionPayload(t, out.State, want)
	second, _, err := RetireDocumentData(out, nil)
	if err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, second.State, want)
	after, _ := json.Marshal(input)
	if !bytes.Equal(before, after) {
		t.Fatal("action retirement changed input")
	}
}

func TestCurrentActionRemapRejectsUnresolvedAddressesAtomically(t *testing.T) {
	for _, payload := range []string{
		`{"Version":1,"Bindings":[{"ID":1,"Object":9,"Missing":false}]}`,
		`{"Version":1,"Bindings":[{"ID":1,"Object":1,"Missing":true}]}`,
		`{"Version":1,"Bindings":[{"ID":1,"Object":0,"Missing":false}]}`,
		`{"Version":1,"Bindings":[{"ID":1,"Object":1.5,"Missing":false}]}`,
		`{"Version":1,"Objects":[{"ID":1,"Object":9}]}`,
		`{"Version":1,"Objects":[{"ID":1,"Object":0}]}`,
		`{"Version":1,"Groups":[{"ID":1,"Object":0}]}`,
		`{"Version":1,"SpellCasters":[{"Object":1,"Caster":9}]}`,
		`{"Version":2}`,
	} {
		var state DocumentStateData
		if err := SetNativeActions(&state, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(state)
		if err := remapNativeActionObjects(&state, []uint16{0, 1}); err == nil {
			t.Fatalf("accepted malformed continuation %s", payload)
		}
		after, _ := json.Marshal(state)
		if !bytes.Equal(before, after) {
			t.Fatal("failed action remap changed input")
		}
	}
}

func TestCurrentGroupAndSpellCasterArchiveRelocation(t *testing.T) {
	var state DocumentStateData
	if err := SetNativeActions(&state, []byte(`{"Version":1,"Groups":[{"Object":1,"Inline":3,"ID":99,"Authored":true}],"SpellCasters":[{"Object":2,"Caster":3},{"Object":4,"Caster":3}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3, 1, 2, 0}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"Groups":[{"Object":3,"Inline":3,"ID":99,"Authored":true}],"SpellCasters":[{"Object":1,"Caster":2}]}`)
	if err := remapNativeActionObjects(&state, []uint16{0, 1, 0, 3}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"Groups":[{"Object":3,"Inline":3,"ID":99,"Authored":true}],"SpellCasters":[{"Object":1,"Caster":0}]}`)
}

func TestCurrentPlayerSlotArchiveRelocation(t *testing.T) {
	var state DocumentStateData
	if err := SetNativeActions(&state, []byte(`{"Version":1,"PlayerSlots":[{"Object":1,"Wire":2,"Native":0}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"PlayerSlots":[{"Object":3,"Wire":2,"Native":0}]}`)
	before, _ := json.Marshal(state)
	if err := remapNativeActionObjects(&state, []uint16{0, 1, 2, 0}); err == nil {
		t.Fatal("retired Player kept a native slot policy")
	}
	after, _ := json.Marshal(state)
	if !bytes.Equal(before, after) {
		t.Fatal("failed Player relocation changed input")
	}
}

func TestCurrentAbsentPlayerArchiveRelocationIsAtomic(t *testing.T) {
	var state DocumentStateData
	if err := SetNativeActions(&state, []byte(`{"Version":1,"AbsentPlayers":[{"Object":1,"Anchor":[19,23]}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"AbsentPlayers":[{"Object":3,"Anchor":[19,23]}]}`)
	before, _ := json.Marshal(state)
	if err := remapNativeActionObjects(&state, []uint16{0, 1, 2, 0}); err == nil {
		t.Fatal("retired Player retained an absence binding")
	}
	after, _ := json.Marshal(state)
	if !bytes.Equal(before, after) {
		t.Fatal("rejected Player relocation changed continuation")
	}
}

func TestCurrentAbsentDiaryArchiveRelocationAndRetirement(t *testing.T) {
	var state DocumentStateData
	if err := SetNativeActions(&state, []byte(`{"Version":1,"AbsentDiaries":[{"Object":1,"Inline":0,"Owner":{"Player":false,"Actor":17},"Anchor":[1,2]},{"Object":2,"Inline":3,"Owner":{"Player":true,"Actor":0},"Anchor":[3,4]}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3, 1}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"AbsentDiaries":[{"Object":3,"Inline":0,"Owner":{"Player":false,"Actor":17},"Anchor":[1,2]},{"Object":1,"Inline":3,"Owner":{"Player":true,"Actor":0},"Anchor":[3,4]}]}`)
	if err := remapNativeActionObjects(&state, []uint16{0, 2, 1, 0}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"AbsentDiaries":[{"Object":2,"Inline":3,"Owner":{"Player":true,"Actor":0},"Anchor":[3,4]}]}`)
}

func TestCurrentArchiveCoordinateRelocationAndDetachment(t *testing.T) {
	var state DocumentStateData
	if err := SetNativeActions(&state, []byte(`{"Version":1,"ArchiveCoordinates":[{"Object":1,"Native":71},{"Object":2,"Native":0}],"ActorGroups":[{"Object":1,"Player":2,"Inline":3,"Native":29,"Anchor":[4,5]}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 3, 1}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"ArchiveCoordinates":[{"Object":3,"Native":71},{"Object":1,"Native":0}],"ActorGroups":[{"Object":3,"Player":1,"Inline":3,"Native":29,"Anchor":[4,5]}]}`)
	if err := remapNativeActionObjects(&state, []uint16{0, 0, 2, 1}); err != nil {
		t.Fatal(err)
	}
	requireActionPayload(t, state, `{"Version":1,"ArchiveCoordinates":[{"Object":1,"Native":71}],"ActorGroups":[]}`)
}
