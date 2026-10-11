package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

const (
	schoolFighterClass = iota
	schoolMageClass
	schoolClassCount
)

// LoadTownSchoolArt resolves the school's immutable surface from the school
// room of description d. It is cosmetic: NewFrontEnd carries an error beside
// a nil result so a missing picture never makes the game unusable, while a
// complete install performs no archive reads in the draw or hit-test path.
//
// Every page art entry is required, and so are the scene's column and
// diamond series; each optional training family is dropped alone. The
// column's endpoints are the two rest faces.
//
// Skill patches use a source-16-bit-zero keyed blit; the rest faces are opaque
// (TOWN-150). The loader's black key instead tests source RGB for pure black.
// cmd/schoolcheck -transparency-only compares those populations over all 30
// shipped patches under standard RGB565/RGB555 channel truncation
// (TERR-LIGHT-019). This is not arbitrary runtime framebuffer-mask support:
// modded near-black RGB can quantize to zero without being pure black.
func LoadTownSchoolArt(d *town.Description, src terrain.EntrySource) (*ui.TownSchoolArt, error) {
	set, err := roomPageArt(d, "school", src)
	if err != nil {
		return nil, err
	}
	a := &ui.TownSchoolArt{
		Background: set.Picture("background"),
		Upper:      set.Picture("upper"),
		UpperSeam:  set.Picture("upper-seam"),
	}
	if a.Scene, err = loadRoomSceneArt(d, "school", src); a.Scene == nil {
		return nil, err
	}
	column := a.Scene["column"]
	a.Faces = [2]image.Image{column[0], column[len(column)-1]}
	for i := range a.Buttons {
		for state := range a.Buttons[i] {
			a.Buttons[i][state] = set.Picture(fmt.Sprintf("button/%d/%d", i, state))
		}
	}
	for class := range a.Masks {
		a.Masks[class] = set.Mask(fmt.Sprintf("mask/%d", class))
		for slot := range a.Skills[class] {
			for state := range a.Skills[class][slot] {
				a.Skills[class][slot][state] = set.Picture(fmt.Sprintf("skill/%d/%d/%d", class, slot, state))
			}
		}
	}
	return a, nil
}
