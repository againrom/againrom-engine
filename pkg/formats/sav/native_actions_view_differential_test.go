package sav

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func nativeActionsOwnedOracle(state DocumentStateData) ([]byte, bool, error) {
	var value []byte
	found := false
	for _, r := range state.ValueRecords {
		if !strings.EqualFold(r.Path, NativeActionsPath) {
			continue
		}
		if found || r.Path != NativeActionsPath || r.Value.Kind != 6 || r.Value.Int32 != 0 {
			return nil, false, fmt.Errorf("sav: ambiguous native action leaf")
		}
		b := r.Value.Bytes
		if len(b) < 8 || len(b) > MaxNativeActions+12 || len(b)%4 != 0 || binary.LittleEndian.Uint32(b) != 1 {
			return nil, false, fmt.Errorf("sav: invalid native action framing")
		}
		n := uint64(binary.LittleEndian.Uint32(b[4:]))
		if n == 0 || n > MaxNativeActions || n > uint64(len(b)-8) || uint64(len(b)) != (n+11)&^3 {
			return nil, false, fmt.Errorf("sav: invalid native action length")
		}
		for _, pad := range b[8+n:] {
			if pad != 0 {
				return nil, false, fmt.Errorf("sav: nonzero native action padding")
			}
		}
		value, found = bytes.Clone(b[8:8+n]), true
	}
	return value, found, nil
}

func setNativeActionsOwnedOracle(state *DocumentStateData, value []byte) error {
	if state == nil || len(value) == 0 || len(value) > MaxNativeActions {
		return fmt.Errorf("sav: native actions require a bounded application leaf")
	}
	if _, _, err := nativeActionsOwnedOracle(*state); err != nil {
		return err
	}
	b := make([]byte, (len(value)+11)&^3)
	binary.LittleEndian.PutUint32(b, 1)
	binary.LittleEndian.PutUint32(b[4:], uint32(len(value)))
	copy(b[8:], value)
	r := CityStateRecordData{Path: NativeActionsPath, Value: CityStateValueData{Kind: 6, Bytes: b}}
	records := slices.Clone(state.ValueRecords)
	index := -1
	for i := range records {
		if records[i].Path == r.Path {
			index = i
		}
	}
	if index < 0 {
		records = append(records, r)
	} else {
		records[index] = r
	}
	slices.SortFunc(records, func(a, b CityStateRecordData) int { return strings.Compare(a.Path, b.Path) })
	state.ValueRecords = records
	return nil
}

