package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
)

func cityTokenObject(class string, id uint32) sav.CityObjectData {
	token := make([]byte, 33)
	binary.LittleEndian.PutUint32(token[12:16], id)
	switch class {
	case "Item":
		return sav.CityObjectData{Class: class, Item: &sav.CityItemData{Token: token}}
	case "Effect":
		return sav.CityObjectData{Class: class, Effect: &sav.CityEffectData{Token: token}}
	}
	return sav.CityObjectData{Class: class, Unit: &sav.CityUnitData{Token: token}}
}

// The ids a town SAVE gives a new Unit were recorded from the allocator this
// one replaced; each case lists the Tokens a document carries and the ids the
// next Units take, one after another.
func TestCityRuntimeIDsStayAsTheTownSaveWroteThem(t *testing.T) {
	type tok struct {
		class string
		id    uint32
	}
	cases := []struct {
		name   string
		tokens []tok
		want   []uint32
	}{
		{"empty document", nil, []uint32{1, 2, 3}},
		{"dense units", []tok{{"Human", 1}, {"Unit", 2}, {"Unit", 3}}, []uint32{4, 5}},
		{"hole is refilled first", []tok{{"Human", 1}, {"Unit", 3}, {"Unit", 4}}, []uint32{2, 5, 6}},
		{"wide unit id also marks its low bits", []tok{{"Human", 1}, {"Unit", 0x10002}}, []uint32{3, 4}},
		{"wide item id marks nothing in the space", []tok{{"Item", 0x10001}, {"Item", 2}}, []uint32{1, 3}},
		{"item and effect ids are taken", []tok{{"Human", 1}, {"Item", 2}, {"Effect", 3}}, []uint32{4}},
		{"zero tokens leave bit zero marked", []tok{{"Human", 0}, {"Item", 0}}, []uint32{1, 2}},
		{"wide effect id below the space edge", []tok{{"Effect", 0xFFFF}, {"Effect", 0x1FFFF}}, []uint32{1, 2}},
	}
	for _, c := range cases {
		var objects []sav.CityObjectData
		objects = append(objects, sav.CityObjectData{Class: "Player"}, sav.CityObjectData{Class: "Spell"})
		for _, k := range c.tokens {
			objects = append(objects, cityTokenObject(k.class, k.id))
		}
		objects = append(objects, sav.CityObjectData{Class: "Human", Unit: &sav.CityUnitData{Token: make([]byte, 8)}})
		for i, want := range c.want {
			got, err := mintCityRuntimeID(objects)
			if err != nil || got != want {
				t.Fatalf("%s: mint %d = %d, %v; want %d", c.name, i, got, err, want)
			}
			objects = append(objects, cityTokenObject("Human", got))
		}
	}
}

func TestCityRuntimeIDsRefuseAFullSpace(t *testing.T) {
	objects := make([]sav.CityObjectData, 0, 1<<16)
	for id := uint32(1); id < 1<<16; id++ {
		objects = append(objects, cityTokenObject("Unit", id))
	}
	if id, err := mintCityRuntimeID(objects); err == nil {
		t.Fatalf("full id space gave %d", id)
	}
}
