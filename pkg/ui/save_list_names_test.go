package ui

import (
	"strings"
	"testing"
)

// Neither save list shows a file extension or a format marker: SAV is the only
// format. File names, selection and the name field's target stay exact.
func TestSaveAndLoadListsShowNoExtension(t *testing.T) {
	entries := []SaveEntry{
		{Name: "game0001.sav", Label: "7 Mission 20 gold"},
		{Name: "Chapter.sav.sav", Label: "Chapter"},
		{Name: "UPPER.SAV", Label: "Upper"},
	}
	spy := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}}
	a := newSaveDialogApp(t, spy, ScreenTown)
	rows := a.HeadlessRows()
	want := []string{"game0001", "Chapter.sav", "UPPER"}
	if len(rows) != len(want) {
		t.Fatalf("SAVE rows = %v", rows)
	}
	for i, row := range rows {
		if row.Text != want[i] {
			t.Fatalf("SAVE row %d shows %q, want %q", i, row.Text, want[i])
		}
	}
	state, _ := a.HeadlessSaveState()
	for i, e := range state.Entries {
		if e.Name != entries[i].Name {
			t.Fatalf("entry %d lost its disk name: %q", i, e.Name)
		}
	}
	for i, name := range []string{"game0001", "Chapter.sav.sav", "UPPER"} {
		if err := a.HeadlessSaveSelect(i); err != nil {
			t.Fatal(err)
		}
		mustSaveAction(t, a, "open")
		if got, _ := a.HeadlessSaveState(); got.Request.Name != name {
			t.Fatalf("row %d filled the name field with %q, want %q", i, got.Request.Name, name)
		}
	}

	b := newTestApp(t, nil, nil)
	b.SetSaveSeams(nil, func() []SaveEntry { return entries }, nil)
	b.flow.openLoad(ScreenMenu)
	for i, row := range b.HeadlessRows() {
		if row.Text != entries[i].Label || strings.Contains(strings.ToLower(row.Text), ".sav") {
			t.Fatalf("LOAD row %d shows %q", i, row.Text)
		}
	}
}