func remapNativeActionsOwnedOracle(state *DocumentStateData, permutation []uint16, dropRetiredOwnership bool) error {
	data, present, err := nativeActionsOwnedOracle(*state)
	if err != nil || !present {
		return err
	}
	if len(permutation) == 0 || permutation[0] != 0 {
		return fmt.Errorf("sav: current action remap lacks the null slot")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("sav: current action remap: %w", err)
	}
	var version uint32
	if err := json.Unmarshal(root["Version"], &version); err != nil || version != 1 {
		return fmt.Errorf("sav: invalid current action remap version")
	}
	var identityRoot map[string]any
	if !dropRetiredOwnership {
		identity := true
		for i, object := range permutation {
			if int(object) != i {
				identity = false
				break
			}
		}
		if identity {
			identityRoot = make(map[string]any, len(root))
			for name, raw := range root {
				identityRoot[name] = raw
			}
		}
	}
	for _, name := range []string{"Bindings", "StructureBindings", "Objects", "Groups", "PlayerSlots", "AbsentPlayers", "SpellCasters", "Ownership", "NativeAreas", "NativeDeliveries", "AbsentDiaries", "ArchiveCoordinates", "ActorGroups", "EffectWidths"} {
		raw, ok := root[name]
		if !ok {
			continue
		}
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return fmt.Errorf("sav: invalid current action %s", name)
		}
		next := make([]map[string]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			var object uint16
			if err := json.Unmarshal(row["Object"], &object); err != nil || int(object) >= len(permutation) {
				return fmt.Errorf("sav: current action object exceeds remap")
			}
			if name == "Bindings" {
				var missing bool
				if err := json.Unmarshal(row["Missing"], &missing); err != nil || missing != (object == 0) {
					return fmt.Errorf("sav: inconsistent current action missing object")
				}
			} else if object == 0 && name != "Ownership" {
				return fmt.Errorf("sav: null current reserved object")
			}
			current := permutation[object]
			if name == "ActorGroups" {
				if current == 0 {
					continue
				}
				var player uint16
				if err := json.Unmarshal(row["Player"], &player); err != nil || int(player) >= len(permutation) {
					return fmt.Errorf("sav: current actor Group exceeds remap")
				}
				if player != 0 && permutation[player] == 0 {
					continue
				}
				row["Player"], _ = json.Marshal(permutation[player])
			}
			if name == "Ownership" && object != 0 && current == 0 {
				if !dropRetiredOwnership {
					return fmt.Errorf("sav: current ownership node was retired")
				}
				continue
			}
			if name == "EffectWidths" {
				var actor uint16
				if err := json.Unmarshal(row["Actor"], &actor); err != nil || actor == 0 || int(actor) >= len(permutation) {
					return fmt.Errorf("sav: current effect width actor exceeds remap")
				}
				if current == 0 || permutation[actor] == 0 {
					continue
				}
				row["Actor"], _ = json.Marshal(permutation[actor])
			}
			if name == "SpellCasters" {
				if current == 0 {
					continue
				}
				var caster uint16
				if err := json.Unmarshal(row["Caster"], &caster); err != nil || int(caster) >= len(permutation) {
					return fmt.Errorf("sav: current spell caster exceeds remap")
				}
				row["Caster"], _ = json.Marshal(permutation[caster])
			}
			if current == 0 && (name == "Groups" || name == "StructureBindings" || name == "PlayerSlots" || name == "AbsentPlayers") {
				return fmt.Errorf("sav: current Group container was retired")
			}
			if current == 0 && (name == "Objects" || name == "NativeAreas" || name == "NativeDeliveries" || name == "AbsentDiaries" || name == "ArchiveCoordinates") {
				continue
			}
			row["Object"], _ = json.Marshal(current)
			if name == "Bindings" && current == 0 {
				row["Missing"] = json.RawMessage("true")
			}
			next = append(next, row)
		}
		if rows != nil {
			if identityRoot != nil {
				identityRoot[name] = next
			} else {
				root[name], err = json.Marshal(next)
				if err != nil {
					return err
				}
			}
		}
	}
	if raw, ok := root["Pending"]; ok {
		relocated, err := remapPendingActionsOwnedOracle(raw, permutation)
		if err != nil {
			return err
		}
		if identityRoot != nil {
			identityRoot["Pending"] = relocated
		} else {
			root["Pending"] = relocated
		}
	}
	if identityRoot != nil {
		data, err = json.Marshal(identityRoot)
	} else {
		data, err = json.Marshal(root)
	}
	if err != nil {
		return err
	}
	if err := setNativeActionsOwnedOracle(state, data); err != nil {
		return err
	}
	return nil
}

func remapPendingActionsOwnedOracle(raw json.RawMessage, permutation []uint16) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return raw, nil
	}
	var queue map[string]json.RawMessage
	if err := json.Unmarshal(raw, &queue); err != nil {
		return nil, fmt.Errorf("sav: invalid pending command queue")
	}
	remap := func(raw json.RawMessage) (json.RawMessage, error) {
		var binding map[string]json.RawMessage
		if err := json.Unmarshal(raw, &binding); err != nil {
			return nil, fmt.Errorf("sav: invalid pending endpoint")
		}
		var object uint16
		var missing bool
		if err := json.Unmarshal(binding["Object"], &object); err != nil || int(object) >= len(permutation) {
			return nil, fmt.Errorf("sav: pending endpoint exceeds remap")
		}
		if err := json.Unmarshal(binding["Missing"], &missing); err != nil || missing != (object == 0) {
			return nil, fmt.Errorf("sav: inconsistent pending missing endpoint")
		}
		binding["Object"], _ = json.Marshal(permutation[object])
		binding["Missing"], _ = json.Marshal(permutation[object] == 0)
		return json.Marshal(binding)
	}
	var commands []map[string]json.RawMessage
	if err := json.Unmarshal(queue["Commands"], &commands); err != nil || len(commands) > 65536 {
		return nil, fmt.Errorf("sav: invalid pending commands")
	}
	for _, command := range commands {
		for _, name := range []string{"Issuer", "Target"} {
			if endpoint, ok := command[name]; ok {
				value, err := remap(endpoint)
				if err != nil {
					return nil, err
				}
				command[name] = value
			}
		}
	}
	var commanded []json.RawMessage
	if err := json.Unmarshal(queue["Commanded"], &commanded); err != nil || len(commanded) > 65536 {
		return nil, fmt.Errorf("sav: invalid pending commanded actors")
	}
	for i, raw := range commanded {
		value, err := remap(raw)
		if err != nil {
			return nil, err
		}
		commanded[i] = value
	}
	queue["Commands"], _ = json.Marshal(commands)
	queue["Commanded"], _ = json.Marshal(commanded)
	return json.Marshal(queue)
}

