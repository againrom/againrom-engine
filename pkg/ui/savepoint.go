package ui

import "time"

// saveKind is a place in this package that asks whether the game may be
// saved now.
type saveKind uint8

const (
	saveMenuAction  saveKind = iota // the game menu's Save action
	saveMenuWindow                  // the save dialog or save page opening
	saveMenuRow                     // whether the town menu's Save row is enabled
	saveKey                         // the Save function key
	saveKeyWithLoad                 // the Save function key pressed with the Load key
	saveLoadKey                     // the Load function key
	saveQuickKey                    // the quick save and quick load keys
	saveAutosave                    // the timed autosave poll
	saveKinds
)

// saveAnswer is what one gate row does to one save kind.
type saveAnswer uint8

const (
	answerPass   saveAnswer = iota // the row does not close this kind
	answerSilent                   // closed; nothing is shown
	answerAsk                      // closed; the save dialog admission's reason is shown
)

// savePointRow names the gate row that closed a save point.
type savePointRow uint8

const (
	pointOpen savePointRow = iota
	pointNoScreen
	pointNoViewer
	pointNoTown
	pointMapPopup
	pointMapNotice
	pointTownDialogue
	pointMissionShop
	pointTownCannotSave
	pointNotCampaign
)

// saveGate is a save point's answer: open, or the row that closed it and
// whether that row showed a reason. The zero value is open.
type saveGate struct {
	row   savePointRow
	shown bool
}

func (g saveGate) open() bool { return g.row == pointOpen }

// saveGateRow closes the kinds it answers for when closed holds at the save
// point's screen.
type saveGateRow struct {
	row    savePointRow
	closed func(f *flow, at Screen) bool
	answer [saveKinds]saveAnswer
}

// noSavePlace is the screen of a menu save asked while no game menu is open.
const noSavePlace Screen = -1

var keyKindsSilent = [saveKinds]saveAnswer{saveKey: answerSilent, saveKeyWithLoad: answerSilent, saveLoadKey: answerSilent, saveQuickKey: answerSilent}

func everyKind(a saveAnswer) (answers [saveKinds]saveAnswer) {
	for k := range answers {
		answers[k] = a
	}
	return answers
}

// saveGateRows are the save point gates in the order they are tested. The
// first row that answers for the kind and holds closes the point.
var saveGateRows = []saveGateRow{
	{pointNoScreen, func(f *flow, at Screen) bool { return at != ScreenMap && at != ScreenTown }, everyKind(answerSilent)},
	{pointNoViewer, func(f *flow, at Screen) bool { return at == ScreenMap && f.viewer == nil },
		[saveKinds]saveAnswer{saveKey: answerSilent, saveKeyWithLoad: answerSilent, saveLoadKey: answerSilent, saveQuickKey: answerSilent, saveAutosave: answerSilent}},
	{pointNoTown, func(f *flow, at Screen) bool { return at == ScreenTown && f.town == nil },
		[saveKinds]saveAnswer{saveAutosave: answerSilent}},
	{pointMapPopup, func(f *flow, at Screen) bool { return at == ScreenMap && f.viewer.popupOpen() },
		[saveKinds]saveAnswer{saveKey: answerAsk, saveKeyWithLoad: answerAsk, saveLoadKey: answerSilent, saveQuickKey: answerSilent}},
	{pointMapNotice, func(f *flow, at Screen) bool { return at == ScreenMap && f.viewer.NoticeOpen() },
		[saveKinds]saveAnswer{saveAutosave: answerSilent}},
	{pointTownDialogue, func(f *flow, at Screen) bool {
		if at != ScreenTown {
			return false
		}
		_, open := townDialogue(f.town)
		return open
	}, [saveKinds]saveAnswer{saveKey: answerAsk, saveKeyWithLoad: answerAsk, saveLoadKey: answerSilent, saveQuickKey: answerSilent, saveAutosave: answerSilent}},
	{pointMissionShop, func(f *flow, at Screen) bool { return at == ScreenTown && missionShopOpen(f.town) }, keyKindsSilent},
	{pointTownCannotSave, func(f *flow, at Screen) bool { return at == ScreenTown && !townCanSave(f.town) },
		[saveKinds]saveAnswer{saveMenuAction: answerAsk, saveMenuWindow: answerSilent, saveMenuRow: answerSilent, saveKey: answerAsk,
			saveKeyWithLoad: answerSilent, saveQuickKey: answerSilent, saveAutosave: answerSilent}},
	{pointNotCampaign, func(f *flow, at Screen) bool { return at == ScreenMap && !f.viewer.readGameMenuContext().Campaign }, keyKindsSilent},
}

// savePlace is the screen whose save point kind k asks about: the screen the
// game menu was opened over for the menu's own kinds, the town for the town
// menu's Save row, and the open screen for the keys and the autosave.
func (f *flow) savePlace(k saveKind) Screen {
	switch k {
	case saveMenuAction, saveMenuWindow:
		if f.screen != ScreenGameMenu {
			return noSavePlace
		}
		return f.menuBack
	case saveMenuRow:
		return ScreenTown
	}
	return f.screen
}

// savePoint is the one gate for whether the game may be saved now. A row
// that asks shows the save dialog admission's reason; a silent row shows
// nothing.
func (f *flow) savePoint(k saveKind) saveGate {
	at := f.savePlace(k)
	for i := range saveGateRows {
		r := &saveGateRows[i]
		answer := r.answer[k]
		if answer == answerPass || !r.closed(f, at) {
			continue
		}
		g := saveGate{row: r.row}
		if answer == answerAsk {
			g.shown = f.showSaveRefusal(at == ScreenMap)
		}
		return g
	}
	return saveGate{}
}

func townCanSave(t TownScreen) bool {
	if s, ok := t.(interface{ CanSave() bool }); ok {
		return s.CanSave()
	}
	return true
}

// showSaveRefusal puts the save dialog admission's reason for a closed save
// point on the map's message line, else the screen's message, and reports
// whether there was one.
func (f *flow) showSaveRefusal(onMap bool) bool {
	if f.saveDialogSeams.Prepare == nil {
		return false
	}
	_, err := f.saveDialogSeams.Prepare(SaveRequest{OnMap: onMap, Admission: true})
	if err == nil {
		return false
	}
	message := f.menuText(err.Error())
	if f.screen == ScreenMap && f.viewer != nil {
		f.viewer.PostMessage(message, MessageWhite, 15*time.Second)
	} else {
		f.msg, f.saveRefused = message, message
	}
	return true
}
