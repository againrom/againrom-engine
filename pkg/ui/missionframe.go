package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
)

// missionLayer is one step of the mission frame. The window executor
// (drawFrame for the canvas steps, Viewer.Draw for the placed ones) runs
// window; the CPU executor (headlessMissionFrame) runs cpu.
type missionLayer struct {
	name string
	// placed steps draw on the window after the canvas is settled; the
	// others draw on the canvas.
	placed bool
	// editor steps also run in the map editor's frame.
	editor bool
	window func(*Viewer, *ebiten.Image, *windowPass)
	// cpu is nil for a step the CPU frame does not compose.
	cpu func(*Viewer, *cpuPass) error
	// cpuSkips names what the CPU frame leaves out of the step.
	cpuSkips string
}

// missionLayers is the mission frame in paint order.
var missionLayers = []missionLayer{
	{name: "map", editor: true, window: (*Viewer).layerMap, cpu: (*Viewer).cpuMap, cpuSkips: "the Heal/Drain shower and the fog"},
	{name: "marks", window: (*Viewer).layerMarks, cpu: (*Viewer).cpuMarks, cpuSkips: "the pause wash and the grid"},
	{name: "path strokes", window: (*Viewer).layerPathStrokes, cpuSkips: "the route strokes"},
	{name: "numerals", window: (*Viewer).layerNumerals, cpu: (*Viewer).cpuNumerals},
	{name: "backings", window: (*Viewer).layerBackings, cpu: (*Viewer).cpuBackings},
	{name: "panel", window: (*Viewer).layerPanel, cpu: (*Viewer).cpuPanel},
	{name: "spellbook", window: (*Viewer).layerSpellbook, cpu: (*Viewer).cpuSpellbook},
	{name: "readout", window: (*Viewer).layerReadout, cpuSkips: "the debug readout"},
	{name: "frame rate", window: (*Viewer).layerFrameRate, cpuSkips: "the frame rate readout"},
	{name: "minimap", window: (*Viewer).layerMinimap, cpu: (*Viewer).cpuMinimap},
	{name: "mission tip", window: (*Viewer).layerMissionTip, cpuSkips: "the mission tip"},
	{name: "cursor mode", window: (*Viewer).layerCursorMode, cpuSkips: "the system cursor mode, not a picture"},
	{name: "column filler", window: (*Viewer).layerColumnFiller, cpu: (*Viewer).cpuColumnFiller},
	{name: "mission card", window: (*Viewer).layerMissionCard, cpuSkips: "the mission card"},
	{name: "structure readout", window: (*Viewer).layerStructureReadout, cpuSkips: "the structure readout"},
	{name: "worn", window: (*Viewer).layerWorn, cpuSkips: "the worn items window"},
	{name: "pack bar", window: (*Viewer).layerPackBar, cpu: (*Viewer).cpuPackBar},
	{name: "command panel", window: (*Viewer).layerCommandPanel, cpu: (*Viewer).cpuCommandPanel},
	{name: "gold", window: (*Viewer).layerGold, cpuSkips: "the drop gold editor"},
	{name: "message", window: (*Viewer).layerMessage, cpu: (*Viewer).cpuMessage},
	{name: "tooltip", window: (*Viewer).layerTooltip, cpuSkips: "the hover tooltip"},
	{name: "popup backdrop", window: (*Viewer).layerPopupBackdrop, cpuSkips: "the popup dim, the dialogue backdrop and the notice"},
	{name: "pointer", window: (*Viewer).layerPointer, cpuSkips: "the pointer"},
	{name: "map to window", placed: true, editor: true, window: (*Viewer).layerMapToWindow, cpu: (*Viewer).cpuMapToWindow},
	{name: "canvas to window", placed: true, editor: true, window: (*Viewer).layerCanvasToWindow, cpu: (*Viewer).cpuCanvasToWindow},
	{name: "text overlay", placed: true, editor: true, window: (*Viewer).layerTextOverlay, cpuSkips: "the window-resolution text overlay"},
}

// windowPass is what the window executor's steps pass on.
type windowPass struct {
	placed       windowMap
	canvas       *ebiten.Image
	op           *ebiten.DrawImageOptions
	text         []text.DrawCall
	ox, oy       float64
	readoutPic   *image.RGBA
	readoutAt    image.Point
	readoutShown bool
}

// drawLayers is the window executor: the canvas steps, or with placed the
// window steps, in list order. The map editor's frame runs the editor steps.
func (v *Viewer) drawLayers(screen *ebiten.Image, f *windowPass, placed bool) {
	for _, l := range missionLayers {
		if l.placed == placed && (l.editor || !v.editorView) {
			l.window(v, screen, f)
		}
	}
}