type nativeActionsViewFixture struct {
	name  string
	state DocumentStateData
}

func nativeActionsOwnedState(t *testing.T, payload []byte) DocumentStateData {
	t.Helper()
	state := DocumentStateData{
		RootKind:         7,
		DirectoryRecords: []CityStateDirectoryData{{Path: "/CurrentState", Kind: 3}},
		ValueRecords: []CityStateRecordData{
			{Path: "/Z", Value: CityStateValueData{Kind: 6, Bytes: []byte{1, 2, 3, 4}}},
			{Path: "/A", Value: CityStateValueData{Kind: 2, Int32: -9}},
		},
	}
	if payload != nil {
		if err := setNativeActionsOwnedOracle(&state, payload); err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func nativeActionsViewFixtures(t *testing.T) []nativeActionsViewFixture {
	t.Helper()
	fixtures := []nativeActionsViewFixture{{"zero-state", DocumentStateData{}}, {"absent", nativeActionsOwnedState(t, nil)}}
	for _, n := range []int{1, 2, 3, 4, 7, 8} {
		fixtures = append(fixtures, nativeActionsViewFixture{fmt.Sprintf("length-%d", n), nativeActionsOwnedState(t, bytes.Repeat([]byte{'x'}, n))})
	}
	for _, name := range []string{"case", "duplicate", "duplicate-case", "kind", "int32", "nil", "short", "header-only", "alignment", "version", "zero-length", "length-overflow", "length-outside", "length-short", "extra-word", "padding", "frame-oversize"} {
		state := nativeActionsOwnedState(t, []byte(`{"Version":1}`))
		index := -1
		for i, record := range state.ValueRecords {
			if record.Path == NativeActionsPath {
				index = i
			}
		}
		r := &state.ValueRecords[index]
		switch name {
		case "case":
			r.Path = "/CurrentState/againromActions"
			state.ValueRecords = append(state.ValueRecords, CityStateRecordData{Path: NativeActionsPath, Value: r.Value})
		case "duplicate", "duplicate-case":
			duplicate := *r
			if name == "duplicate-case" {
				duplicate.Path = "/CurrentState/againromActions"
			}
			state.ValueRecords = append(state.ValueRecords, duplicate)
		case "kind":
			r.Value.Kind = 2
		case "int32":
			r.Value.Int32 = -1
		case "nil":
			r.Value.Bytes = nil
		case "short":
			r.Value.Bytes = r.Value.Bytes[:4]
		case "header-only":
			r.Value.Bytes = r.Value.Bytes[:8]
		case "alignment":
			r.Value.Bytes = r.Value.Bytes[:len(r.Value.Bytes)-1]
		case "version":
			binary.LittleEndian.PutUint32(r.Value.Bytes, 2)
		case "zero-length":
			binary.LittleEndian.PutUint32(r.Value.Bytes[4:], 0)
		case "length-overflow":
			binary.LittleEndian.PutUint32(r.Value.Bytes[4:], ^uint32(0))
		case "length-outside":
			binary.LittleEndian.PutUint32(r.Value.Bytes[4:], uint32(len(r.Value.Bytes)))
		case "length-short":
			binary.LittleEndian.PutUint32(r.Value.Bytes[4:], 1)
		case "extra-word":
			r.Value.Bytes = append(r.Value.Bytes, 0, 0, 0, 0)
		case "padding":
			r.Value.Bytes[len(r.Value.Bytes)-1] = 1
		case "frame-oversize":
			r.Value.Bytes = make([]byte, MaxNativeActions+16)
		}
		fixtures = append(fixtures, nativeActionsViewFixture{name, state})
	}
	return fixtures
}

func TestNativeActionPrivateViewFramingMatchesOwnedReader(t *testing.T) {
	fixtures := nativeActionsViewFixtures(t)
	fixtures = append(fixtures, nativeActionsViewFixture{"maximum", nativeActionsOwnedState(t, bytes.Repeat([]byte{'x'}, MaxNativeActions))})
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			before := nativeRemapCopyState(fixture.state)
			want, wantPresent, wantErr := nativeActionsOwnedOracle(fixture.state)
			got, present, err := NativeActions(fixture.state)
			view, viewPresent, viewErr := nativeActionsView(fixture.state)
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || fmt.Sprint(viewErr) != fmt.Sprint(wantErr) || present != wantPresent || viewPresent != wantPresent || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(view, want) {
				t.Fatal("framing, presence or exact payload differs from the frozen reader", err, viewErr, wantErr)
			}
			if !reflect.DeepEqual(fixture.state, before) {
				t.Fatal("reading changed the state")
			}
			if present {
				if &got[0] == &view[0] {
					t.Fatal("public payload aliases the private view")
				}
				clear(got)
				again, againPresent, againErr := NativeActions(fixture.state)
				if fmt.Sprint(againErr) != fmt.Sprint(wantErr) || againPresent != wantPresent || !bytes.Equal(again, want) || !reflect.DeepEqual(fixture.state, before) {
					t.Fatal("public payload mutation reached the state or a later reader", againErr)
				}
			}
		})
	}
}

