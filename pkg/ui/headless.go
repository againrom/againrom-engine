package ui

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/menu"
)

// HeadlessStep uses Update's dispatch and the shared headless clock, with no
// window creation or physical input sampling.
func (a *App) HeadlessStep() error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless step: nil application")
	}
	if a.step(a.headlessIdleInput(), a.headlessAt()) {
		return fmt.Errorf("headless step: application requested exit")
	}
	return nil
}

func (a *App) HeadlessGameOptionsFrame() (*image.RGBA, error) {
	if a == nil || a.flow == nil || a.flow.screen != ScreenGameMenu || a.flow.menuPage != gameMenuGameOptionsPage || a.flow.gameOptions.draft == nil {
		return nil, fmt.Errorf("headless game options: page is not open")
	}
	return a.gameOptionsPicture(), nil
}

// An idle frame or key edge does not move the mouse to the window origin.
// Keep the viewer's last observed window position, or remain outside before
// any pointer was observed. Explicit pointer (0,0) remains a real edge input.
func (a *App) headlessIdleInput() appInput {
	x, y := -1, -1
	if a.hasHeadlessPointer {
		x, y = a.headlessPointer.X, a.headlessPointer.Y
	} else if v := a.flow.viewer; v != nil && v.hasWinCursor {
		x, y = v.winCursorX, v.winCursorY
	}
	return appInput{CursorX: x, CursorY: y, Viewer: Input{CursorX: x, CursorY: y}, Unfocused: a.headlessUnfocused}
}

// HeadlessFocus drives Update's focus gate without moving the pointer.
func (a *App) HeadlessFocus(focused bool) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless focus: nil application")
	}
	a.headlessUnfocused = !focused
	in := a.headlessIdleInput()
	if a.step(in, a.headlessAt()) {
		return fmt.Errorf("headless focus requested exit")
	}
	return nil
}

// HeadlessCursor is where the next Draw puts the front-end's own cursor
// picture, from the position the latest step was given. It answers false where
// Draw places none: on the map screen, whose pointer the viewer places, and
// under a cutscene.
func (a *App) HeadlessCursor() (CursorPlacement, bool) {
	if a == nil || a.flow == nil || a.cutscene != nil || a.flow.mapShowing() {
		return CursorPlacement{}, false
	}
	return a.cursorPlacement()
}

// HeadlessMapCursor is the slot name the cursor manager holds after the latest
// step on the mission map.
func (a *App) HeadlessMapCursor() (string, bool) {
	if a == nil || a.flow == nil || a.cutscene != nil || !a.flow.mapShowing() {
		return "", false
	}
	return a.flow.cursor.CurrentName(), true
}

// HeadlessKey dispatches one key edge through App.step.  The names are part of
// the scenario vocabulary rather than a second set of game commands. enter and
// numpad-enter are read through the Enter binding a windowed tick asks.
func (a *App) HeadlessKey(name string) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless key: nil application")
	}
	in := a.headlessIdleInput()
	in.AnyKey = true
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "enter":
		in.Enter = enterPressed(onlyKey(ebiten.KeyEnter))
	case "numpad-enter":
		in.Enter = enterPressed(onlyKey(ebiten.KeyNumpadEnter))
	case "escape", "esc":
		in.Escape = true
	case "load", "l":
		in.Load = true
	case "f2":
		in.SaveGame = true
	case "f3":
		in.LoadGame = true
	case "f4":
		in.QuickSave = true
	case "shift-f4":
		in.Reveal, in.ShiftHeld, in.Viewer.Shift = true, true, true
	case "f9":
		in.QuickLoad = true
	case "f11", "readout":
		in.Readout = true
	case "f12", "fps":
		in.FPS = true
	case "alt-backspace":
		in.Viewer.Alt, in.Backspace = true, true
	case "alt-f12":
		in.Viewer.Alt, in.FPS = true, true
	case "f5", "f6", "f7", "f8", "ctrl-f5", "ctrl-f6", "ctrl-f7", "ctrl-f8":
		key := strings.ToLower(strings.TrimSpace(name))
		in.Viewer.Ctrl = strings.HasPrefix(key, "ctrl-")
		in.AttackHeld = in.Viewer.Ctrl
		in.QuickSpell[int(key[len(key)-1]-'5')] = true
	case "numpad-plus":
		in.Faster = true
	case "numpad-minus":
		in.Slower = true
	case "ctrl-numpad-plus":
		in.Unpaced = true
	case "ctrl-numpad-minus":
		in.Paced = true
	case "f1", "help":
		in.Help = true
	case "c", "cast":
		in.Cast = true
	case "pageup":
		in.PageUp = true
	case "pagedown":
		in.PageDown = true
	case "up":
		in.Up = true
	case "down":
		in.Down = true
	case "left":
		in.Left = true
	case "right":
		in.Right = true
	case "shift-left":
		in.Left, in.ShiftHeld, in.Viewer.Shift = true, true, true
	case "shift-right":
		in.Right, in.ShiftHeld, in.Viewer.Shift = true, true, true
	case "shift-home":
		in.Home, in.ShiftHeld, in.Viewer.Shift = true, true, true
	case "shift-end":
		in.End, in.ShiftHeld, in.Viewer.Shift = true, true, true
	case "space":
		in.Panels = true
	case "0", "numpad-0":
		in.Pause = true
	case "ctrl-n":
		in.TimeFlow, in.Viewer.Ctrl = true, true
	case "ctrl-o":
		in.Smoothing, in.Viewer.Ctrl = true, true
	case "ctrl-u":
		in.AutoHealing, in.Viewer.Ctrl = true, true
	case "ctrl-h":
		in.ShowHealth, in.Viewer.Ctrl = true, true
	case "ctrl-l":
		in.Numerals, in.Viewer.Ctrl = true, true
	case "ctrl-f":
		in.Formation, in.Viewer.Ctrl = true, true
	case "ctrl-w":
		in.Retreat, in.Viewer.Ctrl = true, true
	case "a", "attack":
		in.Attack = true
	case "r", "retreat":
		in.PlayerRetreat = true
	case "d", "defend":
		in.Defend = true
	case "p", "patrol":
		in.Patrol = true
	case "f", "grab":
		in.Grab = true
	case "e", "select-all":
		in.SelectAll = true
	case "tab":
		// App dispatches Tab by screen: town pane mode or dialog focus.
		in.PaneMode = true
	case "shift-tab":
		in.PaneMode, in.ShiftHeld = true, true
	case "backspace":
		in.Backspace = true
	case "delete":
		in.Delete = true
	case "home":
		in.Home = true
	case "end":
		in.End = true
	case "ctrl-a":
		in.SelectText = true
	case "book", "b", "q":
		in.Book = true
	case "inventory", "i", "backquote", "`":
		in.Inventory = true
	case "doll", "j":
		in.Doll = true
	default:
		key := strings.ToLower(strings.TrimSpace(name))
		if len(key) != 5 || key[:4] != "alt-" || key[4] < 'a' || key[4] > 'z' {
			return fmt.Errorf("headless key: unknown key %q", name)
		}
		in.Viewer.Alt = true
		in.AltLetter = readAltLetter(true, onlyKey(altLetterKeys[key[4]-'a']))
		in.setAltLetter(key[4])
	}
	if a.step(in, a.headlessAt()) {
		return fmt.Errorf("headless key %q requested application exit", name)
	}
	return nil
}

