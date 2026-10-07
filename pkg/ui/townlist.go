package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/debugtext"
	"againrom/pkg/render/frame"
)

func paintTownList(town TownScreen, list *Picker, msg string, button func(int, int, int, int, bool), print func(string, int, int)) {
	print(clipRunes(town.Header(), pickerCols),
		pickerLeft, pickerHeaderY)

	top, n := list.Visible()
	y := pickerTop + (n+1)*pickerLine
	if atTownSquare(town) {
		for i := 0; i < n && i < 4; i++ {
			x, by, w, h := townSquareRect(i)
			button(x, by, w, h, list.Selection() == i)
			print(list.RowText(i), x+14, by+h/2-8)
		}
		y = townSquareTop + 2*(townSquareH+townSquareGap) + pickerLine
	} else {
		for k := 0; k < n; k++ {
			row := top + k
			by := pickerTop + k*pickerLine - 1
			if list.Rows()[row].Choosable {
				button(pickerLeft-6, by, 580, pickerLine-1,
					list.Selection() == row)
			}
			print(list.RowText(row), pickerLeft, pickerTop+k*pickerLine)
		}
	}

	// The footer starts one blank line under the last row drawn.
	for _, line := range town.Footer() {
		if y+pickerLine > pickerMessageY {
			break
		}
		print(clipRunes(line, pickerCols), pickerLeft, y)
		y += pickerLine
	}

	if msg != "" {
		print(clipRunes(msg, pickerCols), pickerLeft, pickerMessageY)
	}

}

func composeTownList(town TownScreen, list *Picker, msg string) *image.RGBA {
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	box := func(rect image.Rectangle, c color.Color) {
		draw.Draw(pix, rect, image.NewUniform(c), image.Point{}, draw.Src)
	}
	box(pix.Bounds(), pickerBackground)
	paintTownList(town, list, msg, func(x, y, w, h int, selected bool) {
		fill := townButtonFill
		if selected {
			fill = townButtonSelected
		}
		box(image.Rect(x, y, x+w, y+h), fill)
		box(image.Rect(x, y, x+w, y+2), townButtonBorder)
		box(image.Rect(x, y+h-2, x+w, y+h), townButtonBorder)
		box(image.Rect(x, y, x+2, y+h), townButtonBorder)
		box(image.Rect(x+w-2, y, x+w, y+h), townButtonBorder)
	}, func(s string, x, y int) { debugtext.Draw(pix, s, x, y) })
	return pix
}
