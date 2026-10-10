package ui

import (
	"fmt"
	"strings"

	"againrom/pkg/render/text"
)

// SaveGame is the legacy automatic-save seam. Applications configuring only
// SetSaveSeams retain it; the runtime opts into SaveDialogSeams. The bool names
// the screen behind the menu, leaving persistence to the game package.
type SaveGame func(onMap bool) (name string, err error)

// SaveEntry holds the disk token and the label from its header.
type SaveEntry struct {
	Name  string
	Label string
}

// SaveList is what is on disk, newest first.
//
// IT ANSWERS A SLICE AND NOT A SLICE AND AN ERROR. A store that cannot be read
// is an empty list here, and the load window says so with its own header — a
// second failure channel would give this screen two ways to show nothing and
// nothing to distinguish them by.
type SaveList func() []SaveEntry

// LoadGame loads the named save into whatever is behind the seam and reports
// where the player lands: a map opener when the save was taken in a mission,
// or town when it was taken between missions.
//
// THREE RETURNS, ALL IN THIS PACKAGE'S OWN VOCABULARY. A MapOpener is the shape
// four doors here already accept, a bool is a bool, and an error is a sentence.
// Nothing that crosses names a world, a mission, a party or a town model.
//
// town IS AN EXPLICIT BOOL AND NOT A NIL OPENER. "The save was taken in the
// town" and "the save could not produce an opener" are two different answers
// and a caller that had to tell them apart by a nil would get it wrong once.
type LoadGame func(name string) (open MapOpener, town bool, err error)