// HeadlessSelection reports the map controller's current selection without
// changing it. Key and pointer gestures remain the only selection producers.
func (a *App) HeadlessSelection() []uint32 {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return nil
	}
	return append([]uint32(nil), a.flow.viewer.sel...)
}

// HeadlessRows reports the rows the current production list controller owns.
// The returned slice is a copy so a scenario can never mutate the controller.
func (a *App) HeadlessRows() []PickerRow {
	if a == nil || a.flow == nil {
		return nil
	}
	var p *Picker
	switch a.flow.screen {
	case ScreenCutsceneLibrary:
		var rows []PickerRow
		for _, entry := range a.seenCutscenes() {
			rows = append(rows, PickerRow{Text: entry.Title, Choosable: true})
		}
		return rows
	case ScreenEnding:
		var rows []PickerRow
		for _, label := range a.flow.endingButtons() {
			rows = append(rows, PickerRow{Text: label, Choosable: true})
		}
		return rows
	case ScreenPicker:
		p = a.flow.picker
	case ScreenTown:
		p = a.flow.townList
	case ScreenGameMenu:
		p = a.flow.menuList
	case ScreenLoad:
		p = a.flow.loadList
	case ScreenSave:
		if a.flow.saveDialog != nil {
			p = a.flow.saveDialog.list
		}
	}
	if p == nil {
		return nil
	}
	return append([]PickerRow(nil), p.Rows()...)
}

// HeadlessGameplayScreen reports the map or town behind a game/load menu. It
// is used only to choose the production state source for a snapshot; Screen()
// remains the visible screen reported to the scenario.
func (a *App) HeadlessGameplayScreen() Screen {
	if a == nil || a.flow == nil {
		return ScreenMenu
	}
	switch a.flow.screen {
	case ScreenGameMenu, ScreenSave:
		return a.flow.menuBack
	case ScreenMod:
		if a.flow.modUI.back == ScreenGameMenu {
			return a.flow.menuBack
		}
		return a.flow.modUI.back
	case ScreenLoad:
		if a.flow.loadBack == ScreenGameMenu {
			return a.flow.menuBack
		}
		return a.flow.loadBack
	default:
		return a.flow.screen
	}
}

// HeadlessMessage returns the production screen's current refusal/status line.
func (a *App) HeadlessMessage() string {
	if a == nil || a.flow == nil {
		return ""
	}
	return a.flow.msg
}

