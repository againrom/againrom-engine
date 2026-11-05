package game

import "againrom/pkg/render/terrain"

// boundaryFrames pairs a sibling with the base frames, using the base palette.
// A missing or mismatched sheet leaves the base unchanged.
func (c *sheetCache) boundaryFrames(base []*terrain.StaticFrame, basePath, boundaryPath string) []*terrain.StaticFrame {
	if boundaryPath == "" || len(base) == 0 {
		return nil
	}
	sheet, sibling := c.sheet(basePath), c.sheet(boundaryPath)
	if sheet == nil || sibling == nil || !sheet.HasPalette || len(sibling.Frames) != len(base) {
		return nil
	}
	key := basePath + "\x00" + boundaryPath + "\x00paired"
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
	for i, f := range base {
		f.Boundary = paired[i]
	}
	return paired
}
