package ui

import (
	"errors"
	"fmt"
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
	// ModMismatchChanged: a town save loaded although its mods changed their
	// files; the rows are what the active mods no longer hold.
	ModMismatchChanged
	// ModMismatchKeyless: a town save whose mods changed their files holds mod
	// items, and was written before saves named them; the rows are the
	// differences.
	ModMismatchKeyless
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
	// ModMismatchItemPack: Who's pack lost item What of mod Mod, which the
	// mods no longer define.
	ModMismatchItemPack
	// ModMismatchItemWorn: Who wore item What of mod Mod, which the mods no
	// longer define; the slot is empty.
	ModMismatchItemWorn
	// ModMismatchLayer: Who wore clothing layer What of mod Mod, which the
	// mods no longer define.
	ModMismatchLayer
	// ModMismatchSkill: Who's skill What fell from Saved to the cap Current.
	ModMismatchSkill
	// ModMismatchGuest: the guest waiting for mission What of mod Mod no
	// longer waits.
	ModMismatchGuest
	// ModMismatchMission: campaign mission What is no longer declared.
	ModMismatchMission
)

// ModMismatchRow is one difference: its kind, the mod's id, the setting's key
// where the kind names one, the value the save holds and the value active now.
// A row of a changed mods' LOAD names Who held what was lost and What it was;
// a skill's What is its key under load.mods.skill.
type ModMismatchRow struct {
	Kind           ModMismatchKind
	Mod, Setting   string
	Saved, Current string
	Who, What      string
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

// ModLoadNotice is what a LOAD seam answers in place of a nil error when the
// save loaded and the active mods took something from it: the load is
// complete, and Changes (reason ModMismatchChanged) lists what went.
type ModLoadNotice struct{ Changes ModMismatch }

func (n *ModLoadNotice) Error() string {
	return fmt.Sprintf("loaded under changed mods; %d removed or lowered", len(n.Changes.Rows))
}

// loadNoticeOf splits a LOAD seam's answer: the list a completed load
// carries, and the error that remains, nil for that load.
func loadNoticeOf(err error) (*ModMismatch, error) {
	var n *ModLoadNotice
	if errors.As(err, &n) {
		return &n.Changes, nil
	}
	return nil, err
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
	case ModMismatchItemPack:
		return f.word("load.mods.item_pack")
	case ModMismatchItemWorn:
		return f.word("load.mods.item_worn")
	case ModMismatchLayer:
		return f.word("load.mods.layer")
	case ModMismatchSkill:
		return f.word("load.mods.skill")
	case ModMismatchGuest:
		return f.word("load.mods.guest")
	case ModMismatchMission:
		return f.word("load.mods.mission")
	}
	return ""
}

// modMismatchRow fills a row word: the values replace their names, and the
// brackets split the text into plain and alert runs.
func modMismatchRow(word string, r ModMismatchRow) noticeRow {
	values := strings.NewReplacer("{mod}", r.Mod, "{setting}", r.Setting, "{saved}", r.Saved, "{current}", r.Current,
		"{who}", r.Who, "{what}", r.What)
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
// one row per difference, and the hint for what the player can do. A load
// under changed mods has its own title and hint over the rows of what went.
func (f *flow) modMismatchWindow(m ModMismatch) *dialogRun {
	w := &dialogRun{}
	w.Title, w.Answers, w.Hint = f.word("load.mods.title"), []string{f.word("load.mods.ok")}, f.word("load.mods.hint")
	switch m.Reason {
	case ModMismatchUnmarked:
		w.Runs = append(w.Runs, modMismatchRow(f.word("load.mods.unmarked"), ModMismatchRow{}))
		w.Hint = f.word("load.mods.hint_unmarked")
	case ModMismatchMark:
		w.Runs = append(w.Runs, modMismatchRow(f.word("load.mods.unreadable"), ModMismatchRow{}))
	case ModMismatchKeyless:
		w.Runs = append(w.Runs, modMismatchRow(f.word("load.mods.keyless"), ModMismatchRow{}))
	case ModMismatchChanged:
		w.Title, w.Hint = f.word("load.mods.changed_title"), f.word("load.mods.changed_hint")
	}
	for _, r := range m.Rows {
		if r.Kind == ModMismatchSkill {
			r.What = f.word("load.mods.skill." + r.What)
		}
		if word := f.modMismatchWord(r.Kind); word != "" {
			w.Runs = append(w.Runs, modMismatchRow(word, r))
		}
	}
	return w
}

// modMismatchText is a window's rows as plain lines, for a message line.
func (w *dialogRun) modMismatchText() string {
	var lines []string
	for _, row := range w.Runs {
		var b strings.Builder
		for _, r := range row {
			b.WriteString(r.Text)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "; ")
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
		a.flow.closeModMismatch()
		return
	}
	l := w.layout(font, v.page.Panel)
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	if _, closed := w.step(in, p, ok, l, len(w.noticeLines(font, l, a.modMismatchDisplay))); closed {
		a.flow.closeModMismatch()
	}
}

// closeModMismatch closes the Load page's mod window. A window over a
// completed town load enters that town, as the load would have at once.
func (f *flow) closeModMismatch() {
	town := f.loadUI.mismatchTown
	f.loadUI.mismatch, f.loadUI.mismatchTown = nil, false
	f.loadUI.resetPointer()
	if town {
		if err := f.loadPrepared(nil, true); err != nil {
			f.msg = err.Error()
		}
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
	for _, row := range w.Runs {
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
		l := w.layout(font, a.flow.loadPage().Panel)
		out.Lines, out.List, out.Visible = len(w.noticeLines(font, l, a.modMismatchDisplay)), l.List.Rect, l.List.Rows
	}
	return out, true
}