// HeadlessFrame returns the CPU composite for whatever screen the
// application currently shows, for a caller with no ebiten canvas
// (cmd/screenshot). It calls composeScreen (app.go), the exact selection
// Draw itself calls through drawMenu, drawChargen and drawTown, so a
// headless capture and the windowed frame it stands in for cannot select a
// different composer for the same on-screen state — see composeScreen's own
// header for the six developer tools that each held their own copy of that
// decision before this function existed.
//
// note is non-empty only when pix is a correct but incomplete frame: today
// that is ScreenGameMenu alone, where the town or world beneath the mini-menu
// composes but the menu's own dim and panel (paintGameMenu, gamemenu.go) do
// not — that overlay draws through ebiten's vector package directly onto the
// windowed canvas and has no CPU composite. A caller that needs those pixels
// has none to read; it has the note's name for what is missing instead.
//
// err is non-nil, and pix nil, for a screen this build cannot compose on the
// CPU at all: the gameplay/mission screen (Viewer.Draw paints straight onto
// the ebiten canvas with vector.StrokeLine and screen.DrawImage, no RGBA
// composite anywhere), the map picker (ebitenutil.DebugPrintAt only), a load
// list without its installed font.
// It never returns a partial or plausible-looking frame for one of these:
// a screenshot a reader cannot tell from a correct one is worse than a
// refusal that names itself, because the whole reason to call this function
// is to compare the result against the original game.
//
// It reads a.flow and the App's own composed-image seams and touches no
// ebiten-owned field: a.canvas is a *ebiten.Image, allocated lazily by Draw
// alone, and is nil in a headless process that never calls Draw.
func (a *App) HeadlessFrame() (pix *image.RGBA, note string, err error) {
	if a == nil || a.flow == nil {
		return nil, "", fmt.Errorf("headless frame: nil application")
	}
	if a.cutscene != nil {
		pix := a.composeCutscene()
		a.finishCutscenePresentation()
		return pix, "", nil
	}
	if a.flow.mapShowing() {
		return nil, "", fmt.Errorf("headless frame: %s screen draws through Viewer.Draw directly onto the ebiten canvas, no CPU composite", a.Screen())
	}
	var dialogue *image.RGBA
	if detached, ok := a.detachedTownDialogue(); ok {
		// A wide town separates its native room columns. Compose that room
		// without the modal so the modal can remain one centred rectangle after
		// widening, matching Draw's GPU order.
		pix, err = a.composeTownRoom()
		dialogue = detached
	} else {
		pix, err = a.composeScreen()
	}
	if err != nil {
		return nil, "", fmt.Errorf("headless frame: %s screen: %w", a.Screen(), err)
	}
	menuPanel := a.GameMenuPanel()
	if a.flow.screen == ScreenGameMenu && menuPanel == nil {
		note = "the in-game menu's dim and panel are not in this frame: paintGameMenu draws with ebiten's vector package, no CPU composite"
	}
	// The same segment layout the windowed GPU blit consumes widens this CPU
	// witness. cmd/screenshot therefore sees the actual logical width selected
	// by Layout, including an anchored town column or centred native surface.
	wide := a.widenNativeFrame(pix)
	if dialogue != nil {
		a.dialogueBackdrop.apply(wide, townDialogueShows(a.flow.town))
		art, body := townDialogueFrame(a.flow.town)
		at := image.Pt((wide.Bounds().Dx()-dialogue.Bounds().Dx())/2, (frame.H-dialogue.Bounds().Dy())/2)
		a.dialogueBackdrop.applyFrameShadows(wide, art, body, at, nil)
	}
	if dialogue != nil {
		at := image.Pt((wide.Bounds().Dx()-dialogue.Bounds().Dx())/2, (frame.H-dialogue.Bounds().Dy())/2)
		composeDialogueImageWithPolicy(wide, dialogue, at, a.dialogueBackdrop.policy)
	}
	if menuPanel != nil {
		draw.Draw(wide, wide.Bounds(), &image.Uniform{C: AuthoredNoticeBackdrop()}, image.Point{}, draw.Over)
		copyNativeOver(wide, menuPanel, image.Pt((wide.Bounds().Dx()-frame.W)/2, 0), wide.Bounds())
	}
	a.paintTooltip(wide)
	return wide, note, nil
}

func (a *App) HeadlessNoticeFrame() (*image.RGBA, error) {
	if a == nil || a.flow == nil || a.cutscene != nil || a.flow.viewer == nil {
		return nil, fmt.Errorf("headless notice: no visible map notice")
	}
	pic, _, _, visible := a.flow.viewer.noticePresent()
	if !visible {
		return nil, fmt.Errorf("headless notice: no visible map notice")
	}
	return pic, nil
}

// HeadlessCharacterPane composes the MISSION screen's character pane on its
// own, and reports which of its two presentation modes it is in.
//
// IT EXISTS BECAUSE HeadlessFrame REFUSES THE WHOLE MISSION SCREEN. Viewer.Draw
// paints the ebiten canvas directly, so no committed command can render the
// mission at all; the pane is the one part of it that is composed on the CPU as
// an *image.RGBA (panel.go's panelPresent), by the same call Draw itself makes,
// so a reader can see the widget and its six corner controls without playing.
// That is `AGENTS.md` rule 7 applied to a screen whose only composed part is
// this one: the two modes are a screen each, and a mode no committed command
// renders is a mode nobody checks.
//
// It composes through panelPresent and selects nothing of its own. A viewer
// that is not on the mission screen, or that has no font, answers false.
func (a *App) HeadlessCharacterPane() (pix *image.RGBA, statistics bool, err error) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return nil, false, fmt.Errorf("headless character pane: nil application")
	}
	if !a.flow.mapShowing() {
		return nil, false, fmt.Errorf("headless character pane: %s screen has no mission character pane", a.Screen())
	}
	v := a.flow.viewer
	pic, _, ok := v.panelPresent()
	if !ok {
		return nil, false, fmt.Errorf("headless character pane: the viewer composed none (no card font, or no slot at %dx%d)", v.frameW, v.frameH)
	}
	return pic, v.characterPaneStatistics(), nil
}

// HeadlessInventoryPack returns the same CPU pack composite used by Draw.
// It neither issues input nor changes the selected inventory subject.
func (a *App) HeadlessInventoryPack() (*image.RGBA, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return nil, err
	}
	pic, _, ok := v.packBarPresent()
	if !ok || pic == nil {
		return nil, fmt.Errorf("headless inventory pack: no pack is showing")
	}
	return pic, nil
}

// HeadlessBottomHUD joins the same book and inventory layers uploaded by
// Viewer.Draw. It preserves their native origins and crops to the map band.
func (a *App) HeadlessBottomHUD() (*image.RGBA, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return nil, err
	}
	book, bookAt, bookOK := v.spellbookPresent()
	pack, packAt, packOK := v.packBarPresent()
	y := v.frameH
	if bookOK {
		y = min(y, bookAt.Y)
	}
	if packOK {
		y = min(y, packAt.Y)
	}
	if y == v.frameH {
		return nil, fmt.Errorf("headless bottom HUD: both bars are hidden")
	}
	pic := image.NewRGBA(image.Rect(0, 0, v.frameW-sidebarWidth, v.frameH-y))
	if bookOK {
		blit(pic, book, bookAt.X, bookAt.Y-y)
	}
	if packOK {
		blit(pic, pack, packAt.X, packAt.Y-y)
	}
	return pic, nil
}

// HeadlessMinimap returns the exact framed minimap uploaded by Viewer.Draw.
func (a *App) HeadlessMinimap() (*image.RGBA, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return nil, err
	}
	pic, _, ok := v.minimapPresent()
	if !ok {
		return nil, fmt.Errorf("minimap has no drawable map")
	}
	return pic, nil
}

