package ui

import (
	"errors"
	"image"
	"strings"
)

// ModMismatchReason is why a LOAD refused a save over the mods.
type ModMismatchReason int

const (
	// ModMismatchSet: the save records a mod set and the active one differs.
	ModMismatchSet ModMismatchReason = iota + 1
	// ModMismatchUnmarked: the save records no mods and mods are active.
	ModMismatchUnmarked
	// ModMismatchMark: the save's mod record cannot be read or does not fit
	// the save.
	ModMismatchMark
)

// ModMismatchKind is one way the active mods differ from the save's.
type ModMismatchKind int

const (
	// ModMismatchBase: another base game; Saved and Current are the bases.
	ModMismatchBase ModMismatchKind = iota + 1
	// ModMismatchMissing: a mod the save used is not active; Saved is its
	// version.
	ModMismatchMissing
	// ModMismatchAdded: an active mod the save did not use; Current is its
	// version.
	ModMismatchAdded
	// ModMismatchVersion: the same mod in another version.
	ModMismatchVersion
	// ModMismatchFiles: the same version with other files.
	ModMismatchFiles
	// ModMismatchSettingGone: a saved setting the active mod does not declare.
	ModMismatchSettingGone
	// ModMismatchSetting: a setting with another value.
	ModMismatchSetting
	// ModMismatchSettingNew: an active setting the save does not record.
	ModMismatchSettingNew
	// ModMismatchOrder: the same mods in another load order.
	ModMismatchOrder
)

// ModMismatchRow is one difference: its kind, the mod's id, the setting's key
// where the kind names one, the value the save holds and the value active now.
type ModMismatchRow struct {
	Kind           ModMismatchKind
	Mod, Setting   string
	Saved, Current string
}

// ModMismatch is a LOAD refusal over the mods as data: the reason and every
// difference.
type ModMismatch struct {
	Reason ModMismatchReason
	Rows   []ModMismatchRow
}

// modMismatchError is what a LOAD seam's error answers when it refuses a save
// over the mods. The error's text stays the refusal's prose; the window draws
// the same differences from ModMismatch.
type modMismatchError interface {
	error
	ModMismatch() ModMismatch
}

// modMismatchOf is the mismatch err carries, if any.
func modMismatchOf(err error) (ModMismatch, bool) {
	var m modMismatchError
	if errors.As(err, &m) {
		return m.ModMismatch(), true
	}
	return ModMismatch{}, false
}

// modMismatchWord is the engine word of a row's kind, empty for a kind it
// does not know. In a word, {mod}, {setting}, {saved} and {current} are the
// row's values and the text between [ and ] is the mismatched part, drawn in
// the alert ink.
func (f *flow) modMismatchWord(k ModMismatchKind) string {
	switch k {
	case ModMismatchBase:
		return f.word("load.mods.base")
	case ModMismatchMissing:
		return f.word("load.mods.missing")
	case ModMismatchAdded:
		return f.word("load.mods.added")
	case ModMismatchVersion:
		return f.word("load.mods.version")
	case ModMismatchFiles:
		return f.word("load.mods.files")
	case ModMismatchSettingGone:
		return f.word("load.mods.setting_gone")
	case ModMismatchSetting:
		return f.word("load.mods.setting")
	case ModMismatchSettingNew:
		return f.word("load.mods.setting_new")
	case ModMismatchOrder:
		return f.word("load.mods.order")
	}
	return ""
}

// modMismatchRow fills a row word: the values replace their names, and the
// brackets split the text into plain and alert runs.
func modMismatchRow(word string, r ModMismatchRow) noticeRow {
	values := strings.NewReplacer("{mod}", r.Mod, "{setting}", r.Setting, "{saved}", r.Saved, "{current}", r.Current)
	var out noticeRow
	alert := false
	for word != "" {
		n := strings.IndexAny(word, "[]")
		if n < 0 {
			n = len(word)
		}
		if n > 0 {
			out = append(out, inkRun{Text: values.Replace(word[:n]), Alert: alert})
		}
		if n < len(word) {
			alert = word[n] == '['
			n++
		}
		word = word[n:]
	}
	return out
}

// modMismatchWindow builds the notice window of a mismatch from the engine's
// words: a lead row for a save without a record and for an unreadable record,
// one row per difference, and the hint for what the player can do.
func (f *flow) modMismatchWindow(m ModMismatch) *noticeWindow {
	w := &noticeWindow{Title: f.word("load.mods.title"), OK: f.word("load.mods.ok"), Hint: f.word("load.mods.hint")}
	switch m.Reason {
	case ModMismatchUnmarked:
		w.Rows = append(w.Rows, modMismatchRow(f.word("load.mods.unmarked"), ModMismatchRow{}))
		w.Hint = f.word("load.mods.hint_unmarked")
	case ModMismatchMark:
		w.Rows = append(w.Rows, modMismatchRow(f.word("load.mods.unreadable"), ModMismatchRow{}))
	}
	for _, r := range m.Rows {
		if word := f.modMismatchWord(r.Kind); word != "" {
			w.Rows = append(w.Rows, modMismatchRow(word, r))
		}
	}
	return w
}

// modMismatchDisplay is a notice text as the menu font draws it: a font
// without the install's alphabet shows a rune it cannot draw as '?'.
func (a *App) modMismatchDisplay(s string) string {
	return a.flow.menuDisplayText(a.drawableLoadText(s))
}

// stepModMismatch drives the open mismatch window; Enter and OK close it and
// leave the Load page as it was.
func (a *App) stepModMismatch(v pageView, in appInput) {
	w := v.ui.mismatch
	font := a.flow.menuFont
	if font == nil {
		v.ui.mismatch = nil
		return
	}
	l := newNoticeWindowLayout(font, v.page.Panel)
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if w.step(in, p, ok, l, len(w.noticeLines(font, l, a.modMismatchDisplay))) {
		v.ui.mismatch = nil
		v.ui.resetPointer()
	}
}

// HeadlessModMismatch is the open mismatch window as a scenario reads it: its
// title, each row with its alert runs inside [ ], the hint, the first visible
// line and the wrapped line count.
type HeadlessModMismatch struct {
	Title, Hint string
	Rows        []string
	Top, Lines  int
	// List is the list well's frame rectangle and Visible its line count.
	List    image.Rectangle
	Visible int
}

// HeadlessModMismatch reports the Load page's open mismatch window.
func (a *App) HeadlessModMismatch() (HeadlessModMismatch, bool) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenLoad || a.flow.loadUI.mismatch == nil {
		return HeadlessModMismatch{}, false
	}
	w := a.flow.loadUI.mismatch
	out := HeadlessModMismatch{Title: w.Title, Hint: w.Hint, Top: w.top}
	for _, row := range w.Rows {
		var b strings.Builder
		for _, r := range row {
			if r.Alert {
				b.WriteString("[" + r.Text + "]")
			} else {
				b.WriteString(r.Text)
			}
		}
		out.Rows = append(out.Rows, b.String())
	}
	if font := a.flow.menuFont; font != nil {
		l := newNoticeWindowLayout(font, a.flow.loadPage().Panel)
		out.Lines, out.List, out.Visible = len(w.noticeLines(font, l, a.modMismatchDisplay)), l.List.Rect, l.List.Rows
	}
	return out, true
}
