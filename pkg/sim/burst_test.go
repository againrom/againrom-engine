package sim

import (
	"slices"
	"testing"
)

// burstWorld is a Fire_Ball caster at cell (1,1) with an area row of speed 128
// and a target standing four cells east.
func burstWorld(t *testing.T) (*World, SpellRule) {
	t.Helper()
	w := deliveryTestWorld(t, fireBallSpell)
	r := w.spells[0]
	r.Area, r.TargetsUnit, r.Distribution, r.Radius = true, false, distributionDiamond, 1
	w.SetBurstPhases(11)
	return w, r
}

// The burst's expected leaves are written out from the claims, not read from
// the code under test: SAV-1144 (leaves), SAV-1147 (start at -1 with 22
// segments), ANIM-103 (frame = (actionphase / 2) mod phases) and SAV-1133 (a
// record built with N segments lives N + 1 ticks).
func TestFireBallBlastBuildsThePictureThirteenRecord(t *testing.T) {
	w, r := burstWorld(t)
	if !w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil) {
		t.Fatal("area preparation")
	}
	// Distance 4 cells = 1024 units over speed 128 is 8 ticks; the released
	// child blasts on the pass after the transport fires.
	for i := range 8 {
		Step(w, nil)
		if len(w.SavedProjectiles().Items) != 0 || w.SavedProjectiles().FreeIndex != 0 {
			t.Fatalf("a record exists on step %d, before the blast", i+1)
		}
	}
	Step(w, nil)
	got := w.SavedProjectiles()
	if got.FreeIndex != 1 || !slices.Equal(got.IDs, []uint16{0}) || len(got.Items) != 1 {
		t.Fatalf("allocator %d, IDs %v, %d items", got.FreeIndex, got.IDs, len(got.Items))
	}
	cx, cy := int32(5*256+128), int32(1*256+128)
	want := SavedProjectile{ID: 0, X: cx, Y: cy, Picture: 13, Action: 1, LastAction: 1,
		ActionX: cx, ActionY: cy, ActionPhase: 0, ActionSegments: 21}
	if got.Items[0] != want {
		t.Fatalf("record %+v, want %+v", got.Items[0], want)
	}
	caster, _ := w.Entity(1)
	casterOwner := caster.Owner
	if casterOwner == 0 {
		t.Fatal("fixture: the caster has no owner slot")
	}
	if d := w.SavedWorldEffectDrivers(); d == nil || len(d.Projectiles) != 1 || d.Projectiles[0] != (SavedProjectileDriver{ID: 0, Phases: 11, Owner: casterOwner}) {
		t.Fatalf("driver %+v", d)
	}
	// Calls 2 to 22 each add one to actionphase and take one from the
	// segments; the burst never moves.
	for call := 2; call <= 22; call++ {
		Step(w, nil)
		p := w.SavedProjectiles().Items
		if len(p) != 1 {
			t.Fatalf("record gone after call %d", call)
		}
		phase := int32(call - 1)
		if p[0].ActionPhase != phase || p[0].ActionSegments != int32(22-call) || p[0].Phase != (phase/2)%11 || p[0].X != cx || p[0].Y != cy {
			t.Fatalf("call %d: %+v", call, p[0])
		}
	}
	Step(w, nil)
	if len(w.SavedProjectiles().Items) != 0 || w.SavedProjectiles().FreeIndex != 1 || len(w.SavedProjectiles().IDs) != 0 {
		t.Fatalf("the record outlived its 23rd call: %+v", w.SavedProjectiles())
	}
}

