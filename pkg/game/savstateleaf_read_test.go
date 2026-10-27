package game

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
)

func readLeafFixture(path, variant string, kind uint32, value sav.CityStateValueData) sav.DocumentStateData {
	root, leaf := splitStatePath(path)
	state := sav.DocumentStateData{RootKind: 17, DirectoryRecords: []sav.CityStateDirectoryData{{Path: "/" + root, Kind: kind}}}
	if variant == "missing" {
		return state
	}
	if variant == "case" || variant == "duplicate" {
		leaf = strings.ToUpper(leaf)
		if leaf == strings.TrimPrefix(path, "/"+root+"/") {
			leaf = strings.ToLower(leaf)
		}
	}
	state.ValueRecords = []sav.CityStateRecordData{{Path: "/" + root + "/" + leaf, Value: value}}
	if variant == "duplicate" {
		state.ValueRecords = append(state.ValueRecords, sav.CityStateRecordData{Path: path, Value: value})
	}
	return state
}

func TestStateLeafReadApplication(t *testing.T) {
	for _, path := range []string{"/GameOptions/Wimpy", "/GameOptions/ShowHP", "/GameOptions/FlyingHP", "/GameOptions/Formation", "/GameOptions/Speed", "/GameOptions/ShowTimeFlow", "/Inventory/IsOpen", "/SpellBook/IsOpen", "/SpellBook/Pressed", "/View/X", "/View/Y", "/CurrentState/InBattle", "/Character/Name", "/Objects/Selection", "/Fog/FirstState", "/Fog/Data"} {
		t.Run(path, func(t *testing.T) {
			kind := uint32(2)
			value := sav.CityStateValueData{Kind: kind, Int32: 19}
			if path == "/Character/Name" {
				kind = 0
				value = sav.CityStateValueData{Kind: 0, Bytes: []byte("Hero\x00")}
			}
			if path == "/Objects/Selection" || path == "/Fog/Data" {
				kind = 6
				value = sav.CityStateValueData{Kind: 6, Bytes: []byte{1, 0, 0, 0}}
			}
			for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
				t.Run(variant, func(t *testing.T) {
					doc := sav.DocumentData{State: readLeafFixture(path, variant, 1, value)}
					got, err := readOriginalStateLeaf(&doc, path, kind)
					wantErr := variant == "missing" || variant == "duplicate"
					if (err != nil) != wantErr || !wantErr && (got.Int32 != value.Int32 || !slices.Equal(got.Bytes, value.Bytes)) {
						t.Fatalf("got %+v, %v", got, err)
					}
				})
			}
		})
	}
}

func TestStateLeafReadTownOptions(t *testing.T) {
	for _, path := range append([]string{"/GameOptions/Formation"}, cityApplicationLeaves...) {
		for _, unavailable := range []bool{false, true} {
			for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
				t.Run(path+"/"+variant+"/"+map[bool]string{true: "fallback", false: "document"}[unavailable], func(t *testing.T) {
					state := readLeafFixture(path, variant, 1, sav.CityStateValueData{Kind: 2, Int32: 9})
					city := &SnapshotOriginalCity{Document: sav.CityData{State: sav.CityStateData{RootKind: state.RootKind, DirectoryRecords: state.DirectoryRecords, ValueRecords: state.ValueRecords}}}
					if unavailable {
						city.Unavailable = "unsupported topology"
						city.loadedState = state
					}
					got, err := townApplicationOptions(city)
					if (err != nil) != (variant == "duplicate") {
						t.Fatal(err)
					}
					if variant == "exact" || variant == "case" {
						if len(got) != 1 || got[0].Path != path || got[0].Value.Int32 != 9 {
							t.Fatalf("options %+v", got)
						}
					}
					if variant == "missing" && len(got) != 0 {
						t.Fatal(got)
					}
					if variant == "case" {
						state.DirectoryRecords[0].Kind = 17
						got, err = townApplicationOptions(city)
						if err != nil || len(got) != 0 {
							t.Fatalf("sorted case: %+v %v", got, err)
						}
					}
				})
			}
		}
	}
}

