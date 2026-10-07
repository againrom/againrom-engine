package menu

import (
	"fmt"
	"image"
)

func CaptionEntry(n int) string { return fmt.Sprintf("text%d.bmp", n) }

// Overlay and caption offsets: where each bitmap equals the base bitmap.
var (
	secondPlace = [ButtonCount]placement{
		{204, 52, 104, 96},
		{124, 156, 108, 76},
		{124, 252, 96, 88},
		{208, 340, 100, 100},
		{340, 52, 88, 100},
		{424, 152, 84, 88},
		{412, 260, 96, 84},
		{344, 348, 72, 80},
	}
	captionRect = image.Rect(232, 200, 232+180, 200+80)
)

type secondPlacement struct {
	hover, pressed [ButtonCount]image.Rectangle
	caption        [ButtonCount]*image.RGBA
}

// LoadSecond is Load at the second game's sizes, plus the captions.
func LoadSecond(src EntrySource) (*Assets, error) {
	r := rects(secondPlace)
	a, err := load(src, r, r)
	if err != nil {
		return nil, err
	}
	p := &secondPlacement{hover: r, pressed: r}
	for i := 0; i < ButtonCount; i++ {
		name := EntryPrefix + CaptionEntry(i+1)
		img, err := loadRGBA(src, name)
		if err != nil {
			return nil, err
		}
		if err := checkSize(name, img.Bounds(), captionRect.Dx(), captionRect.Dy()); err != nil {
			return nil, err
		}
		p.caption[i] = img
	}
	a.place = p
	return a, nil
}
