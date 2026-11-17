package sim

import "testing"

// A source-class hero's statistic potion gains only up to the class-and-sex
// cap of the second game; the first game's rule is unchanged.
func TestSourcePotionStopsAtTheClassCap(t *testing.T) {
	for _, c := range []struct {
		name   string
		rules  Rules
		stored uint16
		want   uint16
	}{
		{"second game at the cap", Rules{}.WithClassStatCaps(), 52, 52},
		{"second game under the cap", Rules{}.WithClassStatCaps(), 50, 52},
		{"first game", Rules{}, 50, 55},
	} {
		w := cheatWorld(t)
		w.SetRules(c.rules)
		w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, Source: SourceActor{Class: 2, Fighter: true, TypeID: 0x21}}
		w.entities[0].ActorLoad.Source.Stats[0] = c.stored
		w.entities[0].Capacity = 11
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) { return s, nil })
		if !w.sourcePotion(0, 2, 5) {
			t.Fatalf("%s: potion refused", c.name)
		}
		if got := w.entities[0].ActorLoad.Source.Stats[0]; got != c.want {
			t.Errorf("%s: Body %d, want %d", c.name, got, c.want)
		}
	}
}
