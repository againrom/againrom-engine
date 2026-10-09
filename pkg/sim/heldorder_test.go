package sim

import (
	"bytes"
	"testing"
)

func TestHeldOrdersSurviveBinaryAndLapseAsTheWorldAdvances(t *testing.T) {
	w := mustWorld(t, 3, Bounds{Width: 8, Height: 8}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20},
		{ID: 2, X: 2, Y: 1, HP: -20, MaxHP: 20, Decay: DecayBones},
		{ID: 3, X: 4, Y: 4, HP: -30, MaxHP: 20, Decay: DecayBones},
	})
	body := HeldOrder{Kind: HeldOrderBody, Target: 2, Phase: 5, Countdown: 9, Complete: 1}
	frozen := HeldOrder{Kind: HeldOrderFrozen, Phase: 7, Complete: 1}
	before := w.Hash()
	if err := w.RestoreHeldOrders([]HeldOrderRecord{{ID: 1, Order: body}, {ID: 3, Order: frozen}}); err != nil {
		t.Fatal(err)
	}
	if w.Hash() == before {
		t.Fatal("held orders are not hashed World state")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if form[0] != heldOrderFormVersion || CheckSaveForm(form) != nil {
		t.Fatal("the held order form is refused at the save boundary")
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	again, err := cold.MarshalBinary()
	if err != nil || !bytes.Equal(again, form) || cold.entities[0].HeldOrder != body || cold.entities[2].HeldOrder != frozen {
		t.Fatal("held orders did not survive the byte form", err)
	}
	Step(w, nil)
	if w.entities[0].HeldOrder != (HeldOrder{}) || w.entities[2].HeldOrder != frozen {
		t.Fatal("an advancing World must end the body order and keep the frozen one", w.entities[0].HeldOrder, w.entities[2].HeldOrder)
	}
	w.entities[2].HP = 5
	if form, err := w.MarshalBinary(); err != nil || form[0] == heldOrderFormVersion {
		t.Fatal("a living actor's frozen order must leave the byte form", err)
	}
}

func TestHeldOrdersRefuseAnImpossibleOrder(t *testing.T) {
	for name, row := range map[string]HeldOrderRecord{
		"no actor":           {ID: 9, Order: HeldOrder{Kind: HeldOrderFrozen, Phase: 5}},
		"frozen on a living": {ID: 1, Order: HeldOrder{Kind: HeldOrderFrozen, Phase: 5}},
		"empty frozen words": {ID: 3, Order: HeldOrder{Kind: HeldOrderFrozen}},
		"body on a dead":     {ID: 3, Order: HeldOrder{Kind: HeldOrderBody, Target: 1, Phase: 5}},
		"body without phase": {ID: 1, Order: HeldOrder{Kind: HeldOrderBody, Target: 3}},
		"unknown kind":       {ID: 1, Order: HeldOrder{Kind: 3}},
	} {
		w := mustWorld(t, 3, Bounds{Width: 8, Height: 8}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20},
			{ID: 3, X: 4, Y: 4, HP: -30, MaxHP: 20, Decay: DecayBones},
		})
		before := w.Hash()
		if err := w.RestoreHeldOrders([]HeldOrderRecord{row}); err == nil || w.Hash() != before {
			t.Error(name, "was admitted or changed the World")
		}
	}
}
