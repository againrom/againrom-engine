package sim

import (
	"reflect"
	"testing"
)

// The installed shape of Prismatic Spray: delivery 2, fixed ten-tick flight.
var prismaticTestRule = SpellRule{ID: 14, ManaCost: 4, School: 1, MaxRange: 15, TargetsUnit: true,
	Damaging: true, DamageMin: 5, DamageMax: 5, Delivery: 2, EffectSpeed: 128}

const prismaticFoe, prismaticAlly = 4, 2

func prismaticFoeAt(id EntityID, x, y int32) Entity {
	e := spEnt(id, x, y)
	e.TokenSize, e.Owner = 1, prismaticFoe
	return e
}

// prismaticWorld places a player mage (entity 1) at (1,1) whose school level
// is the spell power, so the cap is min(level/20+2,7).
func prismaticWorld(t *testing.T, level int32, ents ...Entity) *World {
	t.Helper()
	caster := effectMage(1, 1, 1, 1<<14)
	caster.Skill[1] = level
	rel := engRel(t, [3]uint32{SelfSlot, prismaticFoe, relationHostile},
		[3]uint32{SelfSlot, prismaticAlly, relationLocked})
	return hlWorld(t, 0x5e1ec7, rel, []SpellRule{prismaticTestRule}, append([]Entity{caster}, ents...)...)
}

// prismaticCast orders the spray at primary and returns the admission
// observation's ray cells and the queued deliveries' victims, in order.
func prismaticCast(t *testing.T, w *World, primary EntityID) ([]CellPoint, []EntityID) {
	t.Helper()
	events := StepObserved(w, []Command{spCast(1, primary, 14)})
	var ids []EntityID
	for _, d := range w.deliveries {
		ids = append(ids, d.Target)
	}
	if len(events) != 1 {
		t.Fatalf("admission produced %d observations, want 1 (deliveries %v)", len(events), ids)
	}
	return events[0].Victims, ids
}

func prismaticCells(t *testing.T, w *World, ids []EntityID) []CellPoint {
	var out []CellPoint
	for _, id := range ids {
		e := spAt(t, w, id)
		out = append(out, CellPoint{X: e.X, Y: e.Y})
	}
	return out
}

func TestPrismaticSprayTakesTheFoesNearestTheCasterUpToTheCap(t *testing.T) {
	// Primary at (9,1). Foes 3 and 4 stand nearest the caster and five or
	// more cells from the primary, outside any primary-centred radius.
	// Foe 6 is edge distance 4 away; foe 5 edge distance 3 and later in the list.
	w := prismaticWorld(t, 20, // cap 3
		prismaticFoeAt(2, 9, 1),
		prismaticFoeAt(3, 3, 1), // edge 2
		prismaticFoeAt(4, 2, 2), // edge 1
		prismaticFoeAt(5, 4, 3), // edge 3
		prismaticFoeAt(6, 5, 5), // edge 4
	)
	cells, ids := prismaticCast(t, w, 2)
	if want := []EntityID{2, 4, 3}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("victims %v, want the primary then the two foes nearest the caster %v", ids, want)
	}
	if want := prismaticCells(t, w, ids); !reflect.DeepEqual(cells, want) {
		t.Fatalf("ray cells %v, want one per victim in victim order %v", cells, want)
	}
	if got := edgeDistance(w.entities[0], spAt(t, w, 4)); got != 1 {
		t.Fatalf("adjacent foe edge distance %d, want 1", got)
	}
}

func TestPrismaticSprayEqualDistancesFallToListOrder(t *testing.T) {
	w := prismaticWorld(t, 0, // cap 2
		prismaticFoeAt(2, 10, 10),
		prismaticFoeAt(3, 1, 4), // edge 3
		prismaticFoeAt(4, 4, 1), // edge 3
	)
	if _, ids := prismaticCast(t, w, 2); !reflect.DeepEqual(ids, []EntityID{2, 3}) {
		t.Fatalf("victims %v, want the primary then the first equal-distance foe in list order", ids)
	}
}

