package ui

import "image"

// playerCount is how many players the open map has: its player records.
func (e *MapEditor) playerCount() int {
	n := 0
	if e.doc != nil {
		for _, r := range e.doc.Records {
			if r.Kind == KindPlayer {
				n = max(n, r.Index+1)
			}
		}
	}
	return min(n, len(editorPlayerTints))
}

// playerTint is the sprite tint the map view applies while the player colours
// are on: the colour of the player that owns a unit or structure of the open
// map, and none for an object of owner 0 or of an owner the map has no player
// for. It reads the records as they are now, so a move, an undo or a redo
// keeps the tint with the object. Off, it is nil. While a player record is
// selected only that player's objects answer, and with the brighter focus tint;
// every other object answers none.
func (e *MapEditor) playerTint() func(InspectionSubject) spriteTint {
	if !e.view.playerColours || e.doc == nil {
		return nil
	}
	players := e.playerCount()
	owners := map[InspectionSubject]int{}
	for _, r := range e.doc.Records {
		switch r.Kind {
		case KindUnit:
			owners[InspectionSubject{Kind: InspectionUnit, ID: uint32(r.Index)}] = r.Player
		case KindStructure:
			owners[InspectionSubject{Kind: InspectionStructure, ID: uint32(r.Index)}] = r.Player
		}
	}
	focus := e.focusedPlayer()
	return func(s InspectionSubject) spriteTint {
		owner := owners[s]
		if owner < 1 || owner > players || (focus != 0 && owner != focus) {
			return spriteTint{}
		}
		return spriteTint{Color: editorPlayerTints[owner-1], On: true, Focus: focus != 0}
	}
}

// focusedPlayer is the number of the selected player record, or 0 when the
// selection is not a player.
func (e *MapEditor) focusedPlayer() int {
	if kind, index := e.selectedRecord(); kind == KindPlayer && index >= 0 {
		return index + 1
	}
	return 0
}

// applyPlayerTint hands the map view's sprite painter the tint of the moment,
// or none while the player colours are off.
func (e *MapEditor) applyPlayerTint() {
	if e.doc != nil {
		e.doc.Viewer.SetSubjectTint(e.playerTint())
	}
}

// keyPlayerTint turns the player colours on or off.
func (e *MapEditor) keyPlayerTint(string) error {
	e.view.playerColours = !e.view.playerColours
	return nil
}

func (e *MapEditor) legendSpec() (popupSpec, bool) {
	if !e.view.playerColours || e.doc == nil {
		return popupSpec{}, false
	}
	var labels [len(editorPlayerTints)]string
	var have [len(editorPlayerTints)]bool
	for _, r := range e.doc.Records {
		if r.Kind == KindPlayer && r.Index >= 0 && r.Index < len(labels) {
			labels[r.Index], have[r.Index] = r.Label, true
		}
	}
	focus := e.focusedPlayer()
	var rows []popupRow
	for i, label := range labels {
		if have[i] {
			rows = append(rows, playerSwatchRow(i, focus, label))
		}
	}
	if len(rows) == 0 {
		return popupSpec{}, false
	}
	s := playerLegendSpec(e.mapArea(), rows, editorMeasure(e.bodyFont()), *e.bodyPage(popLegend))
	return s, !s.Bounds.Empty()
}

func (e *MapEditor) drawPlayerLegend(pic *image.RGBA) {
	if s, ok := e.legendSpec(); ok {
		e.drawWindow(pic, s)
	}
}
