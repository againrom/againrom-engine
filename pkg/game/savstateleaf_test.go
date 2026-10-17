package game

import (
	"encoding/binary"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestStateNameRule(t *testing.T) {
	for _, tc := range []struct {
		kind          uint32
		stored, query string
		want          bool
	}{
		{1, "Formation", "Formation", true},
		{1, "formation", "Formation", true},
		{1, "FORMATION", "Formation", true},
		{17, "Formation", "Formation", true},
		{17, "formation", "Formation", false},
		{1, "ABCDEFGHIJKLMNO", "abcdefghijklmnoP", true},
		{17, "ABCDEFGHIJKLMNO", "ABCDEFGHIJKLMNOP", true},
		{17, "ABCDEFGHIJKLMNOP", "ABCDEFGHIJKLMNOP", false},
		{1, "\xc0", "\xe0", false},
		{1, "Formatio", "Formation", false},
	} {
		if got := stateNameMatches(tc.kind, tc.stored, tc.query); got != tc.want {
			t.Errorf("kind %d stored %q query %q = %v, want %v", tc.kind, tc.stored, tc.query, got, tc.want)
		}
	}
}

func respellLeaf(t *testing.T, dirs []sav.CityStateDirectoryData, values []sav.CityStateRecordData, from, to string, dirKind uint32) {
	t.Helper()
	root, _ := splitStatePath(from)
	found := false
	for i := range values {
		if values[i].Path == from {
			values[i].Path, found = to, true
		}
	}
	for i := range dirs {
		if dirs[i].Path == "/"+root {
			dirs[i].Kind = dirKind
		}
	}
	if !found {
		t.Fatalf("fixture lacks %s", from)
	}
}

func storedLeafValue(t *testing.T, values []sav.CityStateRecordData, path string) sav.CityStateValueData {
	t.Helper()
	for _, r := range values {
		if r.Path == path {
			return r.Value
		}
	}
	t.Fatalf("no stored %s", path)
	return sav.CityStateValueData{}
}

var leafSpellings = []struct {
	name    string
	respell func(string) string
	dirKind uint32
	patched bool
}{
	{"exact", func(s string) string { return s }, 1, true},
	{"scanned_case", strings.ToLower, 1, true},
	{"sorted_case", strings.ToLower, 17, false},
}

func lowerLeaf(path string, respell func(string) string) string {
	at := strings.LastIndexByte(path, '/')
	return path[:at+1] + respell(path[at+1:])
}

func TestStateLeafShortcuts(t *testing.T) {
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			state := sav.NewCityStateData("Hero")
			doc := sav.DocumentData{State: sav.DocumentStateData{RootKind: state.RootKind, DirectoryRecords: state.DirectoryRecords, ValueRecords: state.ValueRecords}}
			path := lowerLeaf("/SpellBook/Shortcuts", sp.respell)
			respellLeaf(t, doc.State.DirectoryRecords, doc.State.ValueRecords, "/SpellBook/Shortcuts", path, sp.dirKind)
			writeQuickSpellLeaf(&doc, sav.CityShortcuts{3, -1, 0, 7})
			got := storedLeafValue(t, doc.State.ValueRecords, path).Bytes
			patched := len(got) == 16 && int32(binary.LittleEndian.Uint32(got[0:])) == 3 && int32(binary.LittleEndian.Uint32(got[12:])) == 7
			if patched != sp.patched {
				t.Fatalf("Shortcuts patched=%v, want %v: % x", patched, sp.patched, got)
			}
		})
	}
}

func cityFormationFixture(t *testing.T, path string, dirKind uint32) sav.CityData {
	t.Helper()
	data := sav.CityData{Players: []uint16{1}, Objects: []sav.CityObjectData{{Class: "Player", Player: &sav.CityPlayerData{Raw32: make([]byte, 32)}}},
		State: sav.NewCityStateData("Hero")}
	respellLeaf(t, data.State.DirectoryRecords, data.State.ValueRecords, "/GameOptions/Formation", path, dirKind)
	return data
}