func TestPrismaticSprayNeverChoosesAnAllyNeutralSelfOrUndetectedInvisibleSecondary(t *testing.T) {
	ally := prismaticFoeAt(3, 2, 1)
	ally.Owner = prismaticAlly
	neutral := prismaticFoeAt(4, 1, 2)
	neutral.Owner = 3
	w := prismaticWorld(t, 100, // cap 7
		prismaticFoeAt(2, 8, 8),
		ally, neutral,
		prismaticFoeAt(5, 2, 2), // invisible, outside the caster's zero detector
		prismaticFoeAt(6, 6, 6),
	)
	w.attached = []attachedEffect{{Target: 5, Spell: 15, Kind: EffectInvisible, Mode: EffectDuration, Remaining: 500}}
	if _, ids := prismaticCast(t, w, 2); !reflect.DeepEqual(ids, []EntityID{2, 6}) {
		t.Fatalf("victims %v, want only the primary and the visible foe", ids)
	}
}

func TestPrismaticSprayAimedAtAnAllyKeepsItAsThePrimary(t *testing.T) {
	ally := prismaticFoeAt(2, 3, 1)
	ally.Owner = prismaticAlly
	w := prismaticWorld(t, 0, ally, prismaticFoeAt(3, 6, 6), prismaticFoeAt(4, 12, 12))
	if _, ids := prismaticCast(t, w, 2); !reflect.DeepEqual(ids, []EntityID{2, 3}) {
		t.Fatalf("victims %v, want the aimed ally then the nearest foe", ids)
	}
}

func TestPrismaticSprayCapFollowsPower(t *testing.T) {
	var foes []Entity
	for id := EntityID(3); id <= 10; id++ {
		foes = append(foes, prismaticFoeAt(id, int32(id)-1, 1))
	}
	for _, c := range []struct {
		level int32
		want  int
	}{{0, 2}, {100, 7}} {
		w := prismaticWorld(t, c.level, append([]Entity{prismaticFoeAt(2, 14, 1)}, foes...)...)
		cells, ids := prismaticCast(t, w, 2)
		if len(ids) != c.want || len(cells) != c.want || ids[0] != 2 {
			t.Fatalf("level %d: victims %v rays %d, want %d with the primary first", c.level, ids, len(cells), c.want)
		}
		for k := 1; k < len(ids); k++ {
			if ids[k] != EntityID(2+k) {
				t.Fatalf("level %d: victims %v, want the nearest foes 3.. in distance order", c.level, ids)
			}
		}
	}
}

func TestPrismaticSprayFallsBackToBodiesWhenTheGroupSeesNoLivingFoe(t *testing.T) {
	ally := prismaticFoeAt(2, 3, 1)
	ally.Owner = prismaticAlly
	near := prismaticFoeAt(3, 2, 2)
	near.HP, near.Decay = 0, DecayFallen
	far := prismaticFoeAt(4, 5, 5)
	far.HP, far.Decay = -4, DecayFallen
	w := prismaticWorld(t, 20, ally, far, near)
	control := prismaticWorld(t, 20, ally, far, near)
	cells, ids := prismaticCast(t, w, 2)
	if want := []EntityID{2, 3, 4}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("victims %v, want the aimed ally then the bodies by distance %v", ids, want)
	}
	if len(cells) != 3 {
		t.Fatalf("%d rays, want one per victim", len(cells))
	}
	Step(control, nil)
	for n := 0; n < 12; n++ {
		Step(w, nil)
		Step(control, nil)
	}
	// A body victim takes the ordinary finishing damage of a damaging row.
	for _, id := range []EntityID{3, 4} {
		if got, idle := spAt(t, w, id).HP, spAt(t, control, id).HP; got >= idle {
			t.Fatalf("body %d at %d beside %d without the spray, want the finishing damage of a damaging row", id, got, idle)
		}
	}
}

