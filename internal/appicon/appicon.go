// Package appicon holds the three program icons, which are project art licensed
// with the engine, and answers each program's icon images. It is the only source
// of an icon for a window and for an executable's resource.
package appicon

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"sync"

	"golang.org/x/image/draw"
)

// Name selects one program's icon.
type Name string

const (
	Game      Name = "againrom"
	MapEditor Name = "mapedit"
	Starter   Name = "starter"
)

//go:embed againrom.png
var gamePNG []byte

//go:embed mapedit.png
var mapeditPNG []byte

//go:embed starter.png
var starterPNG []byte

// Sizes are the square sizes Windows shows, smallest first. The last is the
// embedded picture's own size.
var Sizes = []int{16, 24, 32, 48, 64, 128, 256}

// PNG returns the embedded 256x256 PNG of the icon, or nil for an unknown name.
func PNG(n Name) []byte {
	switch n {
	case Game:
		return gamePNG
	case MapEditor:
		return mapeditPNG
	case Starter:
		return starterPNG
	}
	return nil
}

var (
	mu    sync.Mutex
	cache = map[Name][]image.Image{}
)

// Images returns the icon at every size in Sizes, smallest first, each smaller
// size scaled once from the 256 picture with a Catmull-Rom filter. An unknown
// name returns nil.
func Images(n Name) []image.Image {
	mu.Lock()
	defer mu.Unlock()
	if out, ok := cache[n]; ok {
		return out
	}
	data := PNG(n)
	if data == nil {
		return nil
	}
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		panic("appicon: embedded " + string(n) + ".png: " + err.Error())
	}
	full := image.NewNRGBA(src.Bounds())
	draw.Draw(full, full.Bounds(), src, src.Bounds().Min, draw.Src)
	out := make([]image.Image, len(Sizes))
	for i, s := range Sizes {
		if s == full.Bounds().Dx() {
			out[i] = full
			continue
		}
		dst := image.NewNRGBA(image.Rect(0, 0, s, s))
		draw.CatmullRom.Scale(dst, dst.Bounds(), full, full.Bounds(), draw.Src, nil)
		out[i] = dst
	}
	cache[n] = out
	return out
}
