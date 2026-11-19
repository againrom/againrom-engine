package game

import (
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func setJoin(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(ui.ModCompanionsScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("%s made %d undo steps", key, len(s.undoWorkspace)-steps)
	}
}

func heldJoins(t *testing.T, s *mapInspectionEdits) []mod.CompanionJoin {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, nil, mod.CompanionsFile)
	if err != nil {
		t.Fatal(err)
	}
	return d.Companions.Joins
}

// The companions page of a mod with no file shows the Add button and writes
// nothing; a row is added, changed and removed, each edit one undo step, and
// the file the page makes reads back as the example mod's.
func TestModCompanionsPageEditsJoinRows(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, folder := charactersPage(t, lang)
			before := tree(t, folder)
			if fields := s.WorkspaceFields(ui.ModCompanionsScope); len(fields) != 1 || fields[0].Key != companionsAddKey || !fields[0].Action {
				t.Fatalf("the empty page holds %+v", fields)
			}
			if !maps.Equal(tree(t, folder), before) {
				t.Fatal("an empty page wrote")
			}
			setJoin(t, s, companionsAddKey, "")
			setJoin(t, s, "join.0.talk", "22")
			got := heldJoins(t, s)
			if len(got) != 1 || got[0].Chapter != 30 || got[0].Companion != 22 || got[0].Talk != 22 || got[0].Building != mod.BuildingTavern {
				t.Fatalf("the row: %+v", got)
			}
			want, err := os.ReadFile(filepath.Join("..", "modrt", "testdata", "mods", "reniesta-joins-on-talk", "data", "companions.toml"))
			if err != nil {
				t.Fatal(err)
			}
			example, err := mod.ParseCompanions("m", mod.CompanionsFile, want)
			if err != nil {
				t.Fatal(err)
			}
			page, err := mod.ParseCompanions("m", mod.CompanionsFile, s.workspace.Project.Files[mod.CompanionsFile])
			if err != nil || len(page.Joins) != 1 || len(example.Joins) != 1 {
				t.Fatalf("page file %q: %v", s.workspace.Project.Files[mod.CompanionsFile], err)
			}
			a, b := page.Joins[0], example.Joins[0]
			if a.Chapter != b.Chapter || a.Companion != b.Companion || a.Talk != b.Talk || a.Building != b.Building {
				t.Fatalf("page row %+v, example row %+v", a, b)
			}
			steps := len(s.undoWorkspace)
			if _, err := s.SetWorkspaceField(ui.ModCompanionsScope, companionsAddKey, ""); err == nil || len(s.undoWorkspace) != steps {
				t.Fatalf("a row for a companion that has one: %v", err)
			}
			if _, err := s.SetWorkspaceField(ui.ModCompanionsScope, "join.0.talk", "0"); err == nil || len(s.undoWorkspace) != steps {
				t.Fatalf("a talk of 0: %v", err)
			}
			setJoin(t, s, "join.0.talk", "23")
			if got := heldJoins(t, s); len(got) != 1 || got[0].Talk != 23 {
				t.Fatalf("the changed row: %+v", got)
			}
			setJoin(t, s, "join.0.remove", "")
			if _, held := s.workspace.Project.Files[mod.CompanionsFile]; held || len(heldJoins(t, s)) != 0 {
				t.Fatal("removing every row left the file")
			}
			for range len(s.undoWorkspace) {
				if _, err := s.Undo(); err != nil {
					t.Fatal(err)
				}
			}
			if len(heldJoins(t, s)) != 0 || !maps.Equal(tree(t, folder), before) {
				t.Fatal("undo left a row")
			}
		})
	}
}

// The page offers the campaign's chapters, companions and conversations, and
// a save is refused naming a row the campaign cannot run.
func TestModCompanionsPageOffersCampaignChoicesAndGatesSave(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	x.flow = newCampaignFlow(Campaign{Chapters: map[int]Chapter{
		30: {AddHero: []int{22}, InnNPC: []int{22, 5}},
		31: {AddHero: []int{22}, InnNPC: []int{7}},
	}}, mod.MissionData{})
	setJoin(t, s, companionsAddKey, "")
	f := func(key string) ui.InspectionField {
		for _, g := range s.WorkspaceFields(ui.ModCompanionsScope) {
			if g.Key == key {
				return g
			}
		}
		t.Fatalf("no field %s", key)
		return ui.InspectionField{}
	}
	if c := f("join.0.chapter"); len(c.Choices) != 2 || c.Value != 30 {
		t.Fatalf("chapters: %+v", c)
	}
	if c := f("join.0.talk"); len(c.Choices) != 2 || c.Value != 22 {
		t.Fatalf("talks: %+v", c)
	}
	setJoin(t, s, "join.0.chapter", strconv.Itoa(31))
	if got := heldJoins(t, s)[0]; got.Chapter != 31 || got.Talk != 7 {
		t.Fatalf("a new chapter keeps a conversation it lacks: %+v", got)
	}
	setJoin(t, s, "join.0.talk", "99")
	if err := s.modRowsGate(companionsRows); err == nil || !strings.Contains(err.Error(), "chapter 31") || !strings.Contains(err.Error(), "99") {
		t.Fatalf("the gate: %v", err)
	}
	setJoin(t, s, "join.0.talk", "7")
	if err := s.modRowsGate(companionsRows); err != nil {
		t.Fatal(err)
	}
}

// A row added after another row's chapter was changed takes a key no row holds.
func TestModCompanionsPageAddTakesAFreeKey(t *testing.T) {
	x, s, _ := charactersPage(t, "en")
	x.flow = newCampaignFlow(Campaign{Chapters: map[int]Chapter{
		30: {AddHero: []int{22}, InnNPC: []int{22}},
		31: {AddHero: []int{22}, InnNPC: []int{7}},
	}}, mod.MissionData{})
	setJoin(t, s, companionsAddKey, "")
	setJoin(t, s, "join.0.chapter", "31")
	setJoin(t, s, companionsAddKey, "")
	got := heldJoins(t, s)
	if len(got) != 2 || got[0].Key != "join_30_22" || got[1].Chapter != 30 || got[1].Key == got[0].Key {
		t.Fatalf("rows: %+v", got)
	}
}
