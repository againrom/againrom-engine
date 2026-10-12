package sav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// The frozen placement: Actions and Mods prepended, Seed before InBattle, Rng
// in a fixed world list.

func frozenValidateMods(kind uint32, b []byte) error {
	if kind != 6 || len(b) < 8 || len(b) > MaxNativeMods+12 || len(b)%4 != 0 || binary.LittleEndian.Uint32(b) != 1 {
		return fmt.Errorf("sav: invalid mod mark framing")
	}
	n := uint64(binary.LittleEndian.Uint32(b[4:]))
	if n == 0 || n > MaxNativeMods || n > uint64(len(b)-8) || uint64(len(b)) != (n+11)&^3 {
		return fmt.Errorf("sav: invalid mod mark length")
	}
	for _, pad := range b[8+n:] {
		if pad != 0 {
			return fmt.Errorf("sav: nonzero mod mark padding")
		}
	}
	return nil
}

func frozenValidateRandom(kind uint32, value []byte) error {
	if kind != 6 || len(value) != 12 || binary.LittleEndian.Uint32(value) != 1 {
		return fmt.Errorf("sav: AgainromRng requires version 1 and exactly three dwords")
	}
	return nil
}

func frozenValidateSession(kind uint32, b []byte) error {
	if kind != 6 || len(b) < 4 || map[uint32]int{1: 20, 2: 24}[binary.LittleEndian.Uint32(b)] != len(b) {
		return fmt.Errorf("sav: AgainromSeed requires version 1 with five dwords or version 2 with six")
	}
	if mode := binary.LittleEndian.Uint32(b[12:]); mode > 1 {
		return fmt.Errorf("sav: AgainromSeed names random mode %d, which is not defined", mode)
	}
	return nil
}

func frozenSessionLeaf(state *cityState, shape []stateDirectory) ([]stateDirectory, error) {
	value, ok := state.values[NativeSessionPath]
	if !ok {
		return shape, nil
	}
	if err := frozenValidateSession(value.kind, value.bytes); err != nil {
		return nil, err
	}
	for i := range shape {
		if shape[i].directory != "CurrentState" {
			continue
		}
		children := slices.Clone(shape[i].children)
		at := slices.IndexFunc(children, func(l stateLeaf) bool { return l.name == "InBattle" })
		if at < 0 {
			at = len(children)
		}
		shape[i].children = slices.Insert(children, at, stateLeaf{"AgainromSeed", 6})
	}
	return shape, nil
}

func frozenPrependModsActions(state *cityState, shape []stateDirectory) ([]stateDirectory, error) {
	if value, ok := state.values[NativeModsPath]; ok {
		if err := frozenValidateMods(value.kind, value.bytes); err != nil {
			return nil, err
		}
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = append([]stateLeaf{{"AgainromMods", 6}}, shape[i].children...)
			}
		}
	}
	if value, ok := state.values[NativeActionsPath]; ok {
		if _, _, err := nativeActionsOwnedOracle(DocumentStateData{ValueRecords: []CityStateRecordData{{Path: NativeActionsPath, Value: CityStateValueData{Kind: value.kind, Bytes: value.bytes}}}}); err != nil {
			return nil, err
		}
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = append([]stateLeaf{{"AgainromActions", 6}}, shape[i].children...)
			}
		}
	}
	return shape, nil
}

func frozenCityShape(state *cityState) ([]stateDirectory, error) {
	shape, err := frozenSessionLeaf(state, append([]stateDirectory(nil), cityStateShape...))
	if err != nil {
		return nil, err
	}
	return frozenPrependModsActions(state, shape)
}

func frozenWorldShape(state *cityState) ([]stateDirectory, error) {
	if _, err := worldStateSerializedSize(state); err != nil {
		return nil, err
	}
	shape := append([]stateDirectory(nil), cityStateShape...)
	if value, ok := state.values[NativeRandomPath]; ok {
		if err := frozenValidateRandom(value.kind, value.bytes); err != nil {
			return nil, err
		}
		for i := range shape {
			if shape[i].directory == "CurrentState" {
				shape[i].children = []stateLeaf{{"AgainromRng", 6}, {"InBattle", 2}}
			}
		}
	}
	shape, err := frozenSessionLeaf(state, shape)
	if err != nil {
		return nil, err
	}
	if shape, err = frozenPrependModsActions(state, shape); err != nil {
		return nil, err
	}
	shape = append(shape, stateDirectory{"Fog", []stateLeaf{{"FirstState", 2}, {"Data", 6}}})
	sort.Slice(shape, func(i, j int) bool { return shape[i].directory < shape[j].directory })
	return shape, nil
}

