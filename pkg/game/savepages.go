package game

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The second game's page geometry, in the 640x480 frame: the box is the
// original's 476 by 344 pixels at 1024x768, its title centred, the name box,
// heading, list and buttons placed as the EN original draws them. The page
// constructor's own numbers (100, 30, 600, 450) are not read as a rectangle
// (DIV-3078).
var (
	secondPagePanel   = image.Rect(82, 48, 558, 392)
	secondPageTitle   = image.Rect(102, 56, 538, 84)
	secondPageMessage = image.Rect(120, 372, 527, 396)
	secondPageName    = image.Rect(120, 98, 527, 122)
	// secondPageQuestion is the delete question's window as the EN original
	// draws it at 1024x768 (DIV-4176).
	secondPageQuestion = ui.NoticeShape{Size: image.Pt(288, 224), TextTop: 57, TextWidth: 208, Pitch: 17,
		Answer: image.Pt(82, 24), AnswerGap: 46, AnswerBottom: 28}
	// secondSaveList and secondLoadList are the list arguments: 383 wide,
	// the 24-pixel bar after them reaches the name box's right edge.
	secondSaveList = image.Rect(120, 160, 503, 160)
	secondLoadList = image.Rect(120, 120, 503, 120)
)

const (
	secondSaveRows = 7
	secondLoadRows = 10
)

// secondEmptyInk is the Empty Slot row's white, selected or not.
var secondEmptyInk = color.RGBA{240, 240, 240, 255}

func secondPageButtons(top int, ok, del, cancel string) [3]ui.PageButton {
	var out [3]ui.PageButton
	for i, caption := range []string{ok, del, cancel} {
		out[i] = ui.PageButton{Rect: image.Rect(154+i*122, top, 250+i*122, top+24), Caption: caption}
	}
	return out
}

// secondSavePages builds the second game's load and save pages from its
// dialogs.txt and patch.txt rows (R2-ASSET-098).
func secondSavePages(f *FrontEnd) *ui.SavePages {
	dialogs := &TextTable{}
	patch := &TextTable{}
	if f.Archives != nil {
		dialogs = LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
		if p, err := openInstallContainers(f.Archives, patchArchive); err == nil {
			patch = LoadTextTable(p, patchTextPath, f.textCode())
		}
	}
	row := func(t *TextTable, i int, fallback string) string {
		if s, ok := t.At(i); ok {
			return s
		}
		return fallback
	}
	text := func(i int, fallback string, x, y int) ui.PageText {
		return ui.PageText{Text: row(dialogs, i, fallback), At: image.Pt(x, y)}
	}
	title := func(i int, fallback string) ui.PageText {
		return ui.PageText{Text: row(dialogs, i, fallback), Box: secondPageTitle}
	}
	heading := func(t ui.PageText) ui.PageText {
		t.Focus = true
		return t
	}
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	// The question is patch.txt row 50, the label and row 51 (R2-ENGINE-438);
	// its answers are the engine's words.
	confirm := ui.PageConfirm{Before: row(patch, 50, "Delete"), After: row(patch, 51, "?"), Window: secondPageQuestion,
		Yes: encodeEngineWord(f.Words.Engine.Text("load.delete.yes"), selector),
		No:  encodeEngineWord(f.Words.Engine.Text("load.delete.no"), selector)}
	ok, del, cancel := row(dialogs, 0, "Ok"), row(dialogs, 157, "Delete"), row(dialogs, 1, "Cancel")
	load := ui.SavePage{
		Panel: secondPagePanel,
		Texts: []ui.PageText{
			title(151, "Load Saved Game"),
			heading(text(24, "Load Game", 120, 98)),
		},
		List: secondLoadList, Rows: secondLoadRows, Ruled: true, RowHint: true,
		Message: secondPageMessage, Buttons: secondPageButtons(338, ok, del, cancel),
		Confirm: confirm, OKNeedsSave: true,
	}
	// R2-ENGINE-443: the name box holds 0x7c bytes in EN and 0x80 in RU,
	// read as 123 and 127 characters and a terminator.
	limit := 123
	if f.Base().Profile.Language == "russian" {
		limit = 127
	}
	save := ui.SavePage{
		Panel: secondPagePanel,
		Texts: []ui.PageText{
			title(29, "Save Game"),
			listing(heading(text(31, "Saved Games", 120, 134))),
		},
		List: secondSaveList, Rows: secondSaveRows, Ruled: true, RowHint: true,
		Message: secondPageMessage, Buttons: secondPageButtons(318, ok, del, cancel),
		Confirm: confirm,
		NameBox: secondPageName, NameLimit: limit,
		NameHint:  strings.ToUpper(row(dialogs, 30, "Enter a name for the saved game")),
		EmptySlot: fontText(row(dialogs, 28, "<<< Empty Slot >>>"), selector),
		EmptyInk:  secondEmptyInk,
		Hidden:    secondSaveHidden(),
		Slot:      secondSaveSlot,
	}
	return &ui.SavePages{Load: load, Save: &save}
}

