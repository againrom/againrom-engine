package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// maxWindowSide is the longest window side a development tool may ask for.
const maxWindowSide = 8192

// ParseWindowSize is the one reader of a "WxH" window size, each side
// 1..8192. A caller with a default passes it as empty and gets it for an empty
// value; the zero Point makes an empty value an error.
func ParseWindowSize(value string, empty image.Point) (image.Point, error) {
	if value == "" && empty != (image.Point{}) {
		return empty, nil
	}
	x, y, ok := strings.Cut(value, "x")
	width, errX := strconv.Atoi(x)
	height, errY := strconv.Atoi(y)
	if !ok || errX != nil || errY != nil || width <= 0 || height <= 0 || width > maxWindowSide || height > maxWindowSide {
		return image.Point{}, fmt.Errorf("window size %q: want WxH, each side 1..%d", value, maxWindowSide)
	}
	return image.Pt(width, height), nil
}
