package ui

import (
	"math/bits"
	"time"
)

// CoverKind names a screen shown over a running mission: the campaign
// documents panel and the town's Shop page. One row of coverRows describes
// each, and every raise, open and close goes through the record below.
type CoverKind uint8

const (
	// CoverDocuments is the campaign documents panel.
	CoverDocuments CoverKind = iota
	// CoverShop is the town's Shop page opened by a Shop structure use.
	CoverShop
	coverKinds
)

// coverRow is what one screen over the mission is: its screen, whether the map
// is held under it (stepped at the held cadence) or stands, how it is built and
// how it is released, and whether the seam, not the player, ends it.
type coverRow struct {
	shows Screen
	// holdsMap says the map is held under the screen like the in-game menu
	// holds it. A screen that does not stand the map: it takes no tick.
	holdsMap bool
	// build prepares the screen and reports whether it may show.
	build func(f *flow) bool
	// release drops what build made.
	release func(f *flow)
	// clearsMsgOnClose says the close clears the flow message; the open
	// always does.
	clearsMsgOnClose bool
	// shown, when set, is how a close is found: the flow polls it and closes
	// once it turns false. A row without it is closed by its own OK or Escape.
	shown func(f *flow) bool
}

var coverRows = [coverKinds]coverRow{
	CoverDocuments: {
		shows:    ScreenDocuments,
		holdsMap: true,
		build: func(f *flow) bool {
			if f.docSrc == nil {
				return false
			}
			pages := f.docSrc.Documents()
			if len(pages) == 0 {
				return false
			}
			f.docPanel = newDocumentPanel(f.docArt, f.docFont, pages)
			f.setDocUp(true)
			return true
		},
		release: func(f *flow) {
			f.docPanel = nil
			f.setDocUp(false)
		},
		clearsMsgOnClose: true,
	},
	CoverShop: {
		shows: ScreenTown,
		build: func(f *flow) bool {
			s, ok := f.town.(MissionShopScreen)
			if f.screen != ScreenMap || !ok || !s.OpenMissionShop() {
				return false
			}
			f.refreshTown()
			return true
		},
		release: func(*flow) {},
		shown:   func(f *flow) bool { return missionShopOpen(f.town) },
	},
}

// coverShown is the screen over the mission now showing, if any, and the
// screen its close returns to.
type coverShown struct {
	kind CoverKind
	back Screen
	up   bool
}

// openCover shows the row's screen over back. It refuses when the row's build
// does, and the player stays where he was.
func (f *flow) openCover(kind CoverKind, back Screen) bool {
	row := &coverRows[kind]
	if !row.build(f) {
		return false
	}
	f.cover = coverShown{kind: kind, back: back, up: true}
	f.setScreen(row.shows)
	f.msg = ""
	return true
}

// closeCover drops the showing screen and returns to its back screen.
func (f *flow) closeCover() {
	row := &coverRows[f.cover.kind]
	row.release(f)
	f.cover.up = false
	if f.screen == row.shows {
		f.setScreen(f.cover.back)
	}
	if row.clearsMsgOnClose {
		f.msg = ""
	}
}

// settleCover closes a showing screen whose seam has ended it.
func (f *flow) settleCover() {
	row := &coverRows[f.cover.kind]
	if !f.cover.up || row.shown == nil || row.shown(f) {
		return
	}
	f.closeCover()
}

// coverHoldsMap reports a screen over the mission that holds the map under
// it, over back.
func (f *flow) coverHoldsMap(back Screen) bool {
	row := &coverRows[f.cover.kind]
	return f.screen == row.shows && row.holdsMap && f.cover.back == back
}

// holdMapUnderCover holds the map under the showing screen when its row says so.
func (a *App) holdMapUnderCover(in appInput, now time.Time) {
	c := a.flow.cover
	if coverRows[c.kind].holdsMap {
		a.holdMapUnder(c.back, in, now)
	}
}

// RaiseCover asks the front end to show the row's screen over the mission. A
// standing request of the same kind is overwritten, never queued.
func (v *Viewer) RaiseCover(kind CoverKind) {
	if v != nil && kind < coverKinds {
		v.coverRequests |= 1 << kind
	}
}

// TakeCoverRequest reads the lowest-numbered standing request and clears it in
// the same statement.
func (v *Viewer) TakeCoverRequest() (CoverKind, bool) {
	if v == nil || v.coverRequests == 0 {
		return 0, false
	}
	kind := CoverKind(bits.TrailingZeros8(v.coverRequests))
	v.coverRequests &^= 1 << kind
	return kind, true
}
