package ui

import (
	"fmt"
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// Widget 8's structure readout (MENU-070, MENU-071, MENU-072): three text
// lines in the column's fourth slot, drawn for a structure only. Each is at
// the widget's right edge minus 88 pixels, one flat shadow pixel down and
// right of its glyphs, in the card font.
const (
	readoutRightInset = 0x58
	readoutNameY      = 0x1c
	readoutCaptionY   = 0x2c
	readoutRatioY     = 0x36
	readoutShadow     = 1
	// readoutHealthString is main.txt's global string 19, the word Health.
	readoutHealthString = 19
)

var (
	// readoutNameInk is entry 15 of the ramp the name and the caption share.
	readoutNameInk = color.RGBA{R: 185, G: 159, B: 73, A: 255}
	// readoutRatioInk is entry 15 of the ratio's own ramp.
	readoutRatioInk = color.RGBA{R: 107, G: 154, B: 120, A: 255}
)

type structureReadoutKey struct {
	lines [3]string
	font  *text.Font
}

// selectableStructure is whether a plain click selects the structure under it:
// its class is not Indestructible and the structure answers a hover hit
// (MISSION-065).
func (v *Viewer) selectableStructure(ref InspectionSubject) bool {
	if ref.Kind != InspectionStructure {
		return false
	}
	c := v.structureInfo[ref.ID]
	if c == nil || c.Indestructible {
		return false
	}
	_, exists := v.inspectionPanel(ref)
	return exists
}

// selectedStructure is the structure a plain click selected, while no unit is
// selected: the selection holds one object, so any selected unit replaces it.
func (v *Viewer) selectedStructure() (InspectionSubject, bool) {
	if v.selStructure.Kind != InspectionStructure || len(v.sel) != 0 {
		return InspectionSubject{}, false
	}
	if _, exists := v.inspectionPanel(v.selStructure); !exists {
		return InspectionSubject{}, false
	}
	return v.selStructure, true
}

// readoutStructure is the object widget 8 reads: the hovered object whenever
// one resolves, else the single selected structure. Only a structure draws a
// readout; any other hovered class draws nothing here (MENU-070).
func (v *Viewer) readoutStructure() (InspectionSubject, bool) {
	if ref, hover := v.hoverInspection(); hover {
		return ref, ref.Kind == InspectionStructure
	}
	return v.selectedStructure()
}

// structureReadoutLines are the three lines for a structure: the building.txt
// name of its class, the word Health and current/maximum health (MENU-071,
// MENU-072). A zero maximum reads 1000.
func (v *Viewer) structureReadoutLines(ref InspectionSubject) ([3]string, bool) {
	panel, ok := v.inspectionPanel(ref)
	if !ok {
		return [3]string{}, false
	}
	maxHP := panel.MaxHP
	if maxHP == 0 {
		maxHP = 1000
	}
	return [3]string{panel.Name, v.words.PanelCaptions[readoutHealthString], fmt.Sprintf("%d/%d", panel.HP, maxHP)}, true
}

// composeStructureReadout draws the three lines at their relative heights,
// each shadow first.
func composeStructureReadout(lines [3]string, f *text.Font) *image.RGBA {
	if f == nil || f.Height() <= 0 {
		return nil
	}
	width := 0
	for _, l := range lines {
		w, _ := f.Measure(l)
		width = max(width, w, f.Advance(l))
	}
	if width <= 0 {
		return nil
	}
	ys := [3]int{0, readoutCaptionY - readoutNameY, readoutRatioY - readoutNameY}
	inks := [3]color.RGBA{readoutNameInk, readoutNameInk, readoutRatioInk}
	img := image.NewRGBA(image.Rect(0, 0, width+readoutShadow, ys[2]+f.Height()+readoutShadow))
	for i, l := range lines {
		f.DrawFlat(img, l, readoutShadow, ys[i]+readoutShadow, messageShadowColor)
		f.Draw(img, l, 0, ys[i], inks[i])
	}
	return img
}

// structureReadoutPresent is the picture to draw this frame and where its
// top-left corner goes, or false for a frame with no readout: no card font, no
// room for the column's fourth slot or no structure to read. It draws at the
// slot's own right edge minus 88, the name 28 pixels below the slot's top, the
// caption 44 and the ratio 54 (MENU-071).
func (v *Viewer) structureReadoutPresent() (*image.RGBA, image.Point, bool) {
	f := v.cardFont()
	if f == nil {
		return nil, image.Point{}, false
	}
	box, ok := columnFillerRect(image.Pt(v.frameW, v.frameH))
	if !ok {
		return nil, image.Point{}, false
	}
	ref, ok := v.readoutStructure()
	if !ok {
		return nil, image.Point{}, false
	}
	lines, ok := v.structureReadoutLines(ref)
	if !ok {
		return nil, image.Point{}, false
	}
	key := structureReadoutKey{lines: lines, font: f}
	if v.structPic == nil || key != v.structKey {
		var pic *image.RGBA
		v.structText = text.Record(func() { pic = composeStructureReadout(lines, f) })
		v.structPic, v.structFresh, v.structKey = pic, true, key
	}
	if v.structPic == nil {
		return nil, image.Point{}, false
	}
	text.Append(v.structText, 0, 0)
	return v.structPic, image.Pt(box.Max.X-readoutRightInset, box.Min.Y+readoutNameY), true
}
