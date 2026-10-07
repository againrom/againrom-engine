package mod

import (
	"reflect"
	"testing"
)

func TestARecordedSetReadsBackEqual(t *testing.T) {
	set := Set{Base: "rom1-en", Mods: []SetEntry{
		{ID: "a", Version: "1", Digest: "d1", Settings: []SettingValue{
			{"n", Value{Kind: KindInt, Int: -7}}, {"b", Value{Kind: KindBool, Bool: true}}, {"c", Value{Kind: KindChoice, Str: "grim"}}}},
		{ID: "b", Version: "2.0", Digest: "d2"},
	}}
	back, err := set.Record().Set()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, set) || back.Digest() != set.Digest() {
		t.Fatalf("%+v != %+v", back, set)
	}
	if got := Differences(set, back); len(got) != 0 {
		t.Fatal(got)
	}
	empty, err := Set{}.Record().Set()
	if err != nil || !empty.Empty() {
		t.Fatal(empty, err)
	}
}

func TestARecordWithAnUnreadableValueIsRefused(t *testing.T) {
	for _, v := range []SettingRecord{{"k", "int", "x"}, {"k", "bool", "maybe"}, {"k", "float", "1"}} {
		r := SetRecord{Mods: []EntryRecord{{ID: "m", Settings: []SettingRecord{v}}}}
		if _, err := r.Set(); err == nil {
			t.Errorf("accepted %+v", v)
		}
	}
}
