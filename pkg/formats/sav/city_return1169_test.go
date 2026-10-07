package sav

import (
	"reflect"
	"testing"
)

func TestCityReturn1169WritesCurrentTimingAndResiduesWithoutRebindingSource(t *testing.T) {
	f, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	before := p.Data()
	updates := cityTestUpdates(p.Roster())
	h, err := p.Human(updates[0].Identity)
	if err != nil {
		t.Fatal(err)
	}
	h.Fields.Runtime = &CityHumanRuntime{Reach: 9, AttackCharge: 23, AttackRelax: 31, HealthHundredths: 157, ManaHundredths: 255}
	updates[0].Human, updates[0].Returned = &h.Fields, true
	raw, err := p.Marshal(CityUpdate{Characters: updates})
	if err != nil {
		t.Fatal(err)
	}
	out, err := Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, _, err := out.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	var found *Record
	for _, r := range doc.players[0].Refs["Actors"] {
		if r != nil && r.Text["Name"] == updates[0].Name {
			found = r
			break
		}
	}
	if found == nil {
		t.Fatal("output actor absent")
	}
	for name, want := range map[string]uint32{"U12C": 9, "U134": 23, "U135": 31, "UA2": 157, "UA3": 255, "Stage": 0, "U5C": 0, "U64": 0, "U44": 0, "U40": 0} {
		if found.value(name) != want {
			t.Fatal(name, found.value(name), want)
		}
	}
	if !reflect.DeepEqual(p.Data(), before) {
		t.Fatal("writer changed immutable source")
	}
}
