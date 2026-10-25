package game

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func counterSilentDevice(t *testing.T) {
	t.Helper()
	t.Cleanup(ui.SetDeliveryPlayerFactory(func(r io.ReadSeeker) (ui.DeliveryDevicePlayer, error) {
		return &deliveryWitnessPlayer{reader: r}, nil
	}))
}

func TestReleaseProcessProjectileCounterSAVAndNewGame(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	counterSilentDevice(t)
	f, app, _, _, target := flightMageArena(t, false)
	if got := f.live.world.SavedProjectiles(); len(got.Items) != 0 || got.FreeIndex != 0 {
		t.Fatalf("fresh installed mission is not an empty zero store: %+v", got)
	}
	_, _, raw := campaignSave(t, f, true)
	if value, present := counterRawFreeIndex(t, raw); !present || value != uint32(f.live.world.SavedProjectiles().FreeIndex) {
		t.Fatal("ordinary SAV did not write the current raw counter", value, present)
	}
	wrapped := alterSAV(t, raw, func(doc *sav.DocumentData) bool {
		for i, r := range doc.State.ValueRecords {
			if r.Path == "/Projectiles/FreeIndex" {
				doc.State.ValueRecords[i].Value.Int32 = 65535
				return true
			}
		}
		return false
	})
	if value, present := counterRawFreeIndex(t, wrapped); !present || value != 65535 {
		t.Fatal("controlled wrap input lacks its raw word")
	}
	cold := releaseFront(t)
	loaded, _ := openOriginalSAVApp(t, cold, wrapped, "counter-wrap-input.sav")
	t.Cleanup(loaded.StopAudio)
	if got := cold.live.world.SavedProjectiles(); got.FreeIndex != 65535 || len(got.Items) != 0 {
		t.Fatalf("App LOAD did not admit the complete controlled wrap store: %+v", got)
	}
	if id := counterInstalledUnitCast(t, cold, loaded, target.ID); id != 65535 {
		t.Fatalf("wrapped installed cast took id %d", id)
	}
	if got := cold.live.world.SavedProjectiles().FreeIndex; got != 0 {
		t.Fatalf("actual cast did not wrap the counter to zero: %d", got)
	}
	_, _, zeroRaw := campaignSave(t, cold, true)
	if value, present := counterRawFreeIndex(t, zeroRaw); !present || value != 0 {
		t.Fatal("ordinary SAV did not write the wrapped zero", value, present)
	}
	resumed := releaseFront(t)
	resumedApp, _ := openOriginalSAVApp(t, resumed, zeroRaw, "counter-wrapped-current.sav")
	t.Cleanup(resumedApp.StopAudio)
	if id := counterInstalledUnitCast(t, resumed, resumedApp, target.ID); id != 0 {
		t.Fatalf("ordinary SAV/cold LOAD next pointer cast took id %d, want zero", id)
	}
	result := ui.ChargenResult{Name: "Counter new game", Choices: []int{0, 1, 0}, Stats: []int{30, 30, 40, 40}}
	old, hash, before := f.live, f.live.world.Hash(), f.live.world.SavedProjectiles().FreeIndex
	if _, _, _, _, _, _, _, _, _, _, err := f.NewGameOpener(999, result)(); err == nil {
		t.Fatal("invalid installed NEW GAME succeeded")
	}
	open, err := f.prepareNewGame(10, result)
	if err != nil {
		t.Fatal(err)
	}
	if f.live != old || f.live.world.Hash() != hash || f.projectiles.value() != before {
		t.Fatal("failed or deferred installed NEW GAME mutated the old session")
	}
	counterInstalledCast(t, f, app)
	latest := f.live.world.SavedProjectiles().FreeIndex
	if latest == before {
		t.Fatal("old running App produced no late record")
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != 10 || f.live.world.SavedProjectiles().FreeIndex != latest {
		t.Fatalf("installed NEW GAME ignored late counter %d: %+v", latest, f.live.world.SavedProjectiles())
	}
	if id := counterInstalledCast(t, f, app); id != latest {
		t.Fatalf("NEW GAME next record id %d, want %d", id, latest)
	}
	t.Logf("ordinary SAV raw counter, controlled SAV LOAD65535, actual cast65535→0, next resumed record0; deferred NEW GAME %d→latest%d", before, latest)
}

func counterInstalledUnitCast(t *testing.T, f *FrontEnd, app *ui.App, target sim.EntityID) uint16 {
	t.Helper()
	releasePauseMission(t, f, app)
	caster := f.live.mission.ids[0]
	flightLearn(t, f.live, caster, 1)
	e, _ := f.live.entity(caster)
	if err := f.live.world.HeadlessPlace(target, e.X+5, e.Y); err != nil {
		t.Fatal(err)
	}
	f.live.push()
	state := f.live.view.SaveApplication()
	t.Logf("before selection: pending=%+v paused=%t pressed-spell=%d target=%d", f.live.pending, state.PlayerPaused, state.PressedSpell, target)
	castOrderSelect(t, app, f.live, caster)
	if _, _, err := app.HeadlessSpellPoint(1); err != nil {
		if err := app.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	sx, sy, err := app.HeadlessSpellPoint(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, sx, sy); err != nil {
			t.Fatal(err)
		}
	}
	x, y, err := app.HeadlessEntityPoint(uint32(target))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindCast || f.live.pending[0].Y != 1 || f.live.pending[0].X != int32(target) {
		t.Fatal("installed pointer input did not queue Fire Arrow", f.live.pending)
	}
	for _, edge := range []string{"right-press", "right-release"} {
		if err := app.HeadlessPointer(edge, 512, 200); err != nil {
			t.Fatal(err)
		}
	}
	want := f.live.world.SavedProjectiles().FreeIndex
	activePauseResume(t, app)
	if store := f.live.world.SavedProjectiles(); store.FreeIndex != want {
		if store.FreeIndex != want+1 {
			t.Fatal("App resume did not produce exactly one record", store)
		}
		for _, p := range store.Items {
			if p.ID == want && p.Picture == 10 {
				return want
			}
		}
		t.Fatal("App resume advanced without the next Fire Arrow record", store)
	}
	for range 128 {
		for _, ev := range f.LiveAdvanceCasts(1) {
			if ev.Caster == caster && ev.Spell == 1 {
				if !ev.Produced || len(ev.Records) != 1 || ev.Records[0] != want {
					t.Fatalf("installed Fire Arrow records %+v, want first id %d", ev, want)
				}
				return want
			}
		}
	}
	t.Fatal("installed pointer Fire Arrow did not release a record")
	return 0
}

func counterInstalledCast(t *testing.T, f *FrontEnd, app *ui.App) uint16 {
	t.Helper()
	for n := 0; app.HeadlessNoticeOpen() && n < 32; n++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	releasePauseMission(t, f, app)
	caster := f.live.mission.ids[0]
	flightLearn(t, f.live, caster, 21)
	f.live.push()
	at, ok := cityMageNearest(cityMageCastCells(t, f, app, caster, 21))
	if !ok {
		t.Fatal("no visible cell admits the installed spell")
	}
	want := f.live.world.SavedProjectiles().FreeIndex
	cityMageCastInput(t, f, app, caster, 21, at)
	for _, edge := range []string{"right-press", "right-release"} {
		if err := app.HeadlessPointer(edge, 512, 200); err != nil {
			t.Fatal(err)
		}
	}
	activePauseResume(t, app)
	for range 128 {
		for _, ev := range f.LiveAdvanceCasts(1) {
			if ev.Caster != caster || ev.Spell != 21 {
				continue
			}
			if !ev.Produced || len(ev.Records) != 1 || ev.Records[0] != want {
				t.Fatalf("installed pointer cast records %+v, want first id %d", ev, want)
			}
			return want
		}
	}
	t.Fatal("installed pointer cast did not release a record")
	return 0
}

type counterTransitionReceipt struct {
	Route         string
	Before, After uint16
}

func TestReleaseProcessProjectileCounterFollowsTheCampaign(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	counterSilentDevice(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	f.Carried = MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("projectile counter campaign")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	var receipts []counterTransitionReceipt
	counterInstalledCast(t, f, app)
	chainWin(t, f, app, 10)
	want := f.live.world.SavedProjectiles().FreeIndex
	if want == 0 {
		t.Fatal("installed mission produced no counter increment")
	}
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("direct successor", err, f.liveMission)
	}
	got := f.live.world.SavedProjectiles().FreeIndex
	receipts = append(receipts, counterTransitionReceipt{"mission10 to direct successor20", want, got})
	if got != want {
		t.Fatalf("actual mission10→20 reset projectile counter %d to %d", want, got)
	}
	if id := counterInstalledCast(t, f, app); id != want {
		t.Fatalf("successor first record %d, want %d", id, want)
	}
	chainWin(t, f, app, 20)
	campaignReturn(t, f, app)
	want = f.live.world.SavedProjectiles().FreeIndex
	chainEnterByWorldMap(t, f, app, 30)
	got = f.live.world.SavedProjectiles().FreeIndex
	receipts = append(receipts, counterTransitionReceipt{"mission20 to town to mission30", want, got})
	if got != want {
		t.Fatalf("actual mission20→town→30 reset projectile counter %d to %d", want, got)
	}
	if id := counterInstalledCast(t, f, app); id != want {
		t.Fatalf("town successor first record %d, want %d", id, want)
	}
	if out := os.Getenv("AGAINROM_PROJECTILE_COUNTER_WITNESS_DIR"); out != "" {
		if !filepath.IsAbs(out) {
			t.Fatal("counter witness directory must be absolute")
		}
		if err := os.MkdirAll(out, 0700); err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(receipts, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "campaign-counter.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("installed App campaign transitions: %+v; authored spellbook credit, pointer casts and installed objective scripts", receipts)
}
