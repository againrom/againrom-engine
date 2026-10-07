package game

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/formats/bmp"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func defendPanel1087(t *testing.T, f *FrontEnd, a *ui.App, selected bool) {
	t.Helper()
	pic, err := a.HeadlessCommandPanel()
	if err != nil {
		t.Fatal(err)
	}
	if pic.Bounds() != image.Rect(0, 0, 176, 80) {
		t.Fatal("panel dimensions", pic.Bounds())
	}
	// Literal source addresses and cell rectangles, not the production skip
	// mask, selected-cell helper, Loaded art, or commandCellRects.
	for _, part := range []struct {
		path string
		rect image.Rectangle
	}{
		{"graphics/interface/commandbarr.bmp", image.Rect(110, 41, 144, 75)},
		{map[bool]string{false: "graphics/interface/commandbarr.bmp", true: "graphics/interface/commanddnr.bmp"}[selected], image.Rect(110, 7, 144, 41)},
	} {
		raw, err := f.Archives.Containers.ReadFile(part.path)
		if err != nil {
			t.Fatal(err)
		}
		want, err := bmp.Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		for y := part.rect.Min.Y; y < part.rect.Max.Y; y++ {
			for x := part.rect.Min.X; x < part.rect.Max.X; x++ {
				c := want.At(x, y)
				if got := pic.RGBAAt(x+16, y); got != (color.RGBA{c.R, c.G, c.B, 255}) {
					t.Fatalf("Defend/Retreat installed art mismatch %s at%d,%d", part.path, x, y)
				}
			}
		}
	}
}

func defendNative1087(t *testing.T, f *FrontEnd, stage string) {
	t.Helper()
	snap, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	opener, town, err := back.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatalf("%s restore:%v town=%v", stage, err, town)
	}
	a := back.App("1087-restore")
	a.Layout(1024, 768)
	if err := a.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snap.World, mustDefendForm1087(t, back.live.world)) || len(back.live.pending) != 0 {
		t.Fatal(stage, "restore mutated or queued")
	}
	var control sim.World
	if err := control.UnmarshalBinary(snap.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	for n := 0; n < 33; n++ {
		sim.Step(&control, nil)
		sim.Step(back.live.world, nil)
		if control.Hash() != back.live.world.Hash() {
			t.Fatalf("%s continuation differs at%d", stage, n)
		}
	}
	t.Logf("native %s: exact form and33-tick continuation", stage)
}

func mustDefendForm1087(t *testing.T, w *sim.World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReleasePlayerDefend1087AppPanelMapMinimapAndNative(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("1087-player-defend")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	hero := live.mission.ids[0]
	var subject sim.EntityID
	found := false
	for _, e := range live.world.Entities() {
		if e.ID != hero && e.Alive() && !e.OffMap {
			subject = e.ID
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no installed actor subject")
	}
	// Only fixture placement and presentation fog change. The actor, stats,
	// bitmap art, App, group queue, world and native save are production owners.
	for _, p := range []struct {
		id   sim.EntityID
		x, y int32
	}{{hero, 20, 60}, {subject, 26, 60}} {
		if err := live.world.HeadlessPlace(p.id, p.x, p.y); err != nil {
			t.Fatal(err)
		}
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	live.view.Camera().CenterOn(23*32, 60*32)
	if err := a.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	defendPanel1087(t, f, a, false)
	defendNative1087(t, f, "before")
	key := func(k string) {
		t.Helper()
		if err := a.HeadlessKey(k); err != nil {
			t.Fatal(err)
		}
	}
	pointer := func(action string, x, y int) {
		t.Helper()
		if err := a.HeadlessPointer(action, x, y); err != nil {
			t.Fatal(err)
		}
	}
	key("d")
	key("d")
	defendPanel1087(t, f, a, true)
	live.view.Camera().CenterOn(23*32, 60*32)
	x, y, err := a.HeadlessEntityPoint(uint32(subject))
	if err != nil {
		t.Fatal(err)
	}
	beforeSubject, _ := live.entity(subject)
	pointer("press", x, y)
	pointer("release", x, y)
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindGroupDefend || live.pending[0].Entity != hero || uint32(live.pending[0].X) != uint32(subject) {
		t.Fatalf("D map queue:%+v", live.pending)
	}
	defendPanel1087(t, f, a, false)
	key("0")
	for n := 0; n < 32; n++ {
		e, _ := live.entity(hero)
		if e.ActorState == 8 {
			break
		}
		if err := entryStep1084(a); err != nil {
			t.Fatal(err)
		}
	}
	e, _ := live.entity(hero)
	target, _ := live.entity(subject)
	if e.ActorState != 8 || !e.HasEscortTarget || e.EscortTarget != subject || e.EscortRange != 3 {
		t.Fatalf("D did not install defend:%+v", e)
	}
	if target.ActorState != beforeSubject.ActorState || target.CommandGroup != beforeSubject.CommandGroup {
		t.Fatal("unselected protected actor's order changed")
	}
	defendNative1087(t, f, "following")
	startX, startY := e.X, e.Y
	for n := 0; n < 128; n++ {
		e, _ = live.entity(hero)
		if e.X != startX || e.Y != startY {
			break
		}
		if err := entryStep1084(a); err != nil {
			t.Fatal(err)
		}
	}
	if e.X == startX && e.Y == startY {
		t.Fatal("installed defender did not accompany")
	}
	key("0")
	// Minimap navigation preserves the panel's armed mode without an order.
	// A subsequent world click exercises acquire-in-place in the live App.
	x, y, err = a.HeadlessCommandPoint(3)
	if err != nil {
		t.Fatal(err)
	}
	pointer("press", x, y)
	pointer("release", x, y)
	defendPanel1087(t, f, a, true)
	e, _ = live.entity(hero)
	x, y, err = a.HeadlessMinimapPoint(int(e.X), int(e.Y))
	if err != nil {
		t.Fatal(err)
	}
	n := len(live.pending)
	pointer("press", x, y)
	pointer("release", x, y)
	if len(live.pending) != n {
		t.Fatalf("minimap queued an order:%+v", live.pending)
	}
	defendPanel1087(t, f, a, true)
	x, y, err = a.HeadlessEntityPoint(uint32(hero))
	if err != nil {
		t.Fatal(err)
	}
	pointer("press", x, y)
	pointer("release", x, y)
	if len(live.pending) != n+1 || live.pending[n].Kind != sim.KindGroupDefend || uint32(live.pending[n].X) != uint32(hero) {
		t.Fatalf("panel/world queue:%+v", live.pending)
	}
	defendPanel1087(t, f, a, false)
	key("0")
	for n := 0; n < 32; n++ {
		e, _ = live.entity(hero)
		if e.ActorState == 0xc {
			break
		}
		if err := entryStep1084(a); err != nil {
			t.Fatal(err)
		}
	}
	if e.ActorState != 0xc || e.HasEscortTarget {
		t.Fatalf("selected subject not acquiring:%+v", e)
	}
	defendNative1087(t, f, "acquire")
	t.Logf("App D+D/map actor%d ->%d; moved(%d,%d); minimap preserves arm; panel3/world self ->state0xc; installed Defend art enabled/selected, Retreat enabled", hero, subject, startX, startY)
}
