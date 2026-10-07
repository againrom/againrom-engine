package sim

import "testing"

// What a staged area effect reports to the client.

// apRule is one staged row: `Distribution system` 5 is what selects the ring
// mode, never the spell id (areaModeFor).
func apRule(id uint16, rng uint8) SpellRule {
	return SpellRule{ID: id, School: 1, MaxRange: rng, Area: true, Distribution: 5,
		Radius: 2, DamageMin: 1, DamageMax: 1, Damaging: true}
}

func paintWorld(t *testing.T, rule SpellRule) *World {
	t.Helper()
	caster := Entity{ID: 1, X: 40, Y: 40, HP: 100, MaxHP: 100, Owner: 3, TokenSize: 1,
		Mind: 30, Mana: 500, MaxMana: 500, KnownSpells: 1 << rule.ID, ScanRange: 19,
		AttackCharge: 4, AttackRelax: 2}
	w, err := NewSpelledWorld(1, Bounds{Width: 96, Height: 96}, ModeCanonical, nil,
		[]Entity{caster}, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return w
}

// TestAStagedCastReportsEveryStageItPaints is the whole channel, driven through
// a real cast: Fire Sacrifice's two stages arrive as two reports, three ticks
// apart, carrying the claim's own cell counts (`MAGIC-RING-048`: stage 0 is
// eight cells, stage 1 is twelve) and the caster's own roster slot.
//
// THE TERMINAL STAGE IS THE POINT. The record is reaped on the same tick its
// last stage runs, so a client reading CellEffects() alone can never see stage
// 1 — the outer shell of the explosion the owner asked for would simply never
// be drawn.
func TestAStagedCastReportsEveryStageItPaints(t *testing.T) {
	t.Parallel()

	w := paintWorld(t, apRule(4, 0))
	report := StepReported(w, []Command{{Kind: KindCastAt, Entity: 1, X: 40, Y: 40, Spell: 4}})
	stages := append([]AreaPaint(nil), report.AreaPaints...)
	for i := 0; i < 64 && len(stages) < 2; i++ {
		stages = append(stages, StepReported(w, nil).AreaPaints...)
	}

	if len(stages) != 2 {
		t.Fatalf("Fire Sacrifice reported %d stages, want its two", len(stages))
	}
	if got := len(stages[0].Cells); got != 8 {
		t.Errorf("stage 0 painted %d cells, want the claim's eight", got)
	}
	if got := len(stages[1].Cells); got != 12 {
		t.Errorf("stage 1 painted %d cells, want the claim's twelve", got)
	}
	for i, p := range stages {
		if p.Spell != 4 {
			t.Errorf("stage %d reports spell %d, want 4", i, p.Spell)
		}
		if p.Owner != 3 {
			t.Errorf("stage %d reports owner %d, want the caster's own 3", i, p.Owner)
		}
	}
	// The outer shell surrounds the inner one, which is what "spreads outward
	// from the caster's own cell" is: every stage-0 cell touches the caster's
	// cell and no stage-1 cell does.
	for _, c := range stages[0].Cells {
		if abs32(c.X-40) > 1 || abs32(c.Y-40) > 1 {
			t.Errorf("stage 0 cell %v is not adjacent to the caster's own (40,40)", c)
		}
	}
	for _, c := range stages[1].Cells {
		if abs32(c.X-40) <= 1 && abs32(c.Y-40) <= 1 {
			t.Errorf("stage 1 cell %v is inside the inner shell", c)
		}
	}
}

// TestMeteorStormReportsOneRandomCellPerStage is the owner's sixth report at the
// channel: 32 stages of one cell each, three ticks apart, so a client holding
// each cell for its 16-tick object shows rocks accumulating across the area at a
// visible rate rather than one cell blinking.
func TestMeteorStormReportsOneRandomCellPerStage(t *testing.T) {
	t.Parallel()

	w := paintWorld(t, apRule(21, 8))
	var stages []AreaPaint
	StepReported(w, []Command{{Kind: KindCastAt, Entity: 1, X: 44, Y: 44, Spell: 21}})
	for i := 0; i < 220 && len(stages) < 32; i++ {
		stages = append(stages, StepReported(w, nil).AreaPaints...)
	}
	if len(stages) != 32 {
		t.Fatalf("Meteor Storm reported %d stages, want its 32", len(stages))
	}
	distinct := map[CellPoint]bool{}
	for _, p := range stages {
		if len(p.Cells) != 1 {
			t.Fatalf("a Meteor Storm stage painted %d cells, want one", len(p.Cells))
		}
		c := p.Cells[0]
		if c.X-44 < -2 || c.X-44 > 3 || c.Y-44 < -2 || c.Y-44 > 3 {
			t.Errorf("a rock landed at %v, outside the claim's [-2,3] offset domain", c)
		}
		distinct[c] = true
	}
	if len(distinct) < 5 {
		t.Errorf("32 stages landed on %d distinct cells; the points are not random", len(distinct))
	}
}

// TestACloudReportsNoPaint is the mode fork: a cloud creates no object at all
// and is drawn from its retained cell set, so it must report nothing here or
// the client would draw its cells twice.
func TestACloudReportsNoPaint(t *testing.T) {
	t.Parallel()

	rule := SpellRule{ID: 3, School: 1, MaxRange: 8, Area: true, Distribution: 4,
		Radius: 2, AreaDuration: 15, DamageMin: 1, DamageMax: 1, Damaging: true}
	w := paintWorld(t, rule)
	report := StepReported(w, []Command{{Kind: KindCastAt, Entity: 1, X: 44, Y: 40, Spell: 3}})
	paints := len(report.AreaPaints)
	for i := 0; i < 40; i++ {
		paints += len(StepReported(w, nil).AreaPaints)
	}
	if paints != 0 {
		t.Errorf("a cloud reported %d paints over its whole life, want none", paints)
	}
	if len(w.CellEffects()) == 0 {
		t.Errorf("the cloud left no retained record, so nothing would be drawn at all")
	}
}

// TestObservationChangesNoWorld is castObs' own rule for the new field: a world
// advanced through StepReported and one advanced through Step are the same world
// byte for byte.
func TestObservationChangesNoWorld(t *testing.T) {
	t.Parallel()

	rule := apRule(4, 0)
	cmd := []Command{{Kind: KindCastAt, Entity: 1, X: 40, Y: 40, Spell: 4}}
	plain, observed := paintWorld(t, rule), paintWorld(t, rule)
	Step(plain, cmd)
	StepReported(observed, cmd)
	for i := 0; i < 24; i++ {
		Step(plain, nil)
		StepReported(observed, nil)
	}
	a, err := plain.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	b, err := observed.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if string(a) != string(b) {
		t.Errorf("an observed advance produced a different world than an unobserved one")
	}
}
