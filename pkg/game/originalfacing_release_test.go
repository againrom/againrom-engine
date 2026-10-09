package game

import (
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// Literal mover[0] values were read independently from the exact archive
// graph. Map20 includes all four temporary allies plus the starting hero;
// map40 also includes a zero-map-ID companion. These are raw source-state
// and Againrom continuation witnesses, not original-process observations.
func TestReleaseOriginalCurrentFacingAndNativeContinuation(t *testing.T) {
	for _, tc := range []struct {
		path, sha string
		want      map[sim.EntityID]uint8
	}{
		{"2026-08-02/game0001.sav", "c5010cfbbe1f59018452813e7eda192dc5f3ce9834c9246bad23287e168899dd",
			map[sim.EntityID]uint8{48: 162, 49: 137, 50: 171, 51: 83, 56: 160}},
		{"2026-08-15/game0018.sav", "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b",
			map[sim.EntityID]uint8{46: 96, 47: 32, 48: 96}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			f := releaseFront(t)
			path, raw := groundCorpusFile(t, tc.path, tc.sha)
			check := func(w *sim.World) {
				t.Helper()
				found := 0
				for _, e := range w.Entities() {
					want, ok := tc.want[e.ID]
					if !ok {
						continue
					}
					found++
					if e.Facing != want || e.DesiredFacing != want || e.DrawnFacing() != want || e.TurnRemaining != 0 || e.TurnTotal != 0 {
						t.Fatalf("entity%d current/desired/drawn=%d/%d/%d turn=%d/%d want%d and no invented incoming turn", e.ID, e.Facing, e.DesiredFacing, e.DrawnFacing(), e.TurnRemaining, e.TurnTotal, want)
					}
				}
				if found != len(tc.want) {
					t.Fatalf("found%d of%d literal subjects", found, len(tc.want))
				}
			}
			diagnostic, _, err := loadOriginalMission(f, raw)
			if err != nil {
				t.Fatal(err)
			}
			check(diagnostic.World)
			f.SetDeterministicFrames(true)
			app := f.App("original-facing")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			check(f.live.world)
			cold, coldApp := holdingsNativeFresh(t, f, app, store, nil)
			check(cold.live.world)
			second := SaveStore{Dir: t.TempDir()}
			s, l, back := cold.SaveSeams(second, OriginalStore{}, nil)
			coldApp.SetSaveSeams(s, l, back)
			again, _ := holdingsNativeFresh(t, cold, coldApp, second, nil)
			check(again.live.world)
			t.Logf("source=%s sha=%s literalFacing=%v: both original LOAD doors and two ordinary SAVE/fresh LOAD cycles", tc.path, tc.sha, tc.want)
		})
	}
}
