package game

import (
	"fmt"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func (f *FrontEnd) SetModSpells(d mod.SpellData) error {
	if d.Empty() {
		return nil
	}
	if f == nil || f.Table == nil {
		return fmt.Errorf("the front end has no definition table to carry the mods")
	}
	for _, r := range d.Rows {
		if line, msg := mapload.ModSpellRefusal(f.Table, r); msg != "" {
			return itemFileError(r.Mod, r.File, line, "%s", msg)
		}
	}
	f.Table.Mods.Spells = d
	return f.freezeSpellFormulas()
}

func (f *FrontEnd) freezeSpellFormulas() error {
	set, err := mapload.SpellFormulas(f.Table, f.Table.Mods.Spells)
	if err != nil {
		return err
	}
	r, err := f.Table.Rules.WithSpellFormulas(set)
	if err != nil {
		return err
	}
	f.Table.Rules = r
	return nil
}
