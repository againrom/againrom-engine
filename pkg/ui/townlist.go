package ui

import (
	"image"
	"image/draw"

	"againrom/pkg/render/debugtext"
	"againrom/pkg/render/frame"
)

func paintTownList(pix *image.RGBA, town TownScreen, list *Picker, msg string) {
	print := func(s string, x, y int) { debugtext.Draw(pix, s, x, y) }
	button := func(r image.Rectangle, selected bool) {
		drawPushButton(pix, nil, pushButton{Rect: r, Style: &pushButtonRowList, Selected: selected})
	}
	print(clipRunes(town.Header(), pickerCols),
		pickerLeft, pickerHeaderY)

	top, n := list.Visible()
	y := pickerTop + (n+1)*pickerLine
	if atTownSquare(town) {
		for i := 0; i < n && i < 4; i++ {
			x, by, w, h := townSquareRect(i)
			button(image.Rect(x, by, x+w, by+h), list.Selection() == i)
			print(list.RowText(i), x+14, by+h/2-8)
		}
		y = townSquareTop + 2*(townSquareH+townSquareGap) + pickerLine
	} else {
		for k := 0; k < n; k++ {
			row := top + k
			by := pickerTop + k*pickerLine - 1
			if list.Rows()[row].Choosable {
				button(image.Rect(pickerLeft-6, by, pickerLeft-6+580, by+pickerLine-1),
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
	draw.Draw(pix, pix.Bounds(), image.NewUniform(pickerBackground), image.Point{}, draw.Src)
	paintTownList(pix, town, list, msg)
	return pix
}
