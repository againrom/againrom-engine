package game

import (
	"bytes"
	"encoding/hex"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestReleaseOriginalProfile1107NaturalFractionAppAndNative(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		t.Run(map[bool]string{false: "main menu", true: "mission menu"}[fromMap], func(t *testing.T) {
			f := releaseFront(t)
			path, payload := groundCorpusFile(t, "2026-08-02/game0009.sav", "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6")
			source, err := sav.Open(payload)
			if err != nil {
				t.Fatal(err)
			}
			attack, _ := hex.DecodeString("37003200000000000000000000000103050000000000e900")
			defence, _ := hex.DecodeString("2d000000000000000000000000000000000000000000")
			stats, _ := hex.DecodeString("07004b00030005002000000000002c0107000f0028000000000000004b00")
			if !bytes.Equal(source.Body[28206:28230], attack) || !bytes.Equal(source.Body[28230:28252], defence) || bytes.Count(source.Body, stats) != 1 || bytes.Index(source.Body, stats) != 28796 {
				t.Fatal("independent natural profile control changed")
			}
			f.SetDeterministicFrames(true)
			app := f.App("1107-natural-profile")
			if fromMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := nativeContinuationSeams1170(t, f, store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			old := f.live
			e := poolEntity(t, old.world, 8)
			if e.CurrentProfileBasis != sim.ProfileOriginalCurrent || e.HP != 7 || e.MaxHP != 15 || e.HealthRegenPeriod != 40 || e.HealthHundredths != 75 ||
				e.ToHit != 55 || e.DamageBase != 1 || e.DamageSpread != 3 || e.XPSlot != 5 || e.Defence != 45 || e.Reaction != 75 || e.Mind != 3 || e.Spirit != 5 || old.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatalf("natural after-profile changed: %+v tick%d", e, old.world.Tick())
			}
			// This genuine source is already victorious. Its live notice blocks
			// the SAVE key, so invoke the installed production SAVE seam and
			// keep the terminal state; do not invent a playable source variant.
			if old.world.Outcome() != sim.OutcomeWon {
				t.Fatal("natural terminal control changed")
			}
			if _, err := save(true); err != nil {
				t.Fatal(err)
			}
			entries, err := listAGS(store)
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err)
			}
			fresh := releaseFront(t)
			fresh.SetDeterministicFrames(true)
			freshApp := fresh.App("1107-natural-native")
			fs, fl, ff := nativeContinuationSeams1170(t, fresh, store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fs, fl, ff)
			groundAppLoad(t, freshApp, fl, entries[0].Name)
			if old.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("natural profile native LOAD")
			}
			for i := 0; i < 96; i++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				if err := freshApp.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				if old.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("natural continuation", i)
				}
			}
			if old.world.Tick() != rawSavedSubTick1112(t, payload) || poolEntity(t, fresh.live.world, 8).HealthHundredths != 75 {
				t.Fatal("terminal continuation ticked or lost fraction")
			}
			t.Logf("M20 Unit8 source=7/15 period40 fraction75; published-d521 LOAD fraction0; candidate App LOAD fraction75; SAVE seam/fresh App LOAD and 96 terminal frames preserve saved tick/fraction75; path=%s", path)
		})
	}
}
