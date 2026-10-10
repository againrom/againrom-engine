package ui

import "againrom/pkg/words"

// SaveDialogWords is this build's save-dialog vocabulary, in UTF-8. Every
// string crosses menuDisplayText before reaching the installed byte-indexed
// font.
type SaveDialogWords struct {
	Title, Directory, Name, Open, Up, Save, Cancel, Back    string
	Overwrite, Confirm, Replace, Create, Empty, Unavailable string
	Delete, DeleteConfirm                                   string
	TownDetail, MapSAVDetail, ScrollHint, KeyHint           string
	LatinNameHint                                           string
}

// saveDialogWords is the save dialog's vocabulary from the engine words b.
func saveDialogWords(b words.Book) SaveDialogWords {
	return SaveDialogWords{
		Title: b.Text("save.title"), Directory: b.Text("save.directory"), Name: b.Text("save.name"),
		Open: b.Text("save.open"), Up: b.Text("save.up"), Save: b.Text("save.save"),
		Cancel: b.Text("save.cancel"), Back: b.Text("save.back"),
		Overwrite: b.Text("save.overwrite"), Confirm: b.Text("save.confirm"),
		Replace: b.Text("save.replace"), Create: b.Text("save.create"), Empty: b.Text("save.empty"),
		Unavailable: b.Text("save.unavailable"),
		Delete:      b.Text("save.delete"), DeleteConfirm: b.Text("save.delete_confirm"),
		TownDetail:   b.Text("save.town_detail"),
		MapSAVDetail: b.Text("save.map_sav_detail"),
		ScrollHint:   b.Text("save.scroll_hint"), KeyHint: b.Text("save.key_hint"),
		LatinNameHint: b.Text("save.latin_name_hint"),
	}
}

func (f *flow) saveWords() SaveDialogWords {
	w := f.words.SaveDialog
	if w == (SaveDialogWords{}) || w == saveDialogWords(words.Book{}) {
		w = saveDialogWords(f.words.Engine)
	}
	if d := f.saveDialog; d != nil && d.request.Format == SaveSAV && w.LatinNameHint != "" {
		w.KeyHint = w.LatinNameHint + "\n" + w.KeyHint
	}
	return w
}
