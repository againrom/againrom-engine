package ui

import (
	"fmt"
	"strings"
)

// HeadlessSaveDialog is the visible dialog state. Paths and Existing are exact
// backend targets, including an occupied target that has no readable save row.
type HeadlessSaveDialog struct {
	Request      SaveRequest `json:"request"`
	Directories  []string    `json:"directories,omitempty"`
	Entries      []SaveEntry `json:"entries,omitempty"`
	Paths        []string    `json:"paths,omitempty"`
	Existing     []string    `json:"existing,omitempty"`
	Confirmation bool        `json:"confirmation"`
	DeletePath   string      `json:"delete_path,omitempty"`
	Focus        string      `json:"focus"`
	Message      string      `json:"message,omitempty"`
}

func (a *App) HeadlessSaveState() (HeadlessSaveDialog, bool) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenSave || a.flow.saveDialog == nil {
		return HeadlessSaveDialog{}, false
	}
	d := a.flow.saveDialog
	h := HeadlessSaveDialog{
		Request: d.request, Directories: append([]string(nil), d.directory.Directories...),
		Entries:      append([]SaveEntry(nil), d.directory.Entries...),
		Confirmation: d.prepared != nil || d.remove != nil, DeletePath: d.removePath,
		Focus: saveControlName(d.focus), Message: a.flow.msg,
	}
	if d.prepared != nil {
		h.Paths = append([]string(nil), d.prepared.Paths...)
		h.Existing = append([]string(nil), d.prepared.Existing...)
	}
	return h, true
}

// HeadlessSaveEdit edits through the same text and control handlers as the
// window. Empty arguments retain that field's current value.
func (a *App) HeadlessSaveEdit(directory, name string, format SaveFormat) error {
	if h, ok := a.HeadlessSaveState(); !ok || h.Confirmation {
		return saveDialogError("edit")
	}
	if format != "" && !strings.EqualFold(string(format), string(SaveSAV)) {
		return fmt.Errorf("save dialog: format %q is unavailable", format)
	}
	d := a.flow.saveDialog
	if directory != "" {
		d.setFocus(saveDirectoryControl)
		if err := a.HeadlessKey("ctrl-a"); err != nil {
			return err
		}
		if err := a.HeadlessType(directory, false); err != nil {
			return err
		}
		a.flow.activateSaveControl(saveOpenControl)
		if a.flow.msg != "" {
			return fmt.Errorf("save directory: %s", a.flow.msg)
		}
	}
	if name != "" {
		d.setFocus(saveNameControl)
		if err := a.HeadlessKey("ctrl-a"); err != nil {
			return err
		}
		if err := a.HeadlessType(name, false); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) HeadlessSaveAction(action string) error {
	h, ok := a.HeadlessSaveState()
	if !ok {
		return saveDialogError(action)
	}
	var c saveControl
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "save":
		if h.Confirmation {
			return fmt.Errorf("save dialog: existing targets require overwrite confirmation")
		}
		c = saveWriteControl
	case "overwrite":
		if h.DeletePath != "" {
			return saveDialogError(action)
		}
		fallthrough
	case "confirm":
		if !h.Confirmation {
			return saveDialogError(action)
		}
		c = saveWriteControl
		if h.DeletePath != "" {
			c = saveDeleteControl
		}
	case "delete":
		if h.Confirmation || !a.flow.canDeleteSave() {
			return saveDialogError(action)
		}
		c = saveDeleteControl
	case "back", "cancel":
		c = saveCancelControl
	case "up":
		c = saveUpControl
	case "open":
		c = saveOpenControl
	case "select-all":
		return a.HeadlessKey("ctrl-a")
	default:
		return fmt.Errorf("save dialog: unknown action %q", action)
	}
	if h.Confirmation && c != saveWriteControl && c != saveDeleteControl && c != saveCancelControl {
		return saveDialogError(action)
	}
	a.flow.activateSaveControl(c)
	if a.flow.screen == ScreenSave && a.flow.msg != "" {
		return fmt.Errorf("save dialog: %s", a.flow.msg)
	}
	return nil
}

// HeadlessSaveSelect activates one visible browser entry by its list index.
// Directories precede saves, as they do in HeadlessRows and on the screen.
func (a *App) HeadlessSaveSelect(index int) error {
	if h, ok := a.HeadlessSaveState(); !ok || h.Confirmation {
		return saveDialogError("select")
	}
	d := a.flow.saveDialog
	if d.list == nil || !d.list.Select(index) {
		return fmt.Errorf("save dialog: row %d is unavailable", index)
	}
	d.setFocus(saveListControl)
	a.flow.activateSaveControl(saveListControl)
	if a.flow.msg != "" {
		return fmt.Errorf("save directory: %s", a.flow.msg)
	}
	return nil
}

func saveControlName(c saveControl) string {
	switch c {
	case saveDirectoryControl:
		return "directory"
	case saveOpenControl:
		return "open"
	case saveUpControl:
		return "up"
	case saveListControl:
		return "list"
	case saveNameControl:
		return "name"
	case saveDeleteControl:
		return "delete"
	case saveWriteControl:
		return "save"
	case saveCancelControl:
		return "cancel"
	}
	return ""
}
