package game

import (
	"image"
	"image/color"
	"image/draw"

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
	words                         ui.Words
	dialogFrame                   *ui.MenuPanelArt
	font                          *text.Font
	cardFont                      *text.Font
	preserveCreatureSkillsHeading bool
	castNeedsSight                bool
	attackPointer                 *image.RGBA
	commandPanel                  *ui.CommandPanelArt
	bottomHUD                     *ui.BottomHUDArt
	paneFigure                    ui.TownPane
	paneStats                     ui.TownPane
	paneCorners                   *ui.CharacterPaneCornerArt
	paneFiller                    ui.TownPane
	sackFrames                    []*terrain.StaticFrame
	sackBoundaries                []*terrain.StaticFrame
	missionTip                    ui.MissionTipDress
	minimapPaint                  *ui.MinimapPaint
}

// viewerArtSource resolves a viewer's art from the install and the
// presentation's first-use caches. It holds the two components and nothing
// else.
type viewerArtSource struct {
	in *InstallResources
	pr *Presentation
	pc *PersistenceContext
}

// resolve reads the art a mission viewer needs from the install and warms the
// presentation's caches.
func (s viewerArtSource) resolve() viewerArt {
	art := viewerArt{words: s.in.Words, dialogFrame: s.pr.gameMenuArt(s.in)}
	art.minimapPaint = missionMinimapPaint(s.in)
	art.font = s.in.Font.Value()
	art.cardFont = s.in.tipFont()
	art.preserveCreatureSkillsHeading = !s.in.profile().CreatureResistanceHeading()
	art.castNeedsSight = s.in.profile().CastNeedsSight()
	art.attackPointer = s.in.AttackPointer.Value()
	art.commandPanel = s.in.CommandPanelArt.Value()
	art.bottomHUD = s.in.BottomHUDArt.Value()
	panes := s.pr.characterPanes(s.in)
	art.paneFigure, art.paneStats = panes.Figure, panes.Stats
	art.paneCorners = s.pr.characterPaneCorners(s.in)
	art.paneFiller = s.pr.characterPaneFiller(s.in)
	art.sackFrames, art.sackBoundaries = s.in.SackFrames, s.in.SackBoundaries
	art.missionTip = ui.MissionTipDress{
		Art:         s.pr.tipArt(s.in),
		Font:        art.cardFont,
		CloseLabel:  s.in.Words.TipClose,
		ToggleLabel: s.in.Words.TipShowNext,
		CloseClick:  RoomDescription(s.in.profile()) != nil,
	}
	if row := TipTable(s.in.profile().Edition()).Row("dialogue-close", "", ""); row != nil {
		art.missionTip.Rect, art.missionTip.Owner = row.Rectangle(), row.Owner()
	}
	if pc := s.pc; pc != nil {
		art.missionTip.TipsOn = func() bool { return !pc.tipsOffNow() }
		art.missionTip.SetTipsOn = func(on bool) { pc.setTipsOff(!on) }
	}
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
	v.SetMissionTipDress(a.missionTip)
	v.SetMinimapPaint(a.minimapPaint)
	v.SetPreserveCreatureSkillsHeading(a.preserveCreatureSkillsHeading)
	v.SetCastNeedsSight(a.castNeedsSight)
}

// pickerArt is the art a debug picker map is dressed with: viewerArt without
// the words, dialog frame and pane filler.
type pickerArt struct {
	font                          *text.Font
	cardFont                      *text.Font
	preserveCreatureSkillsHeading bool
	castNeedsSight                bool
	attackPointer                 *image.RGBA
	commandPanel                  *ui.CommandPanelArt
	bottomHUD                     *ui.BottomHUDArt
	paneFigure                    ui.TownPane
	paneStats                     ui.TownPane
	paneCorners                   *ui.CharacterPaneCornerArt
	sackFrames                    []*terrain.StaticFrame
	sackBoundaries                []*terrain.StaticFrame
}

// pickerArtSource resolves a picker map's art.
type pickerArtSource struct {
	in *InstallResources
	pr *Presentation
}

func (s pickerArtSource) resolve() pickerArt {
	art := pickerArt{font: s.in.Font.Value(), cardFont: s.in.tipFont()}
	art.preserveCreatureSkillsHeading = !s.in.profile().CreatureResistanceHeading()
	art.castNeedsSight = s.in.profile().CastNeedsSight()
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
	v.SetPreserveCreatureSkillsHeading(a.preserveCreatureSkillsHeading)
	v.SetCastNeedsSight(a.castNeedsSight)
}

// minimapShadeEntry is the palette entry of the owner shade word the first
// game's minimap marks take: byte +0x1148 of a sixteen-level mode-2 table is
// row 8, entry 164, and row 8 is the neutral gain (MISSION-085,
// TERR-LIGHT-018, TERR-LIGHT-060).
const minimapShadeEntry = 164

// missionMinimapPaint reads the minimap data the edition's mission-minimap
// manifest names, or nil when the description names none, the lookup does not
// read, or the install has no owner palettes. A manifest without colours
// keeps the symbolic picture.
func missionMinimapPaint(in *InstallResources) *ui.MinimapPaint {
	if in == nil || in.Archives == nil || in.Units == nil || !in.Units.HasOwnerPalettes {
		return nil
	}
	d := ScreensDescription(in.profile())
	if d.Art("mission-minimap") == nil {
		return nil
	}
	set, err := loadScreen(in.Archives.artSource(), d, "mission-minimap")
	if err != nil {
		return nil
	}
	pic := set.Picture("colours")
	if pic == nil {
		return nil
	}
	lookup := image.NewRGBA(image.Rectangle{Max: pic.Bounds().Size()})
	draw.Draw(lookup, lookup.Bounds(), pic, pic.Bounds().Min, draw.Src)
	p := &ui.MinimapPaint{Colours: lookup}
	for k := range p.OwnerShades {
		c := in.Units.OwnerPalettes[k][minimapShadeEntry]
		p.OwnerShades[k] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xff}
	}
	return p
}
