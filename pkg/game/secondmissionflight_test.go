package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// secondMissionFlightFile is the fixture mission file with a spell in flight
// shaped as game0006 holds one: one top-level SpellTransport with countdown
// 1 nesting a PointEffect at the hero and an Effect_DirectDamage
// (R2-SESSION-260), and the client projectile Prj16 with zero segments
// (R2-SESSION-244).
func secondMissionFlightFile(hero secondOriginalHero, more ...secondSavedFlight) []byte {
	var damage [24]byte
	damage[19], damage[21] = 5, 1
	flight := secondSavedFlight{spell: 1, countdown: 1, target: hero.key, damage: damage}
	shot := sim.SavedProjectile{ID: 16, X: 6272, Y: 10624, Picture: 40, Dir: 4, Phase: 1, LastAction: 1, Action: 1,
		ActionDir: 4, ActionTarget: 1, ActionX: 6272, ActionY: 10624, ActionPhase: 3}
	return labelledContainer(sav.SecondGame.Magic, "mission",
		secondSavedPartyOf(10, 44, []secondOriginalHero{hero}, true, 0, nil, append([]secondSavedFlight{flight}, more...)...),
		append(secondMissionWholeStore(8, 1, 6, shot), secondSavedTail(0, [2]uint32{10, placeMission})...))
}

// secondMissionFlightLoad LOADs name from dir over a mission whose graphics
// archive holds the projectile registry row of picture 40, as an install's
// does, so the projectile's continuation has its phase count.
func secondMissionFlightLoad(t *testing.T, dir, name string) *FrontEnd {
	t.Helper()
	reg := synth.ProjectilesReg(1, []synth.RegNode{{Name: "ID", Kind: 2, Int: 40}, {Name: "Phases", Kind: 2, Int: 8}})
	path := strings.TrimPrefix(ProjectileRegistry, graphicsPrefix)
	f, _ := secondMissionLoadOf(t, secondMissionContent{graphics: []synth.File{{Path: path, Data: reg}}}, dir, name)
	return f
}

// secondMissionFlightLoadFile LOADs the fixture file through
// secondMissionFlightLoad and returns the directory it lies in.
func secondMissionFlightLoadFile(t *testing.T, hero secondOriginalHero) (*FrontEnd, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), secondMissionFlightFile(hero), 0o600); err != nil {
		t.Fatal(err)
	}
	return secondMissionFlightLoad(t, dir, "game0001.sav"), dir
}

// secondFlight is what a World holds of the fixture's flight: the class of
// each top-level spell effect, the countdown of a transport, the projectile
// IDs and the hero's health.
type secondFlight struct {
	roots     []string
	countdown uint16
	shots     []uint16
	hp        int32
}

func secondFlightOf(t *testing.T, w *sim.World) secondFlight {
	t.Helper()
	var s secondFlight
	for _, e := range w.SavedSpellEffects() {
		s.roots = append(s.roots, e.Class)
		if e.Class == "SpellTransport" {
			s.countdown = e.ST4C
		}
	}
	for _, p := range w.SavedProjectiles().Items {
		s.shots = append(s.shots, p.ID)
	}
	s.hp = secondMissionActor(t, w).HP
	return s
}

// A LOAD of a mission file continues its spell in flight as the claims read
// it. The first tick runs the transport's countdown from 1 to 0, which
// queues its PointEffect behind it (R2-ENGINE-660); the projectile with
// zero segments leaves on its first update (R2-ENGINE-661). The second tick
// runs the PointEffect, whose damage payload strikes its target, and the
// list is empty.
func TestSecondMissionFlightContinuesAfterLoad(t *testing.T) {
	hero := secondMissionFixtureHero()
	raw := secondMissionFlightFile(hero)
	if _, err := sav.DecodeDocumentData(raw); err != nil {
		t.Fatalf("the fixture's document does not decode: %v", err)
	}
	f, _ := secondMissionFlightLoadFile(t, hero)
	w := f.live.world
	loaded := secondFlightOf(t, w)
	if !slices.Equal(loaded.roots, []string{"SpellTransport"}) || loaded.countdown != 1 || !slices.Equal(loaded.shots, []uint16{16}) {
		t.Fatalf("the LOAD holds %+v, want the transport at 1 and Prj16", loaded)
	}
	f.live.tick()
	fired := secondFlightOf(t, w)
	if !slices.Equal(fired.roots, []string{"PointEffect"}) || len(fired.shots) != 0 || fired.hp != loaded.hp {
		t.Fatalf("after the first tick %+v, want the queued PointEffect, no projectile and health %d", fired, loaded.hp)
	}
	f.live.tick()
	struck := secondFlightOf(t, w)
	if len(struck.roots) != 0 || struck.hp >= loaded.hp {
		t.Fatalf("after the second tick %+v, want no effect and health below %d", struck, loaded.hp)
	}
}

