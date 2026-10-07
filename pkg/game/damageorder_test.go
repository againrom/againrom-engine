package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestOrderedHurtAppRouteKeepsEveryZeroAcrossCatchUp(t *testing.T) {
	hero := sim.Entity{ID: 0, X: 4, Y: 4, Owner: sim.SelfSlot, TypeID: sim.HeroTypeID(false, false), Class: 14, Humanoid: true, HP: 100, MaxHP: 100}
	mw, app, rec := hurtVoiceFight(t, hero, figureID{Dir: data.FigureDirManFighter, Hero: true}, 0, 1)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	rec.plays = nil
	before := mw.strikes[0]
	mw.pending = append(mw.pending, sim.Damage(0, 40), sim.Damage(0, 40))
	for range 12 {
		mw.tick()
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	zeros := int(mw.strikes[0] - before)
	if zeros < 2 {
		t.Fatalf("fixture produced only %d zeros", zeros)
	}
	var wounds, strikes int
	for _, play := range rec.plays {
		switch play.src {
		case "mf_hero/easy.wav":
			wounds++
		case "class slot 211":
			strikes++
		default:
			t.Fatalf("unexpected source %+v", play)
		}
	}
	if wounds != 1 || strikes != zeros {
		t.Fatalf("wounds=%d zeros delivered=%d produced=%d; %+v", wounds, strikes, zeros, rec.plays)
	}
	got := slices.Clone(rec.plays)
	mw.push()
	mw.stopped = true
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rec.plays, got) {
		t.Fatal("snapshot replayed messages")
	}
}

func TestDamageMessagesPreserveFallenJoltAdmission(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after int32
		jolt          bool
	}{
		{"restoration", 0, 3, false},
		{"terminal loss", -10, -11, false},
		{"terminal equality", -10, -10, false},
		{"finishing loss", -9, -10, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			world, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil, []sim.Entity{{ID: 7, X: 2, Y: 2, HP: test.after, MaxHP: 20}})
			if err != nil {
				t.Fatal(err)
			}
			mw := &mapWorld{world: world}
			mw.observeDamage([]sim.DamageEvent{{Target: 7, BeforeHP: test.before, AfterHP: test.after}})
			if len(mw.pendingDamage) != 1 {
				t.Fatal("jolt admission discarded a causal message")
			}
			if got := len(mw.hurt) != 0; got != test.jolt {
				t.Fatalf("jolt admitted=%t, want %t", got, test.jolt)
			}
		})
	}
}
