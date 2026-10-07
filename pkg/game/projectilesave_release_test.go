package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// savedMidFlight loads raw through the LOAD door, ticks the live World by
// ticks and saves through the save dialog. It returns the live World's record
// set at that instant, the written SAV and the live mapWorld.
func savedMidFlight(t *testing.T, raw []byte, ticks int) (recordSet, []byte, *mapWorld) {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "midflight.sav")
	t.Cleanup(app.StopAudio)
	dir := t.TempDir()
	g.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	mw := g.live
	for k := 0; k < ticks; k++ {
		mw.tick()
	}
	at := recordSetOfWorld(mw.world)
	return at, cityRosterF2Save(t, app, SaveStore{Dir: dir}, "midflight"), mw
}

// A loaded original save played on and saved while records are alive never
// refuses: the written Projectiles equals the live World's records, and a
// cold LOAD continues the Fire_Ball burst leaf by leaf. game0022 holds a rock
// and game0023 a rock and the burst that follows its landing; ticks after the
// load include the burst alone and ticks where a shot the loaded World
// released overlaps the bound record.
func TestReleaseLoadedOriginalSaveDuringFireBallBurst(t *testing.T) {
	saves := map[string][]byte{
		"game0022.sav": flightSave(t, "game0022.sav"),
		"game0023.sav": flightSave(t, "game0023.sav"),
	}
	bursts := 0
	for _, c := range []struct {
		save  string
		ticks int
	}{
		{"game0022.sav", 6}, {"game0022.sav", 18}, {"game0022.sav", 22}, {"game0022.sav", 30},
		{"game0023.sav", 8}, {"game0023.sav", 10}, {"game0023.sav", 14}, {"game0023.sav", 26},
	} {
		at, raw, mw := savedMidFlight(t, saves[c.save], c.ticks)
		if err := recordSetOfStore(kitProjectileStore(t, raw)).diff(at); err != nil {
			t.Errorf("%s +%d: written Projectiles differ from the live records: %v", c.save, c.ticks, err)
			continue
		}
		var burst *sim.SavedProjectile
		for _, p := range mw.world.SavedProjectiles().Items {
			if p.Picture == fireBallBurstPicture {
				p := p
				burst = &p
			}
		}
		if burst == nil {
			t.Logf("%s +%d: %d records, no burst", c.save, c.ticks, len(at.leaves))
			continue
		}
		bursts++
		run := burstRun{id: burst.ID, leaves: [][16]int32{shotLeaves(*burst)}}
		for k := 0; k < 40; k++ {
			mw.tick()
			p, ok := savedProjectileByID(mw.world, burst.ID)
			if !ok {
				break
			}
			run.leaves = append(run.leaves, shotLeaves(p))
		}
		if err := burstContinuation(t, raw, run); err != nil {
			t.Errorf("%s +%d: cold LOAD of the written SAV: %v", c.save, c.ticks, err)
		}
		t.Logf("%s +%d: burst %d lived %d ticks from the SAVE", c.save, c.ticks, burst.ID, len(run.leaves))
	}
	if bursts == 0 {
		t.Fatal("no sampled tick held a burst")
	}
}

// flightAfterLoad loads raw through the LOAD door and answers the record set
// on each of the next n ticks, index 0 being the load instant.
func flightAfterLoad(t *testing.T, raw []byte, n int) []recordSet {
	t.Helper()
	g := releaseFront(t)
	app, _ := openOriginalSAVApp(t, g, raw, "flight-after-load.sav")
	t.Cleanup(app.StopAudio)
	out := []recordSet{recordSetOfWorld(g.live.world)}
	for range n {
		g.live.tick()
		out = append(out, recordSetOfWorld(g.live.world))
	}
	return out
}

// A shot the loaded World built, and the siege transport it queued, survive
// SAVE and a cold LOAD: the continuation equals the live World's record for
// record. game0022 +6 holds the rock, +12 the pending transport, +22 the burst
// and the bolt. A SAV with a changed or dropped shot differs.
func TestReleaseLoadedWorldShotAndSiegeTransportContinueAfterColdLoad(t *testing.T) {
	raw := flightSave(t, "game0022.sav")
	const after = 14
	for _, c := range []struct {
		ticks, shots, pending int
	}{{6, 1, 0}, {12, 0, 1}, {22, 1, 0}} {
		_, written, mw := savedMidFlight(t, raw, c.ticks)
		if got := mw.world.PendingSpellDeliveries(); got != c.pending {
			t.Fatalf("game0022 +%d: %d pending deliveries at the SAVE, want %d", c.ticks, got, c.pending)
		}
		store := kitProjectileStore(t, written)
		live := []recordSet{recordSetOfWorld(mw.world)}
		for range after {
			mw.tick()
			live = append(live, recordSetOfWorld(mw.world))
		}
		compare := func(cold []recordSet) (int, error) {
			for k := range live {
				if err := cold[k].diff(live[k]); err != nil {
					return k, err
				}
			}
			return -1, nil
		}
		cold := flightAfterLoad(t, written, after)
		if k, err := compare(cold); err != nil {
			t.Errorf("game0022 +%d: cold LOAD differs from the live World %d ticks after the SAVE: %v", c.ticks, k, err)
		}
		// Control: a tick late is not the live World's.
		late := 0
		for k := range after {
			if cold[k].diff(live[k+1]) != nil {
				late++
			}
		}
		if late == 0 {
			t.Errorf("game0022 +%d: the cold continuation one tick late still equals the live World", c.ticks)
		}
		if c.pending != 0 {
			// The burst the transport ends in was built by the continuation.
			burst := false
			for _, r := range live[1:] {
				for _, l := range r.leaves {
					burst = burst || l[3] == fireBallBurstPicture
				}
			}
			if !burst {
				t.Errorf("game0022 +%d: no burst followed the pending transport", c.ticks)
			}
		}
		shots := 0
		for _, p := range store.Items {
			if p.Picture >= 1 && p.Picture <= 12 {
				shots++
			}
		}
		if shots != c.shots {
			t.Fatalf("game0022 +%d: the SAVE holds %d unit shots, want %d", c.ticks, shots, c.shots)
		}
		if shots == 0 {
			continue
		}
		// Loss controls on the written store.
		for name, edit := range map[string]func(*sav.ProjectileStore){
			"changed leaf": func(s *sav.ProjectileStore) {
				for i := range s.Items {
					if s.Items[i].Picture >= 1 && s.Items[i].Picture <= 12 {
						s.Items[i].ActionSegments++
						return
					}
				}
			},
			"dropped shot": func(s *sav.ProjectileStore) {
				for i := range s.Items {
					if s.Items[i].Picture >= 1 && s.Items[i].Picture <= 12 {
						id := s.Items[i].ID
						s.IDs = slices.DeleteFunc(s.IDs, func(v uint16) bool { return v == id })
						s.Items = slices.DeleteFunc(s.Items, func(p sav.Projectile) bool { return p.ID == id })
						return
					}
				}
			},
		} {
			file, err := sav.Open(written)
			if err != nil {
				t.Fatal(err)
			}
			changed := store
			changed.Items, changed.IDs = slices.Clone(store.Items), slices.Clone(store.IDs)
			edit(&changed)
			if err := file.SetProjectiles(changed); err != nil {
				t.Fatal(err)
			}
			if _, err := compare(flightAfterLoad(t, file.Marshal(), after)); err == nil {
				t.Errorf("game0022 +%d: a SAV with a %s flew like the live World", c.ticks, name)
			}
		}
	}
}
