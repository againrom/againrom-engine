package ui

import (
	_ "embed"
	"encoding/json"

	"againrom/pkg/render/text"
)

// Authored localization is data. Every string crosses menuDisplayText before
// reaching the installed byte-indexed font.
//
//go:embed save_dialog_ru.json
var saveDialogRussianJSON []byte

var saveDialogRussian = func() SaveDialogWords {
	var words SaveDialogWords
	if err := json.Unmarshal(saveDialogRussianJSON, &words); err != nil {
		panic(err)
	}
	return words
}()

// SaveDialogWords is this build's save-dialog vocabulary, in UTF-8.
type SaveDialogWords struct {
	Title, Directory, Name, Open, Up, Save, Cancel, Back    string
	Overwrite, Confirm, Replace, Create, Empty, Unavailable string
	Delete, DeleteConfirm                                   string
	TownDetail, MapSAVDetail, ScrollHint, KeyHint           string
	LatinNameHint                                           string
}

func authoredSaveDialogWords(russian bool) SaveDialogWords {
	if russian {
		return saveDialogRussian
	}
	return SaveDialogWords{
		Title: "Save game", Directory: "Folder", Name: "Name",
		Open: "Open", Up: "Up", Save: "Save", Cancel: "Cancel", Back: "Back",
		Overwrite: "Replace", Confirm: "Replace existing files?",
		Replace: "Replace:", Create: "Create:", Empty: "No saves here",
		Unavailable: "Saving is unavailable",
		Delete:      "Delete", DeleteConfirm: "Delete selected saved game?",
		TownDetail:   "Resume in this town.",
		MapSAVDetail: "Resume this map at the saved moment.",
		ScrollHint:   "Up / down: scroll", KeyHint: "Tab: focus   Enter: choose   Esc: cancel",
		LatinNameHint: "Use characters supported by the game font.",
	}
}

func (f *flow) saveWords() SaveDialogWords {
	w := f.words.SaveDialog
	if w == (SaveDialogWords{}) || w == authoredSaveDialogWords(false) {
		w = authoredSaveDialogWords(f.menuFont != nil && f.menuFont.Selector == text.SelectorConverting)
	}
	if d := f.saveDialog; d != nil && d.request.Format == SaveSAV && w.LatinNameHint != "" {
		w.KeyHint = w.LatinNameHint + "\n" + w.KeyHint
	}
	return w
}
