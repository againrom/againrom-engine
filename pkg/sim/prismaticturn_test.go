package sim

import (
	"reflect"
	"testing"
)

// At equal edge distance the foe nearest the heading to the primary wins.
func TestPrismaticSpraySecondaryTieBreaksOnTurnFromCasterFacing(t *testing.T) {
	for _, c := range []struct {
		name    string
		primary [2]int32
		want    []EntityID
	}{
		{"facing east", [2]int32{10, 1}, []EntityID{2, 4}},
		{"facing south", [2]int32{1, 10}, []EntityID{2, 3}},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := prismaticWorld(t, 0, // cap 2
				prismaticFoeAt(2, c.primary[0], c.primary[1]),
				prismaticFoeAt(3, 1, 4), // edge 3, listed first
				prismaticFoeAt(4, 4, 1), // edge 3
			)
			if _, ids := prismaticCast(t, w, 2); !reflect.DeepEqual(ids, c.want) {
				t.Fatalf("victims %v, want %v", ids, c.want)
			}
		})
	}
}

func TestPrismaticSprayTurnNeverOutranksDistance(t *testing.T) {
	w := prismaticWorld(t, 0, // cap 2
		prismaticFoeAt(2, 10, 10),
		prismaticFoeAt(3, 1, 3), // edge 2, directly opposite the facing
		prismaticFoeAt(4, 5, 1), // edge 4, dead ahead
	)
	if _, ids := prismaticCast(t, w, 2); !reflect.DeepEqual(ids, []EntityID{2, 3}) {
		t.Fatalf("victims %v, want the nearer foe despite its turn", ids)
	}
}

func TestPrismaticTurnCostIsTheCircularArc(t *testing.T) {
	caster := Entity{X: 50, Y: 50, TokenSize: 1}
	for _, c := range []struct {
		facing uint8
		dx, dy int32
		want   uint32
	}{
		{0, 0, -4, 0},   // heading 0
		{64, 4, 0, 0},   // heading 64
		{0, 4, 0, 64},   // heading 64 from 0
		{250, 4, 0, 70}, // across the wrap: |250-64| = 186 -> 70
		{0, 0, 4, 128},  // opposite
		{0, 0, 0, 32},   // coincident centre heads 224
	} {
		got := (&World{}).prismaticTurnCost(c.facing, caster, Entity{X: 50 + c.dx, Y: 50 + c.dy, TokenSize: 1})
		if got != c.want {
			t.Errorf("facing %d delta (%d,%d): turn %d, want %d", c.facing, c.dx, c.dy, got, c.want)
		}
	}
}

// The selector flips the primary owner hostile before building candidates.
func TestPrismaticSprayFlipsThePrimaryOwnerBeforeSelecting(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<14)
	rel := engRel(t, [3]uint32{SelfSlot, prismaticFoe, relationHostile})
	primary := prismaticFoeAt(2, 6, 1)
	primary.Owner = 3
	mate := prismaticFoeAt(3, 3, 3)
	mate.Owner = 3
	w := hlWorld(t, 0x5e1ec8, rel, []SpellRule{prismaticTestRule}, caster, primary, mate)
	if w.hostileTo(w.entities[0], w.entities[1]) {
		t.Fatal("fixture owner is already hostile")
	}
	_, ids := prismaticCast(t, w, 2)
	if want := []EntityID{2, 3}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("victims %v, want the primary and its owner's other member %v", ids, want)
	}
	if !w.hostileTo(w.entities[0], w.entities[1]) {
		t.Fatal("selector did not flip the primary owner hostile")
	}
}
