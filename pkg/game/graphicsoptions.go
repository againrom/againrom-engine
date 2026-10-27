package game

import (
	"strconv"

	"againrom/pkg/ui"
)

func graphicsValues(v ui.GameOptionValues) ui.GraphicsOptions {
	return ui.GraphicsOptions{Smoothing: v[ui.GameOptionSmoothing] != 0,
		HideShadows: v[ui.GameOptionShadows] == 0, DisableLighting: v[ui.GameOptionLighting] == 0,
		StaticObjects: v[ui.GameOptionAnimation] == 0, HideLootDots: v[ui.GameOptionLootDots] == 0}
}

func defaultGraphicsValues() ui.GameOptionValues {
	var v ui.GameOptionValues
	v[ui.GameOptionSmoothing] = 1
	v[ui.GameOptionShadows] = 1
	v[ui.GameOptionLighting] = 1
	v[ui.GameOptionAnimation] = 1
	// Loot dots stay off until the player asks: the original's minimap marks
	// no sack (MISSION-063, DIV-2954).
	v[ui.GameOptionLootDots] = 0
	return v
}

func (f *FrontEnd) graphicsValues(v *ui.GameOptionValues) {
	v[ui.GameOptionSmoothing] = boolOption(f.graphics.Smoothing)
	v[ui.GameOptionShadows] = boolOption(!f.graphics.HideShadows)
	v[ui.GameOptionLighting] = boolOption(!f.graphics.DisableLighting)
	v[ui.GameOptionAnimation] = boolOption(!f.graphics.StaticObjects)
	v[ui.GameOptionLootDots] = boolOption(!f.graphics.HideLootDots)
}

func (f *FrontEnd) setGraphicsOption(onMap bool, option ui.GameOption, value int) error {
	var v ui.GameOptionValues
	f.graphicsValues(&v)
	v[option] = value
	if v[ui.GameOptionAnimation] == 0 {
		v[ui.GameOptionLighting] = 0
	}
	if f.Options.Path != "" {
		m, err := f.Options.readAll()
		if err != nil {
			return err
		}
		for o := ui.GameOptionSmoothing; o <= ui.GameOptionAnimation; o++ {
			m[gameOptionKeys[o]] = strconv.Itoa(v[o])
		}
		m[gameOptionKeys[ui.GameOptionLootDots]] = strconv.Itoa(v[ui.GameOptionLootDots])
		if err := f.Options.writeAll(m); err != nil {
			return err
		}
	}
	f.graphics = graphicsValues(v)
	if onMap && f.live != nil {
		f.live.view.SetGraphicsOptions(f.graphics)
	}
	return nil
}
