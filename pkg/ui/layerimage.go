package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	newLayerImage     = ebiten.NewImage
	disposeLayerImage = func(img *ebiten.Image) { img.Dispose() }
)

func ensureLayerImage(dst **ebiten.Image, size image.Point) *ebiten.Image {
	if size.X <= 0 || size.Y <= 0 {
		if *dst != nil {
			disposeLayerImage(*dst)
			*dst = nil
		}
		return nil
	}
	if *dst == nil || (*dst).Bounds().Size() != size {
		if *dst != nil {
			disposeLayerImage(*dst)
		}
		*dst = newLayerImage(size.X, size.Y)
	}
	return *dst
}
