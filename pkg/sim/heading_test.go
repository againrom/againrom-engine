package sim

import "testing"

// fineWorld holds each id's sub-cell bytes as a resumed position does.
func fineWorld(fine map[EntityID][2]uint8) *World {
	w := &World{savedMotion: &savedActorMotionState{}}
	for id := EntityID(0); id < 64; id++ {
		if p, ok := fine[id]; ok {
			w.savedMotion.Motions = append(w.savedMotion.Motions, SavedActorMotion{Entity: id, Current: true,
				Position: SavedActorPosition{FineX: p[0], FineY: p[1]}})
		}
	}
	return w
}

// AI-361's 81 replayed outputs, decider at cell 50/50, both actors size 1 and
// centred, through the actor law every turn toward an actor uses. The centre
// entry is the coincident-centre result.
func TestActorHeadingReplaysTheControlledCentres(t *testing.T) {
	want := [9][9]uint8{
		{224, 224, 224, 0, 0, 0, 32, 32, 32},
		{224, 224, 224, 0, 0, 0, 32, 32, 32},
		{224, 224, 224, 224, 0, 32, 32, 32, 32},
		{192, 192, 224, 224, 0, 32, 32, 64, 64},
		{192, 192, 192, 192, 224, 64, 64, 64, 64},
		{192, 192, 160, 160, 128, 96, 96, 64, 64},
		{160, 160, 160, 160, 128, 96, 96, 96, 96},
		{160, 160, 160, 128, 128, 128, 96, 96, 96},
		{160, 160, 160, 128, 128, 128, 96, 96, 96},
	}
	w := &World{}
	decider := Entity{ID: 1, X: 50, Y: 50, TokenSize: 1}
	for dy := int32(-4); dy <= 4; dy++ {
		for dx := int32(-4); dx <= 4; dx++ {
			target := Entity{ID: 2, X: 50 + dx, Y: 50 + dy, TokenSize: 1}
			if got := w.headingBetween(decider, target); got != want[dy+4][dx+4] {
				t.Errorf("(%d,%d): %d, want %d", dx, dy, got, want[dy+4][dx+4])
			}
		}
	}
}

