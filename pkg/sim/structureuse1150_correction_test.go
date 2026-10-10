package sim

import (
	"fmt"
	"testing"
)

func TestReviewStructureUse1150ReleaseBoundary(t *testing.T) {
	for _, prior := range []string{"book", "scroll"} {
		t.Run(prior, func(t *testing.T) {
			w := manualCastWorld(t)
			if prior == "scroll" {
				w = scrollWorld1090(t, 2, 2)
			}
			w.structures = []Structure{{ID: 0, Kind: 28, Width: 1, Height: 1, Col: 12, Row: 12, Field42: 1, MaxHealth: 1}}
			old := spCast(1, 2, 1)
			if prior == "scroll" {
				old = Command{Kind: KindUseScroll, Entity: 1, X: 2}
			}
			Step(w, []Command{old})
			ready := false
			for n := 0; n < 100; n++ {
				if prior == "book" {
					ready = len(w.bookCasts) == 1 && w.bookCasts[0].Phase == bookCharging && w.bookCasts[0].Remaining == 1 && !w.entities[0].Turning()
				} else {
					ready = len(w.scrollCasts) == 1 && w.scrollCasts[0].Started && w.scrollCasts[0].Remaining == 1 && !w.entities[0].Turning()
				}
				if ready {
					break
				}
				Step(w, nil)
			}
			if !ready {
				t.Fatal("precondition: no natural release boundary")
			}
			invalid, control := retreatRoundTrip1089(t, w), retreatRoundTrip1089(t, w)
			badEvents := StepObserved(invalid, []Command{{Kind: KindUseStructure, Entity: 1, X: 999}})
			Step(control, nil)
			if len(badEvents) != 1 || invalid.Hash() != control.Hash() {
				t.Fatal("invalid structure click suppressed or changed the old release")
			}
			if prior == "scroll" {
				unrefundable := retreatRoundTrip1089(t, w)
				unrefundable.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true}
				unrefundable.entities[0].Capacity = 0
				if events := StepObserved(unrefundable, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}}); len(events) != 1 {
					t.Fatal("unrefundable structure replacement suppressed the old release")
				}
			}
			cold := retreatRoundTrip1089(t, w)
			hash := w.Hash()
			if !w.manualActionInterruptions([]Command{{Kind: KindUseStructure, Entity: 1, X: 0}})[1] || w.Hash() != hash {
				t.Fatal("replacement admission probe refused or mutated the world")
			}
			mana, hp := w.entities[0].Mana, w.entities[1].HP
			beforeStock := uint32(0)
			if prior == "scroll" {
				beforeStock = w.carried[0][0].Count
			}
			events := StepObserved(w, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}})
			Step(cold, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}})
			if w.Hash() != cold.Hash() {
				t.Fatal("cold replacement changed the release boundary")
			}
			t.Logf("prior=%s admitted=%d events=%+v mana=%d->%d victimHP=%d->%d stockBefore=%d stockAfter=%+v", prior, len(w.StructureUses()), events, mana, w.entities[0].Mana, hp, w.entities[1].HP, beforeStock, w.carried[0])
			if len(w.StructureUses()) != 1 {
				t.Fatal("use was not admitted")
			}
			if len(events) != 0 || w.entities[0].Mana != mana || w.entities[1].HP != hp {
				t.Fatal("replaced action released on structure-use tick")
			}
			if prior == "scroll" && w.carried[0][0].Count != beforeStock+1 {
				t.Fatal("canceled scroll not refunded")
			}
		})
	}
}

func TestStructureUse1150RefundCandidateRefusal(t *testing.T) {
	w := sourceMutationWorld(t, PlainItem(0xe01))
	w.carried[0] = nil
	w.scrollCasts = []ScrollCast{{Caster: 1, Item: ItemInstance{Code: 0xe01, Weight: 400, WeightPresent: true}}}
	w.structures = []Structure{{ID: 0, Kind: 28, Width: 1, Height: 1}}
	w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
		if acc != 0 {
			return s, fmt.Errorf("refund arithmetic failure")
		}
		return s, nil
	})
	hash := w.Hash()
	if w.manualActionInterruptions([]Command{{Kind: KindUseStructure, Entity: 1, X: 0}})[1] ||
		w.beginStructureUse(0, 0) || w.Hash() != hash {
		t.Fatal("failed refund candidate interrupted or changed the old action")
	}
}
func TestReviewStructureUse1150PostCompletion(t *testing.T) {
	for _, saved := range []bool{false, true} {
		t.Run(fmt.Sprintf("saved-%v", saved), func(t *testing.T) {
			w := manualCastWorld(t)
			w.entities[1].OffMap = true
			w.structures = []Structure{{ID: 0, Kind: 28, Width: 1, Height: 1, Col: 12, Row: 12, Field42: 1, MaxHealth: 1}}
			if saved {
				savedTacticalRegistry(t, w, true)
			}
			origin := cellOf(&w.entities[0])
			Step(w, []Command{{Kind: KindUseStructure, Entity: 1, X: 0}})
			for n := 0; n < 500 && len(w.StructureUses()) > 0; n++ {
				Step(w, nil)
			}
			if len(w.StructureUses()) != 0 || w.structures[0].Field42 != 0 {
				t.Fatal("use did not complete")
			}
			finished := cellOf(&w.entities[0])
			cold := retreatRoundTrip1089(t, w)
			for n := 0; n < 500; n++ {
				Step(w, nil)
				Step(cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatal("cold mismatch")
				}
			}
			t.Logf("saved=%v origin=%+v completed=%+v after500=%+v pending=%+v", saved, origin, finished, cellOf(&w.entities[0]), w.StructureUses())
			if cellOf(&w.entities[0]) != finished {
				t.Fatal("actor walked away without a new command after using structure")
			}
		})
	}
}