// HeadlessMinimapMarkAt reports which mark colours the framed minimap
// uploaded by Viewer.Draw holds on the pixels a mark on cell (col, row) fills:
// local is the local participant's colour, other every other owner's. It
// reads the composed picture, so every mark rule and anything painted over the
// cell afterwards counts.
func (a *App) HeadlessMinimapMarkAt(col, row int) (local, other bool, err error) {
	v, err := a.headlessViewer()
	if err != nil {
		return false, false, err
	}
	g, geomOK := v.minimapGeometry()
	pic, at, ok := v.minimapPresent()
	if !geomOK || !ok {
		return false, false, fmt.Errorf("headless minimap mark: minimap has no drawable map")
	}
	r := minimapMarkRect(image.Pt(col, row).Sub(g.Origin), g.Cols, g.Rows, g.Num, g.Den, image.Rectangle{Max: g.Content.Size()})
	if r.Empty() {
		return false, false, fmt.Errorf("headless minimap mark: cell (%d,%d) has no mark pixel", col, row)
	}
	r = r.Add(g.Content.Min.Sub(at))
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			switch pic.RGBAAt(x, y) {
			case minimapLocalColor:
				local = true
			case minimapOtherColor:
				other = true
			}
		}
	}
	return local, other, nil
}

// HeadlessStructureReadout is widget 8's readout as this frame draws it: the
// composed picture, the frame pixel its top-left corner stands on and the
// three lines, or an error for a frame with no readout.
func (a *App) HeadlessStructureReadout() (pix *image.RGBA, at image.Point, lines [3]string, err error) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return nil, image.Point{}, lines, fmt.Errorf("headless structure readout: nil application")
	}
	if !a.flow.mapShowing() {
		return nil, image.Point{}, lines, fmt.Errorf("headless structure readout: %s screen has no mission column", a.Screen())
	}
	v := a.flow.viewer
	pic, at, ok := v.structureReadoutPresent()
	if !ok {
		return nil, image.Point{}, lines, fmt.Errorf("headless structure readout: no readout is drawn (no card font, no room, or no hovered or selected structure)")
	}
	return pic, at, v.structKey.lines, nil
}

// HeadlessSelectedStructure is the structure a plain click selected, if any.
func (a *App) HeadlessSelectedStructure() (InspectionSubject, bool) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return InspectionSubject{}, false
	}
	return a.flow.viewer.selectedStructure()
}

// HeadlessMissionColumnLayers records the actual column blits of one offscreen
// Draw. Like Draw, it must run on the application's drawing goroutine.
func (a *App) HeadlessMissionColumnLayers() ([]string, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return nil, err
	}
	var layers []string
	previous := blitColumnLayer
	blitColumnLayer = func(_ *ebiten.Image, _ *ebiten.Image, _ *ebiten.DrawImageOptions, layer string) {
		layers = append(layers, layer)
	}
	defer func() { blitColumnLayer = previous }()
	screen := ebiten.NewImage(v.frameW, v.frameH)
	defer screen.Dispose()
	v.Draw(screen)
	return layers, nil
}

func (a *App) HeadlessMissionCard() (pix *image.RGBA, err error) {
	if a == nil || a.flow == nil || a.flow.viewer == nil {
		return nil, fmt.Errorf("headless mission card: nil application")
	}
	if !a.flow.mapShowing() {
		return nil, fmt.Errorf("headless mission card: %s screen has no mission column", a.Screen())
	}
	v := a.flow.viewer
	pic, _, ok := v.missionCardPresent()
	if !ok {
		if v.characterPaneStatistics() {
			return nil, fmt.Errorf("headless mission card: the pane itself is showing the card, so the fourth box composes none")
		}
		return nil, fmt.Errorf("headless mission card: the viewer composed none (no card font, no subject, or no slot at %dx%d)", v.frameW, v.frameH)
	}
	return pic, nil
}

// HeadlessDrawnMissionCard is the lower card box exactly as Draw composes it
// this frame, with the box's frame position: the rows stay off while a
// structure readout is drawn over it.
func (a *App) HeadlessDrawnMissionCard() (*image.RGBA, image.Point, error) {
	if a == nil || a.flow == nil || a.flow.viewer == nil || !a.flow.mapShowing() {
		return nil, image.Point{}, fmt.Errorf("headless drawn mission card: no mission column")
	}
	v := a.flow.viewer
	_, _, readout := v.structureReadoutPresent()
	pic, at, ok := v.missionCardPresentWith(readout)
	if !ok {
		return nil, image.Point{}, fmt.Errorf("headless drawn mission card: the viewer draws none")
	}
	return pic, at, nil
}