// secondSaveHidden are the names the save page does not list: the two the
// original's restore points use (R2-SESSION-179) and the engine's quick and
// timed slots, which only their own keys write (DIV-3080).
func secondSaveHidden() []string {
	out := []string{"game9998.sav", "game9999.sav"}
	for _, k := range []ownedSlotKind{quickSlots, timedSlots} {
		for slot := range 3 {
			out = append(out, string(k.slotBase(slot))+originalSaveExt)
		}
	}
	return out
}

func listing(t ui.PageText) ui.PageText {
	t.Listing = true
	return t
}

// secondSaveSlot names the Empty Slot's file: gameNNNN.sav with the smallest
// number from 0 that no listed save carries in name characters 4..7
// (R2-SESSION-179). A number whose file exists unlisted is skipped too, so the
// slot never overwrites a file the page does not show (DIV-3080). The
// numbers 9998 and 9999 are never chosen.
func secondSaveSlot(dir string, listed []string) string {
	used := map[int]bool{}
	for _, name := range listed {
		if n, ok := saveSlotNumber(name); ok {
			used[n] = true
		}
	}
	for n := 0; n < 9998; n++ {
		if used[n] {
			continue
		}
		name := fmt.Sprintf("game%04d.sav", n)
		if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
			continue
		}
		return name
	}
	return ""
}

// saveSlotNumber reads name characters 4..7 as four decimal digits.
func saveSlotNumber(name string) (int, bool) {
	if len(name) < 8 {
		return 0, false
	}
	n := 0
	for _, c := range name[4:8] {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

// isGameSaveName is the original's game*.sav pattern.
func isGameSaveName(name string) bool {
	return len(name) >= len("game.sav") && strings.EqualFold(name[:4], "game") && strings.EqualFold(filepath.Ext(name), originalSaveExt)
}

// anyGameSave reports whether dir holds a file named game*.sav, whatever its
// content (R2-ENGINE-437).
func anyGameSave(dir string) bool {
	if dir == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if isGameSaveName(e.Name()) {
			return true
		}
	}
	return false
}

// ownGameSaves lists the .sav files in dir that are saves of game g, newest
// last write first, by their label. Detail is the label and the last write
// time as the original's row hint formats it (R2-ENGINE-438). The engine's
// second-game saves are Bsg& files; an older one carries the first game's
// Asg& envelope with the game named in its current-session leaf. Its quick
// and timed saves are listed beside game*.sav (DIV-2404, DIV-3079).
func ownGameSaves(dir string, selector int, g base.Game) []SaveEntry {
	var out []SaveEntry
	for _, e := range scanSaves(saveScan{Dir: dir, Bounded: true, Regular: true, OfGame: true, Game: g}) {
		out = append(out, SaveEntry{Name: e.Name, Label: asciiLabel(e.Label, selector), Mod: e.Mod})
	}
	return out
}

// saveRowDetail is the row hint: label, then day, month, year, hour, minute
// and second of the last write time.
func saveRowDetail(label string, mod time.Time) string {
	t := mod.Local()
	return fmt.Sprintf("%s  %02d.%02d.%d  %02d:%02d:%02d", label, t.Day(), int(t.Month()), t.Year(), t.Hour(), t.Minute(), t.Second())
}

// ownSaveRows are the page rows for ownGameSaves; token maps a file name to
// the name the page passes back.
func ownSaveRows(saves []SaveEntry, token func(string) string) []ui.SaveEntry {
	out := make([]ui.SaveEntry, len(saves))
	for i, e := range saves {
		out[i] = ui.SaveEntry{Name: token(e.Name), Label: e.Label, Detail: saveRowDetail(e.Label, e.Mod)}
	}
	return out
}

// savedGame is the installed game, or the first game without archives.
func (f *FrontEnd) savedGame() base.Game {
	if f.Archives == nil {
		return base.Game("")
	}
	return f.Archives.Game()
}

// listOwnSaveDirectory is the save page's list: game g's saves in path by
// bare file name.
func listOwnSaveDirectory(path string, fences []string, selector int, g base.Game, profiles ...*runtimeProfileAccess) (ui.SaveDirectory, error) {
	dir, err := namedSaveDirectory(path, fences, profiles...)
	if err != nil {
		return ui.SaveDirectory{}, err
	}
	return ui.SaveDirectory{Path: dir, Entries: ownSaveRows(ownGameSaves(dir, selector, g), func(name string) string { return name })}, nil
}

// fontText is font-encoded text as the runes a list row and the name box hold,
// so the Empty Slot row draws and edits like a typed label.
func fontText(s string, selector int) string {
	return asciiLabel([]byte(s), selector)
}
