//go:build sessioncorpusaudit

package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// Census: entities whose group rate term differs from the entity field.
func TestGroupRateTermCorpusCensus(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	assets := os.Getenv("AGAINROM_ASSETS")
	if corpus == "" || assets == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS and AGAINROM_ASSETS must be set")
	}
	f, err := NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	files, termed, differing, entities := 0, 0, 0, 0
	err = filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
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
		if err != nil || source.World == nil {
			return nil
		}
		ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, mapload.DifficultyNormal, nil, f.Bodies)
		if err != nil {
			return nil
		}
		files++
		fileDiffers := 0
		for _, e := range ms.World.Entities() {
			entities++
			term := ms.World.GroupRateTerm(e.ID)
			if term != 0 {
				termed++
			}
			if term != e.GroupSpeed {
				differing++
				fileDiffers++
			}
		}
		if fileDiffers > 0 {
			t.Logf("%s: %d entities whose rate term differs from the entity field", path, fileDiffers)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("world saves %d, entities %d, with a nonzero rate term %d, readout field differs %d", files, entities, termed, differing)
	if files == 0 {
		t.Fatal("no world save resumed")
	}
}
