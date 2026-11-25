package game

import "againrom/pkg/ui"

// mapChoiceWords resolves the multiplayer map choice page's title, button and
// empty-list captions from the engine's word tables (DIV-5197).
func (f *FrontEnd) mapChoiceWords() ui.MapChoiceWords {
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	return ui.MapChoiceWords{
		Title:  encodeEngineWord(f.Words.Engine.Text("mapchoice.title"), selector),
		OK:     encodeEngineWord(f.Words.Engine.Text("mapchoice.ok"), selector),
		Cancel: encodeEngineWord(f.Words.Engine.Text("mapchoice.cancel"), selector),
		Empty:  encodeEngineWord(f.Words.Engine.Text("mapchoice.empty"), selector),
	}
}

// MultiplayerMapChoice filters the front end's map list by the installed game's
// multiplayer map rule and builds its rows through MapPickerRows.
func (f *FrontEnd) MultiplayerMapChoice() (items []ui.MapChoiceItem, failed int) {
	selector := 0
	if f.Font.Value() != nil {
		selector = f.Font.Value().Selector
	}
	rule := filesOf(f.savedGame()).multiplayerMaps
	kept, failed := MultiplayerMaps(f.Maps, rule)
	return MultiplayerMapItems(kept, selector, rule, f.Words.Engine.Text("mapchoice.level")), failed
}

// OpenMultiplayerMapChoice opens the multiplayer map choice page on a with the
// installed game's filtered root maps and returns how many maps failed to decode.
func (f *FrontEnd) OpenMultiplayerMapChoice(a *ui.App, back ui.Screen, onDone func(ui.MapChoiceResult)) int {
	items, failed := f.MultiplayerMapChoice()
	a.SetMapChoiceWords(f.mapChoiceWords())
	a.OpenMapChoice(items, back, onDone)
	return failed
}
