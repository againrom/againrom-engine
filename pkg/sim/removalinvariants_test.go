package sim

import "testing"

func TestABodyRemovedWhileCreditedLeavesAWorldTheByteFormCanCarry(t *testing.T) {
	rule := SpellRule{ID: 20, School: 4}
	caster := killCreditMage(1, rule.School)
	caster.Domain, caster.DyingTime = DomainAir, 1
	victim := killCreditVictim(2, 100)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
	w.pointAttribution(0, 1, rule)
	if !spAt(t, w, 2).HasKillCredit {
		t.Fatal("fixture installed no kill credit on the victim")
	}

	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	if indexOfEntity(w.entities, 1) >= 0 {
		t.Fatal("the credited source was not removed")
	}

	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("a world whose credited source was removed will not load back: %v", err)
	}
}

// TestRaisingACorpseKeepsEveryLaterBodysOwnItems covers the second removal in
// this package. spell.go's raise arm takes the corpse out of w.entities and
// w.routes by hand and leaves w.carried and w.equipment at their old
// alignment, so every record AFTER the corpse is paired with the container and
// the worn set of the record before it.
//
// The bystander is placed after the corpse deliberately: a corpse that is the
// last record shifts nothing, which is why the shipped raise tests do not see
// this.
func TestRaisingACorpseKeepsEveryLaterBodysOwnItems(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 1)
	bystander := spEnt(3, 3, 1)
	tmpl := hlGhostTemplate()
	w := hlGhostWorld(t, 30, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		tmpl, caster, corpse, bystander)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
	w.carried[2] = []ItemStack{{Code: 4242, Count: 3}}
	w.equipment[2] = [EquipSlots]ItemInstance{1: PlainItem(777)}

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})

	if len(w.carried) != len(w.entities) || len(w.equipment) != len(w.entities) {
		t.Fatalf("after the raise: %d entities, %d containers, %d worn sets",
			len(w.entities), len(w.carried), len(w.equipment))
	}
	i := indexOfEntity(w.entities, 3)
	if i < 0 {
		t.Fatal("the bystander was removed by a raise that did not name it")
	}
	if got := w.carried[i]; len(got) != 1 || got[0].Code != 4242 || got[0].Count != 3 {
		t.Fatalf("bystander container = %v, want one stack of three 4242s", got)
	}
	if got := w.equipment[i][1].Code; got != 777 {
		t.Fatalf("bystander slot 1 = %d, want 777", got)
	}
}
