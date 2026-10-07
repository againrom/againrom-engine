package game

import (
	"encoding/binary"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Literals are the independent head census, not SessionState or an importer.
// Both installed consumers read the same SHA-identified original recordings.
func TestReleaseOriginalClock1112NaturalBothDoorsSaveFreshAndNextPhase(t *testing.T) {
	t.Run("untreated-natural-phase6", naturalPhase6Clock1112)
	for _, tc := range []struct {
		path, sha               string
		sub, full, nextS, nextF uint32
	}{
		{"2026-08-02/game0003.sav", "8902ab4b04068af4a8d37107e6c2d5a9a9cc30bbe67a022901583a01da7ee03d", 9343, 584, 9363, 585},
		{"2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b", 3452, 215, 3472, 217},
	} {
		t.Run(tc.path, func(t *testing.T) {
			f := releaseFront(t)
			path, raw := groundCorpusFile(t, tc.path, tc.sha)
			source, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			if binary.LittleEndian.Uint32(source.Body[:4]) != tc.sub || binary.LittleEndian.Uint32(source.Body[4:8]) != tc.full {
				t.Fatal("independent head anchors changed")
			}
			ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
			if err != nil {
				t.Fatal(err)
			}
			checkWorldClock1112(t, ms.World, tc.sub, tc.full)
			f.SetDeterministicFrames(true)
			app := f.App("1112-natural-clock")
			store := SaveStore{Dir: t.TempDir()}
			s, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(s, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			checkWorldClock1112(t, f.live.world, tc.sub, tc.full)
			warm := f.live.view.Sun()
			if tc.sub == 9343 && (warm.Theta < 0.4799 || warm.Theta > 0.4801) {
				t.Fatalf("first source frame sun=%+v", warm)
			}
			cold, _ := holdingsNativeFresh(t, f, app, store, nil)
			checkWorldClock1112(t, cold.live.world, tc.sub, tc.full)
			if cold.live.view.Sun() != warm {
				t.Fatal("first native frame differs from source-load cache")
			}
			// A new, explicit damage command enters both resumed worlds on
			// their next phase. The sim tests separately pin its order against
			// regeneration; here the installed producer and native continuation
			// must remain identical for20 actual driver ticks.
			var target sim.EntityID
			found := false
			for _, e := range f.live.world.Entities() {
				if e.HP > 20 && e.MaxHP > 0 && !e.OffMap {
					target, found = e.ID, true
					break
				}
			}
			if !found {
				t.Fatal("natural command target missing")
			}
			headlessDamage(t, f.live.world, target, 7)
			headlessDamage(t, cold.live.world, target, 7)
			for i := 0; i < 20; i++ {
				f.live.tick()
				cold.live.tick()
				if f.live.world.Hash() != cold.live.world.Hash() {
					t.Fatalf("native continuation diverged at driver step%d", i)
				}
			}
			checkWorldClock1112(t, cold.live.world, tc.nextS, tc.nextF)
			t.Logf("source%s sha%s: S/F%d/%d ->20steps->%d/%d; ordinary App menu SAVE/fresh LOAD; sunTheta%.10f", tc.path, tc.sha, tc.sub, tc.full, tc.nextS, tc.nextF, warm.Theta)
		})
	}
}

// No controlled script, actor relocation or command injection: this witnesses
// the saved mission program executing real checks at entry phase6.
// Particular firings/notices depend on incoming Group state, not clock law.
// Equality is measured after each simulation tick, not each UI frame helper.
func naturalPhase6Clock1112(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(source.Body[:4]) != 372 || binary.LittleEndian.Uint32(source.Body[4:8]) != 23 {
		t.Fatal("natural phase6 head anchors changed")
	}
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	app := f.App("1112-natural-phase6")
	store := SaveStore{Dir: t.TempDir()}
	s, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(s, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	checkWorldClock1112(t, f.live.world, 372, 23)
	if app.HeadlessNoticeOpen() {
		t.Fatal("unexpected initial notice")
	}
	cold, coldApp := holdingsNativeFresh(t, f, app, store, nil)
	// SAVE leaves the source App in its menu; return it to the same map state
	// as fresh LOAD before comparing identical no-input simulation ticks.
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		tr := sim.StepTraced(ms.World, nil)
		if tr.Pass != (i == 2) || tr.Report || (i == 2 && len(tr.Checks) == 0) {
			t.Fatalf("natural entry%d trace %+v", 372+i, tr)
		}
		f.live.tick()
		cold.live.tick()
		checkWorldClock1112(t, f.live.world, uint32(373+i), 23)
		checkWorldClock1112(t, cold.live.world, uint32(373+i), 23)
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("natural continuation differs at simulation tick%d", 373+i)
		}
		if app.HeadlessNoticeOpen() != coldApp.HeadlessNoticeOpen() || (i < 2 && app.HeadlessNoticeOpen()) {
			t.Fatalf("natural dialogue timing at tick%d: source%t native%t", 373+i, app.HeadlessNoticeOpen(), coldApp.HeadlessNoticeOpen())
		}
		if i == 2 {
			dispatched := 0
			for _, check := range tr.Checks {
				if check.Dispatched {
					dispatched++
				}
			}
			if dispatched == 0 {
				t.Fatal("phase6 reached no real script check")
			}
			t.Logf("natural phase6: checks%d dispatched%d firings%d notice%t (incoming Group chronology not claimed)", len(tr.Checks), dispatched, len(tr.Firings), app.HeadlessNoticeOpen())
		}
	}
	t.Log("untreated natural S/F372/23 ->375/23: real phase6 script work and matching notice state after three simulation ticks; SAVE preceded any notice")
}
