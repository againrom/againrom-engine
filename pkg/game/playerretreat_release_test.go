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

func retreatPanel1089(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	pic, err := app.HeadlessCommandPanel()
	if err != nil {
		t.Fatal(err)
	}
	if pic.Bounds() != image.Rect(0, 0, 176, 80) {
		t.Fatal("panel dimensions", pic.Bounds())
	}
	// Independent installed-source expectation: the Retreat cell is active
	// and unselected both before and after its immediate command. No production
	// skip mask, rectangle helper or loaded panel art constructs this oracle.
	raw, err := f.Archives.Containers.ReadFile("graphics/interface/commandbarr.bmp")
	if err != nil {
		t.Fatal(err)
	}
	want, err := bmp.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for y := 41; y < 75; y++ {
		for x := 110; x < 144; x++ {
			c := want.At(x, y)
			if pic.RGBAAt(x+16, y) != (color.RGBA{c.R, c.G, c.B, 255}) {
				t.Fatalf("active Retreat art at %d,%d", x, y)
			}
		}
	}
}

func retreatNative1089(t *testing.T, f *FrontEnd, stage string) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(snapshot, label)
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
		t.Fatal(stage, "restore", err, town)
	}
	app := back.App("1089-restore")
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	form, err := back.live.world.MarshalBinary()
	if err != nil || !bytes.Equal(form, snapshot.World) || len(back.live.pending) != 0 {
		t.Fatal(stage, "native bytes/queue", err)
	}
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	for n := 0; n < 65; n++ {
		sim.Step(&control, nil)
		sim.Step(back.live.world, nil)
		if control.Hash() != back.live.world.Hash() {
			t.Fatal(stage, "continuation", n)
		}
	}
	t.Logf("native %s: exact bytes, empty queue, 65-tick continuation", stage)
}

func TestReleasePlayerRetreat1089AppPanelAndNative(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1089-player-retreat")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	live := f.live
	hero := live.mission.ids[0]
	before, _ := live.entity(hero)
	var hostile sim.EntityID
	found := false
	for _, e := range live.world.Entities() {
		if e.Alive() && !e.OffMap && live.world.Relations().Hostile(before.Owner, e.Owner) {
			hostile, found = e.ID, true
			break
		}
	}
	if !found {
		t.Fatal("installed mission has no hostile")
	}
	// Fixture placement/fog only. Installed identities, stats, diplomacy,
	// command queue, movement and native restore stay production-owned.
	for _, p := range []struct {
		id   sim.EntityID
		x, y int32
	}{{hero, 20, 60}, {hostile, 24, 60}} {
		if err := live.world.HeadlessPlace(p.id, p.x, p.y); err != nil {
			t.Fatal(err)
		}
	}
	for i := range live.fog.visible {
		live.fog.visible[i], live.fog.explored[i] = 1, 1
	}
	live.push()
	live.view.Camera().CenterOn(22*32, 60*32)
	if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	retreatPanel1089(t, f, app)
	retreatNative1089(t, f, "before")
	for n := 0; n < 2; n++ {
		if err := app.HeadlessKey("r"); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 2 {
		t.Fatalf("repeated R queue=%+v", live.pending)
	}
	for _, c := range live.pending {
		if c.Kind != sim.KindGroupRetreat || c.Entity != hero || c.Player != sim.SelfSlot {
			t.Fatal("wrong Retreat command", c)
		}
	}
	still, _ := live.entity(hero)
	if still.ActorState == 0x16 {
		t.Fatal("paused queue mutated world")
	}
	retreatPanel1089(t, f, app)
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 200; n++ {
		e, _ := live.entity(hero)
		if e.ActorState == 0x16 && e.X < 20 {
			break
		}
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
	}
	e, _ := live.entity(hero)
	if e.ActorState != 0x16 || e.X >= 20 {
		t.Fatalf("healthy installed hero did not retreat: state=%d x=%d", e.ActorState, e.X)
	}
	if e.Withdraw != before.Withdraw || e.Wimpy != before.Wimpy {
		t.Fatal("explicit command changed thresholds")
	}
	retreatNative1089(t, f, "retreating")
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	x, y, err := app.HeadlessCommandPoint(7)
	if err != nil {
		t.Fatal(err)
	}
	n := len(live.pending)
	if err := app.HeadlessPointer("press", x, y); err != nil {
		t.Fatal(err)
	}
	if len(live.pending) != n+1 || live.pending[n].Kind != sim.KindGroupRetreat {
		t.Fatal("panel down did not queue Retreat")
	}
	if err := app.HeadlessPointer("release", x, y); err != nil {
		t.Fatal(err)
	}
	if len(live.pending) != n+1 {
		t.Fatal("panel release repeated Retreat")
	}
	retreatPanel1089(t, f, app)
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 32 && len(live.pending) != 0; n++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 0 {
		t.Fatal("panel command never applied")
	}
	retreatNative1089(t, f, "panel-reissue")
	t.Logf("M10 hero %d retreated from x=20 to x=%d away from installed hostile %d; thresholds %d/%d unchanged", hero, e.X, hostile, e.Withdraw, e.Wimpy)
}
