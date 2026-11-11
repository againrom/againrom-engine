package sim

import (
	"encoding/binary"
	"testing"
)

// The second game's skill award (R2-ENGINE-480, R2-ENGINE-486, R2-ENGINE-487,
// R2-ENGINE-488): X = trunc(V x 0.5 x D / HPmax + 1), scaled by Mind/30 + 0.25,
// ceil(X / 8) into a skill that is neither the main skill nor skill 5, capped
// at T(L+1) - T(L), and a raise when E reaches T(L+1).

// awardXP is T(level) of the default rules.
func awardXP(level int32) int32 { return Rules{}.SkillXP(level) }

// awardHero is a player hero of the given class whose trained levels are
// levels, each holding the experience of its level, against no worn bonus.
func awardHero(fighter bool, mind int32, xpSlot uint8, levels [skillSlots]int32) Entity {
	e := cbEnt(1, 0, 0)
	e.Owner, e.GainsXP, e.Mind, e.XPSlot, e.TypeID, e.Humanoid = SelfSlot, true, mind, xpSlot, HumanTypeID, true
	e.NativeClass = NativeClass{Present: true, Fighter: fighter}
	e.NativeTraining = NativeTraining{Present: true, Levels: levels}
	e.Skill = levels
	for s, level := range levels {
		e.SkillXP[s] = awardXP(level)
	}
	return e
}

// awardVictim is a computer player's creature of value 100 and 100 health.
func awardVictim(owner uint32) Entity {
	e := cbEnt(2, 1, 0)
	e.Owner, e.XPValue = owner, 100
	return e
}

// awardWorld is a world of the hero and the victim under game d's profile.
func awardWorld(t *testing.T, d ScriptDialect, hero, victim Entity) *World {
	t.Helper()
	w := cbWorld(t, 1, hero, victim)
	w.game = newGameProfile(d)
	return w
}

// chosenLevels is a generated hero's: the chosen skill at 20, skill 5 at 10.
func chosenLevels(chosen int) [skillSlots]int32 {
	var l [skillSlots]int32
	l[5] = 10
	l[chosen] = 20
	return l
}

func TestSecondGameBlowPaysTheMainSkillWholeAndAnotherAnEighth(t *testing.T) {
	// D 50 of HPmax 100 at V 100: X = 26. Mind 24: trunc(26 x 1.05) = 27;
	// Mind 60: trunc(26 x 2.25) = 58.
	for _, c := range []struct {
		name          string
		dialect       ScriptDialect
		mind          int32
		slot          uint8
		want, wantLvl int32
	}{
		{"second main mind 24", ScriptROM2, 24, 1, 27, 20},
		{"second main mind 60", ScriptROM2, 60, 1, 58, 20},
		{"second other mind 24", ScriptROM2, 24, 3, 4, 0},
		{"second other mind 60", ScriptROM2, 60, 3, 8, 0},
		{"second skill 5 mind 24", ScriptROM2, 24, 5, 27, 10},
		{"second skill 5 mind 60", ScriptROM2, 60, 5, 58, 10},
		{"first main mind 24", ScriptROM1, 24, 1, 27, 21},
		{"first other mind 60", ScriptROM1, 60, 3, 58, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			levels := chosenLevels(1)
			w := awardWorld(t, c.dialect, awardHero(true, c.mind, c.slot, levels), awardVictim(3))
			w.payExperience(0, 1, 50, w.paidBeforeBlow(w.entities[1]))
			a := w.entities[0]
			if got := a.SkillXP[c.slot] - awardXP(levels[c.slot]); got != c.want {
				t.Errorf("slot %d gained %d, want %d", c.slot, got, c.want)
			}
			if got := a.TrainedSkills([skillSlots]int32{})[c.slot]; got != c.wantLvl {
				t.Errorf("slot %d level %d, want %d", c.slot, got, c.wantLvl)
			}
		})
	}
}

func TestSecondGameBlowGates(t *testing.T) {
	levels := chosenLevels(1)
	t.Run("victim at zero health", func(t *testing.T) {
		v := awardVictim(3)
		v.HP = 0
		w := awardWorld(t, ScriptROM2, awardHero(true, 24, 1, levels), v)
		w.payExperience(0, 1, 50, w.paidBeforeBlow(w.entities[1]))
		if got := w.entities[0].SkillXP[1]; got != awardXP(20) {
			t.Errorf("a victim at zero health paid; experience %d", got)
		}
	})
	t.Run("player-owned victim", func(t *testing.T) {
		v := awardVictim(SelfSlot)
		h := awardHero(true, 24, 1, levels)
		h.Owner = 2
		w := awardWorld(t, ScriptROM2, h, v)
		w.payExperience(0, 1, 50, true)
		if got := w.entities[0].SkillXP[1]; got != awardXP(20) {
			t.Errorf("a human player's victim paid; experience %d", got)
		}
	})
	t.Run("attacker without owner", func(t *testing.T) {
		h := awardHero(true, 24, 1, levels)
		h.Owner = 0
		w := awardWorld(t, ScriptROM2, h, awardVictim(3))
		w.payExperience(0, 1, 50, true)
		if got := w.entities[0].SkillXP[1]; got != awardXP(20) {
			t.Errorf("an attacker with no owner was paid; experience %d", got)
		}
	})
}

