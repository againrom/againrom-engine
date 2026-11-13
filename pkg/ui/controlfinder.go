package ui

import "image"

// ControlRole names one kind of control of the map editor.
type ControlRole int

// The roles FindControl resolves, one row of editorControlRows each.
const (
	ControlPopupButton ControlRole = iota
	ControlField
	ControlRowButton
	ControlPromptButton
	ControlBar
	ControlDropdownRow
	ControlToolbar
	ControlPopupRow
	ControlPlaceGridCell
	ControlListButton
	ControlTitle
	ControlListButtonKey
	ControlGroupsRow
	ControlListRow
	ControlObjectSection
	ControlViewSwitch
	ControlPanelButton
	ControlMenuRow
	ControlGoalOpen
	ControlAreaLabel
	ControlSelectionButton
	ControlTreeRow
	ControlGroupsButton
	ControlGroupMember
	ControlPromptListRow
	ControlPanelRow
	ControlAreaTrigger
	ControlGroupDelete
	ControlPlaceGridSwitch
	ControlFieldChoice
	// ControlMapTab is tab N of the open maps' strip; with Flag, the
	// strip's close button.
	ControlMapTab
	controlRoleCount
)

// ControlQuery names one control: its role and, as the role needs, its row or
// button numbers, its key, or its caption.
type ControlQuery struct {
	Role    ControlRole
	Caption string
	Key     string
	N, K    int
	Flag    bool
}

type editorControlRow func(e *MapEditor, q ControlQuery) (image.Rectangle, bool)

// editorControlRows is the one table from a control role to its rectangle.
var editorControlRows [controlRoleCount]editorControlRow

func init() {
	editorControlRows = [controlRoleCount]editorControlRow{
		ControlPopupButton: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPopupButtonRect(q.N) },
		ControlField:       func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlFieldRect(q.N) },
		ControlRowButton: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) {
			return e.controlRowButtonRect(q.N, q.K, q.Flag)
		},
		ControlPromptButton:  func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPromptButtonRect(q.N) },
		ControlBar:           func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlBarRect(q.Key) },
		ControlDropdownRow:   func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlDropdownRowRect(q.N) },
		ControlToolbar:       func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlToolbarRect(q.Key) },
		ControlPopupRow:      func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPopupRowRect(q.N) },
		ControlPlaceGridCell: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPlaceGridCellRect(q.N) },
		ControlListButton:    func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlListButtonRect(q.N, q.K) },
		ControlTitle:         func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlTitleRect() },
		ControlListButtonKey: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlListButtonKeyRect(q.Key) },
		ControlGroupsRow: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) {
			return e.controlGroupsRowRect(q.N, q.K, q.Flag)
		},
		ControlListRow:         func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlListRowRect(q.N) },
		ControlObjectSection:   func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlObjectSectionRect(q.Key) },
		ControlViewSwitch:      func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlViewSwitchRect() },
		ControlPanelButton:     func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPanelButtonRect(q.N) },
		ControlMenuRow:         func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlMenuRowRect(q.N) },
		ControlGoalOpen:        func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlGoalOpenRect(q.N) },
		ControlAreaLabel:       func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlAreaLabelRect(q.N) },
		ControlSelectionButton: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlSelectionButtonRect(q.N) },
		ControlTreeRow:         func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlTreeRowRect(q.N) },
		ControlGroupsButton:    func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlGroupsButtonRect(q.N) },
		ControlGroupMember:     func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlGroupMemberRect(q.N) },
		ControlPromptListRow:   func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPromptListRowRect(q.N) },
		ControlPanelRow:        func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPanelRowRect(q.N) },
		ControlAreaTrigger:     func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlAreaTriggerRect() },
		ControlGroupDelete:     func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlGroupDeleteRect() },
		ControlPlaceGridSwitch: func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlPlaceGridSwitchRect(q.N) },
		ControlFieldChoice:     func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlFieldChoiceRect(q.N, q.K) },
		ControlMapTab:          func(e *MapEditor, q ControlQuery) (image.Rectangle, bool) { return e.controlMapTabRect(q.N, q.Flag) },
	}
}

// FindControl is the one lookup of a map editor control by role or caption. A
// query with a caption stands for the button of that caption in the prompt or
// the object popup's button row; a role with no such button finds nothing.
func (e *MapEditor) FindControl(q ControlQuery) (image.Rectangle, bool) {
	if q.Role < 0 || q.Role >= controlRoleCount {
		return image.Rectangle{}, false
	}
	if q.Caption != "" {
		n, ok := e.captionButton(q.Role, q.Caption)
		if !ok {
			return image.Rectangle{}, false
		}
		q.N = n
	}
	return editorControlRows[q.Role](e, q)
}

// captionButton is the button number of a caption among a role's buttons.
func (e *MapEditor) captionButton(role ControlRole, caption string) (int, bool) {
	var captions []string
	switch role {
	case ControlPopupButton:
		captions = e.PopupButtons()
	case ControlPromptButton:
		if s, ok := e.promptSpec(); ok {
			captions = s.Buttons
		}
	}
	for n, c := range captions {
		if c == caption {
			return n, true
		}
	}
	return 0, false
}
