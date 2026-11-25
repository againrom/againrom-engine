package ui

import (
	"slices"
	"testing"
	"time"
)

// characterStoreCalls records what the page asked of a character store.
type characterStoreCalls struct {
	listed   int
	prepared []string
	removed  []string
	rows     []SaveEntry
}

// fakeCharacterFiles lists rows with one file left out and records each
// delete; a removal drops the row.
func fakeCharacterFiles(c *characterStoreCalls) CharacterFiles {
	return CharacterFiles{
		List: func() ([]SaveEntry, string) {
			c.listed++
			return slices.Clone(c.rows), "left out: 1"
		},
		CanDelete: func(name string) bool { return true },
		PrepareDelete: func(name string) (func() error, error) {
			c.prepared = append(c.prepared, name)
			return func() error {
				c.removed = append(c.removed, name)
				c.rows = slices.DeleteFunc(c.rows, func(e SaveEntry) bool { return e.Name == name })
				return nil
			}, nil
		},
	}
}

func rowTexts(a *App) []string {
	var out []string
	for _, r := range a.HeadlessRows() {
		out = append(out, r.Text)
	}
	return out
}

// The character page lists the store's rows by name in the store's order,
// then the New row, with the store's message; Ok on the New row stays.
// The first game's Delete asks the Load page's confirmation and removes only
// the chosen file; the second game's page has no Delete, by button or key.
// The Load page then lists the saves as before.
func TestCharacterPageListsTheStoreAndDeletesInTheFirstGame(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for name, a := range multiplayerApps(t) {
		t.Run(name, func(t *testing.T) {
			first := name == "first"
			calls := &characterStoreCalls{rows: []SaveEntry{{Name: "b.chr", Label: "Beta"}, {Name: "a.chr", Label: "Alpha"}}}
			a.SetCharacterPage(CharacterPage{Title: "T", OK: "O", New: "New Character", Delete: first})
			a.SetCharacterFiles(fakeCharacterFiles(calls))
			load := a.flow.loadPage()
			if err := a.HeadlessActivate("multi player"); err != nil {
				t.Fatal(err)
			}
			if got := rowTexts(a); !slices.Equal(got, []string{"Beta", "Alpha", "New Character"}) || a.HeadlessMessage() != "left out: 1" {
				t.Fatalf("rows %q message %q", got, a.HeadlessMessage())
			}
			a.flow.loadList.Select(2)
			clickPageButton(t, a, loadOKButton, now)
			if a.Screen() != ScreenLoad || !a.CharacterList() || len(calls.prepared) != 0 {
				t.Fatalf("Ok on the New row left for %v", a.Screen())
			}
			if v := a.flow.loadView(); a.flow.canDeletePage(v) {
				t.Errorf("the New row can be deleted")
			}
			a.flow.loadList.Select(1)
			if err := a.HeadlessKey("delete"); err != nil {
				t.Fatal(err)
			}
			if !first {
				if len(calls.prepared) != 0 || a.flow.loadUI.confirm {
					t.Fatalf("second game: Delete key prepared %v", calls.prepared)
				}
				return
			}
			if !slices.Equal(calls.prepared, []string{"a.chr"}) || len(calls.removed) != 0 || !a.flow.loadUI.confirm {
				t.Fatalf("Delete prepared %v removed %v confirm %v", calls.prepared, calls.removed, a.flow.loadUI.confirm)
			}
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(calls.removed, []string{"a.chr"}) || calls.listed != 2 || !a.CharacterList() {
				t.Fatalf("confirmed Delete removed %v, listed %d times, character list %v", calls.removed, calls.listed, a.CharacterList())
			}
			if got := rowTexts(a); !slices.Equal(got, []string{"Beta", "New Character"}) {
				t.Fatalf("rows after Delete %q", got)
			}
			clickPageButton(t, a, loadCancelButton, now)
			if err := a.HeadlessActivate("load game"); err != nil {
				t.Fatal(err)
			}
			again := a.flow.loadPage()
			if a.CharacterList() || !slices.Equal(rowTexts(a), []string{"saved"}) || again.Buttons != load.Buttons || len(again.Texts) != len(load.Texts) {
				t.Fatalf("Load after the character list: rows %q", rowTexts(a))
			}
		})
	}
}
