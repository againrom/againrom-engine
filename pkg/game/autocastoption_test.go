package game

import (
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestAutoCastOptionIsTheOneSetting is the Game Options row of the
// automatic-casting setting: it writes the profile key, queues the Player's
// setting with the target bits kept, reads the queued and the applied level
// back, a queued level survives a SAVE cut as a pending option, and Off
// clears the stored setting. Values: 0 Off, 1 Minimum, 2 Average, 3 Maximum.
func TestAutoCastOptionIsTheOneSetting(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	if err := mw.world.RestoreCurrentPlayers([]sim.SavedGroupPlayer{{ID: 1, Slot: 0}, {ID: 2, Slot: sim.SelfSlot}}, []sim.PlayerParticipant{{PlayerID: 1, Value: 1}, {PlayerID: 2}}, true); err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{CampaignSession: CampaignSession{live: mw}, PersistenceContext: PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}}
	if got := f.gameOptionValues(true)[ui.GameOptionAutoCast]; got != 0 {
		t.Fatalf("a new first-game world reads choice %d, want Off", got)
	}
	if !mw.world.SetAutoCast(sim.SelfSlot, sim.AutoCastAverage|sim.AutoCastOwn|sim.AutoCastAllies) {
		t.Fatal("setting refused")
	}
	if got := f.gameOptionValues(true)[ui.GameOptionAutoCast]; got != 2 {
		t.Fatalf("Average reads as choice %d", got)
	}
	if err := f.setGameOption(true, ui.GameOptionAutoCast, 3); err != nil {
		t.Fatal(err)
	}
	if got := f.gameOptionValues(true)[ui.GameOptionAutoCast]; got != 3 {
		t.Fatalf("queued Maximum reads as choice %d", got)
	}
	stored, present, err := f.Options.gameOptions()
	if err != nil || !present[ui.GameOptionAutoCast] || stored[ui.GameOptionAutoCast] != 3 {
		t.Fatal("profile key not written", stored, present, err)
	}
	if m, _ := f.Options.readAll(); m["AutoCastMode"] != "3" {
		t.Fatal("profile key name", m)
	}
	if got := pendingGameOptions(mw.pending); len(got) != 1 || got[0] != (PendingGameOption{ui.GameOptionAutoCast, 3}) {
		t.Fatal("queued level is not a pending option", got)
	}
	if err := validatePendingGameOptions(pendingGameOptions(mw.pending)); err != nil {
		t.Fatal(err)
	}
	mw.tick()
	want := sim.AutoCastMaximum | sim.AutoCastOwn | sim.AutoCastAllies
	if got := mw.world.AutoCast(sim.SelfSlot); got != want {
		t.Fatalf("applied setting %#x, want %#x", got, want)
	}
	// The same level again queues nothing.
	if err := f.setGameOption(true, ui.GameOptionAutoCast, 3); err != nil || len(mw.pending) != 0 {
		t.Fatal("an unchanged level queued a command", mw.pending, err)
	}
	// The off-map page reads the profile; without a key it reads the first
	// game's default, Off.
	if got := f.gameOptionValues(false)[ui.GameOptionAutoCast]; got != 3 {
		t.Fatalf("off-map page reads choice %d", got)
	}
	blank := &FrontEnd{PersistenceContext: PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}}
	if got := blank.gameOptionValues(false)[ui.GameOptionAutoCast]; got != 0 {
		t.Fatalf("absent profile key reads choice %d, want Off", got)
	}
	// Off clears the stored setting: the world reads its first-game default.
	if err := f.setGameOption(true, ui.GameOptionAutoCast, 0); err != nil {
		t.Fatal(err)
	}
	mw.tick()
	if mw.world.AutoCastStored(sim.SelfSlot) || mw.world.AutoCast(sim.SelfSlot).Level() != 0 || f.gameOptionValues(true)[ui.GameOptionAutoCast] != 0 {
		t.Fatalf("Off left %#x", mw.world.AutoCast(sim.SelfSlot))
	}
}

// TestAutoCastOptionOffersPerGame is the one row's data: the first game
// offers Off and the three levels and starts Off; the second offers its three
// levels and starts at Average. The second game refuses Off. Each game's
// column is its edition's.
func TestAutoCastOptionOffersPerGame(t *testing.T) {
	o := ui.GameOptionAutoCast
	first, second := base.GameROM1.Edition().Options, base.GameROM2.Edition().Options
	if got := o.Offers(first); !slices.Equal(got, []int{0, 1, 2, 3}) || o.Default(first) != 0 {
		t.Fatalf("first game offers %v from %d", got, o.Default(first))
	}
	if got := o.Offers(second); !slices.Equal(got, []int{1, 2, 3}) || o.Default(second) != 2 || o.Offered(second, 0) {
		t.Fatalf("second game offers %v from %d", got, o.Default(second))
	}
	if sim.AutoCastLevel(o.Default(second)) != sim.DefaultAutoCast.Level() {
		t.Fatal("the second game's row default is not the constructor's level")
	}
}

// TestAutoCastOptionReachesTheNextMissionThroughTheProfile is the fresh map
// door: a level the profile holds reaches the next mission's World, and an
// old profile without the key leaves the first game Off with nothing queued.
func TestAutoCastOptionReachesTheNextMissionThroughTheProfile(t *testing.T) {
	m := worldFixtureMap()
	open := func(p *PersistenceContext) *mapWorld {
		mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
		if err := mw.world.RestoreCurrentPlayers([]sim.SavedGroupPlayer{{ID: 1, Slot: 0}, {ID: 2, Slot: sim.SelfSlot}}, []sim.PlayerParticipant{{PlayerID: 1, Value: 1}, {PlayerID: 2}}, true); err != nil {
			t.Fatal(err)
		}
		if err := p.applyFreshGameOptions(mw, nil); err != nil {
			t.Fatal(err)
		}
		mw.tick()
		return mw
	}
	p := &PersistenceContext{Options: OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}}
	if err := p.Options.writeAll(map[string]string{"AutoCasting": "1", "FormationMode": "2"}); err != nil {
		t.Fatal(err)
	}
	if mw := open(p); mw.world.AutoCastStored(sim.SelfSlot) || mw.world.AutoCast(sim.SelfSlot).Level() != 0 {
		t.Fatalf("an old profile opened at %#x", mw.world.AutoCast(sim.SelfSlot))
	}
	if err := p.Options.setGameOption(ui.GameOptionAutoCast, 2); err != nil {
		t.Fatal(err)
	}
	if mw := open(p); mw.world.AutoCast(sim.SelfSlot) != sim.AutoCastAverage|sim.AutoCastOwn {
		t.Fatalf("the next mission opened at %#x, want Average, Own", mw.world.AutoCast(sim.SelfSlot))
	}
}