func TestStateLeafReadCityFormation(t *testing.T) {
	for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
		t.Run(variant, func(t *testing.T) {
			data := cityFormationFixture(t, "/GameOptions/Formation", 1)
			loaded := readLeafFixture("/GameOptions/Formation", variant, 1, sav.CityStateValueData{Kind: 2, Int32: 2})
			err := nativeCityPlayerFormation(&data, nil, nil, loaded.ValueRecords)
			if (err != nil) != (variant == "duplicate") {
				t.Fatal(err)
			}
			if err != nil {
				return
			}
			want := int32(2)
			mode := byte(1)
			if variant == "missing" {
				want = 1
				mode = 2
			}
			if got := storedLeafValue(t, data.State.ValueRecords, "/GameOptions/Formation").Int32; got != want || data.Objects[0].Player.Raw32[31] != mode {
				t.Fatalf("setting %d, mode %d", got, data.Objects[0].Player.Raw32[31])
			}
		})
	}
}

func TestStateLeafReadCityApplication(t *testing.T) {
	for _, path := range cityApplicationLeaves {
		for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
			t.Run(path+"/"+variant, func(t *testing.T) {
				data := sav.CityData{State: sav.NewCityStateData("Hero")}
				loaded := readLeafFixture(path, variant, 1, sav.CityStateValueData{Kind: 2, Int32: 19})
				before := storedLeafValue(t, data.State.ValueRecords, path).Int32
				err := projectCityApplication(&data, nil, loaded.ValueRecords)
				if (err != nil) != (variant == "duplicate") {
					t.Fatal(err)
				}
				if err != nil {
					return
				}
				want := int32(19)
				if variant == "missing" {
					want = before
				}
				if got := storedLeafValue(t, data.State.ValueRecords, path).Int32; got != want {
					t.Fatalf("value %d, want %d", got, want)
				}
			})
		}
	}
}

func TestStateLeafReadQuickSpells(t *testing.T) {
	value := sav.CityStateValueData{Kind: 6, Bytes: make([]byte, 16)}
	for i, v := range []int32{0, 1, 2, -1} {
		binary.LittleEndian.PutUint32(value.Bytes[4*i:], uint32(v))
	}
	for _, variant := range []string{"exact", "case", "missing", "duplicate"} {
		t.Run("city/"+variant, func(t *testing.T) {
			state := readLeafFixture("/SpellBook/Shortcuts", variant, 1, value)
			got, err := quickSpellsFromState(state)
			wantErr := variant == "duplicate" || variant == "missing"
			if (err != nil) != wantErr {
				t.Fatal(err)
			}
			if !wantErr && got != ([4]uint32{1, 2, 3, 0}) {
				t.Fatal(got)
			}
		})
	}

	for _, variant := range []string{"exact", "case"} {
		model := cityfixture.City(false)
		state := readLeafFixture("/SpellBook/Shortcuts", variant, 1, value)
		delete(model.State.Values, "/SpellBook/Shortcuts")
		model.State.Values[state.ValueRecords[0].Path] = value
		city, err := sav.CityFromData(model)
		if err != nil {
			t.Fatal(err)
		}
		got, err := originalCityQuickSpells(city)
		if err != nil || got != ([4]uint32{1, 2, 3, 0}) {
			t.Fatal(got, err)
		}
		update := sav.CityUpdate{Money: 123}
		for _, actor := range city.Roster() {
			update.Characters = append(update.Characters, originalCityBaselineUpdate(actor))
		}
		raw, err := city.Marshal(update)
		if err != nil {
			t.Fatal(err)
		}
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err = originalQuickSpells(file)
		if err != nil || got != ([4]uint32{1, 2, 3, 0}) {
			t.Fatal(got, err)
		}
	}
}

