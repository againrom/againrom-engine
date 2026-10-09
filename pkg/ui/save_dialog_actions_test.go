package ui

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveDialogTabSkipsFormatSelector(t *testing.T) {
	a := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
	if err := a.HeadlessKey("tab"); err != nil {
		t.Fatal(err)
	}
	h, _ := a.HeadlessSaveState()
	if h.Focus != "save" {
		t.Fatalf("Tab after name focuses %q, want save", h.Focus)
	}
	paint, err := a.saveDialogPaint()
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range paint.texts {
		if label.text == "SAV" {
			t.Fatal("SAVE still displays a format selector")
		}
	}
}

func TestSaveDialogDeleteUsesSelectedListedFile(t *testing.T) {
	for _, input := range []string{"mouse", "keyboard"} {
		t.Run(input, func(t *testing.T) {
			s := &saveDialogSpy{directories: map[string]SaveDirectory{
				"saves": {Path: "saves", Directories: []string{"child"}, Entries: []SaveEntry{{Name: "first.sav"}, {Name: "selected.sav"}, {Name: "last.sav"}}},
			}}
			a := newSaveDialogApp(t, s, ScreenTown)
			var prepared, removed []string
			a.flow.saveDialogSeams.CanDelete = func(directory, name string) bool { return true }
			a.flow.saveDialogSeams.PrepareDelete = func(directory, name string) (func() error, error) {
				path := filepath.Join(directory, name)
				prepared = append(prepared, path)
				return func() error {
					removed = append(removed, path)
					dir := s.directories[directory]
					dir.Entries = []SaveEntry{{Name: "first.sav"}, {Name: "last.sav"}}
					s.directories[directory] = dir
					return nil
				}, nil
			}
			if input == "mouse" {
				saveDialogClick(a, 100, 144)
			} else {
				if err := a.HeadlessKey("shift-tab"); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					if err := a.HeadlessKey("down"); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := a.HeadlessSaveEdit("", "unselected.sav", ""); err != nil {
				t.Fatal(err)
			}
			saveDialogClick(a, 150, 65)
			if err := a.HeadlessKey("ctrl-a"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessType("other-directory", false); err != nil {
				t.Fatal(err)
			}
			startDelete := func() {
				if input == "mouse" {
					saveDialogClick(a, 195, 444)
				} else {
					for i := 0; i < 3; i++ {
						if err := a.HeadlessKey("tab"); err != nil {
							t.Fatal(err)
						}
					}
					if err := a.HeadlessKey("delete"); err != nil {
						t.Fatal(err)
					}
				}
			}
			startDelete()
			h, _ := a.HeadlessSaveState()
			if !h.Confirmation || h.Focus != "cancel" || len(removed) != 0 {
				t.Fatalf("unsafe delete preparation: %+v, removed=%v", h, removed)
			}
			want := filepath.Join("saves", "selected.sav")
			if len(prepared) != 1 || prepared[0] != want {
				t.Fatalf("prepared %v, want %s", prepared, want)
			}
			if got := strings.Join(a.saveConfirmationLines(), ""); !strings.Contains(got, want) || strings.Contains(got, "unselected") || strings.Contains(got, "other-directory") {
				t.Fatalf("confirmation = %q", got)
			}
			if err := a.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			h, _ = a.HeadlessSaveState()
			if h.Confirmation || len(removed) != 0 {
				t.Fatal("default Cancel deleted a file")
			}
			mustSaveAction(t, a, "delete")
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if len(removed) != 0 || a.Screen() != ScreenSave {
				t.Fatal("Escape did not cancel deletion safely")
			}
			mustSaveAction(t, a, "delete")
			if err := a.HeadlessSaveAction("overwrite"); err == nil {
				t.Fatal("overwrite accepted a delete confirmation")
			}
			if err := a.HeadlessSaveEdit("changed", "changed", ""); err == nil {
				t.Fatal("pending delete allowed target editing")
			}
			if err := a.HeadlessSaveAction("open"); err == nil {
				t.Fatal("pending delete allowed directory navigation")
			}
			if input == "mouse" {
				saveDialogClick(a, 195, 444)
			} else {
				if err := a.HeadlessKey("tab"); err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			h, _ = a.HeadlessSaveState()
			if len(removed) != 1 || removed[0] != want || h.Confirmation || len(h.Entries) != 2 || h.Request.Directory != "saves" {
				t.Fatalf("delete/refresh = %v / %+v", removed, h)
			}
			if got := a.flow.saveDialog.list.Selection(); got != 2 {
				t.Fatalf("refreshed selection = %d, want following save row 2", got)
			}
			if len(s.requests) != 0 || len(s.commits) != 0 {
				t.Fatal("delete invoked the save producer")
			}
		})
	}
}

func TestSaveDialogDeleteTracksNewDirectoryAndIgnoresTextDelete(t *testing.T) {
	s := &saveDialogSpy{directories: map[string]SaveDirectory{
		"saves": {Path: "saves", Entries: []SaveEntry{{Name: "old.sav"}}},
		"other": {Path: "other", Directories: []string{"child"}, Entries: []SaveEntry{{Name: "new.sav"}}},
	}}
	a := newSaveDialogApp(t, s, ScreenTown)
	var target string
	a.flow.saveDialogSeams.CanDelete = func(directory, name string) bool { return true }
	a.flow.saveDialogSeams.PrepareDelete = func(directory, name string) (func() error, error) {
		target = filepath.Join(directory, name)
		return func() error { return nil }, nil
	}
	if err := a.HeadlessSaveSelect(0); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("delete"); err != nil {
		t.Fatal(err)
	}
	h, _ := a.HeadlessSaveState()
	if h.Request.Name != "" || h.Confirmation || target != "" {
		t.Fatal("text Delete triggered file deletion")
	}
	if err := a.HeadlessSaveEdit("other", "old.sav", ""); err != nil {
		t.Fatal(err)
	}
	saveDialogClick(a, 195, 444)
	h, _ = a.HeadlessSaveState()
	if h.Confirmation || target != "" {
		t.Fatal("directory row allowed deletion of prior file")
	}
	if err := a.HeadlessSaveSelect(1); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "delete")
	if want := filepath.Join("other", "new.sav"); target != want {
		t.Fatalf("target = %s, want %s", target, want)
	}
	mustSaveAction(t, a, "back")
	if err := a.HeadlessSaveEdit("empty", "old.sav", ""); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("delete"); err == nil {
		t.Fatal("empty directory retained a deletable selection")
	}
}

func saveDialogClick(a *App, x, y int) {
	a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, headlessNow)
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, headlessNow)
}

