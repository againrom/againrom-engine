package game

import (
	"errors"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// BaseID forwards to the one naming of an install's base for mods.
func BaseID(info InstallInfo) string { return info.BaseID() }

// SetMods applies the rules the mods set and records the mod set. Games the
// front end opens afterwards run under those rules, and a save written under a
// non-empty set carries it. It is called once, before any game is opened, and
// leaves the items a SetModItems call already added in place.
func (f *FrontEnd) SetMods(r sim.Rules, set mod.Set, acceptUnmarked bool) error {
	if f == nil || f.Table == nil {
		return errors.New("the front end has no definition table to carry the mods")
	}
	f.Table.Rules = r
	// The items, characters and conditions the other calls added stay: the
	// order of the calls does not matter.
	f.Table.Mods = mapload.ModContext{Set: set, AcceptUnmarked: acceptUnmarked,
		Items: f.Table.Mods.Items, Characters: f.Table.Mods.Characters, Companions: f.Table.Mods.Companions, Spells: f.Table.Mods.Spells,
		Missions: f.Table.Mods.Missions}
	if f.Table.Mods.Spells.Empty() {
		return nil
	}
	return f.freezeSpellFormulas()
}

func (f *FrontEnd) modContext() mapload.ModContext {
	if f == nil {
		return mapload.ModContext{}
	}
	return tableModContext(f.Table)
}

func tableModContext(t *mapload.Table) mapload.ModContext {
	if t == nil {
		return mapload.ModContext{}
	}
	return t.Mods
}

// ModSet is the mod set the front end runs under; the zero Set when none.
func (f *FrontEnd) ModSet() mod.Set {
	if f == nil || f.Table == nil {
		return mod.Set{}
	}
	return f.Table.Mods.Set
}
