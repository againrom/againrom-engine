package game

import (
	"strconv"

	"againrom/pkg/ui"
)

// graphicsValues is the renderer flags the graphics rows of v name.
func graphicsValues(v ui.GameOptionValues) ui.GraphicsOptions {
	var g ui.GraphicsOptions
	for o, r := range gameOptionRows {
		if r.graphics != nil {
			r.graphics.set(&g, v[o])
		}
	}
	return g
}

// graphicsValues writes the session's graphics flags into their rows of v.
func (f *FrontEnd) graphicsValues(v *ui.GameOptionValues) {
	for o, r := range gameOptionRows {
		if r.graphics != nil {
			v[o] = r.graphics.value(f.graphics)
		}
	}
}

// setGraphicsOption writes one graphics option with its group: every
// graphics row is stored, with the rows' needs applied, and the live map
// takes the whole group.
func (f *FrontEnd) setGraphicsOption(onMap bool, option ui.GameOption, value int) error {
	var v ui.GameOptionValues
	f.graphicsValues(&v)
	v[option] = value
	v = v.Constrained()
	if f.Options.Path != "" {
		m, err := f.Options.readAll()
		if err != nil {
			return err
		}
		for o, r := range gameOptionRows {
			if r.graphics != nil {
				m[gameOptionKeys[o]] = strconv.Itoa(v[o])
			}
		}
		if err := f.Options.writeAll(m); err != nil {
			return err
		}
	}
	f.graphics = graphicsValues(v)
	if f.live != nil {
		f.live.view.SetGraphicsOptions(f.graphics)
	}
	return nil
}
