package main

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/ui"
)

// documentsDriveW and documentsDriveH are the window driveSession declares
// before it opens anything. Every other stage composes a fixed-size screen and
// needs no window at all; this one runs through hit tests over a PLACED mission
// frame, and a mission's own start view is applied when the mission opens, so
// the size has to be known before that rather than after.
const documentsDriveW, documentsDriveH = 1600, 1200

// driveDocuments opens the campaign documents panel over the mission screen
// this drive has already reached, so the panel can be seen composed against
// a real install's own sheet, arrows, OK plate and font4 text.
//
// IT USES NO SEAM OF ITS OWN. Every step is the production dispatch a played
// session produces: the hero is selected through the production pick test, and
// the panel is opened by double-clicking the access item in the pack bar —
// App.step's own pointer edges, the same four this story's headless scenario
// sends. A screen reached any other way would not witness the wiring that
// reaches it.
//
// A NOTICE IS DISMISSED WHEREVER ONE IS FOUND, and never waited for. A popup
// takes every map-screen input (popup.go), so a double-click delivered across
// the frame one is raised on is swallowed with nothing to show for it; but
// which mission this drive reaches is the campaign's own first picker row, and
// whether it raises a notice at all is that mission's business. Stepping until
// one appears advances the world hundreds of ticks on a mission that raises
// none, which moves the party out of the view the selection needs.
func driveDocuments(f *game.FrontEnd, a *ui.App) error {
	if a.Screen() != ui.ScreenMap {
		return fmt.Errorf("the documents panel is opened over the mission screen, and this drive is on %s", a.Screen())
	}
	if err := dismissNotice(a); err != nil {
		return err
	}

	state := f.HeadlessSnapshot(a.Screen())
	entity := uint32(0)
	for _, m := range state.Members {
		if m.Membership == "primary" && m.Entity != 0 {
			entity = m.Entity
			break
		}
	}
	if entity == 0 {
		return fmt.Errorf("this mission's party has no primary with an entity")
	}
	if err := a.HeadlessSelectEntity(entity); err != nil {
		return err
	}
	// The selection reaches the inventory's own subject on a later frame than
	// the one the press landed on, so the pack bar is read after a short
	// bounded advance rather than on the next statement.
	for n := 0; n < 8; n++ {
		if err := a.HeadlessStep(); err != nil {
			return err
		}
		if f.HeadlessInventory() != nil {
			break
		}
	}

	inv := f.HeadlessInventory()
	if inv == nil {
		return fmt.Errorf("the map screen reports no inventory subject after selecting entity %d", entity)
	}
	cell := -1
	for i, code := range inv.Carried {
		if data.RaisesDocuments(data.ItemCode(code)) {
			cell = i
			break
		}
	}
	if cell < 0 {
		return fmt.Errorf("the selected hero carries no access item (pack holds %v)", inv.Carried)
	}

	// TWO ATTEMPTS, because a notice raised by the world's own advance
	// between the first press and the second release swallows the whole
	// gesture. The second attempt runs after that notice has been dismissed.
	for attempt := 0; attempt < 2; attempt++ {
		if err := dismissNotice(a); err != nil {
			return err
		}
		x, y, err := a.HeadlessPackCellPoint(cell)
		if err != nil {
			return err
		}
		for _, action := range []string{"press", "release", "press", "release"} {
			if err := a.HeadlessPointer(action, x, y); err != nil {
				return err
			}
		}
		if err := a.HeadlessStep(); err != nil {
			return err
		}
		if a.Screen() == ui.ScreenDocuments {
			return nil
		}
	}
	return fmt.Errorf("the double-click on the access item left screen %s, not the documents panel", a.Screen())
}

// dismissNotice closes a notice standing over the map, if one is open.
func dismissNotice(a *ui.App) error {
	if !a.HeadlessNoticeOpen() {
		return nil
	}
	return a.HeadlessActivate("notice")
}
