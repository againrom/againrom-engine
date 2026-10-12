package menu

import (
	"fmt"
	"image"

	"againrom/pkg/screenart"
)

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

// LoadSecond binds the second game's resolved art to its layout and captions.
func LoadSecond(set *screenart.Set) (*Assets, error) {
	r := rects(secondPlace)
	a, err := Load(set)
	if err != nil {
		return nil, err
	}
	p := &secondPlacement{hover: r, pressed: r}
	for i := range p.caption {
		img, err := picture(set, fmt.Sprintf("caption/%d", i))
		if err != nil {
			return nil, err
		}
		p.caption[i] = img
	}
	a.place = p
	a.actions = SecondActions
	return a, nil
}
