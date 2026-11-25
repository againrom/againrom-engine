package sim

import "testing"

// heroRaiseWorld is a second-game World holding one body of class typeID at
// health hp, under the build's raise or not.
func heroRaiseWorld(t *testing.T, typeID, hp int32, raise bool) *World {
	t.Helper()
	e := Entity{ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, TypeID: typeID, Decay: decayLast}
	e.SourceBinding.RuntimeID = 5
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, secondGameScript(t, nil, nil, nil), Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetBuildRules(BuildRules{HeroBodyRaise: raise})
	return w
}

// The Russian build raises a hero body from -10000..-62 to -61 on the walk,
// so ageing never removes it; the English build removes one below -600 and
// keeps one above it where it is. A body below -10000 is left (R2-ENGINE-1825).
func TestRussianBuildRaisesAHeroBodyToMinusSixtyOne(t *testing.T) {
	for _, c := range []struct {
		name        string
		typeID, hp  int32
		raise, kept bool
		want        int32
	}{
		{"raised from -500", 0x21, -500, true, true, -61},
		{"raised from -10000", 0x23, -10000, true, true, -61},
		{"consumed body", 0x24, -10001, true, false, 0},
		{"not a hero", 0x25, -500, true, true, -500},
		{"English build", 0x21, -500, false, true, -500},
		{"English build past -600", 0x22, -700, false, false, 0},
	} {
		w := heroRaiseWorld(t, c.typeID, c.hp, c.raise)
		Step(w, nil)
		got := w.Entities()
		if len(got) == 1 != c.kept {
			t.Errorf("%s: %d bodies, want kept %v", c.name, len(got), c.kept)
		} else if c.kept && (got[0].HP != c.want || got[0].Decay != decayLast) {
			t.Errorf("%s: health %d stage %d, want %d", c.name, got[0].HP, got[0].Decay, c.want)
		}
	}
}

// The raise is the build's input: a decode keeps the receiver's, and a
// raised body's health and stage survive the round trip.
func TestHeroBodyRaiseSurvivesASaveRoundTrip(t *testing.T) {
	w := heroRaiseWorld(t, 0x21, -500, true)
	Step(w, nil)
	form := mustMarshal(t, w)
	back := heroRaiseWorld(t, 0x21, -500, true)
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !back.BuildRules().HeroBodyRaise || back.Hash() != w.Hash() {
		t.Fatalf("round trip: raise %v, hash equal %v", back.BuildRules().HeroBodyRaise, back.Hash() == w.Hash())
	}
	e := back.Entities()[0]
	if e.HP != decayHeroHP || e.Decay != decayLast {
		t.Fatalf("round trip body: health %d stage %d", e.HP, e.Decay)
	}
}
