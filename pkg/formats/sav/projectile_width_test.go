package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func projectileWidthStore(ids []uint32, leaves map[string]uint32, free uint32) []byte {
	dirs := literalWorldDirectories(ids...)
	for i := range dirs {
		if dirs[i].name == "Projectiles" {
			dirs[i].leaves[0].word = free
		}
		for j := range dirs[i].leaves {
			if w, ok := leaves[dirs[i].name+"/"+dirs[i].leaves[j].name]; ok {
				dirs[i].leaves[j].word = w
			}
		}
	}
	return literalStateStore(dirs)
}

// SAV-1322: LOAD keeps the low byte of action and actiondir and the low word
// of actiontarget; SAVE writes the bytes sign-extended and the word
// zero-extended. SAV-1191: FreeIndex is a word.
func TestProjectileRecordWidthsAtLoadAndSave(t *testing.T) {
	wide := map[string]uint32{"Prj5/action": 0x1fe, "Prj5/actiondir": 0x180, "Prj5/actiontarget": 0x1f00d, "Prj5/actionx": 0x12345678}
	raw := projectileWidthStore([]uint32{5}, wide, 0x10003)
	want := map[string]uint32{"/Prj5/action": 0xfffffffe, "/Prj5/actiondir": 0xffffff80, "/Prj5/actiontarget": 0xf00d, "/Prj5/actionx": 0x12345678, "/Projectiles/FreeIndex": 3}
	source, _ := literalReadState(raw)
	for path, w := range want {
		if path != "/Prj5/actionx" && source[path].word == w {
			t.Fatalf("loss control: the input already holds %s = %#x", path, w)
		}
	}
	f := &File{}
	f.splitTail(raw)
	store, ok, err := f.Projectiles()
	if err != nil || !ok || len(store.Items) != 1 {
		t.Fatal(store, ok, err)
	}
	p := store.Items[0]
	if p.Action != -2 || p.ActionDir != -128 || p.ActionTarget != 0xf00d || p.ActionX != 0x12345678 || store.FreeIndex != 3 {
		t.Fatalf("LOAD kept %+v free %d", p, store.FreeIndex)
	}
	state, err := parseWorldState(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := serializeWorldState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := literalReadState(encoded)
	for path, w := range want {
		if got[path].word != w {
			t.Fatalf("Document SAVE wrote %s = %#x, want %#x", path, got[path].word, w)
		}
	}
	file := open(t, func() fixture { fx := standard(); fx.tail = literalStateStore(literalWorldDirectories()); return fx }())
	item := Projectile{ID: 5, Action: 0x1fe, ActionDir: 0x180, ActionTarget: 0x1f00d, ActionX: 0x12345678}
	if err := file.SetProjectiles(ProjectileStore{FreeIndex: 3, IDs: []uint16{5}, Items: []Projectile{item}}); err != nil {
		t.Fatal(err)
	}
	written, _ := literalReadState(file.Store)
	for path, w := range want {
		if written[path].word != w {
			t.Fatalf("SetProjectiles wrote %s = %#x, want %#x", path, written[path].word, w)
		}
	}
}

// SAV-1322 and SAV-PROJLOAD-429: LOAD builds one record per IDs element, a
// repeated ID replaces the earlier record in the one map SAVE walks, and
// only the low word of each element is kept.
func TestProjectileRepeatedIDLoadsOneRecordAndSavesOnce(t *testing.T) {
	ids := []uint32{5, 0x10005, 9, 5}
	raw := projectileWidthStore(ids, nil, 10)
	source, _ := literalReadState(raw)
	if len(source["/Projectiles/IDs"].data) != 16 {
		t.Fatal("loss control: the input does not repeat an ID")
	}
	f := &File{}
	f.splitTail(raw)
	store, _, err := f.Projectiles()
	if err != nil || !reflect.DeepEqual(store.IDs, []uint16{5, 9}) || len(store.Items) != 2 || store.Items[0].ID != 5 || store.Items[1].ID != 9 {
		t.Fatal("LOAD", store, err)
	}
	state, err := parseWorldState(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := serializeWorldState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := literalReadState(encoded)
	words := []byte(got["/Projectiles/IDs"].data)
	if len(words) != 8 || binary.LittleEndian.Uint32(words) != 5 || binary.LittleEndian.Uint32(words[4:]) != 9 {
		t.Fatalf("SAVE wrote IDs %x, want 5 and 9 once each", words)
	}
	if _, ok := got["/Prj5/x"]; !ok {
		t.Fatal("SAVE lost the repeated ID's record")
	}
}
