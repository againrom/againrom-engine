package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

func TestTownSaveKeepsCurrentExperience(t *testing.T) {
	for _, mode := range []string{"loaded", "current", "retained", "native"} {
		t.Run(mode, func(t *testing.T) {
			const loadedXP, skillXP = uint32(1637), uint32(1400)
			doc, err := sav.DecodeDocumentData(city1099Fixture(t))
			if err != nil {
				t.Fatal(err)
			}
			for i := range doc.Objects {
				r := &doc.Objects[i]
				if r.Class == "Human" {
					mustSetValue(r, "U130", loadedXP)
					binary.LittleEndian.PutUint32(cityGraftRaw(t, r, "H1CC")[4:], skillXP)
				}
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			f := city1099Front(t, raw)
			want := loadedXP
			for i := range f.Carried {
				p := &f.Carried[i]
				if p.Carry == nil || p.Carry.LiveLoad == nil || p.OriginalHuman == nil {
					t.Fatal("fixture lacks current and retained Human state", p.ID)
				}
				if mode == "current" {
					want = loadedXP + 937
					p.Carry.LiveLoad.Inventory.Source.Experience = want
				}
				if mode == "retained" {
					p.Carry.LiveLoad = nil
				}
				if mode == "native" {
					p.Carry.LiveLoad, p.OriginalHuman = nil, nil
					want = skillXP
				}
			}
			for cycle := 0; cycle < 2; cycle++ {
				s, label, err := f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				before := mapload.CloneParty(s.Party)
				raw, err := f.ExportCurrentSave(s, label)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, s.Party) {
					t.Fatal("SAVE changed the input party")
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				var actors []uint16
				for i, r := range doc.Objects {
					if r.Class == "Human" {
						actors = append(actors, uint16(i+1))
					}
				}
				chars, err := sav.ReadDocumentCharacters(doc, actors)
				if err != nil || len(chars) != len(before) {
					t.Fatal("ordinary Human population", len(chars), len(before), err)
				}
				for _, c := range chars {
					if c.Character.Experience != want || c.Character.SkillXP[1] != skillXP {
						t.Fatalf("%s cycle %d: written Experience=%d SkillXP=%v, want %d and %d", c.Character.Name, cycle, c.Character.Experience, c.Character.SkillXP, want, skillXP)
					}
				}
				cold := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
				if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
					t.Fatal("cold town LOAD", town, err)
				}
				for _, p := range cold.Carried {
					if p.Carry == nil || p.Carry.SkillXP[1] != int32(skillXP) {
						t.Fatal("cold LOAD changed current experience", p.ID)
					}
					if mode != "native" && (p.OriginalHuman == nil || p.OriginalHuman.State.Experience != want) {
						t.Fatal("cold LOAD changed Human experience", p.ID)
					}
					if mode != "retained" && mode != "native" && (p.Carry.LiveLoad == nil || p.Carry.LiveLoad.Inventory.Source.Experience != want) {
						t.Fatal("cold LOAD changed actor experience", p.ID)
					}
				}
				f = cold
			}
		})
	}
}