func TestPrismaticSprayRefusesABodyAsThePrimary(t *testing.T) {
	body := prismaticFoeAt(2, 3, 1)
	body.HP, body.Decay = -2, DecayFallen
	w := prismaticWorld(t, 0, body, prismaticFoeAt(3, 4, 4))
	events := StepObserved(w, []Command{spCast(1, 2, 14)})
	if len(events) != 0 || len(w.deliveries) != 0 || len(w.bookCasts) != 0 || w.entities[0].Mana != 100 {
		t.Fatalf("player order at a body: events %d deliveries %d casts %d mana %d, want a refusal",
			len(events), len(w.deliveries), len(w.bookCasts), w.entities[0].Mana)
	}
	if w.beginBookSpell(0, 2, 14) {
		t.Fatal("an AI cast admitted a body as the primary")
	}
	if id, ok := w.autoCastTarget(0, prismaticTestRule); !ok || id != 3 {
		t.Fatalf("AI target %d %v, want the living foe 3", id, ok)
	}
}

// A group sees as one animal: a foe seen by any member of the caster's group
// is a candidate, one no member sees is not, however near the caster it
// stands (AI-GROUPSEE-068). Without the member that sees it the far foe drops
// out, and the unseen foe is never chosen. A caster given a command group by an
// order stands in a group of its own, so the member's sight is not shared with it.
func TestPrismaticSpraySecondariesComeFromTheWholeGroupsSight(t *testing.T) {
	build := func(withMember bool) *World {
		ents := []Entity{
			prismaticFoeAt(2, 2, 1),   // the primary, in the caster's own sight
			prismaticFoeAt(3, 4, 1),   // in the caster's own sight, edge 4
			prismaticFoeAt(4, 13, 12), // seen only by the member at (13,13)
			prismaticFoeAt(5, 8, 8),   // seen by nobody, nearer than foe 4
		}
		if withMember {
			member := spEnt(6, 13, 13)
			member.Owner, member.TokenSize, member.ScanRange = SelfSlot, 1, 3
			ents = append(ents, member)
		}
		w := prismaticWorld(t, 100, ents...)
		w.entities[0].ScanRange = 3
		return w
	}
	ids := func(w *World, caster EntityID) []EntityID {
		var out []EntityID
		for _, i := range w.prismaticVictims(indexOfEntity(w.entities, caster), 1, prismaticTestRule, 100) {
			out = append(out, w.entities[i].ID)
		}
		return out
	}
	if got, want := ids(build(true), 1), []EntityID{2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("with the member, victims %v, want the primary, the foe in the caster's sight, then the foe only the member sees %v", got, want)
	}
	if got, want := ids(build(false), 1), []EntityID{2, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("without the member, victims %v, want %v", got, want)
	}
	commanded := build(true)
	commanded.entities[0].CommandGroup = 77
	if got, want := ids(commanded, 1), []EntityID{2, 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("a caster in a command group of its own: victims %v, want %v", got, want)
	}
}

// A corpse beside a living candidate never scores below the 65530 sentinel:
// the edge distance is at least 1 and at most 255 for every pair of cells and
// footprint sizes, so list B's score, the A score shifted left by eight, is at
// least 1<<16 (MAGIC-SPRAY-137).
func TestPrismaticEdgeDistanceKeepsListBAboveTheSentinel(t *testing.T) {
	for size := uint8(1); size <= 4; size++ {
		for a := int32(0); a < 256; a += 5 {
			for b := int32(0); b < 256; b++ {
				d := edgeDistance(Entity{X: a, Y: a, TokenSize: size}, Entity{X: b, Y: 255 - b, TokenSize: size})
				if d < 1 {
					t.Fatalf("size %d cells %d,%d: edge distance %d", size, a, b, d)
				}
				if score := (uint32(d)<<8 + 128) & 0xffff << 8; score < 65530 {
					t.Fatalf("size %d cells %d,%d: list B score %d is below the sentinel", size, a, b, score)
				}
			}
		}
	}
}
