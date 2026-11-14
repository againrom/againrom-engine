package ui

import (
	"errors"
	"fmt"
	"image"
	"slices"
	"strings"

	"againrom/pkg/render/text"
)

// keysDialog is the Keys page's frame (DIV-4939).
var keysDialog = newDialogGeometry(520, 400)

const keysListRows = 12

// keyWaitMark is a waiting row's key column.
const keyWaitMark = "..."

func keysListBox(f *text.Font) listBox {
	return newListBox(keysDialog.Rect(32, 48, keysDialog.W()-56, 48), keysListRows, f)
}

func keysButtonRect(action gameMenuAction) image.Rectangle {
	g := keysDialog
	w, y := g.W(), g.Dy()-60
	if action == gameMenuKeyDefaults {
		return g.Rect(w/7, y, 3*w/7, y+24)
	}
	return g.Rect(4*w/7, y, 6*w/7, y+24)
}

// keysPage is the Keys page: the list of the game's mission actions with
// their keys, Defaults and Return; the page's message wraps under the list.
func keysPage(f *text.Font) settingsPage {
	g := keysDialog
	top := keysListBox(f).Rect.Max.Y + 6
	return settingsPage{dialog: g, titleSpan: g.W(), title: func(f *flow) string { return f.menuWord("keys.title") },
		control: (*App).keysControl, notice: image.Rect(g.Min.X+32, top, g.Max.X-32, top+40)}
}

func (f *flow) keysPageRows() []gameMenuRow {
	return []gameMenuRow{
		{Action: gameMenuKeyList, Enabled: true, Literal: true},
		{Label: f.menuWord("keys.defaults"), Fallback: 'D', Action: gameMenuKeyDefaults, Enabled: true},
		{Label: f.menuWord("keys.return"), Fallback: 'R', Action: gameMenuPageReturn, Enabled: true},
	}
}

func (a *App) keysControl(row gameMenuRow) settingsControl {
	c := settingsControl{kind: settingsPush, rect: keysButtonRect(row.Action), label: row.Label, literal: row.Literal, disabled: !row.Enabled}
	if row.Action == gameMenuKeyList {
		box := keysListBox(a.flow.menuFont)
		c.kind, c.rect, c.list, c.box, c.wheel = settingsList, box.Rect, a.keysList(), box, settingsPageList
		c.text, c.picked = a.keyRowText, a.startKeyCapture
	}
	return c
}

// keysList is the page's list model, one row per listed action.
func (a *App) keysList() *Picker {
	slots := listedMissionKeys(a.missionKeys)
	if a.player.list == nil || a.player.list.Len() != len(slots) {
		rows := make([]PickerRow, len(slots))
		for i, s := range slots {
			rows[i] = PickerRow{Text: s.name()}
		}
		a.player.list = NewPicker(rows).SetWindow(keysListRows)
	}
	return a.player.list
}

// keyRowText is an action's caption with its keys at the row's right end.
func (a *App) keyRowText(row, width int) string {
	f := a.flow
	slots := listedMissionKeys(a.missionKeys)
	if row < 0 || row >= len(slots) {
		return ""
	}
	s := slots[row]
	keys := f.menuWord("keys.none")
	if a.player.capture == s.name() {
		// The prompt stands under the list; the row keeps room for its name.
		keys = keyWaitMark
	} else if k := missionKeyCaptions(a.missionKeys, a.player.keys, s); k != "" {
		keys = k
	}
	font := f.menuFont
	if font == nil {
		return keyActionCaption(f, s) + "  " + keys
	}
	gap := max(font.Advance(" "), 1)
	caption := menuFit(font).Fit(keyActionCaption(f, s), width-font.Advance(keys)-2*gap)
	pad := max((width-font.Advance(caption)-font.Advance(keys))/gap, 1)
	return caption + strings.Repeat(" ", pad) + keys
}

func keyActionCaption(f *flow, s missionKeySlot) string {
	w := f.menuWord("keys.action." + missionKeyActionNames[s.action])
	switch s.action {
	case missionKeyShortcut, missionKeyBindSpell, missionKeyBindItem:
		return fmt.Sprintf(w, s.arg+1)
	case missionKeyGroup:
		return fmt.Sprintf(w, s.arg)
	}
	return w
}

// missionKeyCaptions are slot's keys in layout k: its rows without a held
// modifier, or every row when each holds one.
func missionKeyCaptions(p missionKeyAction, k PlayerMissionKeys, s missionKeySlot) string {
	var plain, all []string
	for _, b := range missionKeyRows(p, k) {
		if b.fixed && !b.listed || b.slot() != s {
			continue
		}
		c := MissionKeyChord{Key: b.key, Ctrl: b.ctrl == modHeld, Shift: b.shift == modHeld, Alt: b.alt == modHeld}.caption()
		if !strings.Contains(c, "+") && !slices.Contains(plain, c) {
			plain = append(plain, c)
		}
		if !slices.Contains(all, c) {
			all = append(all, c)
		}
	}
	if len(plain) == 0 {
		plain = all
	}
	return strings.Join(plain, ", ")
}

// startKeyCapture waits for the selected action's new key.
func (a *App) startKeyCapture() {
	slots := listedMissionKeys(a.missionKeys)
	if i := a.keysList().Selection(); i >= 0 && i < len(slots) {
		a.player.capture, a.flow.msg = slots[i].name(), a.flow.word("keys.press")
	}
}

// capturedByKeysPage is the part of a tick a waiting row leaves to others: the
// pointer, the focus and the modifiers. Every pressed key goes to the row.
func (in appInput) capturedByKeysPage() appInput {
	out := appInput{AnyKey: in.AnyKey, Keys: in.Keys, Close: in.Close, Unfocused: in.Unfocused, Escape: in.Escape,
		CursorX: in.CursorX, CursorY: in.CursorY, PrimaryPressed: in.PrimaryPressed,
		PrimaryReleased: in.PrimaryReleased, SecondaryPressed: in.SecondaryPressed, SecondaryReleased: in.SecondaryReleased}
	out.Viewer.Ctrl, out.Viewer.Shift, out.Viewer.Alt = in.Viewer.Ctrl, in.Viewer.Shift, in.Viewer.Alt
	return out
}

// stepKeyCapture gives the waiting action the next key: Escape or a click
// cancels, a lone modifier waits for its key, a reserved key is refused.
func (a *App) stepKeyCapture(in appInput) {
	p := &a.player
	if in.Escape || in.PrimaryPressed || in.Unfocused {
		p.capture, a.flow.msg = "", ""
		return
	}
	for _, key := range in.Keys {
		if loneModifier(key) {
			continue
		}
		err := a.bindMissionKey(p.capture, MissionKeyChord{Key: key, Ctrl: in.Viewer.Ctrl, Shift: in.Viewer.Shift, Alt: in.Viewer.Alt})
		if errors.Is(err, errMissionKeyFixed) {
			a.flow.msg = a.flow.word("keys.reserved")
			return
		}
		a.flow.msg = ""
		if err != nil {
			a.flow.msg = a.flow.word("keys.not_saved") + err.Error()
		}
		p.capture = ""
		return
	}
}

func (a *App) restoreKeysPage() {
	a.player.capture = ""
	if err := a.restoreMissionKeys(); err != nil {
		a.flow.msg = a.flow.word("keys.not_saved") + err.Error()
	}
}
