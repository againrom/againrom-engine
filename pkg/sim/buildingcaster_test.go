package sim

import (
	"bytes"
	"testing"
)

func casterWorld(t *testing.T, defensive bool) *World {
	t.Helper()
	return casterWorldOf(t, defensive, &Script{dialect: ScriptROM2})
}

func casterWorldOf(t *testing.T, defensive bool, script *Script) *World {
	t.Helper()
	rel := Relations{}
	rel.turnHostile(2, 1)
	rel.turnHostile(1, 2)
	ents := []Entity{
		{ID: 1, Owner: 1, X: 10, Y: 10, TokenSize: 1, HP: 50, MaxHP: 50},
		{ID: 2, Owner: 2, X: 11, Y: 10, TokenSize: 1, HP: 50, MaxHP: 50},
		{ID: 3, Owner: 1, X: 60, Y: 60, TokenSize: 1, HP: 50, MaxHP: 50},
	}
	spells := []SpellRule{{ID: 2, Damaging: !defensive, Defensive: defensive}}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 80, Height: 80}, ModeCanonical, Terrain{}, ents, script, rel, nil, nil, spells)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func oneCaster() BuildingCaster {
	return BuildingCaster{X: 10, Y: 10, Spell: 2, Power: 99, Reach: 3, Owner: 2, Cell: cellKey(5, 5)}
}

func TestBuildingCasterPicksHostileOfOwner(t *testing.T) {
	w := casterWorld(t, false)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	w.buildingCasterPass()
	casts := w.ScriptCasts()
	if len(casts) != 1 || casts[0].Target != 1 || !casts[0].AtUnit || casts[0].Spell != 2 || casts[0].Power != 99 || casts[0].FromX != 10 {
		t.Fatalf("casts = %+v, want one at unit 1", casts)
	}
}

func TestBuildingCasterDefensiveSpellPicksFriendly(t *testing.T) {
	w := casterWorld(t, true)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	w.buildingCasterPass()
	if casts := w.ScriptCasts(); len(casts) != 1 || casts[0].Target != 2 {
		t.Fatalf("casts = %+v, want one at the owner's own unit 2", casts)
	}
}

func TestBuildingCasterNoSurvivorOrNoSpellCastsNothing(t *testing.T) {
	w := casterWorld(t, false)
	far, unknown := oneCaster(), oneCaster()
	far.X, far.Y = 40, 40
	unknown.Spell = 28
	w.DeclareBuildingCasters([]BuildingCaster{far, unknown})
	before := w.RandomState()
	w.buildingCasterPass()
	if len(w.ScriptCasts()) != 0 || w.RandomState() != before {
		t.Fatalf("casts %v state %d, want none and no draw", w.ScriptCasts(), w.RandomState())
	}
}

func TestBuildingCasterRunsOnceOnTheTurnPhase(t *testing.T) {
	w := casterWorld(t, false)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	queued := 0
	for i := 0; i < scriptCycle; i++ {
		before := len(w.casts)
		if w.tick%scriptCycle == scriptPassPhase {
			w.buildingCasterPass()
		}
		queued += len(w.casts) - before
		w.tick++
	}
	if queued != 1 {
		t.Fatalf("queued %d casts in one cycle, want 1", queued)
	}
}

func TestBuildingCasterCostsNoMana(t *testing.T) {
	w := casterWorld(t, false)
	w.entities[0].Mana = 7
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	w.buildingCasterPass()
	if w.entities[0].Mana != 7 || w.entities[1].Mana != 0 {
		t.Fatalf("mana moved: %d, %d", w.entities[0].Mana, w.entities[1].Mana)
	}
}

func TestBuildingCasterRemovedOnDestroyingHitByCellWord(t *testing.T) {
	w := casterWorld(t, false)
	other := oneCaster()
	other.Cell = cellKey(6, 6)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster(), other})
	s := Structure{Col: 5, Row: 5, MaxHealth: 100, Field42: 40}
	w.removeBuildingCastersOn(&s, 39)
	if len(w.buildingCasters) != 2 {
		t.Fatalf("a hit below health removed casters: %v", w.buildingCasters)
	}
	w.removeBuildingCastersOn(&s, 40)
	if len(w.buildingCasters) != 1 || w.buildingCasters[0].Cell != cellKey(6, 6) {
		t.Fatalf("after a destroying hit: %v", w.buildingCasters)
	}
}

func TestBuildingCasterFirstGameKeepsNoList(t *testing.T) {
	w := casterWorldOf(t, false, nil)
	base, _ := w.MarshalBinary()
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	declared, _ := w.MarshalBinary()
	w.settleBuildingCasters()
	raw, _ := w.MarshalBinary()
	if !bytes.Equal(base, declared) {
		t.Fatal("a declared list moved a first-game byte form before the script settled it")
	}
	if len(w.buildingCasters) != 0 || !bytes.Equal(base, raw) {
		t.Fatalf("a first-game World kept %d casters or moved its bytes", len(w.buildingCasters))
	}
}

