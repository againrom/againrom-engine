package sim

import "testing"

// drawWorld is a computer mage of owner 7, id 1, at (2, 2) with a full pool
// of 60, holding Heal, Bless and, in the first game, Slow and an unaffordable
// Protection; others stand beside it.
func drawWorld(t *testing.T, second bool, mana int32, others ...Entity) (w *World, heal, bless, slow uint16) {
	t.Helper()
	healRow, blessRow, slowRow, dear := hlHeal(), acsBuff(23), spSlowRow(), acsBuff(5)
	healRow.ManaCost, dear.ManaCost = 3, 100
	rows := []SpellRule{dear, healRow, blessRow, slowRow}
	if second {
		healRow.ID, blessRow.ID = 24, 20
		rows = []SpellRule{secondRow(healRow), secondRow(blessRow)}
	}
	mage := spMage(1, 2, 2, 30, 60, mana, 0)
	mage.Owner, mage.Book.State, mage.Humanoid = 7, BookPresent, true
	for _, r := range rows {
		mage.KnownSpells |= 1 << r.ID
		var defensive uint8
		if r.Defensive || r.Restorative {
			defensive = 1
		}
		mage.Book.Slots[r.ID-1] = BookSpell{Range: uint8(r.MaxRange), Defensive: defensive, ManaCost: uint16(r.ManaCost)}
	}
	var script *Script
	if second {
		var err error
		if script, err = newScript(ScriptROM2, nil, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	companion := spEnt(99, 6, 3)
	companion.Owner = SelfSlot
	w, err := NewStockedSpelledWorld(31, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, append(append([]Entity{mage}, others...), companion), script, engRel(t,
		[3]uint32{7, SelfSlot, relationLocked}, [3]uint32{SelfSlot, 7, relationLocked}), nil, nil, rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.RestoreCurrentPlayers([]SavedGroupPlayer{{ID: 91, Slot: 7}, {ID: 92, Slot: SelfSlot}},
		[]PlayerParticipant{{PlayerID: 91, Value: 1}, {PlayerID: 92, Value: 0}}, true); err != nil {
		t.Fatal(err)
	}
	holdGroups(w)
	return w, healRow.ID, blessRow.ID, slowRow.ID
}

func ownUnit(id EntityID, x, y, hp int32) Entity {
	e := spEnt(id, x, y)
	e.Owner, e.HP = 7, hp
	return e
}

// mageCasts steps w n times and returns the computer mage's casts.
func mageCasts(w *World, n int) []CastEvent {
	var out []CastEvent
	for range n {
		for _, ev := range StepObserved(w, nil) {
			if ev.Caster == 1 {
				out = append(out, ev)
			}
		}
	}
	return out
}

func TestComputerMageBuffsADrawnOwnUnit(t *testing.T) {
	for _, second := range []bool{false, true} {
		t.Run(map[bool]string{false: "first game", true: "second game"}[second], func(t *testing.T) {
			w, _, bless, _ := drawWorld(t, second, 60, ownUnit(2, 3, 2, 100))
			got := 0
			for _, ev := range mageCasts(w, 300) {
				if ev.Spell != bless || ev.Target != 2 {
					t.Fatalf("cast %d at %d, want only Bless at the unit", ev.Spell, ev.Target)
				}
				got++
			}
			if got == 0 {
				t.Fatal("the own unit took no Bless")
			}
		})
	}
}

func TestFirstGameComputerHealKeepsNoReserveAndTakesTheAreaListOrder(t *testing.T) {
	w, heal, _, _ := drawWorld(t, false, 11, ownUnit(2, 3, 2, 40), ownUnit(3, 1, 3, 40))
	casts := mageCasts(w, 100)
	if len(casts) != 1 || casts[0].Spell != heal || casts[0].Target != 3 {
		t.Fatalf("casts %+v, want one Heal at unit 3, first in the area list", casts)
	}
}

func TestFirstGameComputerEmptyDrawCastsNothing(t *testing.T) {
	w, _, bless, slow := drawWorld(t, false, 60, ownUnit(2, 3, 2, 100))
	w.attached = append(w.attached, attachedEffect{Target: 2, Spell: bless, Remaining: 400, Kind: EffectProtectionFire, Mode: EffectDuration, Magnitude: 5})
	if casts := mageCasts(w, 300); len(casts) != 0 {
		t.Fatalf("casts %+v while every kept spell stands or is refused, want none", casts)
	}
	w.attached = nil
	casts := mageCasts(w, 300)
	for _, ev := range casts {
		if ev.Spell == slow || ev.Spell != bless || ev.Target != 2 {
			t.Fatalf("cast %d at %d, want only Bless at the unit", ev.Spell, ev.Target)
		}
	}
	if len(casts) == 0 {
		t.Fatal("the unit took no Bless once its effect lapsed")
	}
}
