package ui

import "image"

func TipPanelBodyRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X, r.Min.Y, r.Max.X-8, r.Max.Y-8)
}