// AI-444 on fine centres: the 2:1 boundary on sub-cell bytes, the footprint
// centre for sizes 1 to 4, the 16-bit wrap and the zero vector.
func TestActorHeadingLawOnFineCentres(t *testing.T) {
	for _, tc := range []struct {
		name         string
		a, b         Entity
		fineA, fineB [2]uint8
		want         uint8
	}{
		{"exact 2:1 on sub-cells is the diagonal", Entity{X: 10, Y: 10}, Entity{X: 12, Y: 9}, [2]uint8{128, 128}, [2]uint8{128, 128}, 32},
		{"one fine unit past 2:1 is the axis", Entity{X: 10, Y: 10}, Entity{X: 12, Y: 9}, [2]uint8{128, 128}, [2]uint8{129, 128}, 64},
		{"one fine unit short of 2:1 is the diagonal", Entity{X: 10, Y: 10}, Entity{X: 12, Y: 9}, [2]uint8{128, 128}, [2]uint8{128, 127}, 32},
		{"sub 0 to sub 255 one fine unit north is the axis", Entity{X: 10, Y: 10}, Entity{X: 10, Y: 9}, [2]uint8{0, 0}, [2]uint8{255, 255}, 64},
		{"nine by four is the axis", Entity{X: 10, Y: 10}, Entity{X: 19, Y: 14}, [2]uint8{128, 128}, [2]uint8{128, 128}, 64},
		{"a size-2 target on the actor's own anchor", Entity{X: 10, Y: 10, TokenSize: 1}, Entity{X: 10, Y: 10, TokenSize: 2}, [2]uint8{128, 128}, [2]uint8{128, 128}, 96},
		{"a size-4 actor beside a size-1 target", Entity{X: 10, Y: 10, TokenSize: 4}, Entity{X: 11, Y: 11, TokenSize: 1}, [2]uint8{128, 128}, [2]uint8{128, 128}, 224},
		{"a size-3 actor measured from its centre", Entity{X: 10, Y: 10, TokenSize: 3}, Entity{X: 14, Y: 12, TokenSize: 1}, [2]uint8{128, 128}, [2]uint8{128, 128}, 64},
		{"a size-3 centre past 65,535 wraps", Entity{X: 10, Y: 10, TokenSize: 1}, Entity{X: 255, Y: 9, TokenSize: 3}, [2]uint8{128, 128}, [2]uint8{200, 128}, 192},
		{"coincident centres", Entity{X: 5, Y: 5, TokenSize: 1}, Entity{X: 4, Y: 4, TokenSize: 3}, [2]uint8{128, 128}, [2]uint8{128, 128}, 224},
	} {
		tc.a.ID, tc.b.ID = 1, 2
		w := fineWorld(map[EntityID][2]uint8{1: tc.fineA, 2: tc.fineB})
		if got := w.headingBetween(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

// A turn toward an actor takes the actor law and a cast at the caster itself
// turns nothing (AI-443).
func TestTurnToActorTakesTheActorLaw(t *testing.T) {
	w := &World{entities: []Entity{{ID: 1, X: 10, Y: 10, Facing: 96}, {ID: 2, X: 19, Y: 14}}}
	if w.turnToActor(0, 1) || w.entities[0].Facing != 64 {
		t.Errorf("nine by four: facing %d, want the east axis 64 written at once", w.entities[0].Facing)
	}
	w.entities[0].Facing = 96
	if w.turnToActor(0, 0) || w.entities[0].Facing != 96 {
		t.Errorf("a cast at the caster itself turned it to %d", w.entities[0].Facing)
	}
}

// AI-445: a structure is a token of size 1 at its anchor cell's centre.
func TestStructureHeadingIsATokenOfSizeOne(t *testing.T) {
	w := &World{}
	if got := w.structureHeading(Entity{X: 10, Y: 10, TokenSize: 1}, Structure{Col: 19, Row: 14, Width: 3, Height: 3}); got != 64 {
		t.Errorf("nine by four: %d, want 64", got)
	}
	if got := w.structureHeading(Entity{X: 10, Y: 10, TokenSize: 2}, Structure{Col: 11, Row: 9}); got != 0 {
		t.Errorf("a size-2 attacker below its target's anchor: %d, want 0", got)
	}
}

// AI-446: whole-cell differences from the anchor cell; sub-cell and size do
// not enter, and the anchor cell itself gives 224.
func TestCellHeadingIgnoresSubCellAndSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		x, y int32
		want uint8
	}{
		{"exact 2:1", 12, 11, 96},
		{"three to one", 13, 11, 64},
		{"one to three north", 11, 7, 0},
		{"the anchor cell", 10, 10, 224},
		{"a far west cell", -245, 10, 192},
	} {
		w := fineWorld(map[EntityID][2]uint8{1: {200, 3}})
		w.entities = []Entity{{ID: 1, X: 10, Y: 10, TokenSize: 3}}
		if w.turnToCell(0, tc.x, tc.y) || w.entities[0].Facing != tc.want {
			t.Errorf("%s: facing %d, want %d", tc.name, w.entities[0].Facing, tc.want)
		}
	}
	if got := areaDirection(10, 10, 13, 11); got != 2 {
		t.Errorf("a wall cast three east and one south points %d, want 2 (east)", got)
	}
	if got := areaDirection(10, 10, 10, 10); got != 7 {
		t.Errorf("a wall cast on the caster's cell points %d, want 7", got)
	}
}

// MOVE-115: signs alone of the next cell's centre minus the anchor fine point,
// no footprint term, and (facing << 5) & 0xff for the zero vector.
func TestStepHeadingTakesTheSignsOfTheFineVector(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fine   [2]uint8
		size   uint8
		x, y   int32
		facing uint8
		want   uint8
	}{
		{"two right one down is the diagonal", [2]uint8{128, 128}, 1, 12, 11, 0, 96},
		{"a size-3 mover has no footprint term", [2]uint8{128, 128}, 3, 11, 10, 0, 64},
		{"a sub-cell east of centre steps west within its cell", [2]uint8{129, 128}, 1, 10, 11, 0, 160},
		{"a sub-cell on the next cell's centre line", [2]uint8{128, 0}, 1, 10, 10, 0, 128},
		{"zero vector at facing 64", [2]uint8{128, 128}, 1, 10, 10, 64, 0},
		{"zero vector at facing 0x77", [2]uint8{128, 128}, 1, 10, 10, 0x77, 224},
	} {
		w := fineWorld(map[EntityID][2]uint8{1: tc.fine})
		e := Entity{ID: 1, X: 10, Y: 10, TokenSize: tc.size, Facing: tc.facing}
		if got := w.stepHeading(e, tc.x, tc.y); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}

// A ranged attacker faces a victim nine east and four south on the east
// axis (AI-444), not the south-east diagonal.
func TestARangedAttackerNineByFourFacesTheAxis(t *testing.T) {
	a := cbFighter(1, 2, 2, 8, 4)
	a.Reach = 12
	a.RotationSpeed = 32
	v := cbEnt(2, 11, 6)
	v.HP, v.MaxHP = 1_000_000, 1_000_000
	w := cbWorld(t, 1, a, v)
	Step(w, []Command{cbOrder(1, 2)})
	for n := 0; n < 10; n++ {
		Step(w, nil)
	}
	got := cbAt(t, w, 1)
	if got.X != 2 || got.Y != 2 {
		t.Fatalf("attacker moved to (%d,%d); the victim is inside its reach", got.X, got.Y)
	}
	if got.Facing != 64 {
		t.Errorf("attacker faces %d, want 64 toward a victim at (9,4)", got.Facing)
	}
}
