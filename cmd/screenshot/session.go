package main

import (
	"fmt"
	"os"

	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// driveSession drives one live ui.App through the menu/load/picker/chargen/
// gameplay/game-menu sequence and returns whatever frame each stage
// produced. It always runs the whole sequence — see capture's own comment
// in main.go for why — and stops early only when a stage this build expects
// to leave does not, recording the reason for every screen downstream of it.
func driveSession(f *game.FrontEnd, root string) map[string]capturedFrame {
	out := map[string]capturedFrame{}

	scratch, err := os.MkdirTemp("", "screenshot-saves")
	if err != nil {
		return failAll(out, err, "menu", "load", "picker", "chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	defer os.RemoveAll(scratch)

	a := f.App("screenshot")
	// The window, declared before any screen is opened. See documentsDriveW.
	a.Layout(documentsDriveW, documentsDriveH)
	a.SetSaveSeams(f.SaveSeams(game.SaveStore{Dir: scratch}, game.OriginalStore{Dir: root}, nil))

	out["menu"] = snap(a)

	if err := a.HeadlessKey("load"); err != nil {
		return failAll(out, err, "load", "picker", "chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	out["load"] = snap(a)
	if err := a.HeadlessKey("escape"); err != nil {
		return failAll(out, err, "picker", "chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}

	if err := a.HeadlessActivate("new game"); err != nil {
		return failAll(out, err, "picker", "chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	out["picker"] = snap(a)

	if err := a.HeadlessActivate("@first"); err != nil {
		return failAll(out, fmt.Errorf("activate the first map row: %w", err),
			"chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	if a.Screen() != ui.ScreenChargen {
		return failAll(out, fmt.Errorf("the first row opened %s, not the generation screen (this campaign's own first row is not a mission)", a.Screen()),
			"chargen-precreate", "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	out["chargen-precreate"] = snap(a)

	state, ok := a.HeadlessChargenState()
	if !ok {
		return failAll(out, fmt.Errorf("HeadlessChargenState answered false right after opening the generation screen"),
			"chargen-detailed", "gameplay", "documents", "game-menu")
	}
	pictures := state.Labels(ui.ChargenControlPicture)
	if len(pictures) == 0 {
		return failAll(out, fmt.Errorf("the pre-create page offers no picture control"),
			"chargen-detailed", "gameplay", "documents", "game-menu")
	}
	if _, err := chargenPress(a, ui.ChargenControlPicture, pictures[0]); err != nil {
		return failAll(out, err, "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	state, err = chargenPress(a, ui.ChargenControlForward, "")
	if err != nil {
		return failAll(out, err, "chargen-detailed", "gameplay", "documents", "game-menu")
	}
	if a.Screen() != ui.ScreenChargen || state.Stage != ui.ChargenStageDetailed {
		return failAll(out, fmt.Errorf("forward left screen %s stage %q, not the detailed page", a.Screen(), state.Stage),
			"chargen-detailed", "gameplay", "documents", "game-menu")
	}
	out["chargen-detailed"] = snap(a)

	if _, err := chargenPress(a, ui.ChargenControlPlay, ""); err != nil {
		return failAll(out, err, "gameplay", "documents", "game-menu")
	}
	if a.Screen() != ui.ScreenMap {
		return failAll(out, fmt.Errorf("play left screen %s, not the mission screen: %s", a.Screen(), a.HeadlessMessage()),
			"gameplay", "game-menu")
	}
	out["gameplay"] = snap(a)

	// A SUBJECT, so the pane is photographed carrying one. This selects the
	// first live party member by the same press the command path resolves
	// (HeadlessSelectEntity), off the front end's own snapshot, and records the
	// failure in the note rather than stopping: a mission whose party has no
	// live entity is still worth photographing.
	paneNote := ""
	if members := f.LiveHeadlessSnapshot().Members; len(members) > 0 {
		picked := false
		for _, m := range members {
			if m.Entity == 0 {
				continue
			}
			if err := a.HeadlessSelectEntity(m.Entity); err != nil {
				paneNote = "no subject: " + err.Error()
			} else {
				paneNote = "subject " + m.Name
				picked = true
			}
			break
		}
		if !picked && paneNote == "" {
			paneNote = "no subject: no party member has a live entity"
		}
	} else {
		paneNote = "no subject: the front end reports no party"
	}
	// These captures describe the selected party member. A selection click
	// can leave the pointer over a different opaque sprite, whose hover card
	// correctly takes precedence in play. Leave the map before both modes.
	if err := a.HeadlessPointer("hover", -1, -1); err != nil {
		return failAll(out, err, "mission-pane-doll", "mission-column-card", "mission-pane-stats", "documents", "game-menu")
	}

	// Both mission panels are visible together. Tab no longer swaps the
	// figure for statistics; retain the old capture name as a card alias.
	out["mission-pane-doll"] = paneSnap(a, paneNote)
	out["mission-column-card"] = cardSnap(a, paneNote)
	out["mission-pane-stats"] = out["mission-column-card"]

	// THE DOCUMENTS PANEL COMES BEFORE THE IN-GAME MENU, because it leaves the
	// mission screen exactly as it found it: its own Escape returns to
	// ScreenMap, so the menu stage below still opens over a running mission.
	// Reaching it consumes mission frames and dismisses this mission's first
	// notice, neither of which any later stage reads.
	if err := driveDocuments(f, a); err != nil {
		out["documents"] = capturedFrame{err: err}
	} else {
		out["documents"] = snap(a)
		if err := a.HeadlessKey("escape"); err != nil {
			return failAll(out, err, "game-menu")
		}
		if a.Screen() != ui.ScreenMap {
			return failAll(out, fmt.Errorf("escape over the documents panel left screen %s, not the mission screen", a.Screen()), "game-menu")
		}
	}

	if err := a.HeadlessKey("escape"); err != nil {
		return failAll(out, err, "game-menu")
	}
	if a.Screen() != ui.ScreenGameMenu {
		return failAll(out, fmt.Errorf("escape over a running mission left screen %s, not the in-game menu", a.Screen()), "game-menu")
	}
	out["game-menu"] = snap(a)

	return out
}

// paneSnap captures the mission character pane through HeadlessCharacterPane,
// which composes it by the same call Viewer.Draw makes. It records the mode the
// pane reported rather than the mode this tool asked for, so a Tab that did not
// arrive shows up as two identical notes instead of two identical pictures
// under different names.
// cardSnap photographs the mission column's own fourth box.
func cardSnap(a *ui.App, subject string) capturedFrame {
	pix, err := a.HeadlessMissionCard()
	if err != nil {
		return capturedFrame{err: err, tried: true}
	}
	note := "the column's fourth box, beside the pane's figure"
	if subject != "" {
		note += ", " + subject
	}
	return capturedFrame{pix: pix, note: note, tried: true}
}

func paneSnap(a *ui.App, subject string) capturedFrame {
	pix, statistics, err := a.HeadlessCharacterPane()
	mode := "figure mode"
	if statistics {
		mode = "statistics mode"
	}
	if err != nil {
		return capturedFrame{err: err, tried: true}
	}
	note := "the pane reported " + mode
	if subject != "" {
		note += ", " + subject
	}
	return capturedFrame{pix: pix, note: note, tried: true}
}

// snap captures a's current screen through HeadlessFrame, the one seam this
// tool and pkg/ui/app.go's Draw both go through (see main.go's own header).
func snap(a *ui.App) capturedFrame {
	pix, note, err := a.HeadlessFrame()
	return capturedFrame{pix: pix, note: note, err: err, tried: true}
}

// chargenPress walks the generation screen's focus to the control named by
// kind and label with its own up/down keys, then presses Enter — the same
// production dispatch a keyboard session uses (ui.App.HeadlessKey). It is
// this tool's own copy of pkg/game's unexported headlessChargenFocus/
// headlessChargenPress, built from HeadlessChargenState's exported
// vocabulary rather than by importing pkg/game's scenario driver, since
// that driver also asserts the spread it builds is legal and this tool
// only needs to reach a page, not build a specific character.
func chargenPress(a *ui.App, kind, label string) (ui.HeadlessChargen, error) {
	state, ok := a.HeadlessChargenState()
	if !ok {
		return state, fmt.Errorf("chargen press %s %q: not on the generation screen (screen is %s)", kind, label, a.Screen())
	}
	control, ok := state.Control(kind, label)
	if !ok {
		return state, fmt.Errorf("chargen press: the %s page offers no %s %q (offers %v)",
			state.Stage, kind, label, state.Labels(kind))
	}
	bound := len(state.Controls) + 2
	for n := 0; state.Focus != control.Focus; n++ {
		if n > bound {
			return state, fmt.Errorf("chargen press: focus stuck at %d short of %s %q at %d",
				state.Focus, kind, label, control.Focus)
		}
		key := "down"
		if state.Focus > control.Focus {
			key = "up"
		}
		if err := a.HeadlessKey(key); err != nil {
			return state, err
		}
		if state, ok = a.HeadlessChargenState(); !ok {
			return state, fmt.Errorf("chargen press: left the generation screen mid-walk (screen is %s)", a.Screen())
		}
	}
	if err := a.HeadlessKey("enter"); err != nil {
		return state, err
	}
	if next, ok := a.HeadlessChargenState(); ok {
		state = next
	}
	return state, nil
}

// failAll records err against every name in names and returns out, for the
// case a stage this drive needed did not produce the state the next stage
// depends on. Every name downstream of a broken stage is unreached, not
// silently skipped: a caller asking for "gameplay" alone still learns why
// the drive that would have reached it stopped three stages earlier.
func failAll(out map[string]capturedFrame, err error, names ...string) map[string]capturedFrame {
	for _, name := range names {
		out[name] = capturedFrame{err: err}
	}
	return out
}
