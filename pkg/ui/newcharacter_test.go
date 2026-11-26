package ui

import (
	"slices"
	"testing"
	"time"
)

// acceptChargen presses Enter on the generator until it leaves, through the
// pre-create page where the game has one.
func acceptChargen(t *testing.T, a *App) {
	t.Helper()
	for i := 0; i < 3 && a.Screen() == ScreenChargen; i++ {
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
}

// Ok on the New row opens the generator the store's New makes. Cancel
// returns to the list having begun nothing; Accept begins the store's write
// and returns to the list, which shows the new row selected. Ok on a
// character row opens nothing.
func TestNewCharacterRowRunsTheStoresGenerator(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for name, a := range multiplayerApps(t) {
		t.Run(name, func(t *testing.T) {
			calls := &characterStoreCalls{rows: []SaveEntry{{Name: "b.chr", Label: "Beta"}}}
			files := fakeCharacterFiles(calls)
			var begun []string
			files.New = func() *ChargenEntry {
				return &ChargenEntry{Model: NewChargen(chargenNamedSetup()), Begin: func(res ChargenResult) (MapOpener, error) {
					begun = append(begun, res.Name)
					calls.rows = append([]SaveEntry{{Name: "a.chr", Label: res.Name}}, calls.rows...)
					return nil, nil
				}}
			}
			a.SetCharacterFiles(files)
			if err := a.HeadlessActivate("multi player"); err != nil {
				t.Fatal(err)
			}
			clickPageButton(t, a, 0, now)
			if a.Screen() != ScreenLoad {
				t.Fatalf("Ok on a character row opened %v", a.Screen())
			}
			a.flow.loadList.Select(1)
			clickPageButton(t, a, 0, now)
			if a.Screen() != ScreenChargen {
				t.Fatalf("Ok on the New row opened %v, want the generator", a.Screen())
			}
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if a.Screen() != ScreenLoad || len(begun) != 0 || !slices.Equal(rowTexts(a), []string{"Beta", "New Character"}) {
				t.Fatalf("Cancel left %v with %d begun and rows %q", a.Screen(), len(begun), rowTexts(a))
			}
			a.flow.loadList.Select(1)
			clickPageButton(t, a, 0, now)
			acceptChargen(t, a)
			if a.Screen() != ScreenLoad || !slices.Equal(begun, []string{"Hero"}) {
				t.Fatalf("Accept left %v with %q begun", a.Screen(), begun)
			}
			if rows := rowTexts(a); !slices.Equal(rows, []string{"Hero", "Beta", "New Character"}) || a.HeadlessListSelection() != 0 {
				t.Fatalf("the list after Accept shows %q with row %d selected", rows, a.HeadlessListSelection())
			}
		})
	}
}