func supplementTestValues(t *testing.T) map[string][]byte {
	t.Helper()
	values := map[string][]byte{}
	for path, set := range map[string]func(*DocumentStateData) error{
		NativeActionsPath: func(s *DocumentStateData) error { return SetNativeActions(s, []byte(`{"Version":1}`)) },
		NativeModsPath:    func(s *DocumentStateData) error { return SetNativeMods(s, []byte(`{"format":1}`)) },
		NativeRandomPath:  func(s *DocumentStateData) error { return SetNativeRandomState(s, 0x0123456789abcdef) },
		NativeSessionPath: func(s *DocumentStateData) error {
			return SetNativeSession(s, NativeSession{Seed: 7, Mode: 1, Shared: 9, Reseeds: 2})
		},
	} {
		var s DocumentStateData
		if err := set(&s); err != nil || len(s.ValueRecords) != 1 {
			t.Fatal(path, err)
		}
		values[path] = s.ValueRecords[0].Value.Bytes
	}
	return values
}

func supplementShapeState(present []string, values map[string][]byte, invalid map[string]cityStateValue) *cityState {
	state := cityTestState("Shape")
	state.directoryKinds["/Fog"] = 1
	state.values["/Fog/FirstState"] = cityStateValue{kind: 2}
	state.values["/Fog/Data"] = cityStateValue{kind: 6, bytes: make([]byte, 8)}
	for _, path := range present {
		state.values[path] = cityStateValue{kind: 6, bytes: bytes.Clone(values[path])}
	}
	for path, value := range invalid {
		state.values[path] = value
	}
	return state
}

func copyStateShape(shape []stateDirectory) []stateDirectory {
	out := slices.Clone(shape)
	for i := range out {
		out[i].children = slices.Clone(out[i].children)
	}
	return out
}

// Every leaf subset, both forms: shape, refusal and store bytes.
func TestSupplementLeafPlacementMatchesFrozenShape(t *testing.T) {
	values := supplementTestValues(t)
	paths := []string{NativeActionsPath, NativeModsPath, NativeRandomPath, NativeSessionPath}
	invalid := map[string]cityStateValue{
		NativeActionsPath: {kind: 2},
		NativeModsPath:    {kind: 6, bytes: []byte{1, 0, 0, 0, 0, 0, 0, 0}},
		NativeRandomPath:  {kind: 6, bytes: make([]byte, 12)},
		NativeSessionPath: {kind: 6, bytes: binary.LittleEndian.AppendUint32(make([]byte, 16), 0)},
	}
	frozenCity := copyStateShape(cityStateShape)
	compared := 0
	for mask := range 1 << len(paths) {
		var present []string
		for i, path := range paths {
			if mask&(1<<i) != 0 {
				present = append(present, path)
			}
		}
		cases := []map[string]cityStateValue{nil}
		for _, a := range paths {
			cases = append(cases, map[string]cityStateValue{a: invalid[a]})
			for _, b := range paths {
				if a < b {
					cases = append(cases, map[string]cityStateValue{a: invalid[a], b: invalid[b]})
				}
			}
		}
		for _, bad := range cases {
			name := fmt.Sprint(present, bad)
			city := supplementShapeState(present, values, bad)
			delete(city.directoryKinds, "/Fog")
			delete(city.values, "/Fog/FirstState")
			delete(city.values, "/Fog/Data")
			got, err := currentCityStateShape(city)
			want, wantErr := frozenCityShape(city)
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) {
				t.Fatalf("city %s: shape %v %v, frozen %v %v", name, got, err, want, wantErr)
			}
			gotBytes, err := serializeCityState(city)
			wantBytes, wantErr := []byte(nil), wantErr
			if wantErr == nil {
				if wantErr = validateStateShape(city, want); wantErr == nil {
					wantBytes, wantErr = serializeStateStore(city, want)
				}
			}
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || !bytes.Equal(gotBytes, wantBytes) {
				t.Fatalf("city %s: bytes or refusal differ: %v, frozen %v", name, err, wantErr)
			}
			world := supplementShapeState(present, values, bad)
			got, err = worldStateShape(world)
			want, wantErr = frozenWorldShape(world)
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || !reflect.DeepEqual(got, want) {
				t.Fatalf("world %s: shape %v %v, frozen %v %v", name, got, err, want, wantErr)
			}
			gotBytes, err = serializeWorldState(world)
			wantBytes = nil
			if wantErr == nil {
				if wantErr = validateStateShape(world, want); wantErr == nil {
					wantBytes, wantErr = serializeStateStore(world, want)
				}
			}
			if fmt.Sprint(err) != fmt.Sprint(wantErr) || !bytes.Equal(gotBytes, wantBytes) {
				t.Fatalf("world %s: bytes or refusal differ: %v, frozen %v", name, err, wantErr)
			}
			if err == nil {
				compared++
			}
		}
	}
	if !reflect.DeepEqual(cityStateShape, frozenCity) {
		t.Fatal("placement edited the shared city shape")
	}
	if compared < 16 {
		t.Fatalf("only %d world stores serialized", compared)
	}
}

