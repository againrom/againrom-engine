package game

import (
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// ModScreens converts the screens the mods declare into the presentation tier's
// form. The text is already in the game's language; the conversion only maps
// the kind names. The screens are presentation: no save, hash or simulation
// state reads them.
func ModScreens(d mod.ScreenData) []ui.ModScreen {
	out := make([]ui.ModScreen, 0, len(d.Screens))
	for _, s := range d.Screens {
		kind := ui.ModScreenInfo
		switch s.Kind {
		case mod.ScreenList:
			kind = ui.ModScreenList
		case mod.ScreenTable:
			kind = ui.ModScreenTable
		case mod.ActionAbandon:
			kind = ui.ModActionAbandon
		case mod.ActionRestart:
			kind = ui.ModActionRestart
		}
		out = append(out, ui.ModScreen{
			Mod: s.Mod, Key: s.Key, Kind: kind,
			Title: s.Title, MenuLabel: s.MenuLabel,
			Main: s.Main, Game: s.Game,
			Paragraphs: s.Paragraphs, Items: s.Items, Rows: s.Rows,
		})
	}
	return out
}
