package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// InspectionWholeMap is the optional seam of a host whose document takes the
// whole-map terrain commands. Each call that changes the map is one edit: one
// undo step, one document; a call that changes nothing answers nil. One seed
// on one map gives one result.
type InspectionWholeMap interface {
	RandomTerrain(seed uint32) (*InspectionDocument, error)
	RandomHeights(seed uint32) (*InspectionDocument, error)
}

// InspectionAreaCopy is the optional seam of a host whose copy also takes the
// terrain under a cell rectangle: the tiles, heights and objects of box with
// the records refs, as one clip, whose paste puts box's top-left cell on the
// target cell. refs may be empty. MoveArea moves the records refs and the
// objects of box by offset cells as one edit: the objects go as a paste of
// box's area puts them, the cells they leave hold none, and the tiles and
// heights stay.
type InspectionAreaCopy interface {
	CopyArea(refs []RecordRef, box image.Rectangle) (InspectionClip, string, error)
	MoveArea(refs []RecordRef, box image.Rectangle, offset image.Point) (SelectionEdit, error)
}

// The starting height a new map offers: 64 by default, 0 to 126 typed
// (TERR-246). A source of another height carries NewMapAltitude and the
// number before any NewMapMultiplayer.
const (
	NewMapAltitude        = "/altitude"
	NewMapDefaultAltitude = 64
	NewMapTopAltitude     = 126
)

// wholeMapHost is the document's whole-map seam while it can be edited.
func (e *MapEditor) wholeMapHost() (InspectionWholeMap, bool) {
	if e.doc == nil || e.catalog || !e.editable() {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionWholeMap)
	return h, ok
}

// keyRandom opens the seed prompt of a whole-map command, holding the seed
// after the last one this window used, 1 at first.
func (e *MapEditor) keyRandom(key string) error {
	if e.doc == nil || e.catalog {
		return nil
	}
	if _, ok := e.wholeMapHost(); !ok {
		e.noteSelection(e.word("editor.random.nohost"), nil)
		return nil
	}
	e.leave(leaveDrag | leavePick)
	e.openPrompt(editorPrompt{kind: promptSeed, scope: key, path: strconv.FormatUint(uint64(e.lastSeed+1), 10)})
	return nil
}

// applyRandom runs the prompt's command with the typed seed, a whole number
// from 0 to 4294967295, and names the seed on the status line.
func (e *MapEditor) applyRandom(text string) error {
	key, typed := e.prompt.scope, strings.TrimSpace(text)
	n, err := strconv.ParseUint(typed, 10, 32)
	e.leave(leavePrompt)
	if err != nil {
		e.message = fmt.Sprintf(e.word("editor.random.badseed"), typed)
		return nil
	}
	if _, ok := e.wholeMapHost(); !ok {
		return nil
	}
	seed := uint32(n)
	done := e.word("editor.random.terraindone")
	if key == "randomheights" {
		done = e.word("editor.random.heightsdone")
	}
	if _, err := e.commandRandomize(wholeMap{Heights: key == "randomheights", Seed: seed}); err != nil {
		e.message = err.Error()
		return err
	}
	e.lastSeed = seed
	e.noteSelection(fmt.Sprintf(done, seed), nil)
	return nil
}

// seedTitle is the seed prompt's title.
func (e *MapEditor) seedTitle() string {
	if e.prompt.scope == "randomheights" {
		return e.word("editor.prompt.randomheights")
	}
	return e.word("editor.prompt.randomterrain")
}

// newAltitude is the starting height typed into the new-map question, and
// whether it is a whole number from 0 to NewMapTopAltitude.
func newAltitude(text string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	return n, err == nil && n >= 0 && n <= NewMapTopAltitude
}

// newMapAltitudeSource is source with the starting height h; the default
// height leaves source as it is.
func newMapAltitudeSource(source string, h int) string {
	if h == NewMapDefaultAltitude {
		return source
	}
	return source + NewMapAltitude + strconv.Itoa(h)
}