// HeadlessGameMenuAction chooses a menu action by the production row's
// action, rather than by its displayed label. The scenario vocabulary is
// stable across installs, while the root labels come from the install's
// localized dialogs.txt. Selection and activation still travel through the
// same arrow and Enter dispatch as a windowed session.
func (a *App) HeadlessGameMenuAction(name string) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless game-menu action: nil application")
	}
	if a.flow.screen != ScreenGameMenu || a.flow.menuList == nil {
		return fmt.Errorf("headless game-menu action %q: screen %s has no game menu", name, a.flow.screen)
	}
	var want gameMenuAction
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "save":
		want = gameMenuSave
	case "load":
		want = gameMenuLoad
	case "diplomacy":
		want = gameMenuDiplomacy
	case "game-options":
		want = gameMenuGameOptions
	case "sound-options":
		want = gameMenuSoundOptions
	case "objectives":
		want = gameMenuQuestObjectives
	case "end":
		want = gameMenuEndQuest
	case "abort":
		want = gameMenuAbortGame
	case "return":
		want = gameMenuReturn
	case "page-return":
		want = gameMenuPageReturn
	case "options-cancel":
		want = gameMenuOptionsCancel
	case "timed-autosave":
		want = gameMenuTimedAutosave
	case "autosave-minutes":
		want = gameMenuAutosaveMinutes
	case "toggle-tips":
		want = gameMenuToggleTips
	case "toggle-sound":
		want = gameMenuToggleSound
	case "acknowledgments":
		want = gameMenuAcknowledgments
	case "music-tracks":
		want = gameMenuMusicTracks
	case "music-random":
		want = gameMenuMusicRandom
	case "music-play":
		want = gameMenuMusicPlay
	case "music-stop":
		want = gameMenuMusicStop
	case "volume-down":
		want = gameMenuVolumeDown
	case "volume-up":
		want = gameMenuVolumeUp
	case "speed-down":
		want = gameMenuSpeedDown
	case "speed-up":
		want = gameMenuSpeedUp
	case "tooltip-delay":
		want = gameMenuTooltipDelay
	case "day-night":
		want = gameMenuDayNight
	case "show-health":
		want = gameMenuHealth
	case "flying-damage":
		want = gameMenuDamage
	case "formation":
		want = gameMenuFormation
	case "retreat-mode":
		want = gameMenuRetreat
	case "smoothing":
		want = gameMenuSmoothing
	case "shadows":
		want = gameMenuShadows
	case "lighting":
		want = gameMenuLighting
	case "animation":
		want = gameMenuAnimation
	case "autohealing":
		want = gameMenuAutoHealing
	case "pathfinding":
		want = gameMenuPathfinding
	case "test-sound":
		want = gameMenuTestSound
	case "confirm-end":
		want = gameMenuConfirmEndQuest
	case "confirm-abort":
		want = gameMenuExitMain
	case "victory":
		want = gameMenuVictory
	case "exit-main":
		want = gameMenuExitMain
	case "exit-windows":
		want = gameMenuExitWindows
	default:
		return fmt.Errorf("headless game-menu action: unknown action %q", name)
	}
	for i, row := range a.flow.menuRows() {
		if row.Action != want {
			continue
		}
		if !row.Enabled {
			return fmt.Errorf("headless game-menu action %q: row %d is not choosable", name, i)
		}
		return a.headlessChooseRow(a.flow.menuList, i)
	}
	return fmt.Errorf("headless game-menu action %q: screen %s has no such action", name, a.flow.screen)
}

// HeadlessActivate selects exactly one current row by its displayed text and
// sends Enter through App.step.  Dialogue and notice activation also use the
// same Enter arm a player uses.  Missing or ambiguous text is an error.
func (a *App) HeadlessActivate(target string) error {
	if a == nil || a.flow == nil {
		return fmt.Errorf("headless activate: nil application")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("headless activate: empty target")
	}
	if a.flow.screen == ScreenMenu {
		return a.headlessMenuButton(target)
	}
	if a.flow.screen == ScreenMap {
		if a.flow.noticeOpen() && (a.flow.viewer.noticeKind == NoticeFailure || a.flow.viewer.noticeKind == NoticeSuccess) && !strings.EqualFold(target, "notice") {
			v := a.flow.viewer
			l := v.noticeLayout()
			var button image.Rectangle
			if v.noticeKind == NoticeSuccess {
				switch strings.ToLower(target) {
				case "victory":
					button = l.Button
				case "continue":
					button = l.SecondaryButton
				}
			} else {
				switch strings.ToLower(target) {
				case "exit to main menu":
					button = l.Button
				case "load game":
					if !l.SecondaryDisabled {
						button = l.SecondaryButton
					}
				}
			}
			if button.Empty() {
				return fmt.Errorf("headless activate %q: outcome panel has no enabled control", target)
			}
			// Resolve the visible control, then drive the ordinary pointer
			// dispatch. No headless-only destination bypasses the panel gates.
			p := button.Min.Add(button.Max).Div(2)
			_, at, scale, visible := v.noticePresent()
			if !visible {
				return fmt.Errorf("headless activate %q: outcome panel is not visible", target)
			}
			x, y, err := v.frameToWindow(at.Add(image.Pt(int(float64(p.X)*scale), int(float64(p.Y)*scale))), "headless outcome control")
			if err != nil {
				return err
			}
			if err := a.HeadlessPointer("press", x, y); err != nil {
				return err
			}
			return a.HeadlessPointer("release", x, y)
		}
		if !strings.EqualFold(target, "notice") || !a.flow.noticeOpen() {
			return fmt.Errorf("headless activate %q: map has no such active control", target)
		}
		return a.HeadlessKey("enter")
	}
	if a.flow.screen == ScreenTown {
		if _, dialogue := townDialogue(a.flow.town); dialogue {
			if !strings.EqualFold(target, "dialogue") && !strings.EqualFold(target, "notice") {
				return fmt.Errorf("headless activate %q: town dialogue is the only active control", target)
			}
			return a.HeadlessKey("enter")
		}
		if world, onMap := townWorldMapScreen(a.flow.town); onMap {
			return a.headlessActivateWorldMap(world, target)
		}
		if surface, onSurface := townSurfaceScreen(a.flow.town); onSurface {
			return a.headlessActivateTownSurface(surface, target)
		}
		if _, ready := townSquareView(a.flow.town); ready {
			return a.headlessChooseTownSquareRow(target)
		}
	}

	var p *Picker
	switch a.flow.screen {
	case ScreenPicker:
		p = a.flow.picker
	case ScreenTown:
		p = a.flow.townList
	case ScreenGameMenu:
		p = a.flow.menuList
	case ScreenLoad:
		p = a.flow.loadList
	default:
		if a.flow.screen == ScreenCutsceneLibrary {
			return a.headlessCutscene(target)
		}
		if a.flow.screen == ScreenEnding {
			return a.headlessEnding(target)
		}
		return fmt.Errorf("headless activate %q: screen %s has no row controls", target, a.flow.screen)
	}
	if p == nil {
		return fmt.Errorf("headless activate %q: screen %s has no list", target, a.flow.screen)
	}
	if strings.EqualFold(target, "@first") {
		for i, row := range p.Rows() {
			if row.Choosable {
				return a.headlessChooseRow(p, i)
			}
		}
		return fmt.Errorf("headless activate @first: screen %s has no choosable row", a.flow.screen)
	}
	match, err := matchPickerRow(p, target, a.flow.screen)
	if err != nil {
		return err
	}
	return a.headlessChooseRow(p, match)
}

