package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"againrom/pkg/namefold"
)

// HeadlessSaveDialog reports the same state the save screen draws.
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
	if a == nil || a.flow == nil || a.flow.screen != ScreenSave || a.flow.saveScreen == nil {
		return HeadlessSaveDialog{}, false
	}
	s := a.flow.saveScreen
	h := HeadlessSaveDialog{Request: s.request, Entries: append([]SaveEntry(nil), s.saves...),
		Confirmation: s.confirming(), Focus: saveControlName(s.focus), Message: a.flow.msg}
	if s.page.directories {
		h.Directories = append([]string(nil), s.directory.Directories...)
	}
	if s.page.reportDeletePath {
		h.DeletePath = s.removePath
	}
	if s.prepared != nil {
		h.Paths = append([]string(nil), s.prepared.Paths...)
		h.Existing = append([]string(nil), s.prepared.Existing...)
	}
	return h, true
}

func (s *savePageState) errorName() string {
	if s.page.errorName != "" {
		return s.page.errorName
	}
	return "save page"
}

func (a *App) HeadlessSaveEdit(directory, name string, format SaveFormat) error {
	if h, ok := a.HeadlessSaveState(); !ok || h.Confirmation {
		return saveDialogError("edit")
	}
	if format != "" && !strings.EqualFold(string(format), string(SaveSAV)) {
		return fmt.Errorf("save dialog: format %q is unavailable", format)
	}
	s := a.flow.saveScreen
	if directory != "" {
		if s.page.DirectoryBox.Empty() {
			if !sameSavePath(directory, s.request.Directory) {
				return fmt.Errorf("save page: it writes only to %s", s.request.Directory)
			}
		} else {
			s.setFocus(saveDirectoryControl)
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
	}
	if name != "" {
		if s.page.keyboardFocus {
			s.setFocus(saveNameControl)
		}
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
	s := a.flow.saveScreen
	var c saveControl
	normalized := strings.ToLower(strings.TrimSpace(action))
	switch normalized {
	case "save":
		if h.Confirmation {
			if s.page.overwriteQuestion {
				return fmt.Errorf("save dialog: existing targets require overwrite confirmation")
			}
			return saveDialogError(action)
		}
		if !s.page.keyboardFocus && a.flow.saveControlDisabled(s, saveWriteControl) {
			return fmt.Errorf("save page: Ok is disabled")
		}
		c = saveWriteControl
	case "overwrite":
		if !s.page.overwriteQuestion {
			return s.unknownAction(action)
		}
		if h.DeletePath != "" {
			return saveDialogError(action)
		}
		fallthrough
	case "confirm":
		if !h.Confirmation {
			return saveDialogError(action)
		}
		c = saveWriteControl
		if s.ui.remove != nil {
			c = saveDeleteControl
		}
	case "delete":
		if h.Confirmation || !a.flow.canDeleteSave() {
			return saveDialogError(action)
		}
		c = saveDeleteControl
	case "back", "cancel":
		c = saveCancelControl
	case "up", "open":
		if s.page.DirectoryBox.Empty() {
			return s.unknownAction(action)
		}
		if h.Confirmation {
			return saveDialogError(action)
		}
		c = saveOpenControl
		if normalized == "up" {
			c = saveUpControl
		}
	case "select-all":
		return a.HeadlessKey("ctrl-a")
	default:
		return s.unknownAction(action)
	}
	if !s.page.keyboardFocus && c == saveWriteControl {
		a.acceptPage(s.view())
	} else {
		a.flow.activateSaveControl(c)
	}
	if a.flow.screen == ScreenSave && a.flow.msg != "" {
		return fmt.Errorf("%s: %s", s.errorName(), a.flow.msg)
	}
	return nil
}

func (s *savePageState) pageActions() string {
	actions := "save, "
	if s.page.overwriteQuestion {
		actions += "overwrite, "
	}
	actions += "confirm, delete, back, cancel, "
	if !s.page.DirectoryBox.Empty() {
		actions += "up, open, "
	}
	return actions + "select-all"
}

func (a *App) HeadlessSaveSelect(index int) error {
	if h, ok := a.HeadlessSaveState(); !ok || h.Confirmation {
		return saveDialogError("select")
	}
	s := a.flow.saveScreen
	if s.list == nil || !s.list.Select(index) {
		return fmt.Errorf("%s: row %d is unavailable", s.errorName(), index)
	}
	if s.page.activateRows {
		s.setFocus(saveListControl)
		a.flow.activateSaveControl(saveListControl)
		if a.flow.msg != "" {
			return fmt.Errorf("save directory: %s", a.flow.msg)
		}
	} else {
		s.pick()
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

func sameSavePath(a, b string) bool {
	return namefold.HostEqual(filepath.Clean(a), filepath.Clean(b))
}

func (s *savePageState) unknownAction(action string) error {
	return fmt.Errorf("%s: unknown action %q; accepted actions: %s", s.errorName(), action, s.pageActions())
}
