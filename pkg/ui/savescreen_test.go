package ui

import "testing"

func TestSavePageDeleteWithoutCommitNeverSaves(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0001.sav", Label: "Alpha"}}}
	a := lookApp(t, fake)
	a.flow.saveDialogSeams.PrepareDelete = func(string, string) (func() error, error) { return nil, nil }
	if err := a.HeadlessSaveSelect(1); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "delete")
	mustSaveAction(t, a, "confirm")
	h, ok := a.HeadlessSaveState()
	if !ok || !h.Confirmation || len(fake.requests) != 0 || len(fake.overwrite) != 0 {
		t.Fatalf("missing delete commit became SAVE: %+v requests %v commits %v", h, fake.requests, fake.overwrite)
	}
}

func TestSaveScreenDescriptionCanAskBeforeASlotOverwrite(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0001.sav", Label: "Alpha"}}, existing: map[string]bool{"game0001.sav": true}}
	a := lookApp(t, fake)
	page := lookPage(fake)
	page.overwriteQuestion = true
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})
	openTestSavePage(t, a)
	if err := a.HeadlessSaveSelect(1); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	h, ok := a.HeadlessSaveState()
	if !ok || !h.Confirmation || len(fake.overwrite) != 0 {
		t.Fatalf("description did not ask before overwriting: %+v %v", h, fake.overwrite)
	}
	mustSaveAction(t, a, "confirm")
	if len(fake.requests) != 1 || fake.requests[0].Name != "game0001.sav" || fake.requests[0].Label != "Alpha" || len(fake.overwrite) != 1 || !fake.overwrite[0] || a.Screen() != ScreenGameMenu {
		t.Fatalf("slot overwrite: %v %v %v", fake.requests, fake.overwrite, a.Screen())
	}
}
