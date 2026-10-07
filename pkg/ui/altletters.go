package ui

// suppressAltLetters clears the letter-driven map actions of a frame with Alt
// held: Alt plus a letter reaches no map action and the debug console is not
// built (MENU-062, AI-378, DIV-2077).
func (in *appInput) suppressAltLetters() {
	if !in.Viewer.Alt {
		return
	}
	in.Kill, in.Grab, in.Chip, in.Load, in.Numerals = false, false, false, false, false
	in.Attack, in.Autocast, in.ShowHealth, in.Formation, in.Retreat = false, false, false, false, false
	in.TimeFlow, in.Smoothing, in.AutoHealing = false, false, false
	in.Inventory, in.Worn, in.SelectAll = false, false, false
	in.Guard, in.PlayerRetreat, in.StandGround, in.Patrol = false, false, false, false
	in.March, in.Defend, in.Move, in.Cast = false, false, false, false
	in.Book, in.Doll = false, false
	// Alt+Backspace and Alt+F12 arrive as system key messages and reach
	// neither the Backspace nor the F12 arm (MENU-082).
	in.Backspace, in.FPS = false, false
}
