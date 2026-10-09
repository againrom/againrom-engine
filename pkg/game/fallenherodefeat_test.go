package game

import (
	"fmt"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// fallenHeroEntity is a human player character at full health with the given
// dying time: 0 for a body with no dwell, 8 for a hero's Humans row.
func fallenHeroEntity(id sim.EntityID, x, y, dyingTime int32) sim.Entity {
	return sim.Entity{ID: id, Owner: sim.SelfSlot, X: x, Y: y, HP: 40, MaxHP: 40, DyingTime: dyingTime,
		TypeID: sim.HumanTypeID, Humanoid: true}
}

// The K key is the fall: mapWorld.affect issues sim.Kill, which leaves health
// -1. The body then lies through its dwell and the corpse walk until -10.
// Every tick of that lie with health above -10 is healable and is not a loss;
// the tick the walk reaches -10 is.
func TestAFallenHeroLosesTheMissionOnlyWhenHeCanNoLongerBeHealed(t *testing.T) {
	for _, dyingTime := range []int32{0, 8} {
		t.Run(fmt.Sprintf("dying_time_%d", dyingTime), func(t *testing.T) {
			w := heroWorld(t, nil, nil, []sim.Entity{fallenHeroEntity(1, 20, 20, dyingTime)})
			mw := grabWorld(t, w, 1, missionSource{})

			mw.affect(1, true)
			mw.tick()
			e, ok := mw.entity(1)
			if !ok || e.Alive() || e.Dying() != (dyingTime > 0) || e.HP != -1 {
				t.Fatalf("setup: K left hero present=%v alive=%v dying=%v hp=%d, want a fallen body at -1",
					ok, e.Alive(), e.Dying(), e.HP)
			}
			afterWindow := 0
			for tick := 1; ; tick++ {
				if tick > 2000 {
					t.Fatalf("hero still at health %d after %d ticks; the corpse walk never reached -10", e.HP, tick)
				}
				if e.HP <= -10 {
					break
				}
				if mw.mission.announced {
					t.Fatalf("settleNotices -> guardedCharacterLost announced outcome %v on tick %d while the hero lay "+
						"at health %d with dwell %d: he can still be healed", mw.mission.outcome, tick, e.HP, e.Dwell)
				}
				if !e.Dying() {
					afterWindow++
				}
				mw.tick()
				if e, ok = mw.entity(1); !ok {
					t.Fatalf("hero removed on tick %d before his body reached -10", tick)
				}
			}
			if afterWindow < 200 {
				t.Fatalf("only %d ticks after the dying window were observed; the control does not reach past it", afterWindow)
			}
			if !mw.mission.announced || mw.mission.outcome != sim.OutcomeLost {
				t.Fatalf("hero reached health %d: announced=%v outcome=%v, want the loss on that tick",
					e.HP, mw.mission.announced, mw.mission.outcome)
			}
			t.Logf("fallen at -1, %d healable ticks after the dying window, lost at health %d on world tick %d",
				afterWindow, e.HP, mw.world.Tick())
		})
	}
}

// A companion mage heals the fallen hero after his dying window has closed.
// The mage's mana equals the Heal cost, below the idle-heal reserve, so only
// the ordered cast can raise him. Raised, he takes a move order and walks.
func TestAHeroHealedAfterHisDyingWindowPlaysOn(t *testing.T) {
	s, err := sim.NewScript(missionChecks(), nil, nil)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	mage := fallenHeroEntity(2, 22, 20, 8)
	mage.HP, mage.MaxHP, mage.TypeID = 30, 30, sim.HeroTypeID(true, false)
	mage.Mind, mage.MaxMana, mage.Mana, mage.KnownSpells, mage.ScanRange = 30, 100, 10, 1<<6, 19
	heal := sim.SpellRule{ID: 6, ManaCost: 10, School: 5, MaxRange: 6,
		DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true}
	w, err := sim.NewSpelledWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical,
		make([]byte, worldFixtureW*worldFixtureH), []sim.Entity{fallenHeroEntity(1, 20, 20, 8), mage}, s,
		[]sim.SpellRule{heal})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	mw := partyWorld(t, w, []mapload.PartyMember{
		{ID: "hero", PlayerCharacter: true, StartingHero: true},
		{ID: "npc:22", PlayerCharacter: true},
	}, []sim.EntityID{1, 2})

	mw.affect(1, true)
	for tick := 0; tick < 40; tick++ {
		mw.tick()
		if mw.mission.announced {
			hero, _ := mw.entity(1)
			t.Fatalf("settleNotices -> guardedCharacterLost announced outcome %v on tick %d while the hero lay "+
				"at health %d with dwell %d: he can still be healed", mw.mission.outcome, tick, hero.HP, hero.Dwell)
		}
	}
	hero, _ := mw.entity(1)
	if hero.Alive() || hero.Dying() || hero.HP <= -10 {
		t.Fatalf("setup: hero alive=%v dying=%v hp=%d, want a healable body past its dwell", hero.Alive(), hero.Dying(), hero.HP)
	}

	mw.attackOrCast(2, 1, 6, 0, 0, false)
	for tick := 0; tick < 400 && !hero.Alive(); tick++ {
		mw.tick()
		hero, _ = mw.entity(1)
	}
	if !hero.Alive() || hero.Decay != sim.DecayNone {
		t.Fatalf("ordered Heal did not raise the hero: hp=%d decay=%d", hero.HP, hero.Decay)
	}
	if caster, _ := mw.entity(2); caster.Mana != 0 {
		t.Fatalf("mage mana %d after the heal, want the one ordered cast's 10 spent", caster.Mana)
	}
	if mw.mission.announced {
		t.Fatalf("healed hero's mission announced outcome %v", mw.mission.outcome)
	}
	t.Logf("healed from a past-dwell body to health %d on world tick %d", hero.HP, mw.world.Tick())

	from := hero.X
	mw.enqueue(1, 30, 20)
	for tick := 0; tick < 400; tick++ {
		mw.tick()
	}
	hero, _ = mw.entity(1)
	if hero.X <= from || !hero.Alive() || mw.mission.announced {
		t.Fatalf("healed hero did not play on: x %d -> %d alive=%v announced=%v", from, hero.X, hero.Alive(), mw.mission.announced)
	}
}
