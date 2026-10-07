package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Prismatic Spray's secondaries on mission 20's own terrain, height plane and
// spell rules. The mage, the foes and the arrangement are fixtures. A group sees as
// one animal and the secondaries are the group's candidates nearest the caster,
// so the foes below, all out of the mage's own sight, are chosen by the sight of
// the mage's group-mate and ranked by distance, not by list order. The mage casts
// unbidden: a player order builds a command group of the caster alone.

const prismaticGroupFoe = 4

type prismaticGroupRig struct {
	m             [3][]byte
	width, height int
	px, py        int
	rules         []sim.SpellRule
	relations     sim.Relations
}

func (r *prismaticGroupRig) world(t *testing.T, mateScan uint8, withMate bool) (*sim.World, [][2]int32) {
	t.Helper()
	at := func(dx, dy int) (int32, int32) { return int32(r.px + dx), int32(r.py + dy) }
	person := func(id sim.EntityID, owner uint32, dx, dy int, scan uint8) sim.Entity {
		x, y := at(dx, dy)
		return sim.Entity{ID: id, X: x, Y: y, PostX: x, PostY: y, HP: 5000, MaxHP: 5000, DyingTime: 200, Owner: owner,
			TokenSize: 1, ScanRange: scan, Reach: 1}
	}
	mage := person(1, sim.SelfSlot, 1, 7, 4)
	mage.Mind, mage.MaxMana, mage.Mana, mage.KnownSpells, mage.AutoSpell = 60, 900, 900, 1<<14, 14
	for i := range mage.Skill {
		mage.Skill[i] = 40
	}
	// The foes are listed from the farthest to the nearest, so list order and
	// distance order disagree. All stand beyond the mage's scan range of 4 and
	// inside the mate's.
	foes := [][2]int{{3, 7}, {13, 11}, {10, 6}, {11, 8}, {9, 9}}
	ents := []sim.Entity{mage, person(2, prismaticGroupFoe, foes[0][0], foes[0][1], 0)}
	for k, c := range foes[1:] {
		ents = append(ents, person(sim.EntityID(3+k), prismaticGroupFoe, c[0], c[1], 0))
	}
	if withMate {
		ents = append(ents, person(7, sim.SelfSlot, 12, 7, mateScan))
	}
	w, err := sim.NewStructuredWorld(2020, sim.Bounds{Width: int32(r.width), Height: int32(r.height)}, sim.ModeCanonical,
		sim.Terrain{Block: r.m[0], Cost: r.m[1], Height: r.m[2]}, ents, nil, r.relations, nil, nil,
		r.rules, sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var cells [][2]int32
	for _, e := range w.Entities() {
		cells = append(cells, [2]int32{e.X, e.Y})
	}
	return w, cells
}

func TestReleasePrismaticSpraySecondariesFollowTheGroupsSightAndRank(t *testing.T) {
	f := releaseFront(t)
	m := releaseMissionMap(t, f, 20)
	r := &prismaticGroupRig{width: m.Width, height: m.Height, rules: mapload.SpellRules(f.Table)}
	r.m = [3][]byte{mapload.PassabilityWith(m, f.Table), mapload.Cost(m), mapload.Height(m)}
	px, py, found := openGroundSquare(r.m[0], r.width, r.height, 15)
	if !found {
		t.Fatal("mission 20 has no open ground square of 15 cells")
	}
	r.px, r.py = px, py
	r.relations.Set(sim.SelfSlot, prismaticGroupFoe, 1)
	r.relations.Set(prismaticGroupFoe, sim.SelfSlot, 1)

	victims := func(w *sim.World) []sim.EntityID {
		var events []sim.CastEvent
		for tick := 0; tick < 4 && len(events) == 0; tick++ {
			events = sim.StepObserved(w, nil)
		}
		if len(events) != 1 {
			t.Fatalf("the unbidden cast produced %d observations, want 1", len(events))
		}
		var ids []sim.EntityID
		for _, c := range events[0].Victims {
			for _, e := range w.Entities() {
				if e.X == c.X && e.Y == c.Y && e.Owner == prismaticGroupFoe {
					ids = append(ids, e.ID)
				}
			}
		}
		return ids
	}
	with, _ := r.world(t, 6, true)
	got := victims(with)
	// Distances from the mage at (1,7): foe 6 is nine cells away, foe 4 nine and
	// the rest further; the order below is the original's score, distance first.
	want := []sim.EntityID{2, 6, 4, 5, 3}
	if len(got) < 3 || len(got) > len(want) || !reflect.DeepEqual(got, want[:len(got)]) {
		t.Errorf("with the group-mate's sight, victims %v, want a prefix of %v (primary, then by distance)", got, want)
	}
	// Loss controls: the mate gone, or present with no sight, leaves no
	// candidate in the group's view and the spray reaches its primary alone.
	absent, _ := r.world(t, 0, false)
	blind, _ := r.world(t, 0, true)
	for name, w := range map[string]*sim.World{"without the mate": absent, "a mate with no sight": blind} {
		if got := victims(w); !reflect.DeepEqual(got, []sim.EntityID{2}) {
			t.Errorf("%s, victims %v, want the primary alone", name, got)
		}
	}
	t.Logf("victims with the mate %v", got)
}
