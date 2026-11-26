package ui

import (
	"reflect"
	"testing"
)

// The Load and Save screens are one list page: the same compose and draw
// methods, a step that ends in the same stepPage, and a page projector each.
func TestSaveScreenSharesTheListPageHandlers(t *testing.T) {
	load, save := screenHandlers[ScreenLoad], screenHandlers[ScreenSave]
	for name, pair := range map[string][2]any{"compose": {load.Compose, save.Compose}, "draw": {load.Draw, save.Draw}} {
		if reflect.ValueOf(pair[0]).Pointer() != reflect.ValueOf(pair[1]).Pointer() {
			t.Errorf("%s differs between Load and Save", name)
		}
	}
	if load.Page == nil || save.Page == nil || save.Step == nil {
		t.Fatal("list page handler lacks a member")
	}
}

// Name entry, the overwrite confirmation, a cancel and the written request go
// through the one page: the page sends exactly the request the edit names and
// commits once with overwrite after the explicit answer.
func TestSavePageEntryOverwriteAndCancelUseTheSharedPage(t *testing.T) {
	s := &saveDialogSpy{existing: []string{"x"}, paths: []string{"x"}}
	a := newSaveDialogApp(t, s, ScreenTown)
	if err := a.HeadlessSaveEdit("dir", "slot", SaveSAV); err != nil {
		t.Fatal(err)
	}
	mustSaveAction(t, a, "save")
	if len(s.requests) != 1 {
		t.Fatalf("requests = %v", s.requests)
	}
	r := s.requests[0]
	if r.Name != "slot" || r.Directory != "dir" || r.Format != SaveSAV || r.OnMap {
		t.Fatalf("request = %+v", r)
	}
	if h, _ := a.HeadlessSaveState(); !h.Confirmation {
		t.Fatal("existing target was not confirmed")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if h, _ := a.HeadlessSaveState(); h.Confirmation || len(s.commits) != 0 {
		t.Fatalf("escape did not leave the confirmation: %+v %v", h, s.commits)
	}
	mustSaveAction(t, a, "save")
	mustSaveAction(t, a, "overwrite")
	if !reflect.DeepEqual(s.commits, []bool{true}) || a.Screen() != ScreenGameMenu {
		t.Fatalf("overwrite = %v on %s", s.commits, a.Screen())
	}
	b := newSaveDialogApp(t, &saveDialogSpy{}, ScreenTown)
	if err := b.HeadlessKey("escape"); err != nil || b.Screen() != ScreenGameMenu {
		t.Fatalf("cancel left %s: %v", b.Screen(), err)
	}
}
