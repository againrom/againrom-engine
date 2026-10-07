package sav

import (
	"reflect"
	"testing"
)

func TestCityAutoHealingOptionalUpdate(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	before := p.Data()
	u := CityUpdate{Label: []byte("healing"), Money: 123, Characters: cityTestUpdates(p.Roster())}
	for _, percent := range []uint32{100, 50, 0, 95, 0xffffffff} {
		u.ManaReservePercent = &percent
		raw, err := p.Marshal(u)
		if err != nil {
			t.Fatal(err)
		}
		file, err := Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		city, err := file.CityProvenance()
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range city.Roster() {
			h, err := city.Human(c.Identity)
			if err != nil || h.ManaReservePercent != percent {
				t.Fatal("owning Player percentage", h.ManaReservePercent, err)
			}
		}
		if !reflect.DeepEqual(before, p.Data()) {
			t.Fatal("city update mutated source")
		}
	}
}
