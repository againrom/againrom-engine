package game

import "againrom/pkg/formats/sav"

// cityApplicationLeaves are the town state-store leaves both original
// application readers consume on LOAD (SAV-914). The spell shortcuts have
// their own producer; nativeCityPlayerFormation writes Formation together
// with its Player byte.
var cityApplicationLeaves = []string{
	"/GameOptions/FlyingHP", "/GameOptions/ShowHP", "/GameOptions/ShowTimeFlow",
	"/GameOptions/Speed", "/GameOptions/Wimpy", "/Inventory/IsOpen",
	"/SpellBook/IsOpen", "/SpellBook/Pressed", "/View/X", "/View/Y",
}

// projectCityApplication fills a town document's option, panel and view
// leaves by the SAV source order instead of leaving NewCityStateData's typed
// zeros (DIV-878): current application state when the session has one, else
// the loaded document's own state-store records. A town with neither keeps the constructor
// value, which is the remaining debt DIV-878 names. Speed 0 is the slowest
// rate (SESS-CLOCK-005), so a zero written over a loaded 8 slows the game.
func projectCityApplication(data *sav.CityData, app *SnapshotApplicationState, loaded []sav.CityStateRecordData) error {
	state := &data.State
	values := map[string]int32{}
	switch {
	case app != nil && !app.LocalOnly:
		raw, err := applicationCurrentRaw(app)
		if err != nil {
			return err
		}
		values = map[string]int32{
			"/GameOptions/FlyingHP": raw.FlyingHP, "/GameOptions/ShowHP": raw.ShowHP,
			"/GameOptions/ShowTimeFlow": raw.ShowTimeFlow, "/GameOptions/Speed": raw.Speed,
			"/GameOptions/Wimpy": raw.Wimpy, "/Inventory/IsOpen": raw.InventoryOpen,
			"/SpellBook/IsOpen": raw.BookOpen, "/SpellBook/Pressed": raw.Pressed,
			"/View/X": raw.ViewX, "/View/Y": raw.ViewY,
		}
	case len(loaded) > 0:
		for _, path := range cityApplicationLeaves {
			value, present, err := sav.ReadStateLeaf(17, nil, loaded, path)
			if err != nil {
				return err
			}
			if present && value.Kind == 2 {
				values[path] = value.Int32
			}
		}
	default:
		return nil
	}
	for _, path := range cityApplicationLeaves {
		v, ok := values[path]
		if !ok {
			continue
		}
		if _, err := patchStateLeaf(state.RootKind, state.DirectoryRecords, state.ValueRecords, path, func(r *sav.CityStateValueData) error {
			if r.Kind == 2 {
				r.Int32 = v
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
