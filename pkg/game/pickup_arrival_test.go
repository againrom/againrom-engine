package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

func pickupArrivalWorld(t *testing.T) (*sim.World, *mapWorld) {
	t.Helper()
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 20, MaxHP: 20, Speed: 16, Facing: 64, DesiredFacing: 64}}, nil, sim.Relations{},
		[]sim.Sack{{X: 4, Y: 3, Gold: 500, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	mw.invParty.table = eqDefsTable(t)
	return w, mw
}

func requirePickupArrivalGround(t *testing.T, w *sim.World, mw *mapWorld) {
	t.Helper()
	sacks := w.Sacks()
	codes, ok := w.Carried(7)
	if len(sacks) != 1 || sacks[0].X != 4 || sacks[0].Y != 3 || sacks[0].Gold != 500 || !reflect.DeepEqual(sacks[0].Items, []uint16{eqSwordCode}) || !ok || len(codes) != 0 || w.Purse(sim.SelfSlot) != 0 || len(mw.view.MessageLines()) != 0 {
		t.Fatalf("transfer occurred before arrival: sacks=%+v carried=%v purse=%d rows=%v", sacks, codes, w.Purse(sim.SelfSlot), mw.view.MessageLines())
	}
}

func requirePickupArrivalTaken(t *testing.T, w *sim.World) {
	t.Helper()
	codes, ok := w.Carried(7)
	if len(w.Sacks()) != 0 || !ok || !reflect.DeepEqual(codes, []uint16{eqSwordCode}) || w.Purse(sim.SelfSlot) != 500 {
		t.Fatalf("arrival did not transfer exactly one sack: sacks=%+v carried=%v purse=%d", w.Sacks(), codes, w.Purse(sim.SelfSlot))
	}
}

func TestPickupArrivalWaitsForRatedStride(t *testing.T) {
	w, mw := pickupArrivalWorld(t)
	mw.grab(7, 4, 3, true)
	requirePickupArrivalGround(t, w, mw)
	for paid := 1; paid <= 15; paid++ {
		mw.tick()
		e, _ := w.Entity(7)
		if e.X != 4 || e.Y != 3 || e.TransitTotal != 16 || int(e.Transit) != 16-paid {
			t.Fatalf("fixture lost the sixteen-tick eastward stride at payment %d: %+v", paid, e)
		}
		requirePickupArrivalGround(t, w, mw)
		if !mw.pickup.set {
			t.Fatal("crossing consumed the pending pickup order")
		}
	}
	mw.tick()
	e, _ := w.Entity(7)
	if e.Transit != 0 {
		t.Fatal("sixteenth payment did not finish the stride", e.Transit)
	}
	requirePickupArrivalTaken(t, w)
	rows := mw.view.MessageLines()
	if mw.pickup.set || !reflect.DeepEqual(rows, announced("Picked up 500 gold", "Picked up Sword")) {
		t.Fatal("arrival did not consume its order and post gold then item once", mw.pickup, rows)
	}
	sim.Step(w, []sim.Command{sim.DropCarried(7, 0, sim.CellPoint{X: 4, Y: 3})})
	for range 3 {
		mw.tick()
	}
	codes, _ := w.Carried(7)
	sacks := w.Sacks()
	if len(sacks) != 1 || !reflect.DeepEqual(sacks[0].Items, []uint16{eqSwordCode}) || sacks[0].Gold != 0 || len(codes) != 0 || w.Purse(sim.SelfSlot) != 500 || !reflect.DeepEqual(mw.view.MessageLines(), rows) {
		t.Fatal("completed pickup repeated on a newly dropped sack", sacks, codes, mw.view.MessageLines())
	}
}

func TestPickupArrivalUnderfootCannotInterruptStride(t *testing.T) {
	for _, route := range []string{"key", "headless"} {
		t.Run(route, func(t *testing.T) {
			w, mw := pickupArrivalWorld(t)
			mission := &Mission{World: w}
			attempt := func() bool {
				if route == "key" {
					mw.grab(0, 0, 0, false)
					return len(w.Sacks()) == 0
				}
				_, taken := mission.PickUpUnderfoot(7)
				return taken
			}
			sim.Step(w, []sim.Command{sim.MoveTo(7, sim.CellPoint{X: 4, Y: 3})})
			for paid := 1; paid <= 15; paid++ {
				e, _ := w.Entity(7)
				if e.X != 4 || e.Y != 3 || int(e.Transit) != 16-paid {
					t.Fatalf("crossing fixture changed at payment %d: %+v", paid, e)
				}
				before := w.Hash()
				if attempt() || w.Hash() != before {
					t.Fatal("underfoot input changed the crossing before arrival")
				}
				requirePickupArrivalGround(t, w, mw)
				sim.Step(w, nil)
			}
			if !attempt() {
				t.Fatal("underfoot input could not take the sack on arrival")
			}
			requirePickupArrivalTaken(t, w)
			before, rows := w.Hash(), mw.view.MessageLines()
			attempt()
			if w.Hash() != before || !reflect.DeepEqual(mw.view.MessageLines(), rows) {
				t.Fatal("repeated underfoot input changed the completed pickup")
			}
		})
	}
}

func TestPickupArrivalWaitsForRetainedFineMotion(t *testing.T) {
	w, mw := pickupArrivalWorld(t)
	order := sim.SavedActorOrder{Entity: 7, State: 0xb}
	order.Raw[8], order.Raw[9] = 1, 3
	group := sim.SavedGroup{ID: 1, Selector: 1, Members: []sim.SavedGroupMember{{Archive: 1, Entity: 7, Bound: true}}}
	if err := w.ImportSavedGroups([]sim.SavedGroup{group}, []sim.SavedActorOrder{order}); err != nil {
		t.Fatal(err)
	}
	motion := sim.SavedActorMotion{Entity: 7, Position: sim.SavedActorPosition{Cell: 0x0304, PackedCell: 0x0304, FineX: 32, FineY: 128}, ActorAction: 1}
	binary.LittleEndian.PutUint16(motion.Mover[0xaa:], 16)
	binary.LittleEndian.PutUint16(motion.Mover[0xac:], 10)
	binary.LittleEndian.PutUint16(motion.Mover[0xae:], 2)
	motion.Mover[0xb0] = 16
	if err := w.ImportOriginalActorMotions([]sim.SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	mission := &Mission{World: w}
	for paid := 0; paid < 6; paid++ {
		x, y, present := w.ActorFinePosition(7)
		if !present || int(x) != 32+16*paid || y != 128 {
			t.Fatalf("retained crossing fixture changed at payment %d: (%d,%d), present=%t", paid, x, y, present)
		}
		before := w.Hash()
		if _, taken := mission.PickUpUnderfoot(7); taken || w.Hash() != before {
			t.Fatal("retained crossing was interrupted by underfoot input")
		}
		requirePickupArrivalGround(t, w, mw)
		sim.Step(w, nil)
	}
	x, y, present := w.ActorFinePosition(7)
	if !present || x != 128 || y != 128 {
		t.Fatal("retained crossing did not reach the center", x, y, present)
	}
	if _, taken := mission.PickUpUnderfoot(7); !taken {
		t.Fatal("centered retained actor could not pick up")
	}
	requirePickupArrivalTaken(t, w)
}

func TestHeadlessPickUpWaitsForCompletedCrossing(t *testing.T) {
	w, _ := pickupArrivalWorld(t)
	p := &PlayWorld{World: w, Party: []sim.EntityID{7}}
	if err := p.pickUp(7, 4, 3); err != nil {
		t.Fatal(err)
	}
	if e, _ := w.Entity(7); e.Transit != 0 || w.Tick() != 16 {
		t.Fatalf("pick-up did not wait for the sixteen crossing payments: tick=%d actor=%+v", w.Tick(), e)
	}
	requirePickupArrivalTaken(t, w)
}
