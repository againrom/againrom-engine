package ui

import (
	"image"
	"image/draw"
)

// The mission command panel (docs/1028-command-panel spec; contract B1-B5).
//
// IT REPLACES hudtoggles.go's FOUR AUTHORED SWITCHES IN THE SAME SLOT. What
// moves out of the slot is only the drawn box: `TOWN-092` gives the
// original's own 4x2 grid at that same slot, `MENU-COMBAT-018` its four
// shipped bitmaps, `MENU-COMBAT-019` its vocabulary and mouse edges, and
// `AI-PANEL-053` what each cell does.
//
// All eight cells are enabled. Retreat issues immediately; Defend and Cast
// arm.

// commandPanelCell names one of the panel's eight cells, in `TOWN-092`'s own
// row-major order and `MENU-COMBAT-019`'s own vocabulary: Attack, Move,
// Guard, Defend, Cast, Swarm, Stand Ground, Retreat.
type commandPanelCell int

const (
	commandCellAttack commandPanelCell = iota
	commandCellMove
	commandCellGuard
	commandCellDefend
	commandCellCast
	commandCellSwarm
	commandCellStandGround
	commandCellRetreat
	commandPanelCellCount
)

// The decoded skip mask remains a seam; no missing order disables a cell now.
const commandPanelSkipMask = 0

// commandPanelCellSkipped refuses an invalid or explicitly skipped cell.
func commandPanelCellSkipped(cell commandPanelCell) bool {
	if cell < 0 || cell >= commandPanelCellCount {
		return true
	}
	return commandPanelSkipMask&(1<<uint(cell)) != 0
}

// commandCellSize and commandCellPitch are `TOWN-092`'s own 34x34 cells at
// pitch 34, and commandCellOriginX/Y its own margin: cell i stands at
// `((i&3)*34+8, (i>>2)*34+7)` panel-local, four columns by two rows.
const (
	commandCellSize    = 34
	commandCellPitch   = 34
	commandCellOriginX = 8
	commandCellOriginY = 7
)

// commandCellRects is one rectangle per cell, in `commandPanelCell` order, in
// the same window-pixel space `bar` is stated in — `TOWN-092`'s own formula,
// carried unchanged from panel-local into the bar's own placement.
func commandCellRects(bar image.Rectangle) [commandPanelCellCount]image.Rectangle {
	var out [commandPanelCellCount]image.Rectangle
	for i := range out {
		col, row := i&3, i>>2
		x := bar.Min.X + commandCellOriginX + col*commandCellPitch
		y := bar.Min.Y + commandCellOriginY + row*commandCellPitch
		out[i] = image.Rect(x, y, x+commandCellSize, y+commandCellSize)
	}
	return out
}

// CommandPanelArt is the panel's four shipped 160x80 bitmaps
// (`MENU-COMBAT-018`) plus the inactive and active grounds' 16x80 closing
// seams. None of the four bodies is a separate per-cell icon — the original
// draws a REGION of one of these four for every cell. The seams never take
// part in cell composition; they close the body's left edge over the map.
type CommandPanelArt struct {
	Heads      image.Image
	HeadsSeam  image.Image
	Active     image.Image
	ActiveSeam image.Image
	Disabled   image.Image
	Selected   image.Image
}

// commandPanelActive is the panel's own gate (contract B2: "The gate is
// ownership of the selection and nothing else"), and it is `canArmAttack`'s
// own two clauses restated for the panel rather than a second computation of
// them: a selection present and alive, and its primary owned by the local
// participant, or no local participant established at all. `AI-PANEL-053`'s
// retracted capability clause leaves ownership as the whole gate for every
// cell but Cast, which carries its own further clause below
// (`commandCastActive`).
func commandPanelActive(cur selection, ents []MapEntity, localOwner uint32) bool {
	return canArmAttack(cur, ents, localOwner)
}

// AI-SPELLCAP-288, DIV-336, AI-PANEL-158
func (v *Viewer) commandCastActive() bool {
	return v.selectionSummary()&selSummarySpell != 0
}

// SetCommandPanelArt adopts the panel's resolved art, or nil for none
// supplied — the developer viewer's state and the state of a front end whose
// install would not yield all four bitmaps (`LoadCommandPanelArt`'s own
// all-or-nothing contract).
func (v *Viewer) SetCommandPanelArt(art *CommandPanelArt) { v.commandPanelArt = art }

// commandPanelBar is where the panel stands, in the same window pixels
// hudToggleBarRect always has — the slot this story does not move.
func (v *Viewer) commandPanelBar() (image.Rectangle, bool) {
	return hudToggleBarRect(image.Pt(v.frameW, v.frameH))
}

// commandPanelCaptures reports that the panel's own slot stands under the
// window pixel (x, y) — hudToggleCaptures' own shape, restated for this
// panel: the whole box, so a press two pixels off a cell does nothing rather
// than ordering the party across the world underneath it.
func (v *Viewer) commandPanelCaptures(x, y int) bool {
	bar, ok := v.commandPanelBar()
	return ok && image.Pt(x, y).In(bar)
}

