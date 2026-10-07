package sav

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func nativeRemapCopyState(in DocumentStateData) DocumentStateData {
	out := in
	if in.DirectoryRecords != nil {
		out.DirectoryRecords = append([]CityStateDirectoryData{}, in.DirectoryRecords...)
	}
	if in.ValueRecords != nil {
		out.ValueRecords = make([]CityStateRecordData, len(in.ValueRecords))
		copy(out.ValueRecords, in.ValueRecords)
		for i := range out.ValueRecords {
			out.ValueRecords[i].Value.Bytes = bytes.Clone(in.ValueRecords[i].Value.Bytes)
		}
	}
	return out
}

func nativeRemapState(t *testing.T, payload string) DocumentStateData {
	t.Helper()
	state := DocumentStateData{RootKind: 7}
	if payload != "" {
		if err := SetNativeActions(&state, []byte(payload)); err != nil {
			t.Fatal(err)
		}
	}
	state.ValueRecords = append(state.ValueRecords,
		CityStateRecordData{Path: "/Z", Value: CityStateValueData{Kind: 6, Bytes: []byte{1, 2, 3, 4}}},
		CityStateRecordData{Path: "/A", Value: CityStateValueData{Kind: 2, Int32: -9}})
	return state
}

func nativeRemapDifferential(t *testing.T, state DocumentStateData, permutation []uint16, city, wantErr bool) DocumentStateData {
	t.Helper()
	input := nativeRemapCopyState(state)
	before := nativeRemapCopyState(input)
	got := input
	beforePermutation := append([]uint16(nil), permutation...)
	err := remapNativeActionObjectsMode(&got, permutation, city)
	if !reflect.DeepEqual(input, before) || !reflect.DeepEqual(permutation, beforePermutation) {
		t.Fatal("candidate mutated input state or permutation")
	}
	legacyInput := nativeRemapCopyState(state)
	legacyBefore := nativeRemapCopyState(legacyInput)
	want := legacyInput
	legacyErr := nativeRemapLegacyMode(&want, permutation, city)
	if !reflect.DeepEqual(legacyInput, legacyBefore) || !reflect.DeepEqual(permutation, beforePermutation) {
		t.Fatal("legacy mutated input state or permutation")
	}
	if fmt.Sprint(err) != fmt.Sprint(legacyErr) || (err != nil) != wantErr {
		t.Fatalf("candidate error %v, legacy error %v, wantErr %t", err, legacyErr, wantErr)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("complete registry state differs from legacy")
	}
	if err != nil {
		if !reflect.DeepEqual(got, before) {
			t.Fatal("failed candidate changed state")
		}
		return got
	}
	actual, actualPresent, actualErr := NativeActions(got)
	expected, expectedPresent, expectedErr := NativeActions(want)
	if actualErr != nil || expectedErr != nil || actualPresent != expectedPresent || !bytes.Equal(actual, expected) {
		t.Fatal("canonical payload bytes differ from legacy")
	}
	if actualPresent {
		owned := nativeRemapCopyState(got)
		for i := range got.ValueRecords {
			if got.ValueRecords[i].Path == NativeActionsPath {
				got.ValueRecords[i].Value.Bytes[8] ^= 0xff
				got.ValueRecords[i].Path = "/Changed"
				break
			}
		}
		if !reflect.DeepEqual(input, before) {
			t.Fatal("candidate native leaf aliases input bytes or record slice")
		}
		return owned
	}
	return got
}

