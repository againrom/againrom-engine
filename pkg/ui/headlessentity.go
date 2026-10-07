package ui

import (
	"fmt"
	"image"
	"math"
)

// HeadlessEntityPoint locates a visible hit; it sends no event and changes no
// selection. A witness still activates the production pointer handler itself.
func (a *App) HeadlessEntityPoint(id uint32) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	for _, e := range v.entities {
		if e.ID != id {
			continue
		}
		r, ok := v.entityPickRect(e)
		if !ok {
			break
		}
		for y := int(math.Floor(r.Y)); y <= int(math.Ceil(r.Y+r.H)); y++ {
			for x := int(math.Floor(r.X)); x <= int(math.Ceil(r.X+r.W)); x++ {
				if hit, ok := targetAt(v.entities, v.entityPickRect, float64(x), float64(y), false); ok && hit == id {
					return v.frameToWindow(image.Pt(x, y), "entity point")
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("entity %d has no visible hit point", id)
}
