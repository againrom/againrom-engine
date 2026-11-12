package game

import (
	"againrom/pkg/mod"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

var baseMenuActions = []struct {
	screen        mod.Screen
	menu, confirm string
	enabled       func(rules.Rules) bool
}{
	{mod.Screen{Key: "abandon", Kind: mod.ActionAbandon, Game: true}, "mission.abandon", "mission.abandon.confirm", rules.Rules.Abandon},
}

// ModScreens converts base and mod menu data to the presentation form.
// Base actions read the rules and engine words; registration resolves replacements.
// No save, hash or simulation state reads these screens.
func ModScreens(d mod.ScreenData, r rules.Rules, book words.Book) []ui.ModScreen {
	screens := append([]mod.Screen(nil), d.Screens...)
	for _, row := range baseMenuActions {
		if row.enabled(r) {
			s := row.screen
			s.MenuLabel, s.Title = book.Text(row.menu), book.Text(row.confirm)
			screens = append(screens, s)
		}
	}
	out := make([]ui.ModScreen, 0, len(screens))
	for _, s := range screens {
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
