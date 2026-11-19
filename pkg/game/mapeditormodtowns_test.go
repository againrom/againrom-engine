package game

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

func townsField(t *testing.T, s *mapInspectionEdits, key string) ui.InspectionField {
	t.Helper()
	for _, f := range s.WorkspaceFields(ui.ModTownsScope) {
		if f.Key == key {
			return f
		}
	}
	t.Fatalf("no Towns field %s", key)
	return ui.InspectionField{}
}

func setTowns(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	steps, before := len(s.undoWorkspace), workspaceBytes(s.workspace)
	if _, err := s.SetWorkspaceField(ui.ModTownsScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 || bytes.Equal(before, workspaceBytes(s.workspace)) {
		t.Fatalf("%s = %q made %d undo steps", key, value, len(s.undoWorkspace)-steps)
	}
}

// The Towns page writes one town change, one tavern offer and the talk text
// of that offer in each language as draft steps with one undo each; the
// towns file the draft writes reads back as the edits.
func TestModTownsPageWritesATownAnOfferAndItsTalkTexts(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, s, _ := handoutEditFixture(t, entry, "")
			name := townsField(t, s, "town.1.name")
			if name.TextValue != "" {
				t.Fatalf("a game town with no name of its own shows %q", name.TextValue)
			}
			if _, err := s.SetWorkspaceField(ui.ModTownsScope, "town.1.name", "Harbor"); err == nil || !strings.Contains(err.Error(), "glyph") {
				t.Fatalf("a name the install cannot draw: %v", err)
			}
			before := workspaceBytes(s.workspace)
			edit := func(field, value string) {
				t.Helper()
				steps := len(s.undoWorkspace)
				if _, err := s.changeWorkspace(func(w *editorWorkspaceState) error { return s.editTown(w, s.townsBook(), 1, field, value) }); err != nil {
					t.Fatalf("%s = %q: %v", field, value, err)
				}
				if len(s.undoWorkspace) != steps+1 {
					t.Fatalf("%s made %d undo steps", field, len(s.undoWorkspace)-steps)
				}
			}
			edit("x", "10")
			edit("y", "20")
			if f := townsField(t, s, "town.1.y"); f.TextValue != "20" || !strings.Contains(f.Group, "changed") && !strings.Contains(f.Group, "изменен") {
				t.Fatalf("the town shows %+v", f)
			}
			if d := s.workspace.Project; len(d.TownRecords) != 1 || !slices.Equal(d.TownRecords[0].Entry.MapPoint.Point, []int{10, 20}) {
				t.Fatalf("records %+v", d.TownRecords)
			}
			if _, err := s.Undo(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Undo(); err != nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
				t.Fatalf("undo of the point: %v", err)
			}
			if _, err := s.Redo(); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Redo(); err != nil {
				t.Fatal(err)
			}
			edit("reset", "")
			if len(s.workspace.Project.TownRecords) != 0 {
				t.Fatal("the town change stayed after Undo changes")
			}
			setTowns(t, s, "add", "5")
			d := s.workspace.Project
			if len(d.Towns) != 1 || d.Towns[0].Row.NPC != 5 || d.Towns[0].Table != town.TavernTable {
				t.Fatalf("the added offer is %+v", d.Towns)
			}
			k := townsOfferName(town.TavernTable.Key(d.Towns[0].Row))
			for _, l := range []string{"en", "ru"} {
				talk := townsField(t, s, k+"talk."+l)
				if talk.TextValue == "" {
					t.Fatalf("the new offer has no %s talk", l)
				}
				setTowns(t, s, k+"talk."+l, "Words in "+l)
			}
			for _, l := range []string{"en", "ru"} {
				path, _ := talkPath(d.Towns[0].Row)
				rel, held := talkFile(s.workspace.Project, l, path)
				if !held || mod.TalkBody(s.workspace.Project.Files[rel]) != "Words in "+l {
					t.Fatalf("talk file %s: %q", rel, s.workspace.Project.Files[rel])
				}
			}
			src, changed, err := s.workspace.Project.townsDraft()
			if err != nil || !changed {
				t.Fatalf("towns draft: %v", err)
			}
			records, offers, err := mod.ParseTowns("m", mod.TownsFile, src)
			if err != nil || len(records) != 0 || len(offers) != 1 || offers[0].Row.NPC != 5 {
				t.Fatalf("written file reads back %+v %+v: %v\n%s", records, offers, err, src)
			}
			setTowns(t, s, k+"drop", "")
			if len(s.workspace.Project.Towns) != 0 {
				t.Fatal("the mod's offer stayed after Delete offer")
			}
		})
	}
}

