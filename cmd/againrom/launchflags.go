package main

import (
	"flag"
	"fmt"

	launchset "againrom/pkg/launch"
)

// launchFlag is where the game keeps one flag of launchset.Settings and the
// flag's help text. parse discards the flag set's output, so these texts have
// this one home and no second copy can drift.
type launchFlag struct {
	target any // *bool, *int or *string, by the row's kind
	usage  string
}

// launchFlags binds every game flag of launchset.Settings to its option; skill
// receives the -skill name that parse resolves.
func launchFlags(o *options, skill *string) map[string]launchFlag {
	return map[string]launchFlag{
		"sound":  {&o.sound, "play sound and music (overrides the saved preference; on by default)"},
		"volume": {&o.volume, fmt.Sprintf("sound and music master volume, %d-%d (overrides the saved preference)", launchset.Volume.Min, launchset.Volume.Max)},
		"movies": {&o.movies, "play installed startup, new-game and mission movies"},
		"4x":     {&o.fourX, "select VIDEO4 mission movies (overrides -8x)"},
		"8x":     {&o.eightX, "select VIDEO8 mission movies"},
		"markers": {&o.markers, "diagnostic: draw a marker on each placed structure, unit and static object " +
			"(off by default; -markers turns all three on)"},
		"chicken": {&o.chicken, "set the ROM1 #Chicken state at every mission start"},
		"window":  {&o.window, "open in a decorated window instead of over the whole screen; Alt+Enter switches between the two"},
		"saves": {&o.saves, "directory for saved games (default: saves/ beside the binary; inside an install, " +
			"Againrom/saves or the user configuration profile)"},
		"mission": {&o.mission, "override the startup campaign mission (default: the first mission of the install's campaign registry)"},
		// -picker makes NEW GAME open the picker instead of generation; a
		// normal launch never reaches the picker.
		"picker": {&o.picker, "debug: make NEW GAME open the former map picker (campaign and loose maps) instead of character generation"},
		"skill":  {skill, skillUsage},
		"shop-stock-multiplier": {&o.shopStockMultiplier, fmt.Sprintf("roll every ROM1 shop shelf a whole number of times from %d to %d",
			launchset.ShopStockMultiplier.Min, launchset.ShopStockMultiplier.Max)},
	}
}

// launchRange is a bounded Number's range as the usage line writes it.
func launchRange(st launchset.Setting) string { return fmt.Sprintf("%d..%d", st.Min, st.Max) }

// registerLaunchFlags defines the flags of every row of launchset.Settings with
// the row's names and defaults, and returns their binding.
func registerLaunchFlags(fs *flag.FlagSet, o *options, skill *string) map[string]launchFlag {
	flags := launchFlags(o, skill)
	bind := func(name string) launchFlag {
		f, ok := flags[name]
		if !ok {
			panic("againrom: launch flag -" + name + " sets no option")
		}
		return f
	}
	for _, st := range launchset.Settings {
		switch st.Kind {
		case launchset.Switch, launchset.Override:
			f := bind(st.Flag)
			fs.BoolVar(f.target.(*bool), st.Flag, st.GameBool(), f.usage)
		case launchset.Choice:
			for _, c := range st.Options[1:] {
				f := bind(c.Flag)
				fs.BoolVar(f.target.(*bool), c.Flag, false, f.usage)
			}
		case launchset.Number:
			f := bind(st.Flag)
			n := f.target.(*int)
			if st.Read != launchset.ReadWhole {
				fs.IntVar(n, st.Flag, st.GameInt(), f.usage)
				continue
			}
			*n = st.GameInt()
			fs.Func(st.Flag, f.usage, func(v string) error {
				got, err := st.Whole(v)
				if err == nil {
					*n = got
				}
				return err
			})
		case launchset.Text:
			f := bind(st.Flag)
			fs.StringVar(f.target.(*string), st.Flag, st.Default, f.usage)
		}
	}
	return flags
}

// checkLaunchBounds refuses a ReadInt number outside its row's bounds.
func checkLaunchBounds(flags map[string]launchFlag) error {
	for _, st := range launchset.Settings {
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
	for _, st := range launchset.Settings {
		x, ok := launchset.ByKey(st.Excludes)
		if ok && given[st.Flag] && *flags[x.Flag].target.(*bool) {
			return fmt.Errorf("-%s and -%s are mutually exclusive", x.Flag, st.Flag)
		}
	}
	return nil
}
