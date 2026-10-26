package menu

// Action is what releasing a main-menu button does. The front end binds each
// action to one screen; ActionNone draws the button and does nothing.
type Action uint8

const (
	ActionNone Action = iota
	ActionNewGame
	ActionLoad
	ActionHallOfFame
	ActionCutscenes
	ActionCredits
	ActionExit
)

// Button is one main-menu button's binding: its action and the scenario word
// that names it, or none.
type Button struct {
	Action Action
	Name   string
}

// Actions is one game's main-menu binding table, indexed by button 1..8 at
// [0..7].
//
// LoadNeedsSave locks every ActionLoad button while the front end reports no
// save file. A locked button shows its hover but a release on it posts
// nothing. EscapeStays makes Escape on the menu do nothing instead of ending
// the program, and HallKeepsEscape makes the Hall of Fame the menu opens
// ignore Escape. CaptionAfterMove shows no button under a resting pointer
// after a return to the menu until the pointer moves.
type Actions struct {
	Buttons          [ButtonCount]Button
	LoadNeedsSave    bool
	EscapeStays      bool
	HallKeepsEscape  bool
	CaptionAfterMove bool
}

// FirstActions binds the first game's six decoded buttons; buttons 2 and 6
// carry no action.
var FirstActions = func() Actions {
	var t Actions
	t.Buttons[NewGameButton-1] = Button{ActionNewGame, "new game"}
	t.Buttons[LoadGameButton-1] = Button{ActionLoad, "load game"}
	t.Buttons[HallOfFameButton-1] = Button{ActionHallOfFame, "hall of fame"}
	t.Buttons[CutscenesButton-1] = Button{ActionCutscenes, "cutscenes"}
	t.Buttons[CreditsButton-1] = Button{ActionCredits, "credits"}
	t.Buttons[ExitButton-1] = Button{ActionExit, "exit"}
	return t
}()

// SecondActions is the second game's table in mask order (R2-ENGINE-436,
// R2-ENGINE-437): button i+1 answers mask value 0x80+0x10*i. Button 2 (0x426,
// unnamed, read as multiplayer) and button 6 (0x487, the web server list)
// belong to multiplayer and carry no action (DIV-2369). Load is locked while
// no game*.sav exists. Escape changes nothing on the menu or on the Hall of
// Fame, and a return to the menu shows no caption until the pointer moves
// (R2-ENGINE-565).
var SecondActions = Actions{
	Buttons: [ButtonCount]Button{
		{ActionNewGame, "new game"},
		{ActionNone, "multi player"},
		{ActionCutscenes, "cutscenes"},
		{ActionCredits, "credits"},
		{ActionLoad, "load game"},
		{ActionNone, "server"},
		{ActionHallOfFame, "hall of fame"},
		{ActionExit, "exit"},
	},
	LoadNeedsSave:    true,
	EscapeStays:      true,
	HallKeepsEscape:  true,
	CaptionAfterMove: true,
}

// Binding is button b's binding, or the zero Button outside 1..8 or without
// assets.
func (a *Assets) Binding(b int) Button {
	if a == nil || b < 1 || b > ButtonCount {
		return Button{}
	}
	return a.table().Buttons[b-1]
}

// table is the loaded table; hand-assembled assets that set none keep the
// first game's bindings, as the menu had before tables existed.
func (a *Assets) table() Actions {
	if a.actions == (Actions{}) {
		return FirstActions
	}
	return a.actions
}

// Bindings is the whole table the assets were loaded with.
func (a *Assets) Bindings() Actions {
	if a == nil {
		return Actions{}
	}
	return a.table()
}

// Locked reports whether button b is locked while the front end reports
// hasSave.
func (a *Assets) Locked(b int, hasSave bool) bool {
	return a != nil && a.table().LoadNeedsSave && !hasSave && a.Binding(b).Action == ActionLoad
}
