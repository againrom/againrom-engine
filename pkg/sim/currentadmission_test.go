package sim

import "testing"

func TestCurrentLivingAdmissionKeepsZeroFootprintAndValidatesExactPresence(t *testing.T) {
	w, err := NewWorld(31, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{{ID: 0, X: 5, Y: 6, HP: 31, MaxHP: 31, Owner: 1}})
	if err != nil {
		t.Fatal(err)
	}
	e := w.entities[0]
	e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x01000001}
	e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	before := w.Hash()
	if err := w.ImportOriginalLivingActors([]OriginalLivingActor{{Entity: e}}); err == nil || w.Hash() != before {
		t.Fatal("ordinary-only zero footprint admission changed", err)
	}
	for _, current := range [][]EntityID{nil, {0, 0}, {0, 9}} {
		if err := w.ImportCurrentLivingActors([]OriginalLivingActor{{Entity: e}}, current); err == nil || w.Hash() != before {
			t.Fatal("inexact current presence changed World", current, err)
		}
	}
	late := e
	late.ID, late.SourceBinding.Identity = 8, 0x01000002
	if err := w.ImportCurrentLivingActors([]OriginalLivingActor{{Entity: e}, {Entity: late, New: true}}, []EntityID{0, 8}); err == nil || w.Hash() != before {
		t.Fatal("late duplicate source binding changed World", err)
	}
	if err := w.ImportCurrentLivingActors([]OriginalLivingActor{{Entity: e}}, []EntityID{0}); err != nil || w.entities[0].TokenSize != 0 {
		t.Fatal("exact current zero-ID actor lost its zero footprint", err)
	}
	e.TokenSize = 3
	if err := w.ImportCurrentLivingActors([]OriginalLivingActor{{Entity: e}}, []EntityID{0}); err != nil || w.entities[0].TokenSize != 3 {
		t.Fatal("current presence overrode the ordinary footprint", err)
	}
}

func TestCurrentLivingAdmissionKeepsOutsidePositionWithoutOffMap(t *testing.T) {
	w, err := NewWorld(31, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 54, Y: 100, HP: 31, MaxHP: 31, Owner: 1, TokenSize: 1}})
	if err != nil {
		t.Fatal(err)
	}
	e := w.entities[0]
	e.SourceBinding = SourceBinding{Class: 1, ArchiveIndex: 1, Identity: 0x01000001}
	e.ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	batch := []OriginalLivingActor{{Entity: e}}
	before := w.Hash()
	if err := w.ImportOriginalLivingActors(batch); err == nil || w.Hash() != before {
		t.Fatal("ordinary-only outside position was admitted or published", err)
	}
	for _, ids := range [][]EntityID{nil, {1, 1}, {2}, {1, 2}} {
		if err := w.ImportCurrentLivingActors(batch, ids); err == nil || w.Hash() != before {
			t.Fatal("inexact current position binding was admitted or published", ids, err)
		}
	}
	if err := w.ImportCurrentLivingActors(batch, []EntityID{1}); err != nil {
		t.Fatal(err)
	}
	if got := w.entities[0]; got.X != 54 || got.Y != 100 || got.OffMap {
		t.Fatal("current position or OffMap changed", got)
	}
	encoded, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(encoded); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("current outside actor did not survive native decode", err)
	}
	for tick := 0; tick < 20; tick++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("outside actor successor diverged", tick)
		}
	}
}