func TestNativeActionIdentityDifferential(t *testing.T) {
	cases := []struct {
		name, payload, canonical string
		wantErr                  bool
	}{
		{"version_only", `{"Version":1}`, `{"Version":1}`, false},
		{"ownership_null", `{"Version":1,"Ownership":[{"x":{"n":1e+03},"Object":null}]}`, `{"Ownership":[{"Object":0,"x":{"n":1e+03}}],"Version":1}`, false},
		{"binding_null_missing", `{"Version":1,"Bindings":[{"Object":1,"Missing":null}]}`, `{"Bindings":[{"Missing":null,"Object":1}],"Version":1}`, false},
		{"binding_null_object", `{"Version":1,"Bindings":[{"Object":null,"Missing":true}]}`, `{"Bindings":[{"Missing":true,"Object":0}],"Version":1}`, false},
		{"secondary_null", `{"Version":1,"ActorGroups":[{"Object":1,"Player":null}],"SpellCasters":[{"Object":2,"Caster":null}]}`, `{"ActorGroups":[{"Object":1,"Player":0}],"SpellCasters":[{"Caster":0,"Object":2}],"Version":1}`, false},
		{"duplicates", `{"Version":2,"Version":1,"Objects":42,"Objects":[{"Object":"bad","Object":2,"object":99,"opaque":{"z":0,"z":1}}],"objects":false}`, `{"Objects":[{"Object":2,"object":99,"opaque":{"z":0,"z":1}}],"Version":1,"objects":false}`, false},
		{"unknown_raw", `{"Version":1,"Objects":[{"Object":1,"unknown":{"b":1e+03,"a":[9007199254740993,-0,"<>&\u2028\u2029","\\u003c"]}}],"Unknown":{"z":2,"a":1}}`, "", false},
		{"literal_separators", "{\"Version\":1,\"Ownership\":[{\"Object\":0,\"unknown\":\"<>&\u2028\u2029\"}]}", "", false},
		{"invalid_utf8_opaque", "{\"Version\":1,\"Objects\":[{\"Object\":1,\"unknown\":\"\xff\"}]}", "", false},
		{"late_invalid_table", `{"Version":1,"Bindings":[{"Object":1,"Missing":false}],"EffectWidths":[{"Object":2,"Actor":9}]}`, "", true},
		{"identity_bad_object", `{"Version":1,"Bindings":[{"ID":1,"Object":65535,"Missing":false}]}`, "", true},
		{"syntax", `{"Version":1,`, "", true},
		{"invalid_unknown_syntax", `{"Version":1,"Unknown":{"x":}}`, "", true},
		{"trailing_value", `{"Version":1} {}`, "", true},
		{"root_null", `null`, "", true},
		{"root_array", `[]`, "", true},
		{"root_string", `"x"`, "", true},
		{"missing_version", `{}`, "", true},
		{"version_case", `{"version":1}`, "", true},
		{"version_null", `{"Version":null}`, "", true},
		{"version_two", `{"Version":2}`, "", true},
		{"version_fraction", `{"Version":1.0}`, "", true},
		{"version_overflow", `{"Version":4294967296}`, "", true},
		{"binding_missing", `{"Version":1,"Bindings":[{"Object":1}]}`, "", true},
		{"binding_inconsistent", `{"Version":1,"Bindings":[{"Object":1,"Missing":true}]}`, "", true},
		{"binding_null_false", `{"Version":1,"Bindings":[{"Object":0,"Missing":false}]}`, "", true},
		{"binding_wrong_type", `{"Version":1,"Bindings":[{"Object":1,"Missing":0}]}`, "", true},
		{"player_bounds", `{"Version":1,"ActorGroups":[{"Object":1,"Player":4}]}`, "", true},
		{"actor_zero", `{"Version":1,"EffectWidths":[{"Object":1,"Actor":0}]}`, "", true},
		{"actor_bounds", `{"Version":1,"EffectWidths":[{"Object":1,"Actor":4}]}`, "", true},
		{"caster_bounds", `{"Version":1,"SpellCasters":[{"Object":1,"Caster":4}]}`, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nativeRemapDifferential(t, nativeRemapState(t, tc.payload), []uint16{0, 1, 2, 3}, false, tc.wantErr)
			if tc.canonical != "" {
				payload, _, err := NativeActions(got)
				if err != nil || string(payload) != tc.canonical {
					t.Fatalf("got %s (%v), want %s", payload, err, tc.canonical)
				}
			}
		})
	}
}