func nativeActionsSetterDifferential(t *testing.T, state *DocumentStateData, value []byte) {
	t.Helper()
	input := bytes.Clone(value)
	var got, want, before *DocumentStateData
	if state != nil {
		gotState, wantState, beforeState := nativeRemapCopyState(*state), nativeRemapCopyState(*state), nativeRemapCopyState(*state)
		got, want, before = &gotState, &wantState, &beforeState
	}
	err, wantErr := SetNativeActions(got, input), setNativeActionsOwnedOracle(want, value)
	if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !bytes.Equal(input, value) {
		t.Fatal("setter changed exact error, state or input bytes", err, wantErr)
	}
	if err != nil && !reflect.DeepEqual(got, before) {
		t.Fatal("failed setter changed the state")
	}
	if err == nil {
		clear(input)
		if !reflect.DeepEqual(got, want) {
			t.Fatal("setter retained caller payload storage")
		}
	}
}

func TestNativeActionPrivateViewSetterMatchesOwnedSetter(t *testing.T) {
	for _, fixture := range nativeActionsViewFixtures(t) {
		t.Run(fixture.name, func(t *testing.T) {
			for _, value := range [][]byte{nil, {}, {'x'}, []byte(`{"Version":1,"Objects":[]}`)} {
				nativeActionsSetterDifferential(t, &fixture.state, value)
			}
		})
	}
	for _, value := range [][]byte{nil, {'x'}, bytes.Repeat([]byte{'x'}, MaxNativeActions+1)} {
		nativeActionsSetterDifferential(t, nil, value)
	}
	state := nativeActionsOwnedState(t, []byte(`{"Version":1}`))
	for _, n := range []int{MaxNativeActions, MaxNativeActions + 1} {
		nativeActionsSetterDifferential(t, &state, bytes.Repeat([]byte{'x'}, n))
	}
	alias := nativeActionsOwnedState(t, []byte(`{"Version":1,"Opaque":"held"}`))
	value, _, err := nativeActionsView(alias)
	if err != nil {
		t.Fatal(err)
	}
	want := nativeRemapCopyState(alias)
	if err := setNativeActionsOwnedOracle(&want, value); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeActions(&alias, value); err != nil || !reflect.DeepEqual(alias, want) {
		t.Fatal("setter changed an input alias of its previous payload", err)
	}
	clear(value)
	if !reflect.DeepEqual(alias, want) {
		t.Fatal("setter retained the previous leaf storage")
	}
}

func clearNativeViewRemaps() {
	nativeRemaps.mu.Lock()
	nativeRemaps.entries = nil
	nativeRemaps.mu.Unlock()
}

