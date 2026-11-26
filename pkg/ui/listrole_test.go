package ui

import (
	"slices"
	"testing"
)

// A role the page has never seen is one row: the page draws the row's page,
// lists the row's rows and message, deletes through the row's seams, and Ok
// runs the row's accept, whose leaving answers the row's caller.
func TestListPageOpensARoleRow(t *testing.T) {
	for name, a := range multiplayerApps(t) {
		var prepared, answered []string
		accepted := 0
		role := &listRole{
			roleName: "probe",
			shownOn:  ScreenLoad,
			chooses:  true,
			page: func(f *flow) SavePage {
				p := f.installedLoadPage()
				p.Texts = []PageText{{Text: "Probe", Box: p.Texts[0].Box}}
				return p
			},
			list: func(*flow, pageView) listing {
				entries := []SaveEntry{{Name: "a.x", Label: "A"}, {Name: "b.x", Label: "B"}}
				return listing{saves: entries, rows: labelRows(entries), message: "two rows"}
			},
			accept: func(f *flow) {
				accepted++
				if i, ok := f.loadList.Choose(); ok {
					f.leaveLoad(f.saves[i].Name, true)
				}
			},
			answer: func(_ *flow, chosen string, ok bool) {
				if ok {
					answered = append(answered, chosen)
				}
			},
			deletes: func(*flow) (func(string) bool, func(string) (func() error, error)) {
				return func(string) bool { return true }, func(n string) (func() error, error) {
					prepared = append(prepared, n)
					return func() error { return nil }, nil
				}
			},
		}
		a.flow.openListPage(ScreenMenu, role)
		if a.Screen() != ScreenLoad || a.CharacterList() || a.flow.msg != "two rows" || a.flow.loadPage().Texts[0].Text != "Probe" {
			t.Fatalf("%s: opened %v, message %q, title %q", name, a.Screen(), a.flow.msg, a.flow.loadPage().Texts[0].Text)
		}
		if got := rowTexts(a); !slices.Equal(got, []string{"A", "B"}) {
			t.Fatalf("%s: rows %q", name, got)
		}
		if err := a.HeadlessKey("delete"); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(prepared, []string{"a.x"}) || !a.flow.loadUI.confirm {
			t.Fatalf("%s: Delete prepared %v, confirm %v", name, prepared, a.flow.loadUI.confirm)
		}
		a.flow.cancelPage(a.flow.loadView())
		if err := a.HeadlessActivate("B"); err != nil {
			t.Fatal(err)
		}
		if accepted != 1 || !slices.Equal(answered, []string{"b.x"}) || a.Screen() != ScreenMenu || a.flow.loadUI.role != nil {
			t.Fatalf("%s: Ok accepted %d, answered %v, screen %v", name, accepted, answered, a.Screen())
		}
	}
}