// menuRows is the open surface's row list, with every disable already
// resolved.
//
// THE PREDICATES WERE ANSWERED AT OPEN TIME AND ARE NOT RE-ASKED HERE, which is
// openLoad's own rule and for its reason: they read the save store, and a row
// that changed under the player's hand between the frame he aimed at and the
// frame he clicked would be picked wrong once. What this rebuilds per call is
// the row VALUES, which are constants of the surface and the two flags.
//
// SAVE IS GATED ON A STORE EXISTING AND LOAD ON THE STORE HOLDING SOMETHING
// (spec AU-7). The original gates them on an undecoded mode word and on a
// directory search for `game*.sav`; the shape — a row refused before it is
// picked rather than a failure reported after — is the same.
func (f *flow) menuRows() []gameMenuRow {
	switch f.menuPage {
	case gameMenuGameOptionsPage:
		return f.gameOptionsRows()
	case gameMenuSoundOptionsPage:
		if f.soundOptions.Read != nil {
			return f.soundOptionRows()
		}
		enabled, volume, available := f.readMenuSound()
		if !available {
			return []gameMenuRow{literalMenuRow("SOUND DEVICE UNAVAILABLE"), pageReturnRow()}
		}
		state := "OFF"
		if enabled {
			state = "ON"
		}
		return []gameMenuRow{
			{Label: "~SOUND: " + state, Fallback: 'S', Action: gameMenuToggleSound, Enabled: f.setMenuSound != nil},
			{Label: fmt.Sprintf("VOLUME ~DOWN: %d", volume), Fallback: 'D', Action: gameMenuVolumeDown, Enabled: f.setMenuSound != nil && volume > 0},
			{Label: fmt.Sprintf("VOLUME ~UP: %d", volume), Fallback: 'U', Action: gameMenuVolumeUp, Enabled: f.setMenuSound != nil && volume < 100},
			{Label: "~TEST SOUND", Fallback: 'T', Action: gameMenuTestSound, Enabled: true},
			pageReturnRow(),
		}
	case gameMenuQuestObjectivesPage:
		return f.questRows()
	case gameMenuDiplomacyPage:
		rows := []gameMenuRow{literalMenuRow("DIPLOMACY")}
		if len(f.menuContext.Relations) == 0 {
			rows = append(rows, literalMenuRow("NO OTHER SIDES ARE PRESENT."))
		} else {
			for _, relation := range f.menuContext.Relations {
				rows = append(rows, literalMenuRow(fmt.Sprintf("SIDE %d: %s", relation.Slot, relation.State)))
			}
		}
		return append(rows, pageReturnRow())
	case gameMenuEndQuestConfirmation:
		if f.menuContext.CampaignVictory {
			return []gameMenuRow{
				{Label: f.words.MenuVictory, Fallback: 'V', Action: gameMenuVictory, Enabled: f.menuContext.VictoryAvailable},
				{Label: f.words.MenuExitMain, Fallback: 'E', Action: gameMenuExitMain, Enabled: true},
				{Label: f.words.MenuExitWindows, Fallback: 'W', Action: gameMenuExitWindows, Enabled: true},
				{Label: f.words.MenuReturn, Fallback: 'R', Action: gameMenuPageReturn, Enabled: true},
			}
		}
		return []gameMenuRow{
			{Label: f.words.MenuChangeMap, Fallback: 'C', Action: gameMenuConfirmEndQuest, Enabled: true},
			{Label: f.words.MenuExitMain, Fallback: 'E', Action: gameMenuExitMain, Enabled: true},
			{Label: f.words.MenuExitWindows, Fallback: 'W', Action: gameMenuExitWindows, Enabled: true},
			{Label: f.words.MenuReturn, Fallback: 'R', Action: gameMenuPageReturn, Enabled: true},
		}
	case gameMenuAbortGameConfirmation:
		return []gameMenuRow{
			{Label: f.words.MenuExitMain, Fallback: 'E', Action: gameMenuExitMain, Enabled: true},
			{Label: f.words.MenuExitWindows, Fallback: 'W', Action: gameMenuExitWindows, Enabled: true},
			{Label: f.words.MenuReturn, Fallback: 'R', Action: gameMenuPageReturn, Enabled: true},
		}
	case gameMenuModActionConfirmation:
		i := f.modUI.pending - 1
		if i < 0 || i >= len(f.modUI.screens) {
			return []gameMenuRow{{Label: f.words.MenuReturn, Fallback: 'R', Action: gameMenuPageReturn, Enabled: true}}
		}
		return []gameMenuRow{
			{Label: f.fitMenuRowLabel(f.menuDisplayText(f.modUI.screens[i].Title)), Action: gameMenuModActionConfirm, Enabled: true, Literal: true, Mod: i + 1},
			{Label: f.words.MenuReturn, Fallback: 'R', Action: gameMenuPageReturn, Enabled: true},
		}
	}
	if f.menuSurface == gameMenuTown {
		rows := townGameMenuRows(f.words)
		for i := range rows {
			if rows[i].Action == gameMenuSave {
				rows[i].Enabled = townCanSave(f.town)
			}
		}
		return f.withModRows(rows)
	}
	return f.withModRows(missionGameMenuRows(f.words, f.menuContext.Campaign, f.menuCanSave, f.menuCanLoad, f.menuCanSound))
}

// withModRows appends one row for each mod screen placed in the in-game menu.
// Without a mod screen the list is returned as it came.
func (f *flow) withModRows(rows []gameMenuRow) []gameMenuRow {
	for _, i := range f.modScreenIndexes(true) {
		action := gameMenuModScreen
		if f.modUI.screens[i].Kind.isAction() {
			// An action entry belongs to a mission that can be left for its
			// town; the town menu and any other mission have no such row.
			if f.menuSurface == gameMenuTown || !f.menuContext.LeaveToTown {
				continue
			}
			action = gameMenuModAction
		}
		rows = append(rows, gameMenuRow{
			Label:   f.fitMenuRowLabel(f.menuDisplayText(f.modUI.screens[i].MenuLabel)),
			Action:  action,
			Enabled: true,
			Literal: true,
			Mod:     i + 1,
		})
	}
	return rows
}