// commandCellAt reports which cell, if any, stands under the window pixel
// (x, y) — hudTogglePanelAt's own shape, restated for eight cells rather
// than four.
func (v *Viewer) commandCellAt(x, y int) (commandPanelCell, bool) {
	bar, ok := v.commandPanelBar()
	if !ok {
		return 0, false
	}
	p := image.Pt(x, y)
	for i, r := range commandCellRects(bar) {
		if p.In(r) {
			return commandPanelCell(i), true
		}
	}
	return 0, false
}

// commandPanelSelected is the cell the panel draws its own selected overlay
// over, derived from the live armed state — Attack, the panel's own Move
// arm, or armed Cast — unless `cmdOverlayHidden` says to draw
// none (below).
func commandPanelSelected(v *Viewer) (commandPanelCell, bool) {
	if v.cmdOverlayHidden {
		return 0, false
	}
	switch {
	case v.armed:
		return commandCellAttack, true
	case v.aimed == commandMove:
		return commandCellMove, true
	case v.aimed == commandSwarm:
		return commandCellSwarm, true
	case v.aimed == commandDefend:
		return commandCellDefend, true
	case v.spellModeLive() || v.itemCast != nil:
		return commandCellCast, true
	}
	return 0, false
}

// pressCommandPanelCell is one press or one drag re-entry naming cell
// (contract B2).
//
// A MARGIN OR A DISABLED CELL — cellOK false, or the cell skipped, or the
// panel is not active — CLEARS ONLY THE OVERLAY (`cmdOverlayHidden = true`)
// AND ARMS NOTHING, leaving whatever mode was already armed intact. This is
// `TOWN-092`'s own stored, signed selected-index field read alongside
// `MENU-COMBAT-018`'s account of a margin or disabled left-down: the drawn
// selection and the armed mode are two different pieces of state, and this
// build keeps them apart in `cmdOverlayHidden` and `v.armed`/`v.aimed`/
// `v.spellArmed` for the same reason.
//
// GUARD AND STAND GROUND ISSUE IMMEDIATELY, onto a one-shot pending flag
// consumed once by app.go's own stance dispatch — the same seam the two
// keyboard bindings already call, so a mouse press and a key press reach
// `a.flow.stance` through the identical statement (`AI-PANEL-053`: buttons 3
// and 7 issue immediately).
//
// ATTACK AND MOVE ARM (`AI-PANEL-053`: buttons 1 and 2, always arm). Attack
// reuses `armAttack` unchanged; Move raises `commandMove`, an armed aimed
// order this story adds beside `commandPatrol` and `commandSwarm` for the
// original's own armed-mode table entry 2 — `decide` resolves it to a plain
// move at the order boundary (command.go), so nothing downstream of this
// package learns a third command byte exists.
//
// CAST ARMS CAST MODE AND CHOOSES NO SPELL (`MENU-055`): the press and the C
// key reach the same arm. The spell selected in the book stays as it was, and
// a click with none selected orders nothing.
//
// AI-SPELLCAP-288
func (v *Viewer) pressCommandPanelCell(cell commandPanelCell, cellOK bool) {
	// Message 0x40c plays click04 on dispatch, before its disabled/skip gates.
	v.PlayUISound(UISoundCommandPanel)
	if !cellOK || commandPanelCellSkipped(cell) || !commandPanelActive(v.sel, v.entities, v.localOwner) ||
		(cell == commandCellCast && !v.commandCastActive()) {
		v.cmdOverlayHidden = true
		return
	}
	v.cmdOverlayHidden = false
	switch cell {
	case commandCellAttack:
		v.armAttack()
	case commandCellMove:
		v.armCommand(commandMove)
	case commandCellDefend:
		v.armDefend()
	case commandCellSwarm:
		// `AI-PANEL-123` cell 5: armed mode 6, click opcode `0x1a`. It arms
		// exactly the way Move does one case up, and `commandSwarm` is the
		// same aimed order the S key raises.
		v.armCommand(commandSwarm)
	case commandCellGuard:
		v.cmdPendingGuard = true
	case commandCellStandGround:
		v.cmdPendingStandGround = true
	case commandCellRetreat:
		v.issuePlayerRetreat()
	case commandCellCast:
		// The same arm as the C key: Cast mode, no spell chosen, and a closed
		// book opens.
		v.castKey()
	}
}

// consumeCommandStancePending answers and clears the two one-shot flags a
// Guard or Stand Ground cell press leaves for app.go's own stance dispatch,
// so each press reaches `a.flow.stance` exactly once.
func (v *Viewer) consumeCommandStancePending() (guard, standGround bool) {
	guard, standGround = v.cmdPendingGuard, v.cmdPendingStandGround
	v.cmdPendingGuard, v.cmdPendingStandGround = false, false
	return guard, standGround
}