func TestStateLeafTownFormation(t *testing.T) {
	loaded := []sav.CityStateRecordData{{Path: "/GameOptions/Formation", Value: sav.CityStateValueData{Kind: 2, Int32: 0}}}
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			path := lowerLeaf("/GameOptions/Formation", sp.respell)
			data := cityFormationFixture(t, path, sp.dirKind)
			leaf := &data.State.ValueRecords[0]
			for i := range data.State.ValueRecords {
				if data.State.ValueRecords[i].Path == path {
					leaf = &data.State.ValueRecords[i]
				}
			}
			leaf.Value.Int32 = 5
			err := nativeCityPlayerFormation(&data, nil, nil, loaded)
			if !sp.patched {
				if err == nil || leaf.Value.Int32 != 5 {
					t.Fatalf("sorted case variant was taken for the Formation leaf: %v %d", err, leaf.Value.Int32)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if leaf.Value.Int32 != 0 || data.Objects[0].Player.Raw32[31] != 0 {
				t.Fatalf("Formation leaf %d, Player byte %d; want the loaded setting 0 and mode 0", leaf.Value.Int32, data.Objects[0].Player.Raw32[31])
			}
			leaf.Value.Int32 = 9
			if err := projectCityApplication(&data, nil, loaded); err != nil {
				t.Fatal(err)
			}
			if leaf.Value.Int32 != 9 {
				t.Fatal("a second writer patched Formation")
			}
		})
	}
}

func TestStateLeafTownApplication(t *testing.T) {
	loaded := []sav.CityStateRecordData{{Path: "/View/X", Value: sav.CityStateValueData{Kind: 2, Int32: 7}}}
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			data := sav.CityData{State: sav.NewCityStateData("Hero")}
			path := lowerLeaf("/View/X", sp.respell)
			respellLeaf(t, data.State.DirectoryRecords, data.State.ValueRecords, "/View/X", path, sp.dirKind)
			if err := projectCityApplication(&data, nil, loaded); err != nil {
				t.Fatal(err)
			}
			if got := storedLeafValue(t, data.State.ValueRecords, path).Int32; (got == 7) != sp.patched {
				t.Fatalf("View/X = %d, patched want %v", got, sp.patched)
			}
		})
	}
}

func TestStateLeafWorldApplication(t *testing.T) {
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			_, snapshot, world := applicationSnapshotFixture(t)
			doc, err := sav.CloneDocumentData(*snapshot.SavedDocument.Document)
			if err != nil {
				t.Fatal(err)
			}
			path := lowerLeaf("/View/X", sp.respell)
			respellLeaf(t, doc.State.DirectoryRecords, doc.State.ValueRecords, "/View/X", path, sp.dirKind)
			snapshot.ApplicationState = nil
			snapshot.CameraSet, snapshot.CameraX, snapshot.CameraY, snapshot.CameraZoom = true, 12.75, 13.25, 2
			err = projectApplicationState(&doc, snapshot, world)
			if !sp.patched {
				if err == nil {
					t.Fatal("sorted case variant accepted as View/X")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := storedLeafValue(t, doc.State.ValueRecords, path).Int32; got != 13 {
				t.Fatalf("View/X = %d, want 13", got)
			}
		})
	}
}

func TestStateLeafSelection(t *testing.T) {
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			state := sav.NewCityStateData("Hero")
			doc := sav.DocumentData{State: sav.DocumentStateData{RootKind: state.RootKind, DirectoryRecords: state.DirectoryRecords, ValueRecords: state.ValueRecords}}
			path := lowerLeaf("/Objects/Selection", sp.respell)
			respellLeaf(t, doc.State.DirectoryRecords, doc.State.ValueRecords, "/Objects/Selection", path, sp.dirKind)
			for i := range doc.State.ValueRecords {
				if doc.State.ValueRecords[i].Path == path {
					doc.State.ValueRecords[i].Value.Bytes = []byte{1, 2, 3}
				}
			}
			err := projectCurrentRuntimeIDs(&doc, Snapshot{})
			if reached := err != nil && strings.Contains(err.Error(), "partial runtime ID"); reached != sp.patched {
				t.Fatalf("Selection reached=%v, want %v: %v", reached, sp.patched, err)
			}
		})
	}
}