func TestBuildingCasterSectionRoundTripAndLossControls(t *testing.T) {
	w := casterWorld(t, false)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	one, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(one); err != nil {
		t.Fatal(err)
	}
	if got := back.BuildingCasters(); len(got) != 1 || got[0] != oneCaster() {
		t.Fatalf("round trip = %+v", got)
	}
	changed := oneCaster()
	changed.Power = 1
	w.DeclareBuildingCasters([]BuildingCaster{changed})
	two, _ := w.MarshalBinary()
	if bytes.Equal(one, two) || w.Hash() == back.Hash() {
		t.Fatal("a changed caster did not change the byte form or the digest")
	}
	w.DeclareBuildingCasters(nil)
	none, _ := w.MarshalBinary()
	if bytes.Equal(one, none) {
		t.Fatal("a dropped list did not change the byte form")
	}
}

// The world step runs the pass on the turn phase and on no other: a cast is
// queued on the step of phase 6, once per cycle. The base does not compile
// this test.
func TestBuildingCasterStepQueuesOnTheTurnPhaseOnly(t *testing.T) {
	w := casterWorld(t, false)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	for i := 0; i < 2*scriptCycle; i++ {
		phase := w.tick % scriptCycle
		Step(w, nil)
		want := 0
		if phase == scriptPassPhase {
			want = 1
		}
		if got := len(w.ScriptCasts()); got != want {
			t.Fatalf("after the step of phase %d: %d casts queued, want %d", phase, got, want)
		}
	}
}

// A unit holding the effect whose bit the claim names is no survivor; another
// effect does not exclude it. The base does not compile this test.
func TestBuildingCasterSkipsTheExcludedEffect(t *testing.T) {
	w := casterWorld(t, false)
	w.DeclareBuildingCasters([]BuildingCaster{oneCaster()})
	w.attached = []attachedEffect{{Target: 1, Spell: buildingCasterExcludedEffect + 1}}
	w.buildingCasterPass()
	if casts := w.ScriptCasts(); len(casts) != 1 || casts[0].Target != 1 {
		t.Fatalf("another effect changed the pick: %+v", casts)
	}
	w.casts = nil
	w.attached = []attachedEffect{{Target: 1, Spell: buildingCasterExcludedEffect}}
	w.buildingCasterPass()
	if casts := w.ScriptCasts(); len(casts) != 0 {
		t.Fatalf("the excluded effect was picked: %+v", casts)
	}
}

// casterStructureWorld is a second-game world with one structure on the cell
// word the caster's Cell names, and one caster on that word and one elsewhere.
func casterStructureWorld(t *testing.T, a Entity, s Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(0, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, []Entity{a}, &Script{dialect: ScriptROM2}, Relations{}, nil, nil, nil, GhostTemplate{}, []Structure{s})
	if err != nil {
		t.Fatal(err)
	}
	on := oneCaster()
	on.Cell = cellKey(s.Col, s.Row)
	off := oneCaster()
	off.Cell = cellKey(s.Col+1, s.Row)
	w.DeclareBuildingCasters([]BuildingCaster{on, off})
	return w
}

// Both ways of hitting a structure remove the casters of its cell word on a
// value not below its health, and leave them below it. The base does not
// compile this test.
func TestBuildingCasterRemovedThroughBothHitPaths(t *testing.T) {
	spell := func(top int32) func(*World, Structure) {
		return func(w *World, s Structure) {
			w.applyStructureSpellAt(s.Col, s.Row, SpellRule{DamageMin: top, DamageMax: top + 1}, 0)
		}
	}
	blow := func(base uint8) func(*World, Structure) {
		return func(w *World, _ Structure) {
			w.entities[0].SecondaryDamage = SecondaryDamage{Base: base, Spread: 1}
			w.orderAttack(0, 0, AttackTargetStructure)
			w.resolveStructureBlow(0)
		}
	}
	for _, tc := range []struct {
		name string
		hit  func(*World, Structure)
		left int
	}{
		{"spell below health", spell(50), 2},
		{"spell at health", spell(150), 1},
		{"blow below health", blow(50), 2},
		{"blow at health", blow(200), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := structureCombatTarget()
			w := casterStructureWorld(t, structureCombatActor(), s)
			tc.hit(w, s)
			got := w.BuildingCasters()
			if len(got) != tc.left {
				t.Fatalf("%d casters left, want %d: %+v", len(got), tc.left, got)
			}
			if tc.left == 1 && got[0].Cell != cellKey(s.Col+1, s.Row) {
				t.Fatalf("the wrong caster left: %+v", got)
			}
		})
	}
}