// Deleting an offer, or changing its character, removes the talk texts the
// page made for it in the same undo step; a text of another format stays.
func TestModTownsPageRemovesTheTalkTextsOfAnOfferItDrops(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, s, _ := handoutEditFixture(t, entry, "")
			talkFiles := func() []string {
				var out []string
				for name := range s.workspace.Project.Files {
					if strings.Contains(name, "/inn/npc/npc") {
						out = append(out, name)
					}
				}
				slices.Sort(out)
				return out
			}
			setTowns(t, s, "add", "5")
			if len(talkFiles()) != 2 {
				t.Fatalf("an added offer makes %v", talkFiles())
			}
			added := talkFiles()
			k := townsOfferName(town.TavernTable.Key(s.workspace.Project.Towns[0].Row))
			setTowns(t, s, k+"drop", "")
			if len(talkFiles()) != 0 {
				t.Fatalf("Delete offer left %v", talkFiles())
			}
			if _, err := s.Undo(); err != nil || !slices.Equal(talkFiles(), added) {
				t.Fatalf("undo of the delete: %v %v", err, talkFiles())
			}
			setTowns(t, s, k+"npc", "6")
			now := talkFiles()
			if len(now) != 2 || slices.Contains(now, added[0]) || slices.Contains(now, added[1]) {
				t.Fatalf("changing the character left %v, was %v", now, added)
			}
			for _, f := range now {
				if !strings.Contains(f, "npc06") {
					t.Fatalf("new text path %s", f)
				}
			}
			k = townsOfferName(town.TavernTable.Key(s.workspace.Project.Towns[0].Row))
			s.workspace.Project.Files[now[0]] = []byte("<part=1 npc=6>\r\nhand\r\n<part=2 npc=6>\r\nmore\r\n")
			setTowns(t, s, k+"drop", "")
			if got := talkFiles(); len(got) != 1 || got[0] != now[0] {
				t.Fatalf("a hand-made text should stay alone, files %v", got)
			}
		})
	}
}

// A town or an offer a mission of the mod needs is refused in plain words.
func TestModTownsPageRefusesATownAndOfferAMissionNeeds(t *testing.T) {
	_, s, _ := handoutEditFixture(t, "english", "")
	d := s.workspace.Project
	d.Missions = append(d.Missions, mod.Mission{Key: "ford", Number: 31, Title: "Ford"})
	book := s.townsBook()
	d.TownRecords = append(d.TownRecords, mod.TownRecord{Entry: town.Entry{ID: 1, Missions: []int{31}}, Fields: []mod.PatchField{{Name: "missions"}}})
	if _, err := s.SetWorkspaceField(ui.ModTownsScope, "town.1.off", ""); err == nil || !strings.Contains(err.Error(), "Ford (31)") {
		t.Fatalf("switching off a town a mission needs: %v", err)
	}
	row := town.ChapterOffer(1, 30, 9, 31)
	rows := []town.OfferRow{row}
	if err := s.offerNeeded(rows, row, s.workspace); err == nil || !strings.Contains(err.Error(), "Ford (31)") {
		t.Fatalf("taking away the only offer of a mission: %v", err)
	}
	if err := s.offerNeeded(append(rows, town.ChapterOffer(1, 40, 10, 31)), row, s.workspace); err != nil {
		t.Fatalf("another offer gives the mission: %v", err)
	}
	if book == nil || !slices.Contains(book.catalog.IDs(), 1) {
		t.Fatal("no first-game town book")
	}
}
