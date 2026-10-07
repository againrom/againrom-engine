package game

import "againrom/pkg/render/terrain"

// boundaryFrames is the frames of the spritesb sibling at boundaryPath, one per
// frame of base. The sibling takes the base sheet's palette, as the sack
// boundary does, and only its coverage is read. A missing sibling, a sibling of
// another frame count and a base without frames each answer nil.
func (c *sheetCache) boundaryFrames(base []*terrain.StaticFrame, basePath, boundaryPath string) []*terrain.StaticFrame {
	if boundaryPath == "" || len(base) == 0 {
		return nil
	}
	sheet, sibling := c.sheet(basePath), c.sheet(boundaryPath)
	if sheet == nil || sibling == nil || !sheet.HasPalette || len(sibling.Frames) != len(base) {
		return nil
	}
	key := boundaryPath + "\x00paired"
	paired, tried := c.converted[key]
	if !tried {
		copy := *sibling
		copy.Palette, copy.HasPalette = sheet.Palette, true
		saved, had := c.decoded[boundaryPath]
		c.decoded[boundaryPath] = &copy
		delete(c.converted, boundaryPath)
		paired = c.frames(boundaryPath)
		delete(c.converted, boundaryPath)
		if had {
			c.decoded[boundaryPath] = saved
		} else {
			delete(c.decoded, boundaryPath)
		}
		c.converted[key] = paired
	}
	if len(paired) != len(base) {
		return nil
	}
	return paired
}
