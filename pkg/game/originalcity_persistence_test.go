package game

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
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

func TestOriginalCityForgedBaselineCannotAuthorizeLossyExport(t *testing.T) {
	changes := map[string]func(*mapload.PartyMember){
		"body":      func(p *mapload.PartyMember) { p.Hero.Body++ },
		"name":      func(p *mapload.PartyMember) { p.Name = "forged" },
		"skill":     func(p *mapload.PartyMember) { p.Hero.Skill[2]++ },
		"skill xp":  func(p *mapload.PartyMember) { p.Carry.SkillXP[2]++ },
		"inventory": func(p *mapload.PartyMember) { p.Carried = append(p.Carried, 123) },
		"item value": func(p *mapload.PartyMember) {
			p.CarriedItems = append(p.CarriedItems, sim.ItemInstance{Code: 123, Price: 7})
		},
		"spell membership": func(p *mapload.PartyMember) { p.KnownSpells ^= 2 },
		"identity":         func(p *mapload.PartyMember) { p.ID = "forged-id" },
		"primary identity": func(p *mapload.PartyMember) { p.StartingHero = !p.StartingHero },
		"profile":          func(p *mapload.PartyMember) { p.FigureFace++ },
		"saved pools":      func(p *mapload.PartyMember) { p.Saved.HP++ },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f, _ := city1095Front(t)
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			beforeTown, beforeState := f.Town, f.originalCity
			beforeParty := mapload.CloneParty(f.Carried)
			id := s.Party[0].ID
			change(&s.Party[0])
			for i := range s.OriginalCity.Bindings {
				b := &s.OriginalCity.Bindings[i]
				if b.PartyID == id {
					change(&b.Baseline)
					b.PartyID = b.Baseline.ID
				}
			}
			if _, _, err := f.Restore(s); err == nil || (!strings.Contains(err.Error(), "original city") && !strings.Contains(err.Error(), "spellbook")) {
				t.Fatalf("forged baseline LOAD: %v", err)
			}
			if f.Town != beforeTown || f.originalCity != beforeState || !reflect.DeepEqual(mapload.CloneParty(f.Carried), beforeParty) {
				t.Fatal("refused forged baseline mutated the active session")
			}
			// Encode/decode have no install table. They validate structure;
			// production Restore must validate the complete import-derived value.
			payload := city1095CorruptEnvelope(t, s)
			if decoded, _, err := DecodeSave(payload); err == nil {
				payload, err = EncodeSave(decoded, "forged")
				if err != nil {
					t.Fatal(err)
				}
			}
			fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
			if out, _, err := fresh.ConvertCitySave(payload, "sav", nil); err == nil || len(out) != 0 {
				t.Fatalf("forged baseline exported %d bytes: %v", len(out), err)
			}
			if fresh.originalCity != nil || len(fresh.Carried) != 0 || fresh.Town != nil {
				t.Fatal("forged conversion installed partial state")
			}
		})
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
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			input := s
			dto := *s.OriginalCity
			input.OriginalCity = &dto
			if legacy {
				old := sav.CityStateData{RootKind: state.RootKind, Directories: map[string]uint32{}, Values: map[string]sav.CityStateValueData{}}
				for _, r := range state.DirectoryRecords {
					old.Directories[r.Path] = r.Kind
				}
				for _, r := range state.ValueRecords {
					old.Values[r.Path] = r.Value
				}
				dto.Document.Version, dto.Document.State = 1, old
			}
			first, err := EncodeSave(input, "stable")
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, err := DecodeSave(first)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.OriginalCity.Document.Version != sav.CityDataVersion || len(decoded.OriginalCity.Document.State.Values) != 0 {
				t.Fatal("legacy state did not normalize")
			}
			if legacy && (dto.Document.Version != 1 || len(dto.Document.State.Values) == 0) {
				t.Fatal("EncodeSave mutated caller provenance")
			}
			var original []byte
			for i := 0; i < 32; i++ {
				encoded, err := EncodeSave(input, "stable")
				if err != nil || !bytes.Equal(first, encoded) {
					t.Fatalf("same snapshot encode %d differs: %v", i, err)
				}
				reencoded, err := EncodeSave(decoded, "stable")
				if err != nil || !bytes.Equal(first, reencoded) {
					t.Fatalf("decoded snapshot encode %d differs: %v", i, err)
				}
				fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
				out, _, err := fresh.ConvertCitySave(reencoded, "sav", nil)
				if err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					original = out
				} else if !bytes.Equal(original, out) {
					t.Fatalf("SAV encode %d differs", i)
				}
			}
			if legacy {
				// A checksum-valid old envelope can still carry its original map
				// type fields; decode upgrades it without a custom opaque gob blob.
				old, _, err := DecodeSave(city1095CorruptEnvelope(t, input))
				if err != nil {
					t.Fatal(err)
				}
				upgraded, err := EncodeSave(old, "stable")
				if err != nil || !bytes.Equal(first, upgraded) {
					t.Fatalf("legacy decode upgrade: %v", err)
				}
			}
		})
	}
}