// A SAVE at each tick of the flight writes what the World then holds, and
// its cold LOAD holds the same and continues as the live World does: before
// the countdown runs the transport and the projectile, after it the queued
// PointEffect, after the strike nothing.
func TestSecondMissionFlightSavesAndColdLoads(t *testing.T) {
	hero := secondMissionFixtureHero()
	f, dir := secondMissionFlightLoadFile(t, hero)
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	saved := map[string]bool{}
	for step := range 3 {
		w := f.live.world
		want := secondFlightOf(t, w)
		if _, err := save(true); err != nil {
			t.Fatalf("step %d: our SAVE: %v", step, err)
		}
		entries, err := secondSaveFiles(dir, "game0001.sav")
		if err != nil {
			t.Fatal(err)
		}
		name := ""
		for _, e := range entries {
			if !saved[e.Name] {
				name = e.Name
			}
		}
		if name == "" {
			t.Fatalf("step %d: our SAVE wrote nothing", step)
		}
		saved[name] = true
		cold := secondMissionFlightLoad(t, dir, name)
		if got := secondFlightOf(t, cold.live.world); !slices.Equal(got.roots, want.roots) || got.countdown != want.countdown || !slices.Equal(got.shots, want.shots) || got.hp != want.hp {
			t.Fatalf("step %d: the cold LOAD holds %+v, want %+v", step, got, want)
		}
		// The SAVE writes no retired record, so once the transport has
		// fired the cold LOAD's spell graph numbers only the PointEffect,
		// and the hashed graph differs from the live one by that renumbering.
		same := step == 0
		if same && cold.live.world.Hash() != w.Hash() {
			t.Fatalf("step %d: the cold LOAD changed the World", step)
		}
		for tick := range 3 {
			f.live.tick()
			cold.live.tick()
			got, want := secondFlightOf(t, cold.live.world), secondFlightOf(t, f.live.world)
			if !slices.Equal(got.roots, want.roots) || got.countdown != want.countdown || !slices.Equal(got.shots, want.shots) || got.hp != want.hp {
				t.Fatalf("step %d, tick %d: the cold LOAD runs to %+v, the live World to %+v", step, tick+1, got, want)
			}
			if same && cold.live.world.Hash() != f.live.world.Hash() {
				t.Fatalf("step %d, tick %d: the cold LOAD's World diverged", step, tick+1)
			}
		}
		// The live World goes back to the next step's tick: a fresh LOAD
		// of the file, run to it.
		f, dir = secondMissionFlightLoadFile(t, hero)
		save, _, _ = f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
		saved = map[string]bool{}
		for range step + 1 {
			f.live.tick()
		}
	}
}

// A held document whose world effects the binder refuses is carried as a
// partial state carries it: the LOAD succeeds, the records stay in the World
// as loaded values, nothing of them is bound, and the SAVE writes none, as
// before. The fixture holds an area cloud with no identity next to the
// transport.
func TestSecondMissionFlightWithARefusedCloudStillLoads(t *testing.T) {
	hero := secondMissionFixtureHero()
	dir := t.TempDir()
	cloud := secondSavedFlight{cloud: true, spell: 3}
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), secondMissionFlightFile(hero, cloud), 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondMissionFlightLoad(t, dir, "game0001.sav")
	w := f.live.world
	got := secondFlightOf(t, w)
	if !slices.Equal(got.roots, []string{"SpellTransport", "AreaEffect"}) || got.countdown != 1 || !slices.Equal(got.shots, []uint16{16}) {
		t.Fatalf("the LOAD carries %+v, want the transport at 1, the cloud and Prj16", got)
	}
	if w.SavedSpellGraph() != nil || w.SavedWorldEffectDrivers() != nil {
		t.Fatal("a refused bind left the World half bound")
	}
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	if _, err := save(true); err != nil {
		t.Fatalf("our SAVE: %v", err)
	}
	entries, err := secondSaveFiles(dir, "game0001.sav")
	if err != nil || len(entries) != 1 {
		t.Fatalf("our SAVE wrote %d files, %v", len(entries), err)
	}
	if cold := secondFlightOf(t, secondMissionFlightLoad(t, dir, entries[0].Name).live.world); len(cold.roots) != 0 {
		t.Fatalf("the cold LOAD of our SAVE holds %+v, want no carried effect", cold)
	}
}
