package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

// Census row "scenario fixture": an authored unit states its cell, owner,
// group, the two health values, damage, sight, the auto-hit mark, the mana
// pair, Mind, book, autocast and turn budget; every other definition field is
// zero. A field the constructor dropped reads back zero here.
func TestScenarioFixtureActorCarriesItsCensusFields(t *testing.T) {
	spec := HeadlessWorldSpec{Width: 12, Height: 12, Units: []HeadlessUnitSpec{{
		Ref: "p0", X: 3, Y: 4, Owner: 5, Group: 6, HP: 70, MaxHP: 90, Damage: 8, ScanRange: 7, AlwaysHits: true,
		Mana: 11, MaxMana: 13, Mind: 17, KnownSpells: []uint32{2, 4}, Autocast: 4, RotationSpeed: 19,
	}}}
	w, err := spec.Build()
	if err != nil {
		t.Fatal(err)
	}
	got, _ := w.World.Entity(0)
	want := sim.ActorDefinition{HP: 70, MaxHP: 90, DamageBase: 8, ScanRange: 7, AlwaysHits: true,
		Mana: 11, MaxMana: 13, Mind: 17, KnownSpells: 1<<2 | 1<<4, AutoSpell: 4, RotationSpeed: 19,
		Reach: 1}
	at := sim.ActorPlacement{X: 3, Y: 4, Owner: 5, Group: 6}
	g := reflect.ValueOf(got)
	for _, in := range []reflect.Value{reflect.ValueOf(want), reflect.ValueOf(at)} {
		for i := 0; i < in.NumField(); i++ {
			name := in.Type().Field(i).Name
			if f := g.FieldByName(name); !reflect.DeepEqual(f.Interface(), in.Field(i).Interface()) {
				t.Errorf("%s = %+v, want %+v", name, f.Interface(), in.Field(i).Interface())
			}
		}
	}
}
