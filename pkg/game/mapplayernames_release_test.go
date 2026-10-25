package game

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseMapEditorPlayerNamesKeepSAVBytes(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, tc := range []struct {
		route string
		wire  []byte
	}{
		{"editor", []byte{0xcc, 0xe5, 0xe4, 0xfc}},
		{"original-editor-bytes", []byte{0xcc, 0xe5, 0xe4, 0xfc}},
		{"undefined-byte", []byte{'M', 0x98, 0xff}},
		{"full-width", bytes.Repeat([]byte{0xdf}, 32)},
	} {
		t.Run(tc.route, func(t *testing.T) {
			x := installedInspector(t)
			class := playerColourClass(t, x)
			root := newMapMod(t)
			if err := x.SetMod(filepath.Join(root, newMapModID)); err != nil {
				t.Fatal(err)
			}
			e := x.Editor()
			if err := e.Open(newMapSource(32, 32)); err != nil {
				t.Fatal(err)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			apply := func(_ *ui.InspectionDocument, err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			apply(s.SetDropCell(image.Pt(16, 16)))
			apply(s.AddUnit(image.Pt(19, 16), int64(class)))
			if err := e.Key("players"); err != nil {
				t.Fatal(err)
			}
			if err := e.Key("player"); err != nil {
				t.Fatal(err)
			}
			addSteps(t, e, nil)
			if err := enterField(e, 0, "\u041c\u0435\u0434\u044c"); err != nil {
				t.Fatal(err)
			}
			apply(s.SetField("Unit", 0, "owner", 3))
			fields := s.Fields("Player", 2)
			if len(fields) == 0 || !fields[0].Text || fields[0].TextValue != "Медь" {
				t.Fatal("text prompt lost the decoded name", fields)
			}
			row := e.Document().Records[recordIndex(t, e, "Player", 2)]
			if !strings.HasSuffix(row.Label, x.inspectionUTF8("Медь")) {
				t.Fatalf("roster display changed: %q", row.Label)
			}
			if err := pastNotice(e, "saveas"); err != nil || e.Dirty() {
				t.Fatalf("save into mod: %v", err)
			}
			path := filepath.Join(root, newMapModID, "mission", "map.alm")
			mapBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sections, _ := playerByteSections(t, mapBytes)
			found := false
			for _, section := range sections {
				if section.typeID != 5 {
					continue
				}
				name := section.payload[2*76+12 : 2*76+44]
				if tc.route != "editor" {
					clear(name)
					copy(name, tc.wire)
				}
				if !bytes.Equal(bytes.TrimRight(name, "\x00"), tc.wire) {
					t.Fatalf("map name = %x, want %x", name, tc.wire)
				}
				found = true
			}
			if !found {
				t.Fatal("saved map lacks roster")
			}
			if tc.route != "editor" {
				if err := os.WriteFile(path, mapBytes, 0600); err != nil {
					t.Fatal(err)
				}
			}
			f, _ := startArmsMission(t, root)
			if actor := playerMissionActor(t, f); actor.Owner != 3 {
				t.Fatal("authored owner", actor.Owner)
			}
			display := alm.PlayerNameText.Decode(tc.wire)
			if got := f.live.cheatDisplayPlayerName(3); got != EncodeInstallText(display, f.live.view.TextSelector()) {
				t.Fatalf("game display name = %x", []byte(got))
			}
			export := func(front *FrontEnd) []byte {
				t.Helper()
				snapshot, label, err := front.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := front.ExportCurrentSave(snapshot, label)
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
			playerName := func(raw []byte, slot uint32) string {
				t.Helper()
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				for _, index := range doc.Players {
					record := &doc.Objects[index-1]
					if got, _ := savedStructureValue(record, "Slot"); got != slot {
						continue
					}
					for _, text := range record.Texts {
						if text.Name == "Name" {
							return text.Value
						}
					}
				}
				t.Fatal("SAV lacks Player Name for slot", slot)
				return ""
			}
			reload := func(raw []byte) *FrontEnd {
				t.Helper()
				cold := guestFront(t, root, newMapModID)
				open, town, err := cold.RestoreOriginal(raw)
				if err != nil || town {
					t.Fatal("cold SAV LOAD", town, err)
				}
				if err := cold.App("player names after LOAD").OpenMission(open); err != nil {
					t.Fatal(err)
				}
				return cold
			}
			raw := export(f)
			for cycle := range 2 {
				if got := playerName(raw, 3); got != string(tc.wire) {
					t.Fatalf("SAV cycle %d Player Name = %x, want Windows-1251 %x", cycle, []byte(got), tc.wire)
				}
				file, err := sav.Open(raw)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(file.Body, tc.wire) || bytes.Contains(file.Body, []byte(display)) {
					t.Fatalf("SAV cycle %d body: Windows-1251 %x present=%t; UTF-8 %x present=%t", cycle, tc.wire, bytes.Contains(file.Body, tc.wire), []byte(display), bytes.Contains(file.Body, []byte(display)))
				}
				t.Logf("%s %s SAV cycle %d: Player Name %x; UTF-8 form absent", filepath.Base(os.Getenv("AGAINROM_ASSETS")), tc.route, cycle, tc.wire)
				if cycle == 0 {
					raw = export(reload(raw))
				}
			}
			if tc.route != "editor" {
				return
			}
			for _, imported := range []struct {
				label string
				slot  uint32
				name  string
			}{
				{"original-cp1251", 3, string([]byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2})},
				{"earlier-utf8", 3, "Медь"},
				{"empty", 3, ""},
				{"local-player", 1, string([]byte{0xff, 0x80, 'P'})},
			} {
				t.Run(imported.label, func(t *testing.T) {
					doc, err := sav.DecodeDocumentData(raw)
					if err != nil {
						t.Fatal(err)
					}
					changed := false
					for _, index := range doc.Players {
						record := &doc.Objects[index-1]
						if slot, _ := savedStructureValue(record, "Slot"); slot == imported.slot {
							mustSetText(record, "Name", imported.name)
							changed = true
						}
					}
					if !changed {
						t.Fatal("import fixture lacks Player", imported.slot)
					}
					current, err := sav.EncodeDocumentData(doc)
					if err != nil {
						t.Fatal(err)
					}
					for cycle := range 2 {
						current = export(reload(current))
						if got := playerName(current, imported.slot); got != imported.name {
							t.Fatalf("imported %s cycle %d Name = %x, want %x", imported.label, cycle, []byte(got), []byte(imported.name))
						}
					}
					t.Logf("imported %s slot %d kept Name %x through two exports", imported.label, imported.slot, []byte(imported.name))
				})
			}
		})
	}
}
