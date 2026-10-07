package game

import (
	"errors"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// BaseID names the base game an install is, as mods name it in applies-to: the
// id of the detected profile ("rom1-en", "rom1-ru", "rom1-demo"), else
// "rom1-en" or "rom1-ru" by the language entry of the install, and plain "rom1"
// when the language is not one of the two.
func BaseID(info InstallInfo) string {
	if id := info.Base.ID(); id != "" {
		return id
	}
	switch info.Language {
	case "english":
		return "rom1-en"
	case "russian":
		return "rom1-ru"
	}
	return "rom1"
}

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
		Items: f.Table.Mods.Items, Characters: f.Table.Mods.Characters, Companions: f.Table.Mods.Companions, Spells: f.Table.Mods.Spells}
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