func TestSupplementLeafPlacementLossControl(t *testing.T) {
	values := supplementTestValues(t)
	state := supplementShapeState([]string{NativeSessionPath, NativeRandomPath}, values, nil)
	shape, err := worldStateShape(state)
	if err != nil {
		t.Fatal(err)
	}
	good, err := serializeStateStore(state, shape)
	if err != nil {
		t.Fatal(err)
	}
	moved := copyStateShape(shape)
	for i := range moved {
		if moved[i].directory == "CurrentState" {
			children := slices.DeleteFunc(moved[i].children, func(l stateLeaf) bool { return l.name == "AgainromSeed" })
			moved[i].children = append(children, stateLeaf{"AgainromSeed", 6})
		}
	}
	if reflect.DeepEqual(moved, shape) {
		t.Fatal("loss control did not move the leaf")
	}
	bad, err := serializeStateStore(state, moved)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(bad, good) {
		t.Fatal("moving AgainromSeed after InBattle left the state-store bytes equal")
	}
}

func TestSupplementLeafFifthDescriptor(t *testing.T) {
	fifth := &supplementLeaf{directory: supplementDirectory, name: "AgainromNext", path: "/CurrentState/AgainromNext",
		kind: 6, framing: framedPayload, bound: 64, forms: cityForm | worldForm,
		refusals: leafRefusals{ambiguous: "ambiguous", kind: "kind", framing: "framing", length: "length", padding: "padding", bound: "bound"}}
	saved := supplementLeaves
	supplementLeaves = append(slices.Clone(saved), fifth)
	t.Cleanup(func() { supplementLeaves = saved })

	var doc DocumentStateData
	if err := fifth.storePayload(&doc, []byte("next")); err != nil {
		t.Fatal(err)
	}
	if err := SetNativeSession(&doc, NativeSession{Seed: 1}); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := fifth.lookupPayload(doc); err != nil || !ok || string(got) != "next" {
		t.Fatal("fifth leaf payload", string(got), ok, err)
	}
	if err := fifth.storePayload(&doc, bytes.Repeat([]byte{'x'}, 65)); err == nil || err.Error() != "bound" {
		t.Fatal("fifth leaf bound", err)
	}
	for _, form := range []string{"city", "world"} {
		state := cityTestState("Next")
		if form == "world" {
			state.directoryKinds["/Fog"] = 1
			state.values["/Fog/FirstState"] = cityStateValue{kind: 2}
			state.values["/Fog/Data"] = cityStateValue{kind: 6, bytes: make([]byte, 8)}
		}
		for _, r := range doc.ValueRecords {
			state.values[r.Path] = cityStateValue{kind: r.Value.Kind, bytes: r.Value.Bytes}
		}
		var shape []stateDirectory
		var err error
		if form == "city" {
			_, err = serializeCityState(state)
			if err == nil {
				shape, err = currentCityStateShape(state)
			}
		} else {
			_, err = serializeWorldState(state)
			if err == nil {
				shape, err = worldStateShape(state)
			}
		}
		if err != nil {
			t.Fatal(form, err)
		}
		i := slices.IndexFunc(shape, func(d stateDirectory) bool { return d.directory == "CurrentState" })
		want := []stateLeaf{{"AgainromNext", 6}, {"AgainromSeed", 6}, {"InBattle", 2}}
		if !reflect.DeepEqual(shape[i].children, want) {
			t.Fatal(form, shape[i].children)
		}
	}
}
