package game

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The second game's page geometry. The page constructor's four numbers
// (100, 30, 600, 450) are read as left, top, right and bottom; the caption,
// name box and list origins are the claimed offsets from the page's corner
// (R2-ENGINE-438, R2-ENGINE-440). List width, buttons, message line and the
// confirmation are chosen (DIV-3078).
var (
	secondPagePanel   = image.Rect(100, 30, 600, 450)
	secondPageMessage = image.Rect(140, 372, 560, 406)
	secondPageConfirm = ui.PageConfirm{At: image.Pt(140, 141), Label: image.Rect(140, 141, 560, 220), Inline: true}
)

func secondPageAt(x, y int) image.Point { return secondPagePanel.Min.Add(image.Pt(x, y)) }

func secondPageButtons(ok, del, cancel string) [3]ui.PageButton {
	var out [3]ui.PageButton
	for i, caption := range []string{ok, del, cancel} {
		out[i] = ui.PageButton{Rect: image.Rect(178+i*124, 410, 274+i*124, 434), Caption: caption}
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
		if p, err := OpenContainers(filepath.Join(f.Archives.Root, patchArchive)); err == nil {
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
		return ui.PageText{Text: row(dialogs, i, fallback), At: secondPageAt(x, y)}
	}
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	confirm := secondPageConfirm
	confirm.Before = row(patch, 50, "Delete")
	confirm.After = row(patch, 51, "?")
	ok, del, cancel := row(dialogs, 0, "Ok"), row(dialogs, 157, "Delete"), row(dialogs, 1, "Cancel")
	load := ui.SavePage{
		Panel: secondPagePanel,
		Texts: []ui.PageText{
			text(151, "Load Saved Game", 40, 32),
			text(24, "Load the Game", 40, 68),
			listing(text(25, "Saved games", 40, 92)),
		},
		List: image.Rect(140, 141, 536, 141), Rows: 10,
		Message: secondPageMessage, Buttons: secondPageButtons(ok, del, cancel),
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
			text(29, "Save Game", 40, 32),
			listing(text(30, "Enter a name for the saved game", 40, 50)),
			listing(text(144, "", 40, 104)),
			listing(text(31, "Saved games", 40, 128)),
		},
		List: image.Rect(140, 177, 536, 177), Rows: 10,
		Message: secondPageMessage, Buttons: secondPageButtons(ok, del, cancel),
		Confirm: confirm,
		NameBox: image.Rect(140, 98, 560, 122), NameLimit: limit,
		EmptySlot: fontText(row(dialogs, 28, "<<< Empty Slot >>>"), selector),
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
// saves carry the first game's Asg& envelope with the game named in their
// current-session leaf, and its quick and timed saves are listed beside
// game*.sav (DIV-2404, DIV-3079).
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
