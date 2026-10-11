package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// LoadBottomHUDArt resolves the two complete mission bars from the
// bottom-hud manifest of description d. Any picture failing fails the whole
// call.
func LoadBottomHUDArt(src terrain.EntrySource, d *Screens) (*ui.BottomHUDArt, error) {
	if src == nil {
		return nil, fmt.Errorf("bottom HUD: no graphics archive")
	}
	set, err := loadScreen(src, d, "bottom-hud")
	if err != nil {
		return nil, err
	}
	pic := func(name string) *image.RGBA { return rgbaOf(set.Picture(name)) }
	art := &ui.BottomHUDArt{
		Book:         pic("book"),
		UnknownSpell: pic("unknown-spell"),
		Pack:         pic("pack"),
		PackItem:     pic("pack-item"),
		PackLeft:     pic("pack-left"),
		PackRight:    pic("pack-right"),
	}
	art.BookLeft[0], art.BookRight[0] = pic("book-left-800"), pic("book-right-800")
	art.BookLeft[1], art.BookRight[1] = pic("book-left-1024"), pic("book-right-1024")
	for i := range art.PackArrow {
		art.PackArrow[i] = pic(fmt.Sprintf("pack-arrow/%d", i))
	}
	return art, nil
}