func TestOriginalCityNativeContinuationUsesProductionSAVE(t *testing.T) {
	f, source := city1095Front(t)
	s, _, err := f.Snapshot(false)
	if err != nil || s.OriginalCity == nil {
		t.Fatalf("snapshot %v", err)
	}
	s.Party[0], s.Party[1] = s.Party[1], s.Party[0] // bindings, not live order
	native, err := EncodeSave(s, "saved city")
	if err != nil {
		t.Fatal(err)
	}
	decoded, label, err := DecodeSave(native)
	if err != nil || label != "saved city" {
		t.Fatalf("decode %q %v", label, err)
	}
	fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
	if _, town, err := fresh.Restore(decoded); err != nil || !town {
		t.Fatalf("restore %t %v", town, err)
	}
	decoded.OriginalCity.Bindings[0].Baseline.Hero.Body++
	decoded.OriginalCity.Document.Objects[1].Unit.Name = "foreign"
	save, _, _ := agsSaveSeams(fresh, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
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
	current.OriginalCity.Document.Objects[1].Unit.Name = "detached"
	if fresh.originalCity.document.Roster()[0].Name == "detached" {
		t.Fatal("Snapshot leaked provenance ownership")
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
			native, err := EncodeSave(s, "changed")
			if err != nil {
				t.Fatal(err)
			}
			fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
			inconsistent := name == "human"
			if output, _, err := fresh.ConvertCitySave(native, "sav", nil); (err != nil) != inconsistent || (len(output) == 0) != inconsistent {
				t.Fatalf("export %d %v", len(output), err)
			} else if !inconsistent {
				cold := &FrontEnd{InstallResources: fresh.InstallResources}
				if _, town, err := cold.RestoreOriginal(output); err != nil || !town || len(cold.Carried) != len(s.Party) {
					t.Fatal("changed city SAV lost current roster", town, err, len(cold.Carried), len(s.Party))
				}
			}
			decoded, _, err := DecodeSave(native)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := fresh.Restore(decoded); err != nil {
				t.Fatal(err)
			}
			save, _, _ := agsSaveSeams(fresh, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
			want := ".sav"
			if file, err := save(false); inconsistent {
				if err == nil {
					t.Fatalf("SAVE of an inconsistent Human wrote %q", file)
				}
			} else if err != nil || !strings.HasSuffix(file, want) {
				t.Fatalf("SAVE %q %v, want %s", file, err, want)
			}
		})
	}
}

// Bypass EncodeSave deliberately: the checksum is valid, but the nested DTO
// is not. DecodeSave and direct Restore must enforce the same full contract.
func city1095CorruptEnvelope(t *testing.T, s Snapshot) []byte {
	t.Helper()
	var body bytes.Buffer
	enc := gob.NewEncoder(&body)
	if err := enc.Encode(s); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(s.randomSession); err != nil {
		t.Fatal(err)
	}
	out := append([]byte(saveMagic), saveVersion, 0, 0)
	out = binary.LittleEndian.AppendUint32(out, crc32.ChecksumIEEE(body.Bytes()))
	out = binary.LittleEndian.AppendUint32(out, uint32(body.Len()))
	return append(out, body.Bytes()...)
}

