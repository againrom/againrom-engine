package main

import (
	"flag"
	"fmt"

	"againrom/pkg/base"
	launchset "againrom/pkg/launch"
	"againrom/pkg/mod"
)

// launchFlag is where the game keeps one flag of launchset.Table: a *bool,
// *int or *string, or the func a repeated row calls; list fills its help.
type launchFlag struct {
	target any
	list   []string
}

// launchFlags binds every flag of launchset.Table; skill and mods receive
// the -skill and -mods text parse resolves.
func launchFlags(o *options, skill, mods *string) map[string]launchFlag {
	return map[string]launchFlag{
		"assets":   {target: &o.assets},
		"base":     {target: &o.base, list: base.IDs()},
		"mods":     {target: mods},
		"mods-dir": {target: &o.modsDir},
		"mod-setting": {target: func(v string) error {
			f, err := mod.ParseSettingFlag(v)
			if err != nil {
				return err
			}
			o.modSettings = append(o.modSettings, f)
			return nil
		}},
		"mods-accept-unmarked":  {target: &o.modsAcceptUnmarked},
		"check":                 {target: &o.check},
		"version":               {target: &o.version},
		"sound":                 {target: &o.sound},
		"volume":                {target: &o.volume},
		"movies":                {target: &o.movies},
		"4x":                    {target: &o.fourX},
		"8x":                    {target: &o.eightX},
		"markers":               {target: &o.markers},
		"chicken":               {target: &o.chicken},
		"window":                {target: &o.window},
		"saves":                 {target: &o.saves},
		"mission":               {target: &o.mission},
		"picker":                {target: &o.picker},
		"skill":                 {target: skill, list: skillNames()},
		"shop-stock-multiplier": {target: &o.shopStockMultiplier},
	}
}

// launchRange is a bounded Number's range as the usage line writes it.
func launchRange(st launchset.Setting) string { return fmt.Sprintf("%d..%d", st.Min, st.Max) }

// registerLaunchFlags defines the flags of every row of launchset.Table with
// the row's names, defaults and help texts, and returns their binding.
func registerLaunchFlags(fs *flag.FlagSet, o *options, skill, mods *string) map[string]launchFlag {
	flags := launchFlags(o, skill, mods)
	bind := func(name string) launchFlag {
		f, ok := flags[name]
		if !ok {
			panic("againrom: launch flag -" + name + " sets no option")
		}
		return f
	}
	for _, st := range launchset.Table {
		switch st.Kind {
		case launchset.Switch, launchset.Override:
			f := bind(st.Flag)
			fs.BoolVar(f.target.(*bool), st.Flag, st.GameBool(), st.Help(f.list...))
		case launchset.Choice:
			for _, c := range st.Options[1:] {
				f := bind(c.Flag)
				fs.BoolVar(f.target.(*bool), c.Flag, false, c.Usage)
			}
		case launchset.Number:
			f := bind(st.Flag)
			n := f.target.(*int)
			if st.Read != launchset.ReadWhole {
				fs.IntVar(n, st.Flag, st.GameInt(), st.Help(f.list...))
				continue
			}
			*n = st.GameInt()
			fs.Func(st.Flag, st.Help(f.list...), func(v string) error {
				got, err := st.Whole(v)
				if err == nil {
					*n = got
				}
				return err
			})
		case launchset.Text, launchset.Verbatim:
			f := bind(st.Flag)
			if st.Repeat {
				fs.Func(st.Flag, st.Help(f.list...), f.target.(func(string) error))
				continue
			}
			fs.StringVar(f.target.(*string), st.Flag, st.Default, st.Help(f.list...))
		}
	}
	return flags
}

// checkLaunchBounds refuses a ReadInt number outside its row's bounds.
func checkLaunchBounds(flags map[string]launchFlag) error {
	for _, st := range launchset.Table {
		if st.Kind == launchset.Number && st.Read == launchset.ReadInt {
			if err := st.Bounds(*flags[st.Flag].target.(*int)); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkLaunchExclusions refuses a given flag whose row excludes a switch that
// is on.
func checkLaunchExclusions(flags map[string]launchFlag, given map[string]bool) error {
	for _, st := range launchset.Table {
		x, ok := launchset.ByKey(st.Excludes)
		if ok && given[st.Flag] && *flags[x.Flag].target.(*bool) {
			return fmt.Errorf("-%s and -%s are mutually exclusive", x.Flag, st.Flag)
		}
	}
	return nil
}
