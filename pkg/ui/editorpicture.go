package ui

import "image"

// pictureFit is how the picture element stands a picture in its box.
type pictureFit uint8

const (
	// pictureCentre shrinks a large picture into the box and grows a small
	// sprite by the largest whole factor that fits, centred.
	pictureCentre pictureFit = iota
	// pictureCorner shrinks a large picture into the box, never grows it,
	// at the box's top right.
	pictureCorner
	// pictureFill draws the picture at the box's size.
	pictureFill
)

// editorPicture is the editor's one picture element: a picture standing in
// Box as Fit says, only its part inside Clip shown.
type editorPicture struct {
	Pic  *image.RGBA
	Box  image.Rectangle
	Clip image.Rectangle
	Fit  pictureFit
}

// drawEditorPicture paints p onto dst; a missing or empty picture draws
// nothing.
func drawEditorPicture(dst *image.RGBA, p editorPicture) {
	if p.Pic == nil || p.Pic.Bounds().Empty() || p.Box.Empty() {
		return
	}
	clip := p.Box.Intersect(p.Clip).Intersect(dst.Bounds())
	if clip.Empty() {
		return
	}
	size := p.Pic.Bounds().Size()
	var at screenRect
	switch p.Fit {
	case pictureCentre:
		at = addPreviewRect(size, p.Box)
	case pictureCorner:
		at = fitRect(size, p.Box)
	default:
		at = screenRect{X: float64(p.Box.Min.X), Y: float64(p.Box.Min.Y), W: float64(p.Box.Dx()), H: float64(p.Box.Dy())}
	}
	editorBlit(dst.SubImage(clip).(*image.RGBA), p.Pic, at, false)
}
