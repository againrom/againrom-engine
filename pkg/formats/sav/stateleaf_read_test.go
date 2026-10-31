package sav

import (
	"encoding/binary"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/reg"
)

func stateReadFile(state DocumentStateData) *File {
	root := &reg.Node{Kind: state.RootKind, Dir: true}
	for _, d := range state.DirectoryRecords {
		section := &reg.Node{Name: strings.TrimPrefix(d.Path, "/"), Kind: d.Kind, Dir: true}
		for _, r := range state.ValueRecords {
			directory, name := splitStatePath(r.Path)
			if "/"+directory != d.Path {
				continue
			}
			v := r.Value
			leaf := &reg.Node{Name: name, Kind: v.Kind, Type: reg.ValueType(v.Kind), Int: v.Int32}
			if v.Kind == 6 {
				leaf.Ints = make([]int32, len(v.Bytes)/4)
				for i := range leaf.Ints {
					leaf.Ints[i] = int32(binary.LittleEndian.Uint32(v.Bytes[4*i:]))
				}
			}
			if v.Kind == 0 {
				leaf.Str = strings.TrimSuffix(string(v.Bytes), "\x00")
			}
			section.Children = append(section.Children, leaf)
		}
		root.Children = append(root.Children, section)
	}
	return &File{store: &reg.Reg{Root: root}}
}

func stateReadVariant(path, variant string, value CityStateValueData) DocumentStateData {
	root, leaf := splitStatePath(path)
	state := DocumentStateData{RootKind: 17, DirectoryRecords: []CityStateDirectoryData{{Path: "/" + root, Kind: 1}}}
	if variant == "missing" {
		return state
	}
	if variant == "case" || variant == "duplicate" {
		leaf = strings.ToUpper(leaf)
		if leaf == strings.TrimPrefix(path, "/"+root+"/") {
			leaf = strings.ToLower(leaf)
		}
	}
	state.ValueRecords = []CityStateRecordData{{Path: "/" + root + "/" + leaf, Value: value}}
	if variant == "duplicate" {
		state.ValueRecords = append(state.ValueRecords, CityStateRecordData{Path: path, Value: value})
	}
	return state
}

func TestStateLeafReadRawFog(t *testing.T) {
	for _, path := range []string{"/Fog/Data", "/Fog/FirstState"} {
		for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
			t.Run(path+"/"+variant, func(t *testing.T) {
				value := CityStateValueData{Kind: 2, Int32: 0x8000}
				if path == "/Fog/Data" {
					value = CityStateValueData{Kind: 6, Bytes: []byte{3, 0, 0, 0}}
				}
				state := stateReadVariant(path, variant, value)
				other := CityStateRecordData{Path: "/Fog/Data", Value: CityStateValueData{Kind: 6, Bytes: []byte{3, 0, 0, 0}}}
				if path == "/Fog/Data" {
					other = CityStateRecordData{Path: "/Fog/FirstState", Value: CityStateValueData{Kind: 2, Int32: 0x8000}}
				}
				state.ValueRecords = append(state.ValueRecords, other)
				got, present, err := stateReadFile(state).Fog()
				wantErr := variant == "duplicate" || variant == "missing" && path == "/Fog/FirstState"
				if (err != nil) != wantErr {
					t.Fatal(err)
				}
				if variant == "exact" || variant == "case" {
					if !present || got.FirstState != 0x8000 || !slices.Equal(got.Cells, []byte{1, 1, 1}) {
						t.Fatal(got, present, err)
					}
				}
				if variant == "missing" && path == "/Fog/Data" && present {
					t.Fatal("missing Data is present")
				}
			})
		}
	}
}

