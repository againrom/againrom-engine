package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseSavedIdleFramesContinueFromTheWorldTick(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.ChargenParty(ui.ChargenResult{Name: "Scene clock", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("saved idle frame")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	if f.live.world.Tick() != 0 || f.live.scene != 0 {
		t.Fatalf("fresh mission starts at tick %d, scene %d", f.live.world.Tick(), f.live.scene)
	}
	f.LiveAdvance(9)
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := loadSAVWindow(t, store, name)
	if g.live.world.Tick() != f.live.world.Tick() || g.live.scene != f.live.scene {
		t.Fatalf("cold LOAD tick %d scene %d; running tick %d scene %d", g.live.world.Tick(), g.live.scene, f.live.world.Tick(), f.live.scene)
	}
	compared := 0
	for step := 0; step <= 5; step++ {
		before := map[uint32]ui.MapEntity{}
		for _, draw := range f.live.entityDraws() {
			before[draw.ID] = draw
		}
		for _, draw := range g.live.entityDraws() {
			orig, ok := before[draw.ID]
			if !ok || orig.Frame == nil || orig.Step != (image.Point{}) || f.live.world.ActorMotionActive(sim.EntityID(draw.ID)) {
				continue
			}
			e, ok := f.live.world.Entity(sim.EntityID(draw.ID))
			if !ok || !e.Alive() {
				continue
			}
			compared++
			if draw.DrawCategory != orig.DrawCategory || draw.Mirror != orig.Mirror || !reflect.DeepEqual(draw.Frame, orig.Frame) {
				t.Fatalf("idle actor %d frame differs at continuation step %d, tick %d", draw.ID, step, f.live.world.Tick())
			}
		}
		if step == 5 {
			break
		}
		f.LiveAdvance(1)
		g.LiveAdvance(1)
		if g.live.world.Tick() != f.live.world.Tick() || g.live.scene != f.live.scene {
			t.Fatalf("continuation step %d tick/scene %d/%d vs %d/%d", step+1, g.live.world.Tick(), g.live.scene, f.live.world.Tick(), f.live.scene)
		}
	}
	if compared == 0 {
		t.Fatal("no drawable idle actor crossed the save and load boundary")
	}
}