func TestNativeActionIdentityTableDifferential(t *testing.T) {
	names := []string{"Bindings", "Objects", "Groups", "PlayerSlots", "AbsentPlayers", "SpellCasters", "Ownership", "NativeAreas", "NativeDeliveries", "AbsentDiaries", "ArchiveCoordinates", "ActorGroups", "EffectWidths"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			tail := `,"Unknown":{"large":184467440737095516160,"ratio":1.00,"list":["<>&",null]}`
			switch name {
			case "Bindings":
				tail += `,"Missing":false`
			case "ActorGroups":
				tail += `,"Player":2`
			case "EffectWidths":
				tail += `,"Actor":2`
			case "SpellCasters":
				tail += `,"Caster":2`
			}
			for _, object := range []string{"1", "3", "4", "65536", "-1", "1.5", "1e0", `"1"`} {
				t.Run("object_"+object, func(t *testing.T) {
					payload := fmt.Sprintf(`{"Version":1,%q:[{"Object":%s%s}]}`, name, object, tail)
					nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1, 2, 3}, false, object != "1" && object != "3")
				})
			}
			for _, rows := range []string{"null", "[]", "{}", "42", `"x"`, "[null]", "[false]", "[{}]"} {
				t.Run("rows_"+rows, func(t *testing.T) {
					payload := fmt.Sprintf(`{"Version":1,%q:%s}`, name, rows)
					nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1, 2, 3}, false, rows != "null" && rows != "[]")
				})
			}
		})
	}
}

func TestNativeActionIdentityFramingDifferential(t *testing.T) {
	for _, name := range []string{"case", "duplicate", "kind", "int32", "short", "alignment", "version", "zero_length", "length", "padding", "oversize"} {
		t.Run(name, func(t *testing.T) {
			state := nativeRemapState(t, `{"Version":1}`)
			r := &state.ValueRecords[0]
			switch name {
			case "case":
				r.Path = "/CurrentState/againromActions"
			case "duplicate":
				state.ValueRecords = append(state.ValueRecords, *r)
			case "kind":
				r.Value.Kind = 2
			case "int32":
				r.Value.Int32 = 1
			case "short":
				r.Value.Bytes = []byte{1, 0, 0, 0}
			case "alignment":
				r.Value.Bytes = r.Value.Bytes[:len(r.Value.Bytes)-1]
			case "version":
				r.Value.Bytes[0] = 2
			case "zero_length":
				binary.LittleEndian.PutUint32(r.Value.Bytes[4:], 0)
			case "length":
				binary.LittleEndian.PutUint32(r.Value.Bytes[4:], 100)
			case "padding":
				r.Value.Bytes[len(r.Value.Bytes)-1] = 1
			case "oversize":
				r.Value.Bytes = make([]byte, MaxNativeActions+16)
				binary.LittleEndian.PutUint32(r.Value.Bytes, 1)
				binary.LittleEndian.PutUint32(r.Value.Bytes[4:], MaxNativeActions+8)
			}
			nativeRemapDifferential(t, state, []uint16{0, 1}, false, true)
		})
	}
	for _, permutation := range [][]uint16{nil, {1}, {0}} {
		t.Run(fmt.Sprint(permutation), func(t *testing.T) {
			nativeRemapDifferential(t, nativeRemapState(t, ""), permutation, false, false)
			nativeRemapDifferential(t, nativeRemapState(t, `{"Version":1}`), permutation, false, len(permutation) == 0 || permutation[0] != 0)
		})
	}
}

func TestNativeActionIdentitySizeDepthDifferential(t *testing.T) {
	t.Run("uint16_maximum", func(t *testing.T) {
		permutation := make([]uint16, 65536)
		for i := range permutation {
			permutation[i] = uint16(i)
		}
		payload := `{"Version":1,"EffectWidths":[{"Object":65535,"Actor":65534}],"ActorGroups":[{"Object":65534,"Player":65535}],"SpellCasters":[{"Object":65535,"Caster":65534}]}`
		nativeRemapDifferential(t, nativeRemapState(t, payload), permutation, false, false)
	})
	t.Run("maximum_input", func(t *testing.T) {
		prefix, suffix := `{"Version":1,"Opaque":"`, `"}`
		payload := prefix + strings.Repeat("x", MaxNativeActions-len(prefix)-len(suffix)) + suffix
		nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1}, false, false)
	})
	t.Run("output_expansion", func(t *testing.T) {
		payload := `{"Version":1,"Ownership":[{"Object":0,"Opaque":"` + strings.Repeat("<", MaxNativeActions/6+1) + `"}]}`
		nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1}, false, true)
	})
	for _, depth := range []int{32, 9997, 10001} {
		t.Run(fmt.Sprint(depth), func(t *testing.T) {
			payload := `{"Version":1,"Ownership":[{"Object":0,"Opaque":` + strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth) + `}]}`
			nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1}, false, depth > 9997)
		})
	}
}

