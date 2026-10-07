package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// LoadBottomHUDArt resolves the two complete mission bars from the install.
// MAGIC-ICON-024/025 name the book family. The inventory frame and its
// 80-pixel bays are matched to the owner's original screenshot; no asset
// bytes are embedded in the engine.
func LoadBottomHUDArt(src terrain.EntrySource) (*ui.BottomHUDArt, error) {
	if src == nil {
		return nil, fmt.Errorf("bottom HUD: no graphics archive")
	}
	art := new(ui.BottomHUDArt)
	for _, asset := range []struct {
		name string
		w, h int
		out  **image.RGBA
	}{
		{"spellbook.bmp", 480, 85, &art.Book},
		{"spellback.bmp", 36, 36, &art.UnknownSpell},
		{"spb800l.bmp", 80, 85, &art.BookLeft[0]},
		{"spb800r.bmp", 96, 85, &art.BookRight[0]},
		{"spb1024l.bmp", 192, 85, &art.BookLeft[1]},
		{"spb1024r.bmp", 208, 85, &art.BookRight[1]},
		{"invframe.bmp", 480, 90, &art.Pack},
		{"backinv.bmp", 80, 80, &art.PackItem},
		{"inv1024l.bmp", 32, 90, &art.PackLeft},
		{"inv1024r.bmp", 48, 90, &art.PackRight},
		{"invarrow1.bmp", 32, 88, &art.PackArrow[0]},
		{"invarrow2.bmp", 32, 88, &art.PackArrow[1]},
		{"invarrow3.bmp", 32, 88, &art.PackArrow[2]},
		{"invarrow4.bmp", 32, 88, &art.PackArrow[3]},
	} {
		path := graphicsPrefix + "interface/" + asset.name
		pic, err := readChargenBMP(src, path)
		if err != nil {
			return nil, err
		}
		if err := chargenSize(pic, asset.w, asset.h, path); err != nil {
			return nil, err
		}
		*asset.out = keyBlack(pic)
	}
	return art, nil
}