func TestStateLeafReadRawQuickSpells(t *testing.T) {
	model := cityfixture.City(false)
	city, err := sav.CityFromData(model)
	if err != nil {
		t.Fatal(err)
	}
	update := sav.CityUpdate{Money: 123}
	for _, actor := range city.Roster() {
		update.Characters = append(update.Characters, originalCityBaselineUpdate(actor))
	}
	raw, err := city.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	base, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, variant := range []string{"exact", "case", "missing", "duplicate", "sorted_case", "root_case"} {
		t.Run(variant, func(t *testing.T) {
			count := 1
			if variant == "missing" {
				count = 0
			}
			if variant == "duplicate" {
				count = 2
			}
			store := make([]byte, 24+32*(1+count)+4+16*count)
			binary.LittleEndian.PutUint32(store, 0x31415926)
			binary.LittleEndian.PutUint32(store[8:], 1)
			binary.LittleEndian.PutUint32(store[12:], 17)
			binary.LittleEndian.PutUint32(store[16:], uint32(1+count))
			node := func(at int, data, size, kind uint32, name string) {
				r := store[24+32*at:]
				binary.LittleEndian.PutUint32(r[4:], data)
				binary.LittleEndian.PutUint32(r[8:], size)
				binary.LittleEndian.PutUint32(r[12:], kind)
				copy(r[16:32], name)
			}
			section := "SpellBook"
			if variant == "root_case" {
				section = "spellbook"
			}
			dirKind := uint32(1)
			if variant == "sorted_case" {
				dirKind = 17
			}
			node(0, 1, uint32(count), dirKind, section)
			for i := 0; i < count; i++ {
				name := "Shortcuts"
				if variant == "case" || variant == "sorted_case" || variant == "duplicate" && i == 0 {
					name = "shortcuts"
				}
				node(i+1, uint32(i*16), 16, 6, name)
				pool := store[24+32*(1+count)+4+i*16:]
				for j, v := range []int32{0, 1, 2, -1} {
					binary.LittleEndian.PutUint32(pool[4*j:], uint32(v))
				}
			}
			binary.LittleEndian.PutUint32(store[24+32*(1+count):], uint32(16*count))
			fileCopy := *base
			fileCopy.Store = store
			file, err := sav.Open(fileCopy.Marshal())
			if err != nil {
				t.Fatal(err)
			}
			got, err := originalQuickSpells(file)
			if (err != nil) != (variant == "duplicate") {
				t.Fatal(err)
			}
			if variant == "exact" || variant == "case" {
				if got != ([4]uint32{1, 2, 3, 0}) {
					t.Fatal(got)
				}
			}
			if variant == "missing" || variant == "sorted_case" || variant == "root_case" {
				if got != ([4]uint32{}) {
					t.Fatal(got)
				}
			}
		})
	}
}

func TestStateLeafReadCompleteProjectile(t *testing.T) {
	paths := []string{"/Projectiles/IDs", "/Projectiles/FreeIndex"}
	for _, name := range projectileFieldNames {
		paths = append(paths, "/Prj7/"+name)
	}
	for _, path := range paths {
		for _, variant := range []string{"exact", "case", "missing", "duplicate", "mixed_duplicate"} {
			t.Run(path+"/"+variant, func(t *testing.T) {
				doc := projectileStateFixture(false, 1)
				value := storedLeafValue(t, doc.State.ValueRecords, path)
				doc.State.ValueRecords = slices.DeleteFunc(doc.State.ValueRecords, func(r sav.CityStateRecordData) bool { return r.Path == path })
				choice := variant
				if variant == "mixed_duplicate" {
					choice = "duplicate"
				}
				replacement := readLeafFixture(path, choice, 1, value)
				if variant == "mixed_duplicate" {
					replacement.ValueRecords[0].Value.Kind = 0
				}
				doc.State.ValueRecords = append(doc.State.ValueRecords, replacement.ValueRecords...)
				want := variant == "exact" || variant == "case" || variant == "missing" && path == "/Projectiles/FreeIndex"
				if got := completeSavedProjectileRecord(&doc, 7); got != want {
					t.Fatalf("complete %v, want %v", got, want)
				}
			})
		}
	}
}

func TestStateLeafReadRuntimeTarget(t *testing.T) {
	for _, variant := range []string{"exact", "case", "missing", "duplicate", "sorted_case"} {
		t.Run(variant, func(t *testing.T) {
			choice := variant
			kind := uint32(1)
			if variant == "sorted_case" {
				choice = "case"
				kind = 17
			}
			state := readLeafFixture("/Prj7/actiontarget", choice, kind, sav.CityStateValueData{Kind: 2, Int32: 1})
			actor := mustNewRecord("Unit")
			mustSetValue(&actor, "RuntimeID", 0)
			doc := sav.DocumentData{State: state, Objects: []sav.DocumentRecordData{actor}}
			err := projectCurrentRuntimeIDs(&doc, Snapshot{})
			if (err != nil) != (variant == "duplicate") {
				t.Fatal(err)
			}
			if err != nil {
				return
			}
			// The leaf is read and the record keeps the id it was written
			// with: SAVE allocates none (SAV-1371).
			got, err := savedStructureValue(&doc.Objects[0], "RuntimeID")
			if err != nil || got != 0 {
				t.Fatalf("actor ID %d, want 0: %v", got, err)
			}
		})
	}
}
