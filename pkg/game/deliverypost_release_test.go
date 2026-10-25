package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func deliveryActor(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := f.live.world.Entity(id)
	if !ok {
		t.Fatal("missing delivery actor", id)
	}
	return e
}

func deliveryFrames(t *testing.T, f *FrontEnd, app *ui.App, until func() bool) {
	t.Helper()
	for k := 0; k < 12000; k++ {
		if until() {
			return
		}
		autoGetStep(t, f, app, "mission delivery")
	}
	t.Fatal("mission delivery did not reach its condition within 12000 frames")
}

func deliveryTicks(t *testing.T, f *FrontEnd, app *ui.App, n uint64) {
	t.Helper()
	end := f.live.world.Tick() + n
	deliveryFrames(t, f, app, func() bool { return f.live.world.Tick() >= end })
}

func deliveryProof(t *testing.T, f *FrontEnd, app *ui.App, witch sim.EntityID) {
	t.Helper()
	delivered := deliveryActor(t, f, witch)
	if delivered.Owner != 2 || delivered.PostX != delivered.X || delivered.PostY != delivered.Y || delivered.ActorState != 0xc || delivered.HasTarget {
		t.Fatalf("delivery did not apply Stand Ground at (%d,%d): %+v", delivered.X, delivered.Y, delivered)
	}
	deliveryTicks(t, f, app, 32)
	stood := deliveryActor(t, f, witch)
	if stood.X < delivered.X-1 || stood.X > delivered.X+1 || stood.Y < delivered.Y-1 || stood.Y > delivered.Y+1 {
		t.Fatal("delivery continued beyond the reserved step", delivered, stood)
	}
	startTick := f.live.world.Tick()
	deliveryTicks(t, f, app, 512)
	got := deliveryActor(t, f, witch)
	if got.X != stood.X || got.Y != stood.Y {
		t.Fatalf("delivered actor left (%d,%d) for (%d,%d)", stood.X, stood.Y, got.X, got.Y)
	}
	dir, raw := autoGetF2Save(t, f, app, "delivered")
	cold, capp := autoGetSession(t, dir)
	if cold.live.world.Hash() != f.live.world.Hash() {
		logCurrentCarrierDiff(t, f.live.world, cold.live.world)
		t.Fatal("cold LOAD changed the delivered World hash")
	}
	for tick := range 512 {
		deliveryTicks(t, f, app, 1)
		deliveryTicks(t, cold, capp, 1)
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("delivery cold continuation diverged at tick %d", tick+1)
		}
		for _, session := range []*FrontEnd{f, cold} {
			if e := deliveryActor(t, session, witch); e.X != stood.X || e.Y != stood.Y {
				t.Fatalf("delivery moved after SAVE/cold LOAD at tick %d", tick+1)
			}
		}
	}
	if out := os.Getenv("AGAINROM_DELIVERY_WITNESS"); out != "" {
		if err := os.WriteFile(filepath.Join(out, strings.ReplaceAll(t.Name(), "/", "-")+".sav"), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("delivery (%d,%d), stood (%d,%d), ticks %d..%d: 512 stationary ticks, F2 SAV/cold LOAD, 512 equal World hashes; final hash=%016x", delivered.X, delivered.Y, stood.X, stood.Y, startTick, startTick+512, f.live.world.Hash())
}

func deliveryFromStart(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	hero, witch := f.live.mission.ids[0], refs[21]
	cleared := 0
	for _, e := range f.live.world.Entities() {
		if e.Alive() && (f.live.world.Relations().Hostile(e.Owner, 1) || f.live.world.Relations().Hostile(e.Owner, 2)) {
			if err := f.live.world.HeadlessKill(e.ID); err != nil {
				t.Fatal(err)
			}
			cleared++
		}
	}
	t.Logf("noncombat setup: %d hostile actors killed; hero and witch walk through ordinary move commands and App frames", cleared)
	f.live.enqueue(uint32(hero), 36, 51)
	deliveryFrames(t, f, app, func() bool { return deliveryActor(t, f, witch).Owner == 1 })
	f.live.enqueue(uint32(witch), 56, 21)
	deliveryFrames(t, f, app, func() bool { return f.live.world.ScriptRegister(50) == 1 })
	deliveryProof(t, f, app, witch)
}

func TestReleaseMissionTenFreshDeliveryPost(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("mission delivery")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	deliveryFromStart(t, f, app)
}

func TestReleaseMissionTenDeliveredActorKeepsDeliveryPost(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, autoGetTenSAV, autoGetTenHash)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "delivery.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, dir)
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	witch := refs[21]
	start := deliveryActor(t, f, witch)
	order, _, _ := f.live.world.FrozenGroupAI(start.Owner, start.Group)
	t.Logf("placed witch: id=%d mapUnit=%d owner=%d group=%d commandGroup=%d groupOrder=%d actorState=%#x position=(%d,%d) post=(%d,%d)", start.ID, start.MapUnitID, start.Owner, start.Group, start.CommandGroup, order, start.ActorState, start.X, start.Y, start.PostX, start.PostY)
	t.Logf("delivery trigger: %+v", f.live.world.Script().Triggers()[3])
	for _, in := range f.live.world.Script().Instants() {
		if in.HasUnit && in.Unit == witch || in.HasGroup && in.Group == start.Group {
			t.Logf("witch instant: %+v", in)
		}
	}
	joined := deliveryActor(t, f, witch)
	t.Logf("joined witch: position=(%d,%d) post=(%d,%d) state=%#x group=%d commandGroup=%d", joined.X, joined.Y, joined.PostX, joined.PostY, joined.ActorState, joined.Group, joined.CommandGroup)
	f.live.enqueue(uint32(witch), 56, 21)
	deliveryFrames(t, f, app, func() bool { return f.live.world.ScriptRegister(50) == 1 })
	delivered := deliveryActor(t, f, witch)
	order, _, _ = f.live.world.FrozenGroupAI(delivered.Owner, delivered.Group)
	t.Logf("delivered witch: tick=%d position=(%d,%d) post=(%d,%d) state=%#x group=%d commandGroup=%d groupOrder=%d target=%v (%d,%d)", f.live.world.Tick(), delivered.X, delivered.Y, delivered.PostX, delivered.PostY, delivered.ActorState, delivered.Group, delivered.CommandGroup, order, delivered.HasTarget, delivered.TargetX, delivered.TargetY)
	deliveryProof(t, f, app, witch)
}

func TestReleaseOwnerMissionTenDeliveryPost(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	input := os.Getenv("AGAINROM_DELIVERY_INPUT")
	if input == "" {
		t.Skip("no AGAINROM_DELIVERY_INPUT")
	}
	dir := t.TempDir()
	raw, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range doc.Objects {
		if object.Class == "Humanoid" {
			t.Logf("owner saved Humanoid texts: %+v", object.Texts)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "owner.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, dir)
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	witch := refs[21]
	e := deliveryActor(t, f, witch)
	t.Logf("owner LOAD: tick=%d register50=%d witch owner=%d group=%d commandGroup=%d state=%#x position=(%d,%d) post=(%d,%d)", f.live.world.Tick(), f.live.world.ScriptRegister(50), e.Owner, e.Group, e.CommandGroup, e.ActorState, e.X, e.Y, e.PostX, e.PostY)
	deliveryFromStart(t, f, app)
}
