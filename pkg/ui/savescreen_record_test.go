package ui

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"os"
	"testing"

	"againrom/pkg/render/text"
)

type saveScreenRecord struct {
	Screen   string
	Frame    string
	State    HeadlessSaveDialog
	Rows     []PickerRow
	Texts    []text.DrawCall
	Requests []SaveRequest
	Commits  []bool
}

func TestSaveScreenRecords(t *testing.T) {
	wantBytes, err := os.ReadFile("testdata/savescreen_records.json")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	for _, game := range []string{"first", "second"} {
		for _, populated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/entries=%v", game, populated), func(t *testing.T) {
				entries := []SaveEntry(nil)
				if populated {
					entries = []SaveEntry{{Name: "game0001.sav", Label: "Alpha", Detail: "Alpha detail"}, {Name: "game0000.sav", Label: "Beta", Detail: "Beta detail"}}
				}
				spy := &saveDialogSpy{directories: map[string]SaveDirectory{"saves": {Path: "saves", Entries: entries}}, paths: []string{"saves/occupied.sav"}, existing: []string{"saves/occupied.sav"}}
				fake := &pageSeamsFake{entries: entries, existing: map[string]bool{"game0001.sav": true, "game0000.sav": true}}
				var a *App
				if game == "first" {
					a = newSaveDialogApp(t, spy, ScreenTown)
					a.SetWords(AuthoredWords(), chargenTestFont(), nil)
				} else {
					a = lookApp(t, fake)
					p := lookPage(fake)
					for i, caption := range []string{"Ok", "Delete", "Cancel"} {
						p.Buttons[i] = PageButton{Rect: image.Rect(154+i*122, 318, 250+i*122, 342), Caption: caption}
					}
					p.Confirm = PageConfirm{Before: "Delete selected saved game?", At: image.Pt(125, 146), Label: image.Rect(122, 180, 520, 210)}
					a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &p})
					openTestSavePage(t, a)
				}
				record := func(label string) {
					t.Helper()
					r := saveScreenRecord{Screen: a.Screen().String(), Rows: a.HeadlessRows()}
					r.State, _ = a.HeadlessSaveState()
					if a.Screen() == ScreenSave {
						r.Texts = text.Record(func() {
							pix, err := a.composeSaveScreen()
							if err != nil {
								t.Fatal(err)
							}
							r.Frame = fmt.Sprintf("%x", sha256.Sum256(pix.Pix))
						})
					}
					if game == "first" {
						r.Requests, r.Commits = spy.requests, spy.commits
					} else {
						r.Requests, r.Commits = fake.requests, fake.overwrite
					}
					b, err := json.Marshal(r)
					if err != nil {
						t.Fatal(err)
					}
					key := fmt.Sprintf("%s/%v/%s", game, populated, label)
					got := fmt.Sprintf("%x", sha256.Sum256(b))
					if os.Getenv("RECORD_SAVE_SCREEN") == "1" {
						t.Logf("RECORD %s %s", key, got)
					} else if got != want[key] {
						t.Errorf("%s: got %s want %s; record %s", key, got, want[key], b)
					}
				}
				key := func(value string) {
					t.Helper()
					if err := a.HeadlessKey(value); err != nil {
						t.Fatal(err)
					}
				}
				record("open")
				if game == "first" {
					for i := 0; i < 8; i++ {
						key("tab")
						record(fmt.Sprintf("focus-%d", i))
					}
				} else {
					for i := 0; i < 3; i++ {
						a.step(appInput{CursorX: 180 + i*122, CursorY: 330, PrimaryPressed: true}, headlessNow)
						record(fmt.Sprintf("button-%d", i))
					}
					a.step(appInput{Unfocused: true}, headlessNow)
					key("down")
					record("list-focus")
				}
				if populated {
					i := 0
					if game == "second" {
						i = 1
					}
					if err := a.HeadlessSaveSelect(i); err != nil {
						t.Fatal(err)
					}
					record("selected")
				}
				if err := a.HeadlessSaveEdit("", "Gamma", SaveSAV); err != nil {
					t.Fatal(err)
				}
				record("typed")
				for _, k := range []string{"left", "backspace", "home", "delete", "end"} {
					key(k)
				}
				if err := a.HeadlessType("Z", false); err != nil {
					t.Fatal(err)
				}
				record("edited")
				if game == "first" {
					a.step(appInput{CursorX: 300, CursorY: 444, PrimaryPressed: true}, headlessNow)
					a.step(appInput{CursorX: 300, CursorY: 444, PrimaryReleased: true}, headlessNow)
					record("overwrite-question")
					key("escape")
					record("overwrite-no")
					key("enter")
					key("tab")
					key("enter")
					record("overwrite-yes")
				} else {
					if populated {
						mustSaveAction(t, a, "delete")
						record("delete-question")
						mustSaveAction(t, a, "back")
						record("delete-no")
					}
					a.step(appInput{CursorX: 180, CursorY: 330, PrimaryPressed: true}, headlessNow)
					a.step(appInput{CursorX: 180, CursorY: 330, PrimaryReleased: true}, headlessNow)
					record("save-click")
					if populated {
						openTestSavePage(t, a)
						if err := a.HeadlessSaveSelect(1); err != nil {
							t.Fatal(err)
						}
						mustSaveAction(t, a, "delete")
						mustSaveAction(t, a, "confirm")
						record("delete-yes")
					}
				}
				a.flow.openGameMenu(ScreenTown)
				a.flow.openSaveDialog()
				key("escape")
				record("escape")
			})
		}
	}
}