// A shot and a burst take ids from one counter, and the newest id stands first
// in its hash bucket (SAV-1146).
func TestFireBallBurstSharesTheProjectileCounter(t *testing.T) {
	w, r := burstWorld(t)
	w.entities[0].HasAttackTarget, w.entities[0].AttackTarget, w.entities[0].AttackTargetKind = true, 2, AttackTargetUnit
	w.entities[0].Reach = 6
	w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil)
	for range 5 {
		Step(w, nil)
	}
	// The shot is built with 5 segments: the call on its birth tick and four
	// more leave it flying when the blast lands on step 9.
	if !w.ReleaseUnitShot(UnitShot{Shooter: 1, Picture: 5, Phases: 1}) {
		t.Fatal("no shot")
	}
	for range 4 {
		Step(w, nil)
	}
	got := w.SavedProjectiles()
	if got.FreeIndex != 2 || !slices.Equal(got.IDs, []uint16{1, 0}) || len(got.Items) != 2 || got.Items[0].Picture != 13 || got.Items[1].Picture != 5 {
		t.Fatalf("allocator %d, IDs %v, items %+v", got.FreeIndex, got.IDs, got.Items)
	}
}

// SAVE and LOAD in the middle of the burst's life continue it to the same end.
func TestFireBallBurstSurvivesTheByteForm(t *testing.T) {
	w, r := burstWorld(t)
	w.landArea(r, 0, 1, true, 1, 1, 5, 1, nil)
	for range 9 + 6 {
		Step(w, nil)
	}
	back := requireSpellGraphBinary(t, w)
	for k := range 20 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("LOAD diverged %d ticks after the byte form", k+1)
		}
	}
	if len(back.SavedProjectiles().Items) != 0 || back.SavedProjectiles().FreeIndex != 1 {
		t.Fatalf("burst did not end: %+v", back.SavedProjectiles())
	}
}

// The Fire_Ball transport waits the truncated Euclidean distance over the
// row's speed (ANIM-111, SAV-1149); a Firebolt keeps the cell metric.
func TestFireBallTransportWaitsTheEuclideanDistance(t *testing.T) {
	for _, tc := range []struct {
		spell    uint16
		toX, toY int32
		want     uint16
	}{
		{fireBallSpell, 5, 1, 8},  // 1024 units straight
		{fireBallSpell, 5, 4, 10}, // 4 cells by 3 cells = 1280 units
		{fireBallSpell, 4, 4, 8},  // 3 cells by 3 cells = 1086 units
		{1, 5, 4, 8},              // Chebyshev: 4 cells
	} {
		w := deliveryTestWorld(t, tc.spell)
		w.spells[0].Area, w.spells[0].TargetsUnit, w.spells[0].Distribution, w.spells[0].Radius = true, false, distributionDiamond, 1
		if !w.landArea(w.spells[0], 0, 1, true, 1, 1, tc.toX, tc.toY, nil) || len(w.deliveries) != 1 {
			t.Fatal("area preparation")
		}
		if got := w.deliveries[0].Remaining; got != tc.want {
			t.Errorf("spell %d to (%d,%d): countdown %d, want %d", tc.spell, tc.toX, tc.toY, got, tc.want)
		}
	}
}

// A retained transport handing a Fire_Ball area to its blast builds the burst
// too (ANIM-111: every Fire_Ball area effect ends in one 0x86).
func TestSavedFireBallAreaBlastBuildsTheBurst(t *testing.T) {
	w := transportGraphWorld(t, 2)
	g := w.SavedSpellGraph()
	g.Nodes[1].Value = SavedSpellEffect{Class: "AreaEffect", AE48: [4]byte{1, 1, 0, 0}, AE44: g.Nodes[1].Value.PE48}
	g.Nodes[1].HasTarget, g.Nodes[1].Target = false, 0
	w.savedWorldEffects.Areas = []SavedAreaDriver{{ID: 2, Root: -1, Identity: 55, Key: cellKey(5, 1), Layer: 255, Mode: AreaModeBlast, Spell: fireBallSpell}}
	if err := w.ImportSavedSpellGraph(g); err != nil {
		t.Fatal(err)
	}
	w.SetBurstPhases(11)
	for range 3 {
		Step(w, nil)
	}
	p := w.SavedProjectiles()
	if len(p.Items) != 1 || p.Items[0].Picture != 13 || p.Items[0].X != 5*256+128 || p.Items[0].ActionSegments != 21 || p.FreeIndex != 1 {
		t.Fatalf("burst after the handoff: %+v", p)
	}
	requireSpellGraphBinary(t, w)
}