// fitMenuRowLabel clips a label already in the install's code page to the room a
// row has for text, marking the cut, so a long mod label cannot run off its row.
func (f *flow) fitMenuRowLabel(label string) string {
	room := gameMenuPanelW - gameMenuRowLeft - gameMenuRowInsetR - gameMenuTextX
	if f.menuFont == nil {
		if max := room / pickerAdvance; len(label) > max {
			return label[:max-len(clipMark)] + clipMark
		}
		return label
	}
	if w, _ := f.menuFont.Measure(label); w <= room {
		return label
	}
	for len(label) > 0 {
		label = label[:len(label)-1]
		if w, _ := f.menuFont.Measure(label + clipMark); w <= room {
			break
		}
	}
	return label + clipMark
}

func (f *flow) menuDisplayText(s string) string {
	encode := f.encodeMenuKey
	if encode == nil {
		encode = defaultMenuKeyEncode
	}
	var b strings.Builder
	for _, r := range s {
		c, ok := encode(r)
		if !ok {
			c = '?'
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (f *flow) readMenuSound() (enabled bool, volume int, available bool) {
	if f.menuSound == nil {
		return false, 0, false
	}
	return f.menuSound()
}

func (f *flow) refreshGameMenuGates() {
	f.menuCanSave = f.saveGame != nil || f.saveDialogSeams.Prepare != nil
	f.menuCanLoad = false
	if f.saveList != nil {
		f.menuCanLoad = len(f.saveList()) > 0
	}
	_, _, f.menuCanSound = f.readMenuSound()
}

func (f *flow) rebuildGameMenu(page gameMenuPage, selection int) {
	if page == gameMenuSoundOptionsPage && f.menuPage != page {
		f.initSoundTracks()
	}
	if page == gameMenuSoundOptionsPage && f.menuPage != page && f.soundOptions.ReadAcknowledgments != nil {
		f.soundOptions.acknowledgments = f.soundOptions.ReadAcknowledgments()
	}
	if page == gameMenuGameOptionsPage && f.menuPage != page && f.gameOptions.Read != nil {
		f.menuPage = page
		f.gameOptions.draft = f.newGameOptionsDraft()
	} else if page != gameMenuGameOptionsPage {
		f.gameOptions.draft = nil
	}
	f.menuPage = page
	f.soundPointer = soundOptionPointer{}
	if page == gameMenuRoot {
		f.refreshGameMenuGates()
	}
	f.menuList = NewPicker(gameMenuPickerRows(f.menuRows()))
	f.menuList.SetWindow(7)
	if page == gameMenuRoot {
		if n := len(f.menuRows()); n > 7 {
			f.menuList.SetWindow(n)
		}
	}
	if page == gameMenuGameOptionsPage && f.gameOptions.Read != nil {
		f.menuList.SetWindow(len(f.menuRows()))
	}
	if page == gameMenuSoundOptionsPage && f.soundOptions.Read != nil {
		f.menuList.SetWindow(len(f.menuRows()))
	}
	if page == gameMenuQuestObjectivesPage {
		selection = len(f.menuRows()) - 1
	}
	f.menuList.Select(selection)
	f.msg = ""
}

// openGameMenu raises the in-game menu OVER the screen it was called from.
//
// IT IS THE ONE PLACE f.screen BECOMES ScreenGameMenu, showTown's own rule and
// for its reason: every method on that arm may then assume the list is not nil
// once the screen is showing.
//
// THE SCREEN BEHIND IT IS REMEMBERED AND NOT ASSUMED (0143 plan D-5). It opens
// from two screens and a constant would send the player somewhere he was not.
// Since 0158 that memory decides two more things: which of the two surfaces is
// built, and what is drawn behind the panel.
//
// THE VIEWER IS TOLD. Raising the viewer's own popup answer is the whole of
// how this menu stops the world, pins the camera, holds the ambient clock
// and darkens the map — one flag, four readers, and no condition beside
// any of them.
func (f *flow) openGameMenu(back Screen) {
	f.menuSurface = gameMenuMission
	if back == ScreenTown {
		f.menuSurface = gameMenuTown
	}
	f.menuContext = GameMenuContext{Campaign: f.menuSurface == gameMenuMission}
	if f.menuSurface == gameMenuMission && f.viewer != nil {
		f.menuContext = f.viewer.readGameMenuContext()
	}
	f.menuBack = back
	f.rebuildGameMenu(gameMenuRoot, 0)
	f.setScreen(ScreenGameMenu)
	f.setMenuUp(true)
}

// setMenuUp raises or lowers the viewer's half of the popup answer. It is safe
// on a front-end holding no viewer, which is every town session and every test
// that never opened a map.
func (f *flow) setMenuUp(up bool) {
	if f.viewer != nil {
		f.viewer.menuUp = up
	}
}

// chooseGameMenu acts on the selected row.
//
// A DISABLED ROW NEVER REACHES THE SWITCH: Choose refuses an unchoosable
// row, which is the one gate for Enter, for the accelerator and for a
// pointer release alike.
//
// END QUEST AND ABORT GAME DO NOT LEAVE HERE. They raise their decoded,
// different confirmation pages; only a destination row runs an exit.
// Cancellation therefore changes no session state and returns through the same
// root rebuild as every nested page.
func (f *flow) chooseGameMenu() {
	if f.screen != ScreenGameMenu || f.menuList == nil {
		return
	}
	rows := f.menuRows()
	i, ok := f.menuList.Choose()
	if !ok || i < 0 || i >= len(rows) {
		return
	}
	f.applyGameMenuAction(rows[i].Action)
}

// applyGameMenuAction is shared by validated menu rows and the original
// function-key routes. Keyboard context gates are not menu-row availability:
// F3 may open an empty Load list even when the root's Load row is disabled.
func (f *flow) applyGameMenuAction(action gameMenuAction) {
	selection := 0
	if f.menuList != nil {
		selection = f.menuList.Selection()
	}
	switch action {
	case gameMenuReturn:
		f.closeGameMenu()
	case gameMenuOptionsCancel:
		f.rebuildGameMenu(gameMenuRoot, 0)
	case gameMenuTimedAutosave:
		if d := f.gameOptions.draft; d != nil {
			d.autosave.Enabled = !d.autosave.Enabled
		}
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
	case gameMenuAutosaveMinutes:
		if d := f.gameOptions.draft; d != nil {
			d.autosave.Minutes = min(d.autosave.Minutes+1, MaxAutosaveMinutes)
		}
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
	case gameMenuPageReturn:
		if f.menuPage == gameMenuGameOptionsPage && f.gameOptions.draft != nil {
			if err := f.commitGameOptions(); err != nil {
				f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
				f.msg = gameOptionsFailure(err)
				return
			}
		}
		if f.menuPage == gameMenuSoundOptionsPage && f.soundOptions.ReadAcknowledgments != nil &&
			f.soundOptions.WriteAcknowledgments != nil && f.soundOptions.acknowledgments != f.soundOptions.ReadAcknowledgments() {
			if err := f.soundOptions.WriteAcknowledgments(f.soundOptions.acknowledgments); err != nil {
				f.msg = f.soundOptions.Words.NotSaved + err.Error()
				return
			}
		}
		f.rebuildGameMenu(gameMenuRoot, 0)
	case gameMenuAcknowledgments:
		f.soundOptions.acknowledgments = !f.soundOptions.acknowledgments
		f.rebuildGameMenu(gameMenuSoundOptionsPage, selection)
	case gameMenuMusicTracks, gameMenuMusicRandom, gameMenuMusicPlay, gameMenuMusicStop:
		f.chooseMusicOption(action)
	case gameMenuSave:
		if !f.atGameSavePoint() {
			return
		}
		if f.saveDialogSeams.Prepare != nil {
			f.openSaveDialog()
			return
		}
		if f.saveGame == nil {
			f.msg = "this build has nowhere to save to"
			return
		}
		_, err := f.saveGame(f.menuBack == ScreenMap)
		if err != nil {
			f.msg = err.Error()
			return
		}
		f.msg = f.words.SaveAcknowledgement
		if f.msg == "" {
			f.msg = AuthoredWords().SaveAcknowledgement
		}
	case gameMenuModScreen:
		if rows := f.menuRows(); selection >= 0 && selection < len(rows) && rows[selection].Mod > 0 {
			f.openModScreen(rows[selection].Mod-1, ScreenGameMenu, selection)
		}
	case gameMenuModAction:
		if rows := f.menuRows(); selection >= 0 && selection < len(rows) && rows[selection].Mod > 0 {
			f.modUI.pending = rows[selection].Mod
			f.rebuildGameMenu(gameMenuModActionConfirmation, 1)
		}
	case gameMenuModActionConfirm:
		if rows := f.menuRows(); selection >= 0 && selection < len(rows) && rows[selection].Mod > 0 {
			f.runModAction(rows[selection].Mod - 1)
		}
	case gameMenuLoad:
		f.setMenuUp(false)
		f.openLoad(ScreenGameMenu)
	case gameMenuGameOptions:
		f.rebuildGameMenu(gameMenuGameOptionsPage, 0)
	case gameMenuSoundOptions:
		f.rebuildGameMenu(gameMenuSoundOptionsPage, 0)
	case gameMenuQuestObjectives:
		f.questTop, f.questPress, f.questBar = 0, buttonLatch{}, scrollBarInput{}
		f.rebuildGameMenu(gameMenuQuestObjectivesPage, 0)
	case gameMenuDiplomacy:
		f.rebuildGameMenu(gameMenuDiplomacyPage, 0)
	case gameMenuEndQuest:
		f.rebuildGameMenu(gameMenuEndQuestConfirmation, 0)
	case gameMenuAbortGame:
		f.rebuildGameMenu(gameMenuAbortGameConfirmation, 0)
	case gameMenuToggleTips:
		if f.gameOptions.draft != nil {
			f.toggleDraftTips()
			f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
		} else if f.menuTips != nil && f.setMenuTips != nil {
			f.setMenuTips(!f.menuTips())
			f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
		}
	case gameMenuDayNight, gameMenuHealth, gameMenuDamage, gameMenuFormation, gameMenuRetreat, gameMenuPathfinding, gameMenuSmoothing, gameMenuShadows, gameMenuLighting, gameMenuAnimation, gameMenuAutoHealing:
		if f.gameOptions.draft != nil {
			f.cycleDraftOption(GameOption(action - gameMenuDayNight))
			f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
		} else {
			f.cycleGameOption(GameOption(action - gameMenuDayNight))
		}
	case gameMenuSpeedDown, gameMenuSpeedUp:
		delta := -1
		if action == gameMenuSpeedUp {
			delta = 1
		}
		if f.gameOptions.draft != nil {
			f.setDraftSpeed(f.gameOptions.draft.speed + delta)
		} else {
			f.stepMenuSpeed(delta)
		}
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
	case gameMenuTooltipDelay:
		if f.gameOptions.draft != nil {
			f.cycleDraftTooltipDelay()
			f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
			return
		}
		err := f.cycleTooltipDelay()
		f.rebuildGameMenu(gameMenuGameOptionsPage, selection)
		if err != nil {
			prefix := "Tooltip setting not saved: "
			if f.menuSelector() == text.SelectorConverting {
				prefix = f.menuDisplayText(tooltipRU.NotSaved)
			}
			f.msg = prefix + err.Error()
		}
	case gameMenuMusicVolume, gameMenuEffectsVolume, gameMenuSpeechVolume:
		if channel, ok := soundActionChannel(action); ok && f.soundOptions.Read != nil {
			f.setSoundOption(channel, f.soundOptions.Read()[channel]+5)
		}
	case gameMenuToggleSound, gameMenuVolumeDown, gameMenuVolumeUp:
		enabled, volume, available := f.readMenuSound()
		if !available || f.setMenuSound == nil {
			return
		}
		switch action {
		case gameMenuToggleSound:
			enabled = !enabled
		case gameMenuVolumeDown:
			volume -= 25
		case gameMenuVolumeUp:
			volume += 25
		}
		if volume < 0 {
			volume = 0
		}
		if volume > 100 {
			volume = 100
		}
		err := f.setMenuSound(enabled, volume)
		f.rebuildGameMenu(gameMenuSoundOptionsPage, selection)
		if err != nil {
			f.msg = "Sound settings not saved: " + err.Error()
			if f.soundOptions.Read != nil {
				f.msg = f.soundOptions.Words.NotSaved + err.Error()
			}
		}
	case gameMenuTestSound:
		// Playback is the action itself and therefore stays in App, the tier
		// that owns the optional audio device. The production wrapper emits
		// UISoundOptionsTest when this accepted action is dispatched.
	case gameMenuConfirmEndQuest:
		f.setMenuUp(false)
		f.setScreen(ScreenPicker)
		f.leaveMap()
		f.msg = ""
	case gameMenuVictory:
		f.finishContinuedMission()
	case gameMenuExitMain:
		f.setMenuUp(false)
		if f.viewer != nil {
			f.viewer.StopAmbient()
		}
		f.toMenu()
		f.msg = ""
	case gameMenuExitWindows:
		f.setMenuUp(false)
		f.menuExit = true
	}
}

// DIV-140
func (f *flow) chooseGameMenuAccelerator(r rune, before ...func(gameMenuAction)) bool {
	if f.screen != ScreenGameMenu || f.menuList == nil {
		return false
	}
	encode := f.encodeMenuKey
	if encode == nil {
		encode = defaultMenuKeyEncode
	}
	sel := f.menuSelector()
	b, ok := encode(r)
	if !ok {
		return false
	}
	c := gameMenuLower(b, sel)
	for i, row := range f.menuRows() {
		accelerator, marked := row.accelerator(sel)
		if !marked || accelerator != c {
			continue
		}
		f.menuList.Select(i)
		if row.Enabled && len(before) > 0 && before[0] != nil {
			before[0](row.Action)
		}
		f.chooseGameMenu()
		return true
	}
	return false
}

// menuSelector is the language selector the open menu's own font carries, 0
// (identity) when no font was set — the debug-font path and every
// hand-assembled test flow.
func (f *flow) menuSelector() int {
	if f.menuFont == nil {
		return 0
	}
	return f.menuFont.Selector
}

// defaultMenuKeyEncode is the ASCII-only encoder chooseGameMenuAccelerator
// falls back to when no install seam was wired (cmd/mapview, every
// hand-assembled test flow) — the same behaviour this package had before
// 1014 for every rune it could reach at all.
func defaultMenuKeyEncode(r rune) (byte, bool) {
	if r < 0x20 || r > 0x7e {
		return 0, false
	}
	return byte(r), true
}

// closeGameMenu returns to the screen the menu was opened from. It is the
// return row and it is Escape on this screen, one statement for both.
func (f *flow) closeGameMenu() {
	f.setMenuUp(false)
	f.setScreen(f.menuBack)
	f.menuPage = gameMenuRoot
	f.gameOptions.draft = nil
	f.menuList = nil
	f.menuContext = GameMenuContext{}
	f.msg = ""
}

func (f *flow) menuPanelSurface() gameMenuSurface {
	if f.menuPage == gameMenuEndQuestConfirmation || f.menuPage == gameMenuAbortGameConfirmation ||
		f.menuPage == gameMenuModActionConfirmation {
		return gameMenuTown
	}
	return f.menuSurface
}

func (f *flow) takeMenuExit() bool {
	exit := f.menuExit
	f.menuExit = false
	return exit
}

// openLoad shows the LOAD GAME window over the screen that armed it.
//
// THE LIST IS READ HERE AND NOT PER FRAME. A save written while the window is
// open is a state this story does not produce — the only writer is the mini-menu
// behind it — and a list rebuilt every frame would move the selection under the
// player's hand.
func (f *flow) openLoad(back Screen) {
	f.loadUI.resetClick()
	f.loadUI.resetPointer()
	f.loadUI.confirm = false
	f.loadUI.remove = nil
	f.saves = nil
	if f.saveList != nil {
		f.saves = f.saveList()
	}
	rows := make([]PickerRow, len(f.saves))
	for i, e := range f.saves {
		rows[i] = PickerRow{Text: e.Label, Choosable: true}
	}
	f.loadList = NewPicker(rows).SetWindow(loadVisibleRows)
	f.loadBack = back
	f.setScreen(ScreenLoad)
	if f.saveList == nil {
		f.msg = "this build has nowhere to load from"
	} else if len(f.saves) == 0 {
		f.msg = "no saved games"
	} else {
		f.msg = ""
	}
}

// closeLoad returns to the screen that opened the load window. A GameMenu
// return is more than a screen assignment: the load window deliberately
// lowered the viewer's popup flag, and the root's availability may have
// changed while the list was visible. Restore both before ScreenGameMenu can
// be driven again so its first visible frame holds, rather than repays, the
// elapsed map and ambient-animation span.
func (f *flow) closeLoad() {
	f.loadUI.resetClick()
	f.loadUI.resetPointer()
	back := f.loadBack
	if back == ScreenGameMenu {
		selection := 0
		if f.menuList != nil {
			selection = f.menuList.Selection()
		}
		f.rebuildGameMenu(gameMenuRoot, selection)
		f.setMenuUp(true)
	}
	f.setScreen(back)
	f.msg = ""
}

// chooseLoad loads the selected save.
//
// A REFUSAL LEAVES THE PLAYER ON THE LIST with the reason on the message line
// and the row STILL CHOOSABLE — unlike the map list, which marks a row that
// would not load. A file that failed to read says nothing about the next one,
// and a mark here would outlive nothing: the list is rebuilt every time the
// window opens.
func (f *flow) chooseLoad() {
	if f.screen != ScreenLoad || f.loadList == nil {
		return
	}
	i, ok := f.loadList.Choose()
	if !ok {
		return
	}
	if f.loadGame == nil || i < 0 || i >= len(f.saves) {
		return
	}
	open, town, err := f.loadGame(f.saves[i].Name)
	if err == nil {
		err = f.loadPrepared(open, town)
	}
	if err != nil {
		f.msg = loadFailure(f.saves[i].Name, err)
	}
}

func (f *flow) loadPrepared(open MapOpener, town bool) error {
	// Prepare the replacement before releasing the old map. An opener can
	// refuse after the file was read; cancellation must still have a complete
	// old session, including a terminal failure panel, to return to.
	if town {
		if f.town == nil {
			return fmt.Errorf("this build has no town to load into")
		}
		f.leaveMap()
		f.showTown("")
		f.loadUI.completed++
		f.resetTimedAutosave()
		return nil
	}
	if open == nil {
		return fmt.Errorf("the save named no game")
	}
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := open()
	if err != nil {
		return err
	}
	f.leaveMap()
	f.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	f.loadUI.completed++
	return nil
}

// loadFailure is what the load window's message line says when a save will
// not open: WHICH save, then why (1032 return 1).
//
// The name is the row the player just chose. It was absent before, and the
// reason alone is not enough on a list of thirty rows named by date: a player
// who pressed the wrong row reads a sentence about a save and cannot tell which
// one it is about. The line is clipped to pickerCols by the draw, which now
// marks the cut.
func loadFailure(name string, err error) string {
	return name + ": " + err.Error()
}
