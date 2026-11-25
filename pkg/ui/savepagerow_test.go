package ui

import (
	"fmt"
	"strings"
	"testing"
)

// saveRowTrace drives the save page through its row on page: a typed name
// and Ok, the overwrite of a listed name, Delete with its question, Cancel and
// Escape. It returns what the page did, one line per step.
func saveRowTrace(t *testing.T, page *SavePage) string {
	t.Helper()
	fake := &pageSeamsFake{
		entries:  []SaveEntry{{Name: "game0001.sav", Label: "Alpha"}, {Name: "game0000.sav", Label: "Beta"}},
		existing: map[string]bool{"game0001.sav": true, "game0000.sav": true, "game0001": true, "game0000": true},
	}
	seams := fake.seams()
	var removed []string
	seams.CanDelete = func(dir, name string) bool { return name != "game0000.sav" }
	seams.PrepareDelete = func(dir, name string) (func() error, error) {
		return func() error { removed = append(removed, dir+"/"+name); return nil }, nil
	}
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetSaveDialogSeams(seams)
	if page != nil {
		a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: page})
	}
	var out []string
	note := func(step string) {
		h, ok := a.HeadlessSaveState()
		out = append(out, fmt.Sprintf("%s: screen=%v open=%v name=%q label=%q confirm=%v remove=%v msg=%q requests=%d commits=%v",
			step, a.Screen(), ok, h.Request.Name, h.Request.Label, h.Confirmation, a.flow.saveScreen != nil && a.flow.saveScreen.ui.remove != nil,
			a.flow.msg, len(fake.requests), fake.overwrite))
	}
	try := func(step string, err error) {
		if err != nil {
			out = append(out, step+": error "+err.Error())
		}
		note(step)
	}
	openTestSavePage(t, a)
	note("open")
	try("type", a.HeadlessSaveEdit("", "Gamma", SaveSAV))
	try("ok", a.HeadlessSaveAction("save"))
	openTestSavePage(t, a)
	try("select", a.HeadlessSaveSelect(len(a.HeadlessRows())-1))
	try("ok existing", a.HeadlessSaveAction("save"))
	try("cancel question", a.HeadlessSaveAction("cancel"))
	try("ok again", a.HeadlessSaveAction("save"))
	if h, _ := a.HeadlessSaveState(); h.Confirmation {
		try("overwrite", a.HeadlessSaveAction("overwrite"))
	}
	openTestSavePage(t, a)
	try("select first", a.HeadlessSaveSelect(len(a.HeadlessRows())-2))
	try("delete", a.HeadlessSaveAction("delete"))
	try("cancel delete", a.HeadlessSaveAction("cancel"))
	try("delete again", a.HeadlessSaveAction("delete"))
	try("confirm delete", a.HeadlessSaveAction("confirm"))
	try("escape", a.HeadlessKey("escape"))
	if a.Screen() == ScreenSave {
		try("cancel", a.HeadlessSaveAction("cancel"))
	}
	out = append(out, fmt.Sprintf("files: requests=%+v removed=%v", fake.requests, removed))
	return strings.Join(out, "\n")
}

// TestSavePageDrivenThroughItsRowInBothGames holds what the save page does
// for the first game's page and the second game's page to the transcript the
// page wrote before it was a row of the list page roles: equal on base.
func TestSavePageDrivenThroughItsRowInBothGames(t *testing.T) {
	second := testSavePage(&pageSeamsFake{})
	for _, c := range []struct {
		name string
		page *SavePage
		want string
	}{
		{"first", nil, wantFirstSaveRow},
		{"second", &second, wantSecondSaveRow},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := saveRowTrace(t, c.page); got != c.want {
				t.Fatalf("transcript:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

const wantFirstSaveRow = `open: screen=save open=true name="save" label="" confirm=false remove=false msg="" requests=0 commits=[]
type: screen=save open=true name="Gamma" label="" confirm=false remove=false msg="" requests=0 commits=[]
ok: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=1 commits=[false]
select: screen=save open=true name="game0000" label="" confirm=false remove=false msg="" requests=1 commits=[false]
ok existing: screen=save open=true name="game0000" label="" confirm=true remove=false msg="" requests=2 commits=[false]
cancel question: screen=save open=true name="game0000" label="" confirm=false remove=false msg="" requests=2 commits=[false]
ok again: screen=save open=true name="game0000" label="" confirm=true remove=false msg="" requests=3 commits=[false]
overwrite: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=3 commits=[false true]
select first: screen=save open=true name="game0001" label="" confirm=false remove=false msg="" requests=3 commits=[false true]
delete: screen=save open=true name="game0001" label="" confirm=true remove=true msg="" requests=3 commits=[false true]
cancel delete: screen=save open=true name="game0001" label="" confirm=false remove=false msg="" requests=3 commits=[false true]
delete again: screen=save open=true name="game0001" label="" confirm=true remove=true msg="" requests=3 commits=[false true]
confirm delete: screen=save open=true name="game0001" label="" confirm=false remove=false msg="" requests=3 commits=[false true]
escape: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="" requests=3 commits=[false true]
files: requests=[{OnMap:true Admission:false Directory:D Name:Gamma Label: Format:SAV} {OnMap:true Admission:false Directory:D Name:game0000 Label: Format:SAV} {OnMap:true Admission:false Directory:D Name:game0000 Label: Format:SAV}] removed=[D/game0001.sav]`

const wantSecondSaveRow = `open: screen=save open=true name="" label="" confirm=false remove=false msg="" requests=0 commits=[]
type: screen=save open=true name="" label="Gamma" confirm=false remove=false msg="" requests=0 commits=[]
ok: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=1 commits=[false]
select: screen=save open=true name="" label="Beta" confirm=false remove=false msg="" requests=1 commits=[false]
ok existing: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=2 commits=[false true]
cancel question: error save dialog: cancel is unavailable on this screen
cancel question: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=2 commits=[false true]
ok again: error save dialog: save is unavailable on this screen
ok again: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="Your character is saved" requests=2 commits=[false true]
select first: screen=save open=true name="" label="Alpha" confirm=false remove=false msg="" requests=2 commits=[false true]
delete: screen=save open=true name="" label="Alpha" confirm=true remove=true msg="" requests=2 commits=[false true]
cancel delete: screen=save open=true name="" label="Alpha" confirm=false remove=false msg="" requests=2 commits=[false true]
delete again: screen=save open=true name="" label="Alpha" confirm=true remove=true msg="" requests=2 commits=[false true]
confirm delete: screen=save open=true name="" label="" confirm=false remove=false msg="" requests=2 commits=[false true]
escape: screen=gamemenu open=false name="" label="" confirm=false remove=false msg="" requests=2 commits=[false true]
files: requests=[{OnMap:true Admission:false Directory:D Name:game0002.sav Label:Gamma Format:SAV} {OnMap:true Admission:false Directory:D Name:game0000.sav Label:Beta Format:SAV}] removed=[D/game0001.sav]`
