package sim

import (
	"reflect"
	"testing"
)

// fillDistinct sets every settable scalar inside v to a distinct nonzero value,
// walking arrays and structs, so a field the constructor drops reads back zero.
func fillDistinct(v reflect.Value, next *uint64) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*next++
		v.SetInt(int64(*next%100 + 1))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		*next++
		v.SetUint(*next%100 + 1)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			fillDistinct(v.Index(i), next)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				fillDistinct(v.Field(i), next)
			}
		}
	}
}

// TestNewActorCarriesEveryDefinitionAndPlacementField fails when NewActor
// drops a field of either input, writes it to a differently named Entity
// field, or sets an Entity field that neither input names.
func TestNewActorCarriesEveryDefinitionAndPlacementField(t *testing.T) {
	var d ActorDefinition
	var at ActorPlacement
	var n uint64
	fillDistinct(reflect.ValueOf(&d).Elem(), &n)
	fillDistinct(reflect.ValueOf(&at).Elem(), &n)
	e := NewActor(d, at)
	got := reflect.ValueOf(e)
	named := map[string]bool{"DesiredFacing": true}
	for _, in := range []reflect.Value{reflect.ValueOf(d), reflect.ValueOf(at)} {
		for i := 0; i < in.NumField(); i++ {
			name := in.Type().Field(i).Name
			named[name] = true
			f := got.FieldByName(name)
			if !f.IsValid() {
				t.Errorf("%s has no Entity field of the same name", name)
				continue
			}
			if !reflect.DeepEqual(f.Interface(), in.Field(i).Interface()) {
				t.Errorf("%s = %v, want %v", name, f.Interface(), in.Field(i).Interface())
			}
		}
	}
	if e.DesiredFacing != at.Facing {
		t.Errorf("DesiredFacing = %d, want the facing %d", e.DesiredFacing, at.Facing)
	}
	for i := 0; i < got.NumField(); i++ {
		name := got.Type().Field(i).Name
		if !named[name] && !got.Field(i).IsZero() {
			t.Errorf("%s = %v; NewActor sets a field no input names", name, got.Field(i).Interface())
		}
	}
}

// The Control Spirit raise: census row "ghost". Every template field the raise
// reads, the seven corpse stores, the caster's owner and group, the corpse's
// cell and facing, guard at that cell; every other definition field zero.
func TestRaisedGhostCarriesItsCensusFields(t *testing.T) {
	g := hlGhostTemplate()
	g.Reach, g.TokenSize = 0, 0
	g.Humanoid = true
	g.NativeBasis = NativeActorBasis{}.WithBody(77)
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner, caster.Group = 7, 9
	corpse := spEnt(3, 2, 2)
	corpse.Reaction, corpse.Mind, corpse.Spirit, corpse.MaxHP = 41, 37, 29, 90
	corpse.ToHit, corpse.Defence, corpse.Facing, corpse.DesiredFacing = 71, 73, 5, 5
	w := hlGhostWorld(t, 30, []SpellRule{csRule()}, g, caster, corpse)
	got, ok := w.raisedGhost(0, 1, 12)
	if !ok {
		t.Fatal("template not raisable")
	}
	want := NewActor(ActorDefinition{}, ActorPlacement{})
	want.ID, want.X, want.Y, want.PostX, want.PostY = 12, 2, 2, 2, 2
	want.Facing, want.DesiredFacing, want.Owner, want.Group = 5, 5, 7, 9
	want.ActorState = actorStateGuard
	want.HP, want.MaxHP = 45, 45
	want.Reaction, want.Mind, want.Spirit, want.ToHit, want.Defence = 21, 37, 29, 71, 73
	want.Class, want.TypeID, want.Domain, want.Speed, want.RotationSpeed = g.Class, g.TypeID, g.Domain, g.Speed, g.RotationSpeed
	want.ScanRange, want.Reach, want.TokenSize, want.DyingTime, want.XPValue = g.ScanRange, 1, 1, g.DyingTime, g.XPValue
	want.Withdraw, want.Wimpy, want.Humanoid, want.NativeBasis = g.Withdraw, g.Wimpy, true, g.NativeBasis
	want.Protection, want.Resistance, want.XPSlot, want.Absorption = g.Protection, g.Resistance, g.XPSlot, g.Absorption
	want.DamageBase, want.DamageSpread, want.AttackCharge, want.AttackRelax, want.AlwaysHits = g.DamageBase, g.DamageSpread, g.AttackCharge, g.AttackRelax, g.AlwaysHits
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("raised ghost\n got %+v\nwant %+v", got, want)
	}
}