func TestSecondGameSpellDamagePaysTheMageBySphere(t *testing.T) {
	// The same X = 26 as the blow, Mind 24: 27 on the main Sphere, 4 on another.
	for _, c := range []struct {
		name    string
		dialect ScriptDialect
		fighter bool
		school  uint8
		slot    int
		want    int32
	}{
		{"main sphere", ScriptROM2, false, 1, 1, 27},
		{"other sphere", ScriptROM2, false, 2, 2, 4},
		{"sphere 5", ScriptROM2, false, 5, 5, 27},
		{"fighter", ScriptROM2, true, 2, 3, 0},
		{"first-game fighter", ScriptROM1, true, 2, 3, 27},
	} {
		t.Run(c.name, func(t *testing.T) {
			levels := chosenLevels(1)
			w := awardWorld(t, c.dialect, awardHero(c.fighter, 24, 3, levels), awardVictim(3))
			w.awardSpellDamage(0, 1, SpellRule{School: c.school}, 50)
			if got := w.entities[0].SkillXP[c.slot] - awardXP(levels[c.slot]); got != c.want {
				t.Errorf("slot %d gained %d, want %d", c.slot, got, c.want)
			}
		})
	}
}

func TestSecondGameCastPaymentOffMainCreditsOne(t *testing.T) {
	// R2-ENGINE-487: cost 10 offers 5; Mind 24 keeps 5; another Sphere takes
	// ceil(5 / 8) = 1.
	w := awardWorld(t, ScriptROM2, awardHero(false, 24, 0, chosenLevels(1)), awardVictim(3))
	w.awardSkill(0, 2, (10+1)/2, -1)
	if got := w.entities[0].SkillXP[2]; got != 1 {
		t.Errorf("cast payment credited %d, want 1", got)
	}
}

func TestSecondGameAwardCapAndRaise(t *testing.T) {
	// X = 5001 at Mind 24 is 5251; a level 0 skill takes T(1) - T(0) = 100,
	// reaches T(1) and rises to 1. Another skill: ceil(5251 / 8) = 657, capped
	// at 100 as well.
	for _, slot := range []uint8{1, 2} {
		var levels [skillSlots]int32
		w := awardWorld(t, ScriptROM2, awardHero(true, 24, slot, levels), awardVictim(3))
		w.awardSkill(0, 0, 5001, 1)
		a := w.entities[0]
		if a.SkillXP[slot] != 100 || a.TrainedSkills([skillSlots]int32{})[slot] != 1 {
			t.Errorf("slot %d: experience %d level %d, want 100 and 1", slot, a.SkillXP[slot], a.TrainedSkills([skillSlots]int32{})[slot])
		}
	}
	// One short of T(21) leaves level 20; reaching it raises.
	levels := chosenLevels(1)
	h := awardHero(true, 0, 1, levels)
	h.SkillXP[1] = awardXP(21) - 2
	w := awardWorld(t, ScriptROM2, h, awardVictim(3))
	w.awardSkill(0, 0, 4, 1) // Mind 0: trunc(4 x 0.25) = 1
	if got := w.entities[0].TrainedSkills([skillSlots]int32{})[1]; got != 20 {
		t.Fatalf("one short of T(21) raised to %d", got)
	}
	w.awardSkill(0, 0, 4, 1)
	if got := w.entities[0].TrainedSkills([skillSlots]int32{})[1]; got != 21 {
		t.Errorf("reaching T(21) left level %d, want 21", got)
	}
}

func TestSecondGameSourceActorAward(t *testing.T) {
	// The first-game fixture of the source sink: base 9 in slot 3 holding
	// T(9) = 1357, Mind 60, offered 100: 225. Under the second game the main
	// skill takes 225 and does not reach T(10) = 1593; with slot 1 at base 20
	// slot 3 is off the main skill and takes ceil(225 / 8) = 29.
	for _, c := range []struct {
		name  string
		main1 bool
		want  uint32
	}{
		{"main", false, 1582},
		{"other", true, 1386},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := cbWorld(t, 1, skAwarder(1, 1), skSource(2, 2))
			w.game = newGameProfile(ScriptROM2)
			s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 60, 10, 18, 0, 0, 101, 100, 100, 100, 0, 0, 50}, Experience: 2300}
			s.Attack[16] = 3
			binary.LittleEndian.PutUint16(s.Base[8:], 9)
			if c.main1 {
				binary.LittleEndian.PutUint16(s.Base[4:], 20)
			}
			s.SkillXP[3] = 1357
			load := ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}
			if err := w.RestoreActorLoad(1, load); err != nil {
				t.Fatal(err)
			}
			w.BindSourceDerive(func(n SourceActor, _ int32, _ Rules) (SourceActor, error) { return n, nil })
			if w.awardSkill(0, 0, 100, 1) {
				t.Fatal("the award raised slot 3")
			}
			got := w.entities[0].ActorLoad.Source
			if got.SkillXP[3] != c.want || binary.LittleEndian.Uint16(got.Base[8:]) != 9 {
				t.Errorf("slot 3 experience %d base %d, want %d and 9", got.SkillXP[3], binary.LittleEndian.Uint16(got.Base[8:]), c.want)
			}
		})
	}
}