func projectileStateFixture(variant bool, dirKind uint32) sav.DocumentData {
	free, x := "FreeIndex", "x"
	if variant {
		free, x = "freeindex", "X"
	}
	doc := sav.DocumentData{State: sav.DocumentStateData{RootKind: 17, DirectoryRecords: []sav.CityStateDirectoryData{{Path: "/Prj7", Kind: dirKind}, {Path: "/Projectiles", Kind: dirKind}}}}
	ids := make([]byte, 4)
	binary.LittleEndian.PutUint32(ids, 7)
	doc.State.ValueRecords = append(doc.State.ValueRecords,
		sav.CityStateRecordData{Path: "/Projectiles/" + free, Value: sav.CityStateValueData{Kind: 2, Int32: 8}},
		sav.CityStateRecordData{Path: "/Projectiles/IDs", Value: sav.CityStateValueData{Kind: 6, Bytes: ids}})
	for _, name := range projectileFieldNames {
		if name == "x" {
			name = x
		}
		doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: "/Prj7/" + name, Value: sav.CityStateValueData{Kind: 2}})
	}
	return doc
}

func TestStateLeafProjectiles(t *testing.T) {
	for _, sp := range leafSpellings {
		t.Run(sp.name, func(t *testing.T) {
			variant := sp.name != "exact"
			doc := projectileStateFixture(variant, sp.dirKind)
			records := len(doc.State.ValueRecords)
			if got := completeSavedProjectileRecord(&doc, 7); got != sp.patched {
				t.Fatalf("complete record = %v, want %v", got, sp.patched)
			}
			current := sim.SavedProjectiles{FreeIndex: 9, IDs: []uint16{7}, Items: []sim.SavedProjectile{{ID: 7, X: 42}}}
			if err := projectProjectileCounter(&doc, current); err != nil {
				t.Fatal(err)
			}
			free := "/Projectiles/FreeIndex"
			if variant {
				free = "/Projectiles/freeindex"
			}
			if got := storedLeafValue(t, doc.State.ValueRecords, free).Int32; (got == 9) != sp.patched {
				t.Fatalf("FreeIndex = %d, patched want %v", got, sp.patched)
			}
			err := projectSavedProjectiles(&doc, current, []uint16{7})
			if !sp.patched {
				if err == nil {
					t.Fatal("sorted case variant x counted as the projectile's x leaf")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.State.ValueRecords) != records {
				t.Fatalf("records %d, want %d: a second leaf was constructed", len(doc.State.ValueRecords), records)
			}
			x := "/Prj7/x"
			if variant {
				x = "/Prj7/X"
			}
			if got := storedLeafValue(t, doc.State.ValueRecords, x).Int32; got != 42 {
				t.Fatalf("%s = %d, want 42", x, got)
			}
		})
	}
}

func TestStateLeafProjectileRetirementKeepsSortedRootSpelling(t *testing.T) {
	doc := projectileStateFixture(false, 1)
	doc.State.DirectoryRecords = append(doc.State.DirectoryRecords, sav.CityStateDirectoryData{Path: "/prj7", Kind: 1})
	doc.State.ValueRecords = append(doc.State.ValueRecords, sav.CityStateRecordData{Path: "/prj7/x", Value: sav.CityStateValueData{Kind: 2}})
	current := sim.SavedProjectiles{IDs: []uint16{}, FreeIndex: 8}
	if err := projectSavedProjectiles(&doc, current, []uint16{7}); err != nil {
		t.Fatal(err)
	}
	var roots []string
	for _, d := range doc.State.DirectoryRecords {
		roots = append(roots, d.Path)
	}
	if strings.Join(roots, ",") != "/Projectiles,/prj7" {
		t.Fatalf("retirement removed %v; the sorted root names Prj7 only", roots)
	}
}