func TestStateLeafReadRawProjectiles(t *testing.T) {
	paths := append([]string{"/Projectiles/FreeIndex", "/Projectiles/IDs"}, func() []string {
		var out []string
		for _, name := range projectileLeafNames {
			out = append(out, "/Prj7/"+name)
		}
		return out
	}()...)
	for _, path := range paths {
		for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
			t.Run(path+"/"+variant, func(t *testing.T) {
				value := CityStateValueData{Kind: 2, Int32: 23}
				if path == "/Projectiles/IDs" {
					value = CityStateValueData{Kind: 6, Bytes: []byte{7, 0, 0, 0}}
				}
				state := stateReadVariant(path, variant, value)
				root, _ := splitStatePath(path)
				if root != "Projectiles" {
					state.DirectoryRecords = append(state.DirectoryRecords, CityStateDirectoryData{Path: "/Projectiles", Kind: 1})
				}
				if path != "/Projectiles/IDs" {
					state.ValueRecords = append(state.ValueRecords, CityStateRecordData{Path: "/Projectiles/IDs", Value: CityStateValueData{Kind: 6, Bytes: []byte{7, 0, 0, 0}}})
				}
				if path != "/Projectiles/FreeIndex" {
					state.ValueRecords = append(state.ValueRecords, CityStateRecordData{Path: "/Projectiles/FreeIndex", Value: CityStateValueData{Kind: 2, Int32: 23}})
				}
				if root == "Projectiles" {
					state.DirectoryRecords = append(state.DirectoryRecords, CityStateDirectoryData{Path: "/Prj7", Kind: 1})
				}
				if !strings.HasPrefix(path, "/Prj7/") {
					state.ValueRecords = append(state.ValueRecords, CityStateRecordData{Path: "/Prj7/x", Value: CityStateValueData{Kind: 2, Int32: 23}})
				}
				got, present, err := stateReadFile(state).Projectiles()
				if (err != nil) != (variant == "duplicate") {
					t.Fatal(err)
				}
				if err != nil {
					return
				}
				if !present {
					t.Fatal("manager missing")
				}
				if variant == "missing" {
					if path == "/Projectiles/IDs" && len(got.IDs) != 0 || path == "/Projectiles/FreeIndex" && got.FreeIndex != 0 {
						t.Fatal(got)
					}
					if strings.HasPrefix(path, "/Prj7/") {
						_, name := splitStatePath(path)
						if len(got.Items) != 1 || reflect.ValueOf(got.Items[0]).Field(1+slices.Index(projectileLeafNames, name)).Int() != 0 {
							t.Fatal(got)
						}
					}
					return
				}
				if path == "/Projectiles/IDs" && !slices.Equal(got.IDs, []uint16{7}) || path == "/Projectiles/FreeIndex" && got.FreeIndex != 23 {
					t.Fatal(got)
				}
				if strings.HasPrefix(path, "/Prj7/") {
					if len(got.Items) != 1 {
						t.Fatal(got)
					}
					_, name := splitStatePath(path)
					if reflect.ValueOf(got.Items[0]).Field(1+slices.Index(projectileLeafNames, name)).Int() != 23 {
						t.Fatal("projectile leaf missing", got)
					}
				}
			})
		}
	}
}

func TestStateLeafReadSupplements(t *testing.T) {
	for _, d := range supplementLeaves {
		t.Run(d.name, func(t *testing.T) {
			payload := []byte{1, 0, 0, 0, 1, 0, 0, 0, 'x', 0, 0, 0}
			if d == nativeRandomLeaf {
				payload = make([]byte, 12)
				binary.LittleEndian.PutUint32(payload, 1)
			}
			if d == nativeSessionLeaf {
				payload = make([]byte, 20)
				binary.LittleEndian.PutUint32(payload, 1)
			}
			if d == nativeRowsLeaf {
				payload = []byte{1, 0, 0, 0, 2, 0, 0, 0, 8, 1, 0, 0}
			}
			for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
				t.Run(variant, func(t *testing.T) {
					state := stateReadVariant(d.path, variant, CityStateValueData{Kind: 6, Bytes: payload})
					got, present, err := d.lookup(state)
					if (err != nil) != (variant == "duplicate") {
						t.Fatal(err)
					}
					if variant == "exact" || variant == "case" {
						if !present || !slices.Equal(got, payload) {
							t.Fatal(got, present)
						}
						if variant == "case" {
							state.DirectoryRecords[0].Kind = 17
							_, present, err = d.lookup(state)
							if present || err != nil {
								t.Fatal("sorted variant matched", err)
							}
						}
					}
					if variant == "missing" && present {
						t.Fatal("missing present")
					}
					if d == nativeActionsLeaf && stateReadFile(state).HasNativeActionsLeaf() != (variant != "missing" && variant != "case") {
						t.Fatal("authored marker detection differs")
					}
				})
			}
		})
	}
}