func TestOriginalCityMalformedSnapshotRollbackAndGobBounds(t *testing.T) {
	changes := map[string]func(*Snapshot){
		"future wrapper":   func(s *Snapshot) { s.OriginalCity.Version++ },
		"future document":  func(s *Snapshot) { s.OriginalCity.Document.Version++ },
		"foreign binding":  func(s *Snapshot) { s.OriginalCity.Bindings[0].Identity = 0xdeadbeef },
		"retarget binding": func(s *Snapshot) { s.OriginalCity.Bindings[0].PartyID = s.OriginalCity.Bindings[1].PartyID },
		"foreign party": func(s *Snapshot) {
			s.OriginalCity.Bindings[0].PartyID = "foreign"
			s.OriginalCity.Bindings[0].Baseline.ID = "foreign"
		},
		"duplicate binding":   func(s *Snapshot) { s.OriginalCity.Bindings[0] = s.OriginalCity.Bindings[1] },
		"missing binding":     func(s *Snapshot) { s.OriginalCity.Bindings = nil },
		"different character": func(s *Snapshot) { s.OriginalCity.Bindings[0].Baseline.Name = "foreign" },
		"bad field":           func(s *Snapshot) { s.OriginalCity.Document.Objects[1].Unit.RawD4 = nil },
		"mission":             func(s *Snapshot) { s.Mission = 10 },
		"baseline gob count":  func(s *Snapshot) { s.OriginalCity.Bindings[0].Baseline.Carried = make([]uint16, maxSaveGobElements+1) },
		"graph gob count": func(s *Snapshot) {
			s.OriginalCity.Document.Objects[1].Unit.Effects = make([]uint16, maxSaveGobElements+1)
		},
		"directory record gob count": func(s *Snapshot) {
			s.OriginalCity.Document.State.DirectoryRecords = make([]sav.CityStateDirectoryData, maxSaveGobElements+1)
		},
		"value record gob count": func(s *Snapshot) {
			s.OriginalCity.Document.State.ValueRecords = make([]sav.CityStateRecordData, maxSaveGobElements+1)
		},
		"forged session offered": func(s *Snapshot) { s.OriginalCity.Offered++; s.Offered++ },
		"forged session difficulty": func(s *Snapshot) {
			s.OriginalCity.Difficulty = mapload.DifficultyHard
			s.Difficulty = mapload.DifficultyHard
		},
		"forged marker latch": func(s *Snapshot) {
			s.OriginalCity.CampaignMarkerSelected = !s.OriginalCity.CampaignMarkerSelected
			s.CampaignMarkerSelected = s.OriginalCity.CampaignMarkerSelected
		},
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f, _ := city1095Front(t)
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			beforeTown, beforeState := f.Town, f.originalCity
			beforeParty := mapload.CloneParty(f.Carried)
			change(&s)
			if _, _, err := f.Restore(s); err == nil {
				t.Fatal("Restore accepted malformed provenance")
			}
			if f.Town != beforeTown || f.originalCity != beforeState || !reflect.DeepEqual(mapload.CloneParty(f.Carried), beforeParty) {
				t.Fatal("failed load changed active state")
			}
			if out, err := EncodeSave(s, ""); err == nil || len(out) != 0 {
				t.Fatal("EncodeSave accepted malformed provenance")
			}
			if _, _, err := DecodeSave(city1095CorruptEnvelope(t, s)); err == nil {
				t.Fatal("DecodeSave accepted malformed provenance")
			}
		})
	}
}