// matchPickerRow resolves target against rows' own displayed text: an exact
// case-insensitive match first, then (if none) one unambiguous label prefix
// ending at whitespace, since some production rows append live status to a
// fixed name (for example "GATES     1 mission(s) available"). screen names
// the caller's own screen, for the error text alone. The matched row must
// also be Choosable.
func matchPickerRow(p *Picker, target string, screen Screen) (int, error) {
	match := -1
	for i, row := range p.Rows() {
		if strings.EqualFold(strings.TrimSpace(row.Text), target) {
			if match >= 0 {
				return 0, fmt.Errorf("headless activate %q: ambiguous rows %d and %d", target, match, i)
			}
			match = i
		}
	}
	if match < 0 {
		for i, row := range p.Rows() {
			text := strings.TrimSpace(row.Text)
			if len(text) <= len(target) || !strings.EqualFold(text[:len(target)], target) ||
				!isHeadlessSpace(text[len(target)]) {
				continue
			}
			if match >= 0 {
				return 0, fmt.Errorf("headless activate %q: ambiguous row prefixes %d and %d", target, match, i)
			}
			match = i
		}
	}
	if match < 0 {
		available := make([]string, 0, len(p.Rows()))
		for _, row := range p.Rows() {
			available = append(available, row.Text)
		}
		return 0, fmt.Errorf("headless activate %q: no row on screen %s; available: %q", target, screen, available)
	}
	if !p.Rows()[match].Choosable {
		return 0, fmt.Errorf("headless activate %q: row %d is not choosable", target, match)
	}
	return match, nil
}

// headlessChooseTownSquareRow resolves target against the square's own door
// list and selects it directly — Select then chooseTown — rather than
// through headlessChooseRow's arrow-key simulation, which stepTownAt no
// longer answers once the composed picture is ready (1016 round-2 review).
func (a *App) headlessChooseTownSquareRow(target string) error {
	p := a.flow.townList
	if p == nil {
		return fmt.Errorf("headless activate %q: screen %s has no list", target, a.flow.screen)
	}
	match, err := matchPickerRow(p, target, a.flow.screen)
	if err != nil {
		return err
	}
	p.Select(match)
	a.flow.chooseTown()
	a.syncViewerLayout()
	return nil
}

// headlessActivateTownSurface selects a visible shared-shell cell or button
// through the same mutation door as a pointer release. Before 1006 a tavern
// NPC row opened its conversation directly, while the shared shell deliberately
// splits selection from Talk; the NPC arm below preserves that semantic
// activation without changing what one player click does.
func (a *App) headlessActivateTownSurface(surface TownSurfaceView, target string) error {
	want := target
	type candidate struct {
		label   string
		control TownSurfaceControl
		enabled bool
	}
	candidates := make([]candidate, 0, len(surface.Cells)+len(surface.Buttons))
	for i, cell := range surface.Cells {
		visible, enabled := townSurfaceCellState(surface, i)
		if !visible {
			continue
		}
		label := cell.Semantic
		if label == "" {
			label = cell.Label
		}
		candidates = append(candidates, candidate{label,
			TownSurfaceControl{Kind: TownSurfaceControlCell, Index: i}, enabled})
	}
	for i, button := range surface.Buttons {
		candidates = append(candidates, candidate{button.Label,
			TownSurfaceControl{Kind: TownSurfaceControlButton, Index: i}, button.Enabled})
	}
	match := -1
	for i, c := range candidates {
		if c.label == "" || !strings.EqualFold(strings.TrimSpace(c.label), want) {
			continue
		}
		if match >= 0 {
			return fmt.Errorf("headless activate %q: ambiguous town surface controls %d and %d", target, match, i)
		}
		match = i
	}
	if match < 0 {
		available := make([]string, 0, len(candidates))
		for _, c := range candidates {
			if c.label != "" {
				available = append(available, c.label)
			}
		}
		return fmt.Errorf("headless activate %q: no control on town surface; available: %q", target, available)
	}
	if !candidates[match].enabled {
		return fmt.Errorf("headless activate %q: town surface control is disabled", target)
	}
	chosen := candidates[match]
	if !a.flow.clickTownSurface(chosen.control, false) {
		return fmt.Errorf("headless activate %q: town surface closed before activation", target)
	}
	// The pre-shell scenario vocabulary names an NPC row as one activation,
	// and that row opened the NPC's conversation. Preserve that meaning by
	// carrying the semantic activation through the shell's two visible
	// controls: select the NPC cell, then press its Talk button. A player click
	// remains selection-only; this is solely the meaning of the headless verb.
	if chosen.control.Kind == TownSurfaceControlCell && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(chosen.label)), "NPC ") {
		// Selection is the operation that enables Talk in the owner-directed
		// tavern. Re-read the production surface after that mutation; the view
		// passed into this function is the pre-click frame and deliberately has
		// Talk disabled on entry.
		selected, ok := townSurfaceScreen(a.flow.town)
		if !ok {
			return fmt.Errorf("headless activate %q: town surface closed after selecting NPC", target)
		}
		for i, button := range selected.Buttons {
			if strings.EqualFold(strings.TrimSpace(button.Label), strings.TrimSpace(a.flow.words.TavernTalk)) && button.Enabled {
				if !a.flow.clickTownSurface(TownSurfaceControl{Kind: TownSurfaceControlButton, Index: i}, false) {
					return fmt.Errorf("headless activate %q: town surface closed before Talk", target)
				}
				return nil
			}
		}
		return fmt.Errorf("headless activate %q: selected NPC has no enabled Talk control", target)
	}
	return nil
}

// headlessWorldMapArrivalFrames bounds the frames a headless walk out to a
// mission waits for its travel to arrive, one frame in five carrying a tick. A
// travel that never arrives fails the step instead of stalling the run.
const headlessWorldMapArrivalFrames = 4000

