package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func city1095Front(t *testing.T) (*FrontEnd, []byte) {
	t.Helper()
	raw, err := cityfixture.Original(false)
	if err != nil {
		t.Fatal(err)
	}
	rows := make(dbCollection, 30)
	rows[28].name, rows[29].name = "PC_Leader", "PC_Companion"
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: &mapload.Table{Humans: rows}}}
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("restore %t %v", town, err)
	}
	return f, raw
}

func bindCurrentCityFixtureRow(t *testing.T, f *FrontEnd) {
	t.Helper()
	if len(f.Carried) != 2 || f.Carried[1].StartingHero || f.Carried[1].Class != data.HeroUnmatchedClass {
		t.Fatal("current city fixture needs its two source members and class1 companion")
	}
	table := *f.Table
	rows := append(dbCollection(nil), table.Humans.(dbCollection)...)
	params := make([]int32, data.MinHumanRow)
	copy(params, rows[29].params)
	params[16] = data.HeroUnmatchedClass
	params[21], params[22] = 1, 1
	rows[1] = dbEntry{name: "PC_current_fixture", params: params}
	table.Humans = rows
	f.Table = &table
	if row := mapload.PartyHumanRow(f.Carried[1], f.Carried[0], f.Table); row != 1 {
		t.Fatalf("current fixture companion constructor row %d, want1", row)
	}
}

func TestOriginalCityNativeEncodingIsStableWithLegacyAndOrderedState(t *testing.T) {
	f, _ := city1095Front(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	state := &s.OriginalCity.Document.State
	if len(state.DirectoryRecords) < 3 || len(state.ValueRecords) < 3 || len(state.Directories) != 0 || len(state.Values) != 0 {
		t.Fatal("fixture does not exercise multiple ordered directories and values")
	}
	first, err := f.ExportCurrentSave(s, "stable")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		encoded, err := f.ExportCurrentSave(s, "stable")
		if err != nil || !bytes.Equal(first, encoded) {
			t.Fatalf("same current SAV encode %d differs: %v", i, err)
		}
	}
	after, _, err := f.Snapshot(false)
	if err != nil || !reflect.DeepEqual(s, after) {
		t.Fatal("current SAV mutated caller provenance", err)
	}
	fresh := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := fresh.RestoreOriginal(first); err != nil || !town {
		t.Fatal("stable SAV LOAD", town, err)
	}
}

func TestOriginalCityNativeContinuationUsesProductionSAVE(t *testing.T) {
	f, source := city1095Front(t)
	bindCurrentCityFixtureRow(t, f)
	s, _, err := f.Snapshot(false)
	if err != nil || s.OriginalCity == nil {
		t.Fatalf("snapshot %v", err)
	}
	s.Party[0], s.Party[1] = s.Party[1], s.Party[0] // bindings, not live order
	native, err := f.ExportCurrentSave(s, "saved city")
	if err != nil {
		t.Fatal(err)
	}
	fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
	if _, town, err := fresh.RestoreOriginal(native); err != nil || !town {
		t.Fatalf("restore %t %v", town, err)
	}
	if len(fresh.Carried) != 2 || fresh.Carried[0].ID != s.Party[0].ID || fresh.Carried[1].ID != s.Party[1].ID {
		t.Fatalf("current SAV changed the two persistent party identities: %+v", fresh.Carried)
	}
	s.OriginalCity.Bindings[0].Baseline.Hero.Body++
	s.OriginalCity.Document.Objects[1].Unit.Name = "foreign"
	save, _, _ := fresh.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !strings.HasSuffix(name, ".sav") {
		t.Fatalf("SAVE = %q %v", name, err)
	}
	current, _, err := fresh.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	first, err := fresh.ExportOriginalSave(current, "saved city")
	if err != nil {
		t.Fatal(err)
	}
	second, err := fresh.ExportOriginalSave(current, "saved city")
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("export nondeterministic %v", err)
	}
	if bytes.Equal(first, source) {
		t.Fatal("source file replayed")
	}
	parsed, err := sav.Open(first)
	if err != nil || string(parsed.Label) != "saved city" {
		t.Fatalf("output label %v", err)
	}
	if len(parsed.Players) != 1 || parsed.Players[0].Money != 123 {
		t.Fatal("purse lost")
	}
	if len(current.Party) != 2 || current.Party[0].Saved == nil || fresh.Carried[0].Saved == nil {
		t.Fatal("current city snapshot lost party/pool state")
	}
	before := *fresh.Carried[0].Saved
	current.Party[0].Saved.HP++
	if *fresh.Carried[0].Saved != before {
		t.Fatal("Snapshot leaked current party ownership")
	}
}

// Changed town state writes SAV. A contradictory legacy input still fails
// validation before it can replace the current party.
func TestOriginalCityChangedStateWritesSAV(t *testing.T) {
	changes := map[string]func(*Snapshot){
		"human":      func(s *Snapshot) { s.Party[0].Hero.Body++ },
		"item":       func(s *Snapshot) { s.Party[0].Carried = append(s.Party[0].Carried, 1) },
		"roster":     func(s *Snapshot) { s.Party = s.Party[:1] },
		"difficulty": func(s *Snapshot) { s.Difficulty = mapload.DifficultyHard },
		"selection":  func(s *Snapshot) { s.Offered++ },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f, _ := city1095Front(t)
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			change(&s)
			before := mapload.CloneParty(f.Carried)
			output, err := f.ExportCurrentSave(s, "changed")
			inconsistent := name == "human"
			if (err != nil) != inconsistent || (len(output) == 0) != inconsistent {
				t.Fatalf("export %d %v", len(output), err)
			}
			if !reflect.DeepEqual(before, f.Carried) {
				t.Fatal("export changed current party")
			}
			if inconsistent {
				return
			}
			fresh := &FrontEnd{InstallResources: f.InstallResources}
			if _, town, err := fresh.RestoreOriginal(output); err != nil || !town || len(fresh.Carried) != len(s.Party) {
				t.Fatal("changed city SAV lost current roster", town, err, len(fresh.Carried), len(s.Party))
			}
			save, _, _ := fresh.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
			if file, err := save(false); err != nil || !strings.HasSuffix(file, ".sav") {
				t.Fatalf("SAVE %q %v, want .sav", file, err)
			}
		})
	}
}