func legacyCityFixtureIdentities(s *Snapshot) {
	// This two-character fixture has its companion on row29. Version6
	// assigned npc22 even without a registry; preserve that old provenance.
	for i := range s.OriginalCity.Bindings {
		binding := &s.OriginalCity.Bindings[i]
		if binding.Baseline.StartingHero {
			continue
		}
		oldID := binding.PartyID
		binding.PartyID, binding.Baseline.ID, binding.Baseline.CompanionNPC = "npc:22", "npc:22", 22
		if binding.Baseline.OriginalHuman != nil {
			binding.Baseline.OriginalHuman.PartyID = "npc:22"
		}
		for j := range s.Party {
			if s.Party[j].ID == oldID {
				s.Party[j].ID, s.Party[j].CompanionNPC = "npc:22", 22
				if s.Party[j].OriginalHuman != nil {
					s.Party[j].OriginalHuman.PartyID = "npc:22"
				}
			}
		}
	}
}

func TestOriginalCityVersionSixCompanionIdentityStillLoads(t *testing.T) {
	f, _ := city1095Front(t)
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	s.OriginalCity.Version = 6
	legacyCityFixtureIdentities(&s)
	encoded, err := EncodeSave(s, "legacy companion")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	// The companion identity is the registry's: row 29 is npc 22 only while
	// the town grants npc 22; with no npc.reg record, or with the town
	// granting another companion, the legacy LOAD is refused.
	for _, npc := range []int{22, 0, 23} {
		cold, _ := city1095Front(t)
		if npc != 0 {
			legacyCompanionRegistry(cold, npc)
		}
		_, town, err := cold.Restore(decoded)
		if npc != 22 {
			if err == nil {
				t.Fatalf("grant %d: legacy LOAD accepted npc:22 for row 29", npc)
			}
			continue
		}
		if err != nil || !town {
			t.Fatalf("legacy LOAD town=%v err=%v", town, err)
		}
		if !reflect.DeepEqual(cold.Carried, s.Party) {
			t.Fatal("legacy LOAD changed saved party progress")
		}
	}
}

// legacyCompanionRegistry gives f the registry a town companion's legacy
// identity is read from: [Mission30] AddHero = npc and an [npc<npc>] record
// Mage,!MySex composed from DataBinID 26, whose rows 28 and 29 carry their
// own server ids. The returned function restores f's table and campaign.
func legacyCompanionRegistry(f *FrontEnd, npc int) func() {
	oldTable, oldCampaign := f.Table, f.Campaign
	humans := append(dbCollection(nil), f.Table.Humans.(dbCollection)...)
	for _, row := range []int{28, 29} {
		params := append([]int32(nil), humans[row].params...)
		for len(params) <= 0x18 {
			params = append(params, -1)
		}
		params[0x18] = int32(row)
		humans[row].params = params
	}
	r := &reg.Reg{Root: &reg.Node{Dir: true, Children: []*reg.Node{{Name: fmt.Sprintf("npc%d", npc), Dir: true, Children: []*reg.Node{
		{Name: "Flags", Type: reg.TypeString, Str: "Hero,Mage,!MySex,Start"},
		{Name: "DataBinID", Type: reg.TypeInt, Int: 26},
	}}}}}
	table := *f.Table
	table.Humans, table.NPC = humans, data.LoadNPCDefs(r)
	f.Table = &table
	c := f.Campaign.Value()
	chapters := make(map[int]Chapter, len(c.Chapters)+1)
	for n, ch := range c.Chapters {
		chapters[n] = ch
	}
	chapters[30] = Chapter{Mission: 30, AddHero: []int{npc}}
	c.Chapters = chapters
	f.Campaign = resolved(c, nil)
	return func() { f.Table, f.Campaign = oldTable, oldCampaign }
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
	if _, _, err := f.Restore(s); err != nil {
		t.Fatal(err)
	}
	if f.originalCity != nil {
		t.Fatal("old AGS inherited provenance")
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
	// AGS is a retired write target, so the legacy fixture the surviving
	// AGS->SAV direction reads below is authored directly from this city's
	// own current Snapshot instead of round-tripped through ConvertCitySave.
	native, err := EncodeSave(citySnapshot, "synthetic city")
	if err != nil {
		t.Fatal(err)
	}
	fresh := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil), Table: f.Table, Bodies: f.Bodies}}
	override := "renamed"
	out, label, err := fresh.ConvertCitySave(native, "sav", &override)
	if err != nil || label != override {
		t.Fatalf("AGS->SAV %q %v", label, err)
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
