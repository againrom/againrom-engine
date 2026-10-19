package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestSpellInnerEffectSingleLandingLifetime(t *testing.T) {
	for _, tc := range []struct {
		name      string
		id        int
		kind      data.SpellEffectKind
		mode      data.SpellEffectMode
		raw, want uint16
	}{
		{"Freezing", 7, data.SpellEffectSpeed, data.SpellEffectDuration, 20, 320},
		{"Light", 12, data.SpellEffectScanRange, data.SpellEffectDuration, 1, 16},
		{"Darkness", 17, data.SpellEffectScanRange, data.SpellEffectDuration, 1, 16},
		{"Poison", 8, data.SpellEffectHealth, data.SpellEffectContinuous, 8, 128},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := make([]data.Spell, tc.id)
			rows[tc.id-1] = data.Spell{TargetsUnit: true, MaxRange: 8, Effect: data.SpellEffect{Kind: tc.kind, Mode: tc.mode, Magnitude: -2, Duration: tc.raw}}
			rule := spellRules(rows, false)[tc.id-1]
			for _, loss := range []bool{false, true} {
				r := rule
				if loss {
					r.EffectDuration = tc.raw
				}
				caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, Mind: 30, KnownSpells: 1 << tc.id, ScanRange: 10, Speed: 10, TokenSize: 1}
				target := caster
				target.ID = 2
				target.X = 3
				w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster, target}, nil, sim.Relations{}, nil, nil, []sim.SpellRule{r})
				if err != nil {
					t.Fatal(err)
				}
				commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(tc.id))}
				var remaining uint16
				for tick := 0; tick < 64; tick++ {
					sim.Step(w, commands)
					commands = nil
					for _, e := range w.ActiveEffects() {
						if e.Target == 2 {
							remaining = e.Remaining
						}
					}
					if remaining > 0 {
						break
					}
				}
				want := tc.want
				if loss {
					want = tc.raw
				}
				if remaining != want {
					t.Fatalf("loss=%v: fresh landing %d, want %d", loss, remaining, want)
				}
				if loss && remaining == tc.want {
					t.Fatal("loss control retained corrected counter")
				}
				for tick := uint16(1); tick <= want; tick++ {
					sim.Step(w, nil)
					if got := w.HasEffectSpell(2, uint16(tc.id)); got != (tick < want) {
						t.Fatalf("loss=%v: tick %d/%d attached=%v", loss, tick, want, got)
					}
				}
			}
		})
	}
}

func TestSpellNumericEffectCountersFollowMode(t *testing.T) {
	for _, mode := range []data.SpellEffectMode{data.SpellEffectDuration, data.SpellEffectContinuous, data.SpellEffectCharges, data.SpellEffectSingleUse, 0} {
		rows := []data.Spell{{Effect: data.SpellEffect{Kind: data.SpellEffectSpeed, Mode: mode, Magnitude: -3, Duration: 4097}}}
		got := spellRules(rows, false)[0]
		want := uint16(4097)
		if mode&(data.SpellEffectDuration|data.SpellEffectContinuous|data.SpellEffectCharges) != 0 {
			want = 16
		}
		if got.EffectDuration != want || got.EffectMagnitude != -3 || got.EffectMode != sim.EffectMode(mode) {
			t.Fatalf("mode %d: %+v, want counter %d", mode, got, want)
		}
	}
}

func TestSecondGameSightCountersKeepBaseDuration(t *testing.T) {
	for _, id := range []int{14, 15} {
		rows := make([]data.Spell, 15)
		rows[id-1] = data.Spell{TargetsUnit: true, MaxRange: 8, Effect: data.SpellEffect{Kind: data.SpellEffectScanRange, Mode: data.SpellEffectDuration, Magnitude: -2, Duration: 1}}
		for _, loss := range []bool{false, true} {
			rules := spellRules(rows, !loss)
			SecondGameSpellArms(rules)
			r := rules[id-1]
			want := uint16(1)
			if loss {
				want = 16
			}
			if !r.Second || r.EffectDuration != want {
				t.Fatalf("row %d loss=%v: counter %d, want %d", id, loss, r.EffectDuration, want)
			}
			caster := sim.Entity{ID: 1, X: 2, Y: 2, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, Mind: 30, KnownSpells: 1 << id, ScanRange: 10, TokenSize: 1}
			target := caster
			target.ID, target.X = 2, 3
			w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster, target}, nil, sim.Relations{}, nil, nil, []sim.SpellRule{r})
			if err != nil {
				t.Fatal(err)
			}
			commands := []sim.Command{sim.Cast(1, 2, sim.SpellID(id))}
			var remaining uint16
			for tick := 0; tick < 64 && remaining == 0; tick++ {
				sim.Step(w, commands)
				commands = nil
				for _, e := range w.ActiveEffects() {
					if e.Target == 2 {
						remaining = e.Remaining
					}
				}
			}
			if remaining != want {
				t.Fatalf("row %d loss=%v: landing %d, want %d", id, loss, remaining, want)
			}
			for tick := uint16(1); tick <= want; tick++ {
				sim.Step(w, nil)
				if got := w.HasEffectSpell(2, uint16(id)); got != (tick < want) {
					t.Fatalf("row %d loss=%v tick %d: attached=%v", id, loss, tick, got)
				}
			}
		}
	}
}