func TestNativeActionIdentitySecondaryDifferential(t *testing.T) {
	for _, pair := range [][2]string{{"ActorGroups", "Player"}, {"EffectWidths", "Actor"}, {"SpellCasters", "Caster"}} {
		for _, value := range []string{"", "null", "0", "1", "3", "4", "-1", "65536", "1.5", `"1"`, "false"} {
			t.Run(pair[0]+"_"+value, func(t *testing.T) {
				field := ""
				if value != "" {
					field = fmt.Sprintf(",%q:%s", pair[1], value)
				}
				payload := fmt.Sprintf(`{"Version":1,%q:[{"Object":1%s}]}`, pair[0], field)
				valid := value == "1" || value == "3" || (pair[1] != "Actor" && (value == "null" || value == "0"))
				nativeRemapDifferential(t, nativeRemapState(t, payload), []uint16{0, 1, 2, 3}, false, !valid)
			})
		}
	}
}

func TestNativeActionNonIdentityDifferential(t *testing.T) {
	cases := []struct {
		name, payload string
		permutation   []uint16
		city, wantErr bool
	}{
		{"swap", `{"Version":1,"Bindings":[{"Object":1,"Missing":false}],"ActorGroups":[{"Object":2,"Player":1}],"EffectWidths":[{"Object":1,"Actor":2}],"SpellCasters":[{"Object":2,"Caster":1}]}`, []uint16{0, 2, 1}, false, false},
		{"binding_retired", `{"Version":1,"Bindings":[{"Object":1,"Missing":false}]}`, []uint16{0, 0, 1}, false, false},
		{"ownership_strict", `{"Version":1,"Ownership":[{"Object":1}]}`, []uint16{0, 0, 1}, false, true},
		{"ownership_city", `{"Version":1,"Ownership":[{"Object":1},{"Object":2,"Opaque":{"a":1e3}}]}`, []uint16{0, 0, 1}, true, false},
		{"identity_city", `{"Version":1,"Ownership":[{"Object":null,"Opaque":42}]}`, []uint16{0, 1, 2}, true, false},
		{"group_retired", `{"Version":1,"Groups":[{"Object":1}]}`, []uint16{0, 0, 1}, false, true},
		{"actor_group_skip_before_player", `{"Version":1,"ActorGroups":[{"Object":1,"Player":999}]}`, []uint16{0, 0, 1}, false, false},
		{"effect_validate_before_skip", `{"Version":1,"EffectWidths":[{"Object":1,"Actor":999}]}`, []uint16{0, 0, 1}, false, true},
		{"caster_skip_before_caster", `{"Version":1,"SpellCasters":[{"Object":1,"Caster":999}]}`, []uint16{0, 0, 1}, false, false},
		{"caster_retired", `{"Version":1,"SpellCasters":[{"Object":2,"Caster":1}]}`, []uint16{0, 0, 1}, false, false},
		{"unbounded_target", `{"Version":1,"Objects":[{"Object":1}]}`, []uint16{0, 65535}, false, false},
	}
	for _, name := range []string{"Objects", "NativeAreas", "NativeDeliveries", "AbsentDiaries", "ArchiveCoordinates", "EffectWidths"} {
		payload := fmt.Sprintf(`{"Version":1,%q:[{"Object":1,"Actor":2}]}`, name)
		cases = append(cases, struct {
			name, payload string
			permutation   []uint16
			city, wantErr bool
		}{name + "_retired", payload, []uint16{0, 0, 1}, false, false})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nativeRemapDifferential(t, nativeRemapState(t, tc.payload), tc.permutation, tc.city, tc.wantErr)
		})
	}
}

func nativeRemapLegacyMode(state *DocumentStateData, permutation []uint16, dropRetiredOwnership bool) error {
	data, present, err := NativeActions(*state)
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
	for _, name := range []string{"Bindings", "Objects", "Groups", "PlayerSlots", "AbsentPlayers", "SpellCasters", "Ownership", "NativeAreas", "NativeDeliveries", "AbsentDiaries", "ArchiveCoordinates", "ActorGroups", "EffectWidths"} {
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
			if current == 0 && (name == "Groups" || name == "PlayerSlots" || name == "AbsentPlayers") {
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
			root[name], err = json.Marshal(next)
			if err != nil {
				return err
			}
		}
	}
	data, err = json.Marshal(root)
	if err != nil {
		return err
	}
	return SetNativeActions(state, data)
}
