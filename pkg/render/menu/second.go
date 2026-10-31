package menu

import (
	"fmt"
	"image"
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

// SecondLayout is the second game's row: one rectangle per button for hover and
// pressed alike, the caption bitmaps caption/0..7 and its own binding table.
var SecondLayout = func() Layout {
	r := rects(secondPlace)
	l := Layout{Hover: r, Pressed: r, CaptionAt: captionRect, Actions: SecondActions}
	for i := range l.Captions {
		l.Captions[i] = fmt.Sprintf("caption/%d", i)
	}
	return l
}()
