package sim

import "testing"

type writtenSkill struct {
	trained, skill int32
	base           [2]byte
	known          uint32
}

func writtenOf(e Entity, slot int32) writtenSkill {
	at := 2 + 2*slot
	return writtenSkill{e.NativeTraining.Levels[slot], e.Skill[slot],
		[2]byte{e.NativeBasis.Base[at], e.NativeBasis.Base[at+1]}, e.NativeBasis.BaseKnown & (3 << at)}
}

func TestReturnedAndAwardedHeroCarryTheSameWrittenFields(t *testing.T) {
	const bonus = 7
	awarded := trainingWorld(t, 10, bonus)
	awarded.entities[0].NativeBasis.BasePresent = true
	if !awarded.awardSkill(0, 0, 1_000_000, 1) {
		t.Fatal("no award")
	}
	got := writtenOf(awarded.entities[0], 3)
	if got.trained != 11 || got.skill != 11+bonus || got.known == 0 {
		t.Fatalf("awarded fields %+v", got)
	}

	returned := trainingWorld(t, 10, bonus)
	body := &returned.entities[0]
	body.NativeBasis.BasePresent = true
	body.SkillXP[3] = skillXPFor(11) + 1
	body.HP, body.CorpseDecayed = 0, true
	id := body.ID
	returned.returnHero(0, CellPoint{X: 5, Y: 5})
	i := indexOfEntity(returned.entities, id)
	if i < 0 || returned.entities[i].Dead() {
		t.Fatal("hero not returned")
	}
	h := returned.entities[i]
	held := returned.game.skillAward.heldLevel(returned.rules, h.SkillXP[3])
	want := writtenSkill{trained: held, skill: held + bonus}
	if g := writtenOf(h, 3); g.trained != want.trained || g.skill != want.skill || g.base != [2]byte{byte(held), 0} || g.known != got.known {
		t.Fatalf("returned fields %+v, want %+v with base %d", g, want, held)
	}
	if held != 11 || writtenOf(h, 3) != got {
		t.Fatalf("returned %+v differs from awarded %+v", writtenOf(h, 3), got)
	}
}
