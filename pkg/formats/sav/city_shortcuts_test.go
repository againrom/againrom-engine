package sav

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// Read the physical YA1 table and pool without the semantic shortcut reader.
func rawCityShortcuts1167(t *testing.T, store []byte) CityShortcuts {
	t.Helper()
	n := int(binary.LittleEndian.Uint32(store[16:20]))
	pool := store[24+n*32+4:]
	for i := 0; i < n; i++ {
		r := store[24+32*i : 24+32*(i+1)]
		if string(bytes.TrimRight(r[16:32], "\x00")) != "Shortcuts" {
			continue
		}
		if binary.LittleEndian.Uint32(r[8:12]) != 16 || binary.LittleEndian.Uint32(r[12:16]) != 6 {
			t.Fatal("shortcut record must have kind6 and 16 payload bytes")
		}
		at := int(binary.LittleEndian.Uint32(r[4:8]))
		var got CityShortcuts
		for j := range got {
			got[j] = int32(binary.LittleEndian.Uint32(pool[at+4*j:]))
		}
		return got
	}
	t.Fatal("no raw Shortcuts record")
	return CityShortcuts{}
}

func TestCityShortcutsOptionalCurrentUpdateAndAtomicRefusal1167(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	// Deliberately distinct unrelated controller/view state must survive.
	p.state.values["/SpellBook/Pressed"] = cityStateValue{kind: 2, int32: 19}
	p.state.values["/SpellBook/IsOpen"] = cityStateValue{kind: 2, int32: 1}
	p.state.values["/View/X"] = cityStateValue{kind: 2, int32: 1234}
	before := p.Data()
	u := CityUpdate{Label: []byte("shortcuts"), Money: 123, Characters: cityTestUpdates(p.Roster())}
	baseline, err := p.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := Open(baseline)
	for _, want := range []CityShortcuts{{5, 23, -1, 0}, {-1, 23, 5, 0}, {-1, -1, -1, -1}} {
		u.Shortcuts = &want
		out, err := p.Marshal(u)
		if err != nil {
			t.Fatal(err)
		}
		back, err := Open(out)
		if err != nil {
			t.Fatal(err)
		}
		if got := rawCityShortcuts1167(t, back.Store); got != want {
			t.Fatalf("raw cells = %v, want %v", got, want)
		}
		if !bytes.Equal(base.Body, back.Body) || !bytes.Equal(base.TailRest, back.TailRest) {
			t.Fatal("shortcut update changed unrelated objects/campaign")
		}
		a, _ := parseCityState(base.Store)
		b, _ := parseCityState(back.Store)
		b.values["/SpellBook/Shortcuts"] = a.values["/SpellBook/Shortcuts"]
		if !reflect.DeepEqual(a, b) {
			t.Fatal("shortcut update changed another state-store record")
		}
		if !reflect.DeepEqual(before, p.Data()) {
			t.Fatal("successful update mutated its source document")
		}
		again, err := p.Marshal(u)
		if err != nil || !bytes.Equal(out, again) {
			t.Fatal("non-deterministic current shortcut update", err)
		}
	}
	for _, bad := range []CityShortcuts{{-2, -1, -1, -1}, {24, -1, -1, -1}, {5, 5, -1, -1}, {0, 23, 23, -1}, {2147483647, -1, -1, -1}} {
		u.Shortcuts, u.Money = &bad, 999
		u.Characters[0].Name = "must not commit"
		if out, err := p.Marshal(u); err == nil || out != nil {
			t.Fatalf("malformed cells %v emitted %d bytes: %v", bad, len(out), err)
		}
		if !reflect.DeepEqual(before, p.Data()) {
			t.Fatal("failed update mutated provenance")
		}
	}
	u = CityUpdate{Label: []byte("shortcuts"), Money: 123, Characters: cityTestUpdates(p.Roster())}
	unchanged, err := p.Marshal(u)
	if err != nil || !bytes.Equal(baseline, unchanged) {
		t.Fatal("nil update lost the original shortcut bytes after prior calls", err)
	}
}
