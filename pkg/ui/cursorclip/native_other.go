//go:build !windows && !darwin

package cursorclip

import "image"

type System struct{}

func (*System) Rectangle() (image.Rectangle, bool) { return image.Rectangle{}, false }
func (*System) Clip(*image.Rectangle) bool         { return true }
