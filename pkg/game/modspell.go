package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func (f *FrontEnd) SetModSpells(d mod.SpellData) error {
	_, err := f.setModSpells(d)
	return err
}

// setModSpells refuses a row the table cannot take, then keeps the rows
// merged one per spell over the table's names.
func (f *FrontEnd) setModSpells(d mod.SpellData) ([]mod.Override, error) {
	if d.Empty() {
		return nil, nil
	}
	if f == nil || f.Table == nil {
		return nil, fmt.Errorf("the front end has no definition table to carry the mods")
	}
	for _, r := range d.Rows {
		if line, msg := mapload.ModSpellRefusal(f.Table, r); msg != "" {
			return nil, itemFileError(r.Mod, r.File, line, "%s", msg)
		}
	}
	var names []string
	if spells, err := data.LoadSpells(f.Table.Spells); err == nil {
		for _, sp := range spells {
			names = append(names, sp.Name)
		}
	}
	d, overrides, err := mod.MergeSpells(d, names)
	if err != nil {
		return nil, err
	}
	f.Table.Mods.Spells = d
	return overrides, f.freezeSpellFormulas()
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