// composeCommandPanel paints the panel and returns it: `MENU-COMBAT-018`'s
// own draw order — the inactive ground alone, or the active ground with
// every disabled cell's own rectangle from the disabled bitmap and the
// selected cell's own rectangle from the selected bitmap, each cut from the
// SAME panel-sized picture rather than from a separate per-cell asset.
//
// A NIL ART DRAWS NOTHING: there is no fallback drawing, because there is no
// authored appearance for a panel with no shipped bitmaps behind it — see
// `commandPanelPresent`.
func composeCommandPanel(v *Viewer, bar image.Rectangle) *image.RGBA {
	art := v.commandPanelArt
	size := bar.Size()
	body := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	full := image.Rect(0, 0, size.X, size.Y)
	var seam image.Image

	if !commandPanelActive(v.sel, v.entities, v.localOwner) {
		if art.Heads != nil {
			draw.Draw(body, full, art.Heads, art.Heads.Bounds().Min, draw.Src)
		}
		seam = art.HeadsSeam
	} else {
		if art.Active != nil {
			draw.Draw(body, full, art.Active, art.Active.Bounds().Min, draw.Src)
		}
		if art.Disabled != nil {
			for i, r := range commandCellRects(bar) {
				if !commandPanelCellSkipped(commandPanelCell(i)) {
					continue
				}
				local := r.Sub(bar.Min)
				draw.Draw(body, local, art.Disabled, local.Min.Add(art.Disabled.Bounds().Min), draw.Src)
			}
		}
		if art.Selected != nil {
			if cell, ok := commandPanelSelected(v); ok {
				r := commandCellRects(bar)[cell].Sub(bar.Min)
				draw.Draw(body, r, art.Selected, r.Min.Add(art.Selected.Bounds().Min), draw.Src)
			}
		}
		seam = art.ActiveSeam
	}
	if seam == nil {
		return body
	}
	seamW := seam.Bounds().Dx()
	img := image.NewRGBA(image.Rect(0, 0, seamW+size.X, size.Y))
	draw.Draw(img, image.Rect(0, 0, seamW, size.Y), seam, seam.Bounds().Min, draw.Over)
	draw.Draw(img, image.Rect(seamW, 0, seamW+size.X, size.Y), body, body.Bounds().Min, draw.Src)
	return img
}

// commandPanelPresent is the panel's picture for this frame and where its
// own top-left corner goes, or false for a window with no room for one, or
// no art ever resolved.
//
// IT RECOMPOSES EVERY FRAME IT DRAWS: eight cells and at most two overlays
// costs less to redo than a rebuild key carrying the selection, the armed
// state and the window size would cost to keep honest.
func (v *Viewer) commandPanelPresent() (*image.RGBA, image.Point, bool) {
	if v.commandPanelArt == nil {
		return nil, image.Point{}, false
	}
	bar, ok := v.commandPanelBar()
	if !ok {
		return nil, image.Point{}, false
	}
	pic := composeCommandPanel(v, bar)
	at := bar.Min
	at.X -= pic.Bounds().Dx() - bar.Dx()
	return pic, at, true
}

// commandCellLabel is cell's own vocabulary word, from the resolved install
// word set — main.txt lines 0..7, `MENU-COMBAT-019` — or "" for a viewer
// with no words resolved.
func commandCellLabel(v *Viewer, cell commandPanelCell) string {
	if cell < 0 || int(cell) >= len(v.words.Command) {
		return ""
	}
	return v.words.Command[cell]
}

func commandTooltipLabel(v *Viewer, cell commandPanelCell) string {
	label := commandCellLabel(v, cell)
	if label != "" && cell == commandCellCast && v.selectedSpell != 0 {
		for _, spell := range v.spellbook {
			if spell.ID == v.selectedSpell && spell.Name != "" {
				return label + ": " + spell.Name
			}
		}
	}
	return label
}

// commandHoverPresent is the hovered cell's own label box and where it
// stands, or false for no hover, no font, an inactive panel, or a disabled
// cell.
//
// `MENU-COMBAT-019`'s own tooltip gate is narrower than this — active,
// enabled, no modal or special cursor, and frame-state bits `0x0a` clear.
// This build has no modelled modal cursor or frame-state byte to test, so
// only the two conditions this tier already carries are applied: the panel's
// own active gate and the cell's own skip mask. The narrower original gate
// is not reconstructed (contract "What is not decoded": the message-post
// route).
func (v *Viewer) commandHoverPresent() (*image.RGBA, image.Point, bool) {
	if v.font == nil || !v.hasCursor {
		return nil, image.Point{}, false
	}
	cell, ok := v.commandCellAt(v.cursorX, v.cursorY)
	if !ok || commandPanelCellSkipped(cell) || !commandPanelActive(v.sel, v.entities, v.localOwner) {
		return nil, image.Point{}, false
	}
	label := commandTooltipLabel(v, cell)
	if label == "" {
		return nil, image.Point{}, false
	}
	return v.tooltipPicture(tooltipTarget{tooltipCommand, "command", tooltipLines(label), v.font})
}