// headlessActivateWorldMap resolves target on the gates screen once it draws
// the campaign world map instead of a row list (1007). roomGates's Rows()
// returns nil there by design, so the Picker path above has no rows to match
// against and a scenario naming a mission fails with "available: []". This
// reads the mission set from the same TownWorldMapScreen seam a player's
// mouse and keyboard use, selects the target mission through the same
// WorldMapMove dispatch Up and Down drive in stepTown (pkg/ui/app.go), never
// by writing selection state directly, and then steps App's own ticks until the
// travel arrives. The walk sends no Enter: what Enter does on the map is not
// settled (DIV-1546), so the walk does not depend on it.
func (a *App) headlessActivateWorldMap(world TownWorldMapScreen, target string) error {
	view := world.WorldMapView()
	available := make([]int, 0, len(view.Missions))
	for _, m := range view.Missions {
		if m.Enabled {
			available = append(available, m.Number)
		}
	}
	number, ok := headlessWorldMapMissionNumber(target)
	if !ok {
		return fmt.Errorf("headless activate %q: town world map has no such control; available missions: %v", target, available)
	}
	want := -1
	for i, m := range view.Missions {
		if m.Enabled && m.Number == number {
			want = i
			break
		}
	}
	if want < 0 {
		return fmt.Errorf("headless activate %q: no mission %d on screen town; available missions: %v", target, number, available)
	}
	for i := 0; i <= len(view.Missions) && view.Selected != want; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			return err
		}
		view = world.WorldMapView()
	}
	if view.Selected != want {
		return fmt.Errorf("headless activate %q: world map selection stopped at index %d, wanted mission %d", target, view.Selected, number)
	}
	for i := 0; i < headlessWorldMapArrivalFrames; i++ {
		if err := a.HeadlessStep(); err != nil {
			return err
		}
		if a.flow.screen != ScreenTown || world.WorldMapView().Selected != want {
			// Arrival ends the selection and opens the mission, or returns the
			// party when it cannot. A window's frame sizes an opened map's
			// camera, and a headless run has no such frame.
			a.syncViewerLayout()
			return nil
		}
	}
	return fmt.Errorf("headless activate %q: the travel to mission %d did not arrive within %d frames", target, number, headlessWorldMapArrivalFrames)
}

// headlessWorldMapMissionNumber extracts the mission number from the two
// forms a headless target uses on the gates screen: the scenario
// vocabulary's historical "walk out to mission N" (pre-1007 gateRows text,
// kept so an existing scenario file needs no edit) and the world map's own
// status text, "travelling to mission N" (the arrival's TownAction.Msg,
// pkg/game/worldmap.go).
func headlessWorldMapMissionNumber(target string) (int, bool) {
	lower := strings.ToLower(strings.TrimSpace(target))
	for _, prefix := range []string{"walk out to mission ", "travelling to mission "} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimSpace(lower[len(prefix):])); err == nil {
			return n, true
		}
	}
	return 0, false
}

// headlessChooseRow reaches the requested row through the same arrow-key
// dispatch as a windowed session, then activates it through the same Enter
// dispatch. It never writes the Picker selection directly.
//
// BOUNDED (round-3 review, W-4), headlessActivateWorldMap's own precedent
// (:382): a screen where Down/Up have gone inert — the town square once art
// is ready is one, headlessChooseTownSquareRow above is what routes around
// it there — would otherwise not move the selection at all, and an unbounded
// "for p.Selection() < target" loops forever rather than failing.
// pipeline/check-scenarios.sh sets no per-scenario timeout, so a scenario
// that reached this loop unbounded would stall the gate rather than report
// it. The bound is one more than the row count: enough to cross the whole
// list from either end, never enough to loop.
func (a *App) headlessChooseRow(p *Picker, target int) error {
	down, up := "down", "up"
	if a.flow != nil && a.flow.screen == ScreenGameMenu && p == a.flow.menuList &&
		(a.flow.menuPage == gameMenuSoundOptionsPage && a.flow.soundOptions.Read != nil ||
			a.flow.menuPage == gameMenuGameOptionsPage && a.flow.gameOptions.Read != nil) {
		down, up = "tab", "shift-tab"
	}
	bound := len(p.Rows()) + 1
	for i := 0; i < bound && p.Selection() < target; i++ {
		if err := a.HeadlessKey(down); err != nil {
			return err
		}
	}
	for i := 0; i < bound && p.Selection() > target; i++ {
		if err := a.HeadlessKey(up); err != nil {
			return err
		}
	}
	if p.Selection() != target {
		return fmt.Errorf("headless row %d: production controller stopped at %d after %d key press(es) "+
			"(the picker's own row bound); Down/Up may be inert on this screen", target, p.Selection(), bound)
	}
	return a.HeadlessKey("enter")
}

func isHeadlessSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

