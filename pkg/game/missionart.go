package game

import (
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// viewerArt is the install's words, fonts and pictures a mission viewer is
// dressed with. Resolving it reads the install and warms the first-use art
// caches; applying it is a run of plain setters on a viewer. The two are
// separate so the mission-entry sequence asks for "the viewer's art" once and
// names no picture.
//
// A nil member is the ordinary case for an install whose art would not read:
// the viewer then draws its own authored mark, or nothing, exactly as before.
type viewerArt struct {
	words          ui.Words
	dialogFrame    *ui.MenuPanelArt
	font           *text.Font
	cardFont       *text.Font
	attackPointer  *image.RGBA
	commandPanel   *ui.CommandPanelArt
	bottomHUD      *ui.BottomHUDArt
	paneFigure     ui.TownPane
	paneStats      ui.TownPane
	paneCorners    *ui.CharacterPaneCornerArt
	paneFiller     ui.TownPane
	sackFrames     []*terrain.StaticFrame
	sackBoundaries []*terrain.StaticFrame
}

// viewerArtSource resolves a viewer's art from the install and the
// presentation's first-use caches. It holds the two components and nothing
// else.
type viewerArtSource struct {
	in *InstallResources
	pr *Presentation
}

// resolve reads the art a mission viewer needs from the install and warms the
// presentation's caches.
func (s viewerArtSource) resolve() viewerArt {
	art := viewerArt{words: s.in.Words, dialogFrame: s.pr.gameMenuArt(s.in)}
	art.font = s.in.Font.Value()
	art.cardFont = s.in.tipFont()
	art.attackPointer = s.in.AttackPointer.Value()
	art.commandPanel = s.in.CommandPanelArt.Value()
	art.bottomHUD = s.in.BottomHUDArt.Value()
	panes := s.pr.characterPanes(s.in)
	art.paneFigure, art.paneStats = panes.Figure, panes.Stats
	art.paneCorners = s.pr.characterPaneCorners(s.in)
	art.paneFiller = s.pr.characterPaneFiller(s.in)
	art.sackFrames, art.sackBoundaries = s.in.SackFrames, s.in.SackBoundaries
	return art
}

// apply dresses a viewer with the resolved art and turns its debug readout
// off, which the game does and the standalone developer viewer does not.
func (a viewerArt) apply(v *ui.Viewer) {
	v.SetWords(a.words)
	v.SetDialogFrame(a.dialogFrame)
	v.SetFont(a.font)
	v.SetCardFont(a.cardFont)
	v.ShowReadout(false)
	v.SetAttackPointer(a.attackPointer)
	v.SetCommandPanelArt(a.commandPanel)
	v.SetBottomHUDArt(a.bottomHUD)
	v.SetCharacterPaneArt(a.paneFigure, a.paneStats, a.paneCorners)
	v.SetCharacterPaneFillerArt(a.paneFiller)
	v.SetSackFrames(a.sackFrames)
	v.SetSackBoundaries(a.sackBoundaries)
}

// pickerArt is the art a debug picker map is dressed with: viewerArt without
// the words, dialog frame and pane filler.
type pickerArt struct {
	font           *text.Font
	cardFont       *text.Font
	attackPointer  *image.RGBA
	commandPanel   *ui.CommandPanelArt
	bottomHUD      *ui.BottomHUDArt
	paneFigure     ui.TownPane
	paneStats      ui.TownPane
	paneCorners    *ui.CharacterPaneCornerArt
	sackFrames     []*terrain.StaticFrame
	sackBoundaries []*terrain.StaticFrame
}

// pickerArtSource resolves a picker map's art.
type pickerArtSource struct {
	in *InstallResources
	pr *Presentation
}

func (s pickerArtSource) resolve() pickerArt {
	art := pickerArt{font: s.in.Font.Value(), cardFont: s.in.tipFont()}
	art.attackPointer = s.in.AttackPointer.Value()
	art.commandPanel = s.in.CommandPanelArt.Value()
	art.bottomHUD = s.in.BottomHUDArt.Value()
	panes := s.pr.characterPanes(s.in)
	art.paneFigure, art.paneStats = panes.Figure, panes.Stats
	art.paneCorners = s.pr.characterPaneCorners(s.in)
	art.sackFrames, art.sackBoundaries = s.in.SackFrames, s.in.SackBoundaries
	return art
}

// apply dresses a picker viewer and turns its debug readout off.
func (a pickerArt) apply(v *ui.Viewer) {
	v.SetFont(a.font)
	v.SetCardFont(a.cardFont)
	v.ShowReadout(false)
	v.SetAttackPointer(a.attackPointer)
	v.SetCommandPanelArt(a.commandPanel)
	v.SetBottomHUDArt(a.bottomHUD)
	v.SetCharacterPaneArt(a.paneFigure, a.paneStats, a.paneCorners)
	v.SetSackFrames(a.sackFrames)
	v.SetSackBoundaries(a.sackBoundaries)
}
