//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

// Every original city SAV keeps its permanent mercenary order when rewritten.
func TestPermanentMercenaryOrderCorpusRoundTrip(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name gameversions/saves")
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	if assets == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install")
	}
	cities, listed, differing := 0, 0, 0
	err := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".sav" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source, err := sav.Open(raw)
		if err != nil || source.World != nil {
			return nil
		}
		proj, ok, err := source.Campaign()
		if err != nil || !ok {
			return nil
		}
		g, err := NewFrontEnd(assets)
		if err != nil {
			return err
		}
		cleanupFrontAudio(t, g)
		if _, town, err := g.RestoreOriginal(raw); err != nil || !town {
			return nil
		}
		cities++
		snap, _, err := g.Snapshot(false)
		if err != nil {
			t.Errorf("%s: snapshot: %v", path, err)
			return nil
		}
		out, err := g.ExportCurrentSave(snap, "order")
		if err != nil {
			t.Errorf("%s: export: %v", path, err)
			return nil
		}
		written, err := sav.Open(out)
		if err != nil {
			t.Errorf("%s: reopen: %v", path, err)
			return nil
		}
		wp, _, _ := written.Campaign()
		if len(proj.PermanentMercenaries) > 0 {
			listed++
		}
		if !slices.Equal(wp.PermanentMercenaries, proj.PermanentMercenaries) {
			differing++
			t.Errorf("%s: PermanentMercenaries %v, source %v", path, wp.PermanentMercenaries, proj.PermanentMercenaries)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("city saves %d, with a permanent list %d, differing %d", cities, listed, differing)
	if listed == 0 {
		t.Fatal("no city save in the corpus carries a permanent list; the witness has no population")
	}
}