// headlessMenuButton presses one brooch button on the main menu.
//
// THE MAIN MENU IS POINTER-ONLY. stepMenu reads no key except L, so NEW GAME
// was unreachable to a scenario until this existed and every front-end file had
// to start from a save. This is the same press-and-release pair
// HeadlessSelectEntity dispatches, and the same oracle: the point is found by
// asking the production hit test, App.buttonAt, rather than by computing one
// from the placement table. A button whose mask region moved is therefore still
// found, and a button with no region at all is an error rather than a press
// that lands on the brooch background.
func (a *App) headlessMenuButton(target string) error {
	for k, i := range a.flow.modScreenIndexes(false) {
		if strings.EqualFold(strings.TrimSpace(target), a.flow.modUI.screens[i].MenuLabel) {
			x, y, err := a.modEntryPoint(k)
			if err != nil {
				return fmt.Errorf("headless activate %q: %w", target, err)
			}
			a.step(appInput{CursorX: x, CursorY: y, PrimaryPressed: true}, a.headlessAt())
			a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, a.headlessAt())
			return nil
		}
	}
	want, ok := headlessMenuButtons[strings.ToLower(strings.TrimSpace(target))]
	if !ok {
		names := make([]string, 0, len(headlessMenuButtons))
		for name := range headlessMenuButtons {
			names = append(names, name)
		}
		sort.Strings(names)
		return fmt.Errorf("headless activate %q: the menu offers %s",
			target, strings.Join(names, ", "))
	}
	px, py, found := 0, 0, false
	for y := 0; y < a.winH && !found; y++ {
		for x := 0; x < a.winW; x++ {
			if a.buttonAt(x, y) == want {
				px, py, found = x, y, true
				break
			}
		}
	}
	if !found {
		return fmt.Errorf("headless activate %q: this install's brooch has no hit region for it", target)
	}
	a.step(appInput{CursorX: px, CursorY: py, PrimaryPressed: true}, a.headlessAt())
	if a.step(appInput{CursorX: px, CursorY: py, PrimaryReleased: true}, a.headlessAt()) {
		return fmt.Errorf("headless activate %q requested application exit", target)
	}
	return nil
}

// headlessMenuButtons is the scenario vocabulary for the brooch, keyed in lower
// case. The words are the ones the buttons carry in the shipped artwork; the
// indices are menu's own, computed from the placement table rather than spelled
// out here.
var headlessMenuButtons = map[string]int{
	"new game":     menu.NewGameButton,
	"load game":    menu.LoadGameButton,
	"hall of fame": menu.HallOfFameButton,
	"cutscenes":    menu.CutscenesButton,
	"credits":      menu.CreditsButton,
	"exit":         menu.ExitButton,
}

// HeadlessSelectEntity performs a primary-button press and release at a point
// that the production hit test resolves uniquely to id.  It does not write the
// viewer's selection directly.
//
// IT RETRIES, and on a live mission it has to (1005 round 2, seventh pass): the
// press and the release are two frames, and every frame advances the world, so a
// walking unit leaves the pixel the search chose and the press lands on whoever
// walked into it. Selecting one of six party members standing together in
// mission 30 failed this way about half the time. Each attempt re-runs the whole
// search against the positions as they now stand, so a retry is a fresh gesture
// rather than the same one repeated; the attempts are bounded and the last
// mismatch is what the error reports.
func (a *App) HeadlessSelectEntity(id uint32) error {
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		if err = a.headlessSelectEntityOnce(id); err == nil {
			return nil
		}
	}
	return err
}

func (a *App) headlessSelectEntityOnce(id uint32) error {
	if a == nil || a.flow == nil || a.flow.screen != ScreenMap || a.flow.viewer == nil {
		return fmt.Errorf("headless select entity %d: map is not open", id)
	}
	v := a.flow.viewer
	var chosen *screenRect
	for _, e := range v.entities {
		if e.ID != id {
			continue
		}
		r, ok := v.entityPickRect(e)
		if !ok {
			return fmt.Errorf("headless select entity %d: entity is outside the current view", id)
		}
		chosen = &r
		break
	}
	if chosen == nil {
		return fmt.Errorf("headless select entity %d: entity is not present", id)
	}
	// Search the entity's actual pick rectangle because its centre may be
	// covered by a lower-id entity.  The same topAt used by the command path is
	// the oracle; no scenario-only picking rule is introduced.
	x0, x1 := int(math.Floor(chosen.X)), int(math.Ceil(chosen.X+chosen.W))
	y0, y1 := int(math.Floor(chosen.Y)), int(math.Ceil(chosen.Y+chosen.H))
	px, py, found := 0, 0, false
	// A first exposed edge pixel can be covered by a moving neighbour on
	// release. Prefer the interior without pausing either production frame.
	bestDistance := math.Inf(1)
	cx, cy := chosen.X+chosen.W/2, chosen.Y+chosen.H/2
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if got, ok := topAt(v.entities, v.entityPickRect, float64(x), float64(y)); ok && got == id {
				// A downscaled window cannot address every frame pixel. Keep
				// searching inside the same actor for a real pointer location.
				if _, _, ok := v.place.FrameToWindow(image.Pt(x, y)); !ok {
					continue
				}
				dx, dy := float64(x)-cx, float64(y)-cy
				if distance := dx*dx + dy*dy; distance < bestDistance {
					px, py, found, bestDistance = x, y, true, distance
				}
			}
		}
	}
	if !found {
		return fmt.Errorf("headless select entity %d: every visible point is obscured", id)
	}
	// px, py CAME FROM entityPickRect, WHICH IS FRAME SPACE (1026 B4). The two
	// step calls below take window pixels, so the point is converted here by
	// the same placement the production press will be resolved by.
	wx, wy, err := v.frameToWindow(image.Pt(px, py), fmt.Sprintf("headless select entity %d", id))
	if err != nil {
		return err
	}
	press := appInput{CursorX: wx, CursorY: wy, PrimaryPressed: true,
		Viewer: Input{CursorX: wx, CursorY: wy, PrimaryDown: true}}
	a.step(press, a.headlessAt())
	release := appInput{CursorX: wx, CursorY: wy, PrimaryReleased: true,
		Viewer: Input{CursorX: wx, CursorY: wy}}
	a.step(release, a.headlessAt())
	got, ok := v.SelectedUnit()
	if !ok || got != id {
		return fmt.Errorf("headless select entity %d: production controller selected %d (present=%v)", id, got, ok)
	}
	return nil
}

// setAltLetter raises the plain-letter field of a headless Alt chord's key.
func (in *appInput) setAltLetter(letter byte) {
	switch letter {
	case 'b', 'q':
		in.Book = true
	case 'c':
		in.Cast = true
	case 'd':
		in.Defend = true
	case 'e':
		in.SelectAll = true
	case 'f':
		in.Grab = true
	case 'i':
		in.Inventory = true
	case 'j':
		in.Doll = true
	case 'p':
		in.Patrol = true
	}
}