func TestStateLeafReadCodec(t *testing.T) {
	for _, directory := range cityStateShape {
		for _, child := range directory.children {
			path := "/" + directory.directory + "/" + child.name
			for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
				t.Run(path+"/"+variant, func(t *testing.T) {
					data := NewCityStateData("Hero")
					state, err := cityStateFromData(data, 2)
					if err != nil {
						t.Fatal(err)
					}
					value := state.values[path]
					delete(state.values, path)
					replacement := path
					if variant == "case" || variant == "duplicate" {
						replacement = "/" + directory.directory + "/" + strings.ToLower(child.name)
					}
					if replacement == path && variant != "exact" {
						replacement = "/" + directory.directory + "/" + strings.ToUpper(child.name)
					}
					if variant != "missing" {
						state.values[replacement] = value
					}
					if variant == "duplicate" {
						state.values[path] = value
					}
					err = validateCityState(state)
					if (err != nil) != (variant == "missing" || variant == "duplicate") {
						t.Fatal(err)
					}
					if err != nil {
						return
					}
					raw, err := serializeCityState(state)
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := parseCityState(raw)
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(state, decoded) {
						got := decoded.values[replacement]
						if got.kind != value.kind || value.kind == 2 && got.int32 != value.int32 || !slices.Equal(got.bytes, value.bytes) {
							t.Fatal("codec changed selected leaf")
						}
					}
				})
			}
		}
	}
}

func TestStateLeafReadWorldCodec(t *testing.T) {
	paths := []string{"/Fog/FirstState", "/Fog/Data", "/Projectiles/FreeIndex", "/Projectiles/IDs"}
	for _, name := range projectileLeafNames {
		paths = append(paths, "/Prj7/"+name)
	}
	for _, path := range paths {
		for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
			t.Run(path+"/"+variant, func(t *testing.T) {
				state, err := parseWorldState(literalStateStore(literalWorldDirectories(7)))
				if err != nil {
					t.Fatal(err)
				}
				value := state.values[path]
				delete(state.values, path)
				root, name := splitStatePath(path)
				state.directoryKinds["/"+root] = 1
				name = strings.ToUpper(name)
				if name == strings.TrimPrefix(path, "/"+root+"/") {
					name = strings.ToLower(name)
				}
				stored := path
				if variant == "case" || variant == "duplicate" {
					stored = "/" + root + "/" + name
				}
				if variant != "missing" {
					state.values[stored] = value
				}
				if variant == "duplicate" {
					state.values[path] = value
				}
				if variant == "missing" && path == "/Projectiles/IDs" {
					delete(state.directoryKinds, "/Prj7")
					for p := range state.values {
						if strings.HasPrefix(p, "/Prj7/") {
							delete(state.values, p)
						}
					}
				}
				err = validateWorldState(state)
				optional := path == "/Projectiles/FreeIndex" || path == "/Projectiles/IDs"
				if (err != nil) != (variant == "duplicate" || variant == "missing" && !optional) {
					t.Fatal(err)
				}
				if err != nil {
					return
				}
				raw, err := serializeWorldState(state)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := parseWorldState(raw)
				if err != nil {
					t.Fatal(err)
				}
				if variant != "missing" {
					got := decoded.values[stored]
					if got.kind != value.kind || value.kind == 2 && got.int32 != value.int32 || !slices.Equal(got.bytes, value.bytes) {
						t.Fatal("codec changed selected leaf")
					}
				}
			})
		}
	}
}
