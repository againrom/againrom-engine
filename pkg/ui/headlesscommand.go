package ui

import (
	"fmt"
	"image"
)

// HeadlessCommandPanel is the same CPU composite the mission draw paints.
// It deliberately does not claim to render the whole mission screen.
func (a *App) HeadlessCommandPanel() (*image.RGBA, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return nil, err
	}
	pic, _, ok := v.commandPanelPresent()
	if !ok {
		return nil, fmt.Errorf("headless command panel: no panel")
	}
	return pic, nil
}

func (a *App) HeadlessCommandPoint(cell int) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	bar, ok := v.commandPanelBar()
	if !ok || cell < 0 || cell >= int(commandPanelCellCount) {
		return 0, 0, fmt.Errorf("headless command point: invalid cell %d or no panel", cell)
	}
	r := commandCellRects(bar)[cell]
	return v.frameToWindow(r.Min.Add(r.Max).Div(2), "headless command point")
}

func (a *App) HeadlessMinimapPoint(col, row int) (int, int, error) {
	v, err := a.headlessViewer()
	if err != nil {
		return 0, 0, err
	}
	g, ok := v.minimapGeometry()
	if ok && col >= 0 && row >= 0 && col < v.grid.Width && row < v.grid.Height {
		for y := g.Content.Min.Y; y < g.Content.Max.Y; y++ {
			for x := g.Content.Min.X; x < g.Content.Max.X; x++ {
				if c, hit := v.minimapCellAt(x, y); hit && c == image.Pt(col, row) {
					return v.frameToWindow(image.Pt(x, y), "headless minimap point")
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("headless minimap point: cell (%d,%d) is not drawn", col, row)
}