func TestSaveDialogDeleteFailureRequiresFreshConfirmation(t *testing.T) {
	s := &saveDialogSpy{directories: map[string]SaveDirectory{
		"saves": {Path: "saves", Entries: []SaveEntry{{Name: "selected.sav"}}},
	}}
	a := newSaveDialogApp(t, s, ScreenTown)
	prepared, removed := 0, 0
	a.flow.saveDialogSeams.CanDelete = func(string, string) bool { return true }
	a.flow.saveDialogSeams.PrepareDelete = func(string, string) (func() error, error) {
		prepared++
		return func() error { removed++; return errors.New("selected file changed") }, nil
	}
	mustSaveAction(t, a, "delete")
	if err := a.HeadlessSaveAction("confirm"); err == nil {
		t.Fatal("delete failure disappeared")
	}
	h, _ := a.HeadlessSaveState()
	if h.Confirmation || h.Message != "selected file changed" || len(h.Entries) != 1 {
		t.Fatalf("failed deletion state = %+v", h)
	}
	if err := a.HeadlessSaveAction("confirm"); err == nil {
		t.Fatal("stale delete confirmation remained active")
	}
	mustSaveAction(t, a, "delete")
	if prepared != 2 || removed != 1 {
		t.Fatalf("prepare/remove = %d/%d, want 2/1", prepared, removed)
	}
	saveDialogClick(a, 420, 444)
	if removed != 1 {
		t.Fatal("Back button invoked deletion")
	}
}
