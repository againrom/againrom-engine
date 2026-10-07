package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseMapMarqueeEndsOnCommandPanel(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	other := party[0]
	other.ID = "marquee-companion"
	a := f.App("marquee across HUD")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, []mapload.PartyMember{party[0], other})); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	// The installed layout initially shows the book along the left edge.
	if err := a.HeadlessKey("book"); err != nil {
		t.Fatal(err)
	}
	actor, ok := live.entity(id)
	if !ok {
		t.Fatal("missing party actor")
	}
	cam := live.view.Camera()
	cam.X, cam.Y = float64(actor.X*32-160), float64(actor.Y*32-120)
	cam.Clamp()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	// Both scattered actor figures fit inside the map above this press.
	dragY := 660
	before := live.world.Hash()
	x, y, err := a.HeadlessCommandPoint(0) // active Attack button
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []struct {
		edge string
		x, y int
	}{{"press", 40, dragY}, {"move", x, y}, {"release", x, y}} {
		if err := a.HeadlessPointer(event.edge, event.x, event.y); err != nil {
			t.Fatal(err)
		}
	}
	want := []uint32{uint32(id), uint32(live.mission.ids[1])}
	if got := a.HeadlessSelection(); !reflect.DeepEqual(got, want) {
		t.Fatalf("selection = %v, want both party actors%v", got, want)
	}
	if len(live.pending) != 0 || live.world.Hash() != before {
		t.Fatal("selection issued an order or changed the paused world")
	}
	// A subsequent plain ground tap must still be an ordinary move. Crossing
	// the active Attack button while dragging must not arm attack mode.
	entryPointer1084(t, a, int(actor.X)+1, int(actor.Y))
	if len(live.pending) != 2 || live.pending[0].Kind != sim.KindGroupMoveTo || live.pending[1].Kind != sim.KindGroupMoveTo {
		t.Fatalf("post-marquee ground tap queued %v", live.pending)
	}
}