func TestCityConversionWriterFencesBothFormats(t *testing.T) {
	_, source := city1095Front(t)
	// The install fence inspects the target directory, not the payload or its
	// extension, so both extensions - including the retired "ags" write
	// target - must still be refused before the SAV writer looks at the
	// bytes at all. Any payload proves that; source needs no AGS-format twin.
	for _, format := range []string{"ags", "sav"} {
		for _, configured := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/configured=%t", format, configured), func(t *testing.T) {
				install, assets := t.TempDir(), t.TempDir()
				if configured {
					assets = install
				} else {
					for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
						if err := os.WriteFile(filepath.Join(install, name), nil, 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				dir := filepath.Join(install, "new", "nested")
				if err := WriteConvertedSave(filepath.Join(dir, "output."+format), source, assets); err == nil || !strings.Contains(err.Error(), "game install") {
					t.Fatalf("install write accepted: %v", err)
				}
				if _, err := os.Stat(filepath.Dir(dir)); !os.IsNotExist(err) {
					t.Fatalf("directory created: %v", err)
				}
			})
		}
	}
	missingContext := filepath.Join(t.TempDir(), "new", "output.ags")
	if err := WriteConvertedSave(missingContext, source, ""); err == nil {
		t.Fatal("empty install context accepted")
	}
	if _, err := os.Stat(filepath.Dir(missingContext)); !os.IsNotExist(err) {
		t.Fatal("empty context created directory")
	}
}

func TestCityConversionPublicationRechecksInstallFence(t *testing.T) {
	for _, ext := range []string{".ags", ".sav"} {
		t.Run(ext, func(t *testing.T) {
			dir, assets := t.TempDir(), t.TempDir()
			checks := 0
			validate := func(current saveDirectoryIdentity) error {
				if err := refuseOriginalWriteTarget(current.path, assets); err != nil {
					return err
				}
				checks++
				if checks == 1 {
					// Simulate an install becoming visible after initial validation,
					// before the temporary file is published. Synthetic markers only.
					for _, name := range []string{MainArchive, GraphicsArchive, ScenarioArchive, WorldArchive, MoviesArchive} {
						if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
							return err
						}
					}
				}
				return nil
			}
			_, err := (SaveStore{Dir: dir}).write([]byte("synthetic"), "save-conversion", ext, validate, func(i int) (string, bool) { return "output" + ext, i == 0 })
			if err == nil || !strings.Contains(err.Error(), "game install") {
				t.Fatalf("publication accepted: %v", err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 5 {
				t.Fatalf("output or temporary file leaked: %v %v", entries, err)
			}
		})
	}
}

func TestOriginalCityNoProvenanceAndNewSessionNeverInheritBindings(t *testing.T) {
	f, _ := city1095Front(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	s.OriginalCity = nil
	f.installCandidate(&restoreCandidate{town: testRestoreTown(f.Campaign.Value(), s), carried: mapload.CloneParty(s.Party), townOnly: true})
	if f.originalCity != nil {
		t.Fatal("current candidate inherited provenance")
	}
	out, err := f.ExportCurrentSave(s, "current city without provenance")
	if err != nil {
		t.Fatal("current city without provenance cannot SAVE", err)
	}
	doc, err := sav.DecodeDocumentData(out)
	if err != nil || doc.Head.Mission != 0 {
		t.Fatal("current city did not produce an ordinary city SAV", err)
	}
	cold, _ := city1095Front(t)
	if open, city, err := cold.RestoreOriginal(out); err != nil || !city || open != nil {
		t.Fatal("current city without provenance cannot cold LOAD", city, err)
	}
	f, _ = city1095Front(t)
	f.resetSessionForNewGame()
	if f.originalCity != nil {
		t.Fatal("new game inherited provenance")
	}
	f, _ = city1095Front(t)
	f.installCandidate(&restoreCandidate{town: NewTown(saveCampaign()), townOnly: true})
	if f.originalCity != nil {
		t.Fatal("mission/native candidate inherited provenance")
	}
}

func TestCityConversionPreservesLabelAndNoOverwrite(t *testing.T) {
	f, _ := city1095Front(t)
	citySnapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	override := "renamed"
	out, err := f.ExportCurrentSave(citySnapshot, override)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(out)
	if err != nil || string(file.Label) != override {
		t.Fatalf("current SAV label %v", err)
	}
	path := filepath.Join(t.TempDir(), "converted.sav")
	assets := t.TempDir()
	if err := WriteConvertedSave(path, out, assets); err != nil {
		t.Fatal(err)
	}
	if err := WriteConvertedSave(path, out, assets); err == nil {
		t.Fatal("output replaced")
	}
	kept, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(kept, out) {
		t.Fatal("existing output mutated")
	}
	wrong := filepath.Join(t.TempDir(), "wrong.ags")
	if err := WriteConvertedSave(wrong, out, assets); err == nil {
		t.Fatal("SAV written under AGS extension")
	}
	if _, err := os.Stat(wrong); !os.IsNotExist(err) {
		t.Fatal("failed output exists")
	}
}
