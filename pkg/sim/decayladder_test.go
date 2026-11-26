package sim

import "testing"

// decayBodyAt steps a body of class typeID from health hp through ticks
// sub-ticks and returns its health and stage.
func decayBodyAt(t *testing.T, second bool, typeID, hp int32, ticks uint64) (int32, DecayStage) {
	t.Helper()
	e := Entity{ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, TypeID: typeID, Decay: DecayFallen}
	e.SourceBinding.RuntimeID = 5
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for w.Tick() < ticks {
		Step(w, nil)
	}
	got := w.Entities()[0]
	return got.HP, got.Decay
}

// decayDamagedAt fells a living unit of class typeID to exact health hp and
// returns the resulting entity after that tick.
func decayDamagedAt(t *testing.T, second bool, typeID, hp int32) (Entity, bool) {
	t.Helper()
	e := Entity{ID: 1, X: 3, Y: 3, HP: 50, MaxHP: 50, TypeID: typeID}
	e.SourceBinding.RuntimeID = 5
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{Damage(1, 50-hp)})
	if len(w.Entities()) == 0 {
		return Entity{}, false
	}
	return w.Entities()[0], true
}

func TestBothGamesCorpseStagesBeginBelowTheirThresholds(t *testing.T) {
	for _, c := range []struct {
		hp   int32
		want DecayStage
	}{
		{-10, 1}, {-11, 2}, {-20, 2}, {-21, 3}, {-40, 3}, {-41, 4},
	} {
		for _, second := range []bool{false, true} {
			got, ok := decayDamagedAt(t, second, 10, c.hp)
			if !ok || got.HP != c.hp || got.Decay != c.want || !got.CorpseDecayed {
				t.Errorf("second=%v health %d: ok=%v hp=%d stage=%d decayed=%v, want hp=%d stage=%d decayed=true",
					second, c.hp, ok, got.HP, got.Decay, got.CorpseDecayed, c.hp, c.want)
			}
		}
	}
	for _, second := range []bool{false, true} {
		if got, ok := decayDamagedAt(t, second, 10, -600); !ok || got.Decay != decayLast || !got.CorpseDecayed {
			t.Errorf("second=%v health -600: ok=%v stage=%d decayed=%v, want stage 4 decayed=true", second, ok, got.Decay, got.CorpseDecayed)
		}
		if _, ok := decayDamagedAt(t, second, 10, -601); ok {
			t.Errorf("second=%v health -601: body still present, want removed", second)
		}
	}
}

// Old saves written before the strict ladder hold dead units at -20 (stage 3)
// and -40 (stage 4) without a CorpseDecayed wrapper; both the World byte form
// and held OriginalDeadActors continue to load and derive CorpseDecayed from
// the stage.
func TestOldSaveDeadUnitsAtThresholdBoundariesLoad(t *testing.T) {
	for _, second := range []bool{false, true} {
		ents := []Entity{
			{ID: 1, X: 2, Y: 2, HP: -20, MaxHP: 50, Decay: 3},
			{ID: 2, X: 4, Y: 2, HP: -40, MaxHP: 50, Decay: 4},
		}
		var s *Script
		if second {
			s = secondGameScript(t, nil, nil, nil)
		}
		w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, ents, s, Relations{}, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		if raw[0] == corpseDecayedFormVersion {
			t.Fatalf("second=%v: stage 3/4 bodies wrote unnecessary wrapper %d", second, raw[0])
		}
		var back World
		if err := back.UnmarshalBinary(raw); err != nil {
			t.Fatalf("second=%v: UnmarshalBinary failed: %v", second, err)
		}
		got := back.Entities()
		if len(got) != 2 || got[0].HP != -20 || got[0].Decay != 3 || !got[0].CorpseDecayed ||
			got[1].HP != -40 || got[1].Decay != 4 || !got[1].CorpseDecayed {
			t.Fatalf("second=%v: decoded old dead bodies = %+v, want (-20,stage 3) and (-40,stage 4) both decayed", second, got)
		}
		// Virtual dead actors whose current state stopped at -20 (stage 3) or
		// -40 (stage 4) under the pre-strict ladder still validate on load.
		for _, tc := range []struct {
			name            string
			srcStage, curSt uint8
			srcHP, curHP    int16
		}{
			{"at -20 stage 3", 2, 3, -16, -20},
			{"at -40 stage 4", 3, 4, -24, -40},
		} {
			src := DeadActorState{RuntimeID: 40, Cell: 0x0304, FineX: 128, FineY: 128, Stage: tc.srcStage, HP: tc.srcHP}
			cur := src
			cur.Stage, cur.HP = tc.curSt, tc.curHP
			if !validHeldDeadContinuation(src, cur, w.game.decay) {
				t.Errorf("second=%v %s: validHeldDeadContinuation rejected old-save boundary", second, tc.name)
			}
		}
	}
}

// The CorpseDecayed wrapper encodes only stage-1 decayed corpses, leaves
// living worlds byte-identical, and rejects malformed sections.
func TestCorpseDecayedByteFormRoundTripAndRefusal(t *testing.T) {
	live, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := live.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if plain[0] == corpseDecayedFormVersion {
		t.Fatalf("living world used wrapper %d", plain[0])
	}
	Step(live, []Command{Damage(1, 20)})
	if e := live.Entities()[0]; e.HP != -10 || e.Decay != DecayFallen || !e.CorpseDecayed {
		t.Fatalf("felled at -10: %+v", e)
	}
	wrapped, err := live.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if wrapped[0] != corpseDecayedFormVersion {
		t.Fatalf("stage-1 decayed body form = %d, want %d", wrapped[0], corpseDecayedFormVersion)
	}
	var cold World
	if err := cold.UnmarshalBinary(wrapped); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if cold.Hash() != live.Hash() || !cold.Entities()[0].CorpseDecayed || cold.Entities()[0].Decay != DecayFallen {
		t.Fatalf("round-trip lost CorpseDecayed: %+v", cold.Entities()[0])
	}
	env, ok := readWrapper(wrapped, corpseDecayedFormVersion)
	if !ok {
		t.Fatal("missing corpse flag section")
	}
	if err := cold.UnmarshalBinary(env.base()); err != nil || !cold.entities[0].CorpseDecayed {
		t.Fatal("lost the older spent-loot mark at -10", err)
	}
	// Malformed wrapper naming a living actor must be refused.
	bad := append([]byte(nil), plain...)
	ws := beginWrapper(bad)
	bad = append(bad, 0, 0, 0, 0)
	bad = endWrapper(bad, ws, corpseDecayedFormVersion)
	if err := (&World{}).UnmarshalBinary(bad); err == nil {
		t.Fatal("accepted CorpseDecayed wrapper on a living entity")
	}
}

// A second-game body of class word 0x21..0x24 loses health only at -60 and
// above, so the walk stops it at -61; class 0x25 walks on (R2-ENGINE-1825).
func TestSecondGameHeroBodyStopsAtMinusSixtyOne(t *testing.T) {
	for _, c := range []struct {
		typeID, want int32
	}{
		{0x21, -61}, {0x22, -61}, {0x23, -61}, {0x24, -61}, {0x25, -69},
	} {
		if hp, stage := decayBodyAt(t, true, c.typeID, -59, 32*10); hp != c.want || stage != decayLast {
			t.Errorf("class %#x: health %d stage %d, want %d and %d", c.typeID, hp, stage, c.want, decayLast)
		}
	}
}
