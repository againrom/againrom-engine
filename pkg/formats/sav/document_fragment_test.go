package sav

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func fragmentFixture(t *testing.T) DocumentFragment {
	t.Helper()
	actor, err := NewDocumentRecord("Human", "HasInventory", "HasSpellbook")
	if err != nil {
		t.Fatal(err)
	}
	item, _ := NewDocumentRecord("Weapon")
	effect, _ := NewDocumentRecord("Effect")
	spell, _ := NewDocumentRecord("Spell")
	for i := range actor.Texts {
		actor.Texts[i].Value = string([]byte{0xc4, 0xe0, 0xed, 0, 0xff})
	}
	setRefs := func(r *DocumentRecordData, name string, refs ...uint16) {
		for i := range r.RefSlots {
			if r.RefSlots[i].Name == name {
				r.RefSlots[i].Objects = refs
			}
		}
		for i := range r.Counts {
			if r.Counts[i].Name == name {
				r.Counts[i].Count = uint32(len(refs))
				if name == "Spells" {
					r.Counts[i].Count++
				}
			}
		}
	}
	setRefs(&actor, "HeldWeapon", 2)
	setRefs(&actor, "Inventory", 2, 2)
	setRefs(&actor, "Spells", 4)
	setRefs(&item, "Effects", 3, 3)
	setRefs(&item, "WeaponSpell", 4)
	f, _, err := ReindexDocumentFragment(DocumentFragment{Actor: 1, Objects: []DocumentRecordData{actor, item, effect, spell}})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDocumentFragmentOwnsLocalGraphAndByteText(t *testing.T) {
	f := fragmentFixture(t)
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("\\ufffd")) {
		t.Fatal("byte text was normalized by JSON")
	}
	var decoded DocumentFragment
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	clone, err := CloneDocumentFragment(decoded)
	if err != nil || !reflect.DeepEqual(f, clone) {
		t.Fatal("fragment changed fields, references or byte text", err)
	}
	clone.Objects[0].Raw[0].Bytes[0]++
	if reflect.DeepEqual(clone, f) {
		t.Fatal("fragment clone shares its raw fields")
	}
	leaf, _ := json.Marshal(struct {
		Version uint32
		Party   []DocumentFragment
	}{Version: 1, Party: []DocumentFragment{f}})
	var state DocumentStateData
	if err := SetNativeActions(&state, leaf); err != nil {
		t.Fatal(err)
	}
	if err := remapNativeActionObjects(&state, []uint16{0, 9, 8, 7, 6}); err != nil {
		t.Fatal(err)
	}
	result, _, _ := NativeActions(state)
	var values struct{ Party []DocumentFragment }
	if err := json.Unmarshal(result, &values); err != nil || !reflect.DeepEqual(values.Party, []DocumentFragment{f}) {
		t.Fatal("main document remap changed fragment namespace", err)
	}
}

func TestDocumentFragmentRejectsMalformedGraphWithoutPartialResult(t *testing.T) {
	for name, corrupt := range map[string]func(*DocumentFragment){
		"no root":           func(f *DocumentFragment) { f.Actor = 0 },
		"outside root":      func(f *DocumentFragment) { f.Actor = uint16(len(f.Objects) + 1) },
		"wrong root class":  func(f *DocumentFragment) { f.Actor = 2 },
		"orphan":            func(f *DocumentFragment) { f.Objects = append(f.Objects, f.Objects[1]) },
		"second actor":      func(f *DocumentFragment) { f.Objects = append(f.Objects, f.Objects[0]) },
		"object budget":     func(f *DocumentFragment) { f.Objects = make([]DocumentRecordData, maxDocumentDataObjects+1) },
		"outside reference": func(f *DocumentFragment) { f.Objects[0].RefSlots[0].Objects = []uint16{65535} },
		"bad child class": func(f *DocumentFragment) {
			for i := range f.Objects[0].RefSlots {
				if f.Objects[0].RefSlots[i].Name == "HeldWeapon" {
					f.Objects[0].RefSlots[i].Objects = []uint16{1}
				}
			}
		},
		"wrong raw width": func(f *DocumentFragment) { f.Objects[0].Raw[0].Bytes = nil },
		"repeated text":   func(f *DocumentFragment) { f.Objects[0].Texts = append(f.Objects[0].Texts, f.Objects[0].Texts[0]) },
		"wide scalar": func(f *DocumentFragment) {
			for i := range f.Objects[0].Values {
				if f.Objects[0].Values[i].Name == "Health" {
					f.Objects[0].Values[i].Value = 1 << 16
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := fragmentFixture(t)
			corrupt(&f)
			if got, err := CloneDocumentFragment(f); err == nil || got.Actor != 0 || got.Objects != nil {
				t.Fatal("malformed fragment returned partial records", err)
			}
		})
	}
}

func TestDocumentFragmentJSONRejectsNestedUnknownFieldsAndTrailingValues(t *testing.T) {
	f := fragmentFixture(t)
	raw, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	for name, malformed := range map[string][]byte{
		"root":             bytes.Replace(raw, []byte(`"Actor":`), []byte(`"Unknown":0,"Actor":`), 1),
		"record":           bytes.Replace(raw, []byte(`"Class":`), []byte(`"Unknown":0,"Class":`), 1),
		"text":             bytes.Replace(raw, []byte(`"Texts":[{`), []byte(`"Texts":[{"Unknown":0,`), 1),
		"scalar":           bytes.Replace(raw, []byte(`"Values":[{`), []byte(`"Values":[{"Unknown":0,`), 1),
		"trailing object":  append(bytes.Clone(raw), []byte(` {}`)...),
		"trailing null":    append(bytes.Clone(raw), []byte(` null`)...),
		"trailing garbage": append(bytes.Clone(raw), 'x'),
	} {
		t.Run(name, func(t *testing.T) {
			before, _ := json.Marshal(f)
			if bytes.Equal(raw, malformed) {
				t.Fatal("corruption did not reach its field")
			}
			if err := f.UnmarshalJSON(malformed); err == nil {
				t.Fatal("accepted malformed nested JSON")
			}
			after, _ := json.Marshal(f)
			if !bytes.Equal(before, after) {
				t.Fatal("failed JSON decode changed its receiver")
			}
		})
	}
}