func nativeActionsViewRemapDifferential(t *testing.T, state DocumentStateData, permutation []uint16, city bool) {
	t.Helper()
	clearNativeViewRemaps()
	before := nativeRemapCopyState(state)
	beforePermutation := slices.Clone(permutation)
	want := nativeRemapCopyState(state)
	wantErr := remapNativeActionsOwnedOracle(&want, permutation, city)
	for range 3 {
		got := nativeRemapCopyState(state)
		err := remapNativeActionObjectsMode(&got, permutation, city)
		if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(state, before) || !reflect.DeepEqual(permutation, beforePermutation) {
			t.Fatal("remap changed exact error, state, input or permutation", err, wantErr)
		}
		payload, present, readErr := nativeActionsOwnedOracle(state)
		if readErr == nil && present && len(permutation) > 0 && permutation[0] == 0 {
			if _, hit := nativeRemaps.lookup(payload, permutation, city); hit != (wantErr == nil) {
				t.Fatal("memo acceptance differs from the frozen remap")
			}
		}
		if err != nil && !reflect.DeepEqual(got, before) {
			t.Fatal("failed remap changed the state")
		}
		if err == nil {
			for i := range got.ValueRecords {
				if got.ValueRecords[i].Path == NativeActionsPath {
					clear(got.ValueRecords[i].Value.Bytes)
					got.ValueRecords[i].Path = "/Changed"
				}
			}
			if !reflect.DeepEqual(state, before) {
				t.Fatal("remap output shares its native leaf with the input")
			}
		}
	}
}

func TestNativeActionPrivateViewRemapMatchesOwnedRemap(t *testing.T) {
	t.Cleanup(clearNativeViewRemaps)
	payloads := []struct {
		name, payload string
	}{
		{"version", `{"Version":1}`},
		{"all-tables", `{"Version":1,"Bindings":[{"Object":1,"Missing":false},{"Object":0,"Missing":true}],"StructureBindings":[{"Object":1}],"Objects":[{"Object":2}],"Groups":[{"Object":1}],"PlayerSlots":[{"Object":1}],"AbsentPlayers":[{"Object":1}],"SpellCasters":[{"Object":2,"Caster":1}],"Ownership":[{"Object":1},{"Object":0}],"NativeAreas":[{"Object":2}],"NativeDeliveries":[{"Object":2}],"AbsentDiaries":[{"Object":2}],"ArchiveCoordinates":[{"Object":2}],"ActorGroups":[{"Object":2,"Player":1}],"EffectWidths":[{"Object":2,"Actor":1}],"Pending":{"Commands":[{"Issuer":{"Object":1,"Missing":false},"Target":{"Object":2,"Missing":false}}],"Commanded":[{"Object":1,"Missing":false}]},"Unknown":{"number":1e+03,"text":"<>&"}}`},
		{"ownership", `{"Version":1,"Ownership":[{"Object":1},{"Object":2},{"Object":0}]}`},
		{"retired-optional", `{"Version":1,"Objects":[{"Object":1}],"NativeAreas":[{"Object":1}],"NativeDeliveries":[{"Object":1}],"AbsentDiaries":[{"Object":1}],"ArchiveCoordinates":[{"Object":1}],"ActorGroups":[{"Object":1,"Player":999}],"SpellCasters":[{"Object":1,"Caster":999}],"EffectWidths":[{"Object":1,"Actor":2}]}`},
		{"pending", `{"Version":1,"Pending":{"Commands":[{"Issuer":{"Object":1,"Missing":false},"Target":{"Object":2,"Missing":false}}],"Commanded":[{"Object":0,"Missing":true}]}}`},
		{"bad-json", `{"Version":1,`},
		{"bad-version", `{"Version":2}`},
		{"bad-object", `{"Version":1,"Objects":[{"Object":65535}]}`},
		{"bad-missing", `{"Version":1,"Bindings":[{"Object":1,"Missing":true}]}`},
		{"bad-secondary", `{"Version":1,"EffectWidths":[{"Object":1,"Actor":0}]}`},
		{"bad-pending", `{"Version":1,"Pending":{"Commands":[{"Issuer":{"Object":1,"Missing":true}}]}}`},
	}
	fixtures := nativeActionsViewFixtures(t)
	for _, payload := range payloads {
		fixtures = append(fixtures, nativeActionsViewFixture{payload.name, nativeActionsOwnedState(t, []byte(payload.payload))})
	}
	permutations := [][]uint16{nil, {1}, {0}, {0, 1, 2, 3}, {0, 2, 1, 3}, {0, 0, 1, 2}}
	for _, fixture := range fixtures {
		for _, city := range []bool{false, true} {
			for i, permutation := range permutations {
				t.Run(fmt.Sprintf("%s/city-%t/permutation-%d", fixture.name, city, i), func(t *testing.T) {
					nativeActionsViewRemapDifferential(t, fixture.state, permutation, city)
				})
			}
		}
	}
}
