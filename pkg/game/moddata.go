package game

import (
	"fmt"

	"againrom/pkg/mod"
)

// ModDataReport is what applying the mods' data reports: the fields a later
// mod sets over an earlier one, and the warnings of the rules that refuse
// nothing.
type ModDataReport struct {
	Overrides []string
	Warnings  []string
}

// modDataApply applies one declared kind of the mods' data to the front end.
type modDataApply func(f *FrontEnd, d mod.Data, r *ModDataReport) error

// modDataAppliers apply each kind of mod.DataKinds, by its name. Screens and
// settings reach no front end table: the window takes the screens
// (ui.App.SetModScreens) and the mods' run takes the settings.
var modDataAppliers = map[string]modDataApply{
	"items": func(f *FrontEnd, d mod.Data, _ *ModDataReport) error { return f.SetModItems(d.Items) },
	"characters": func(f *FrontEnd, d mod.Data, r *ModDataReport) error {
		overrides, err := f.setModCharacters(d.Characters)
		r.override(overrides)
		return err
	},
	"screens": func(*FrontEnd, mod.Data, *ModDataReport) error { return nil },
	"companions": func(f *FrontEnd, d mod.Data, r *ModDataReport) error {
		overrides, err := f.setModCompanions(d.Companions)
		r.override(overrides)
		return err
	},
	"spells": func(f *FrontEnd, d mod.Data, _ *ModDataReport) error { return f.SetModSpells(d.Spells) },
	// The body sheets and the weapon bodies apply together: a weapon body
	// may name a sheet a mod supplies.
	"body sheets": func(*FrontEnd, mod.Data, *ModDataReport) error { return nil },
	"weapon bodies": func(f *FrontEnd, d mod.Data, r *ModDataReport) error {
		overrides, err := f.setModBodies(d.Bodies)
		r.override(overrides)
		return err
	},
	"missions": func(f *FrontEnd, d mod.Data, r *ModDataReport) error {
		overrides, warnings, err := f.setModMissions(d.Missions)
		r.override(overrides)
		if err != nil {
			return err
		}
		r.Warnings = append(r.Warnings, warnings...)
		return nil
	},
	"towns": func(f *FrontEnd, d mod.Data, r *ModDataReport) error {
		overrides, err := f.SetModTowns(d.Towns)
		r.Overrides = append(r.Overrides, overrides...)
		return err
	},
	"settings": func(*FrontEnd, mod.Data, *ModDataReport) error { return nil },
}

func (r *ModDataReport) override(list []mod.Override) {
	for _, o := range list {
		r.Overrides = append(r.Overrides, o.String())
	}
}

// SetModData applies what the mods' data files declare, each kind of
// mod.DataKinds in its order, after SetMods and before any game is opened.
// Every refusal names the mod, the file and the line. On a refusal the
// report holds what the kinds applied before it reported.
func (f *FrontEnd) SetModData(d mod.Data) (ModDataReport, error) {
	var r ModDataReport
	for _, k := range mod.DataKinds() {
		apply, ok := modDataAppliers[k.Name]
		if !ok {
			return r, fmt.Errorf("the game has no apply step for the mods' %s", k.Name)
		}
		if err := apply(f, d, &r); err != nil {
			return r, err
		}
	}
	return r, nil
}
