package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/launch"
)

// The values of the sound and video rows as the tests name them.
const (
	soundDefault = launch.SoundDefault
	soundOn      = launch.SoundOn
	soundOff     = launch.SoundOff
	videoNormal  = launch.VideoNormal
	video4x      = launch.Video4x
	video8x      = launch.Video8x
)

// splitArgs is the extra arguments' split with the starter's refusal.
func splitArgs(line string) ([]string, error) {
	args, ok := launch.Split(line)
	if !ok {
		return nil, captionErr(cExtraQuote)
	}
	return args, nil
}

// launchSamples are values of st the game accepts, with the arguments the
// starter must pass for each, and values one past each bound, which both
// programs refuse.
func launchSamples(st launch.Setting) (accepted map[string][]string, refused []string) {
	accepted = map[string][]string{}
	switch st.Kind {
	case launch.Switch:
		for _, on := range []bool{false, true} {
			var want []string
			if on != (st.Default == "true") {
				want = []string{"-" + st.Flag}
				if !on {
					want = []string{"-" + st.Flag + "=false"}
				}
			}
			accepted[launch.FormatBool(on)] = want
		}
	case launch.Override:
		for i, o := range st.Options {
			var want []string
			if i > 0 {
				want = []string{"-" + st.Flag + "=" + strconv.FormatBool(o.On)}
			}
			accepted[o.Value] = want
		}
	case launch.Choice:
		for i, o := range st.Options {
			var want []string
			if i > 0 {
				want = []string{"-" + o.Flag}
			}
			accepted[o.Value] = want
		}
	case launch.Number:
		accepted[""] = nil
		for _, n := range []int{st.Min, st.Min + 1, st.Max} {
			if n == 0 && st.Max == 0 {
				continue
			}
			want := []string{"-" + st.Flag, strconv.Itoa(n)}
			if strconv.Itoa(n) == st.Default {
				want = nil
			}
			accepted[strconv.Itoa(n)] = want
		}
		refused = append(refused, strconv.Itoa(st.Min-1))
		if st.Max > 0 {
			refused = append(refused, strconv.Itoa(st.Max+1))
		}
	case launch.Text:
		accepted[""] = nil
		accepted["some value"] = []string{"-" + st.Flag, "some value"}
	case launch.Arguments:
		accepted[`-x "a b"`] = []string{"-x", "a b"}
		refused = append(refused, `"open`)
	}
	return accepted, refused
}

// TestLaunchSettingsTableReachesTheStarter walks launch.Settings: every row
// has one view, and its default and bounds reach the starter's defaults, its
// settings file round trip, its argument list and its refusals.
func TestLaunchSettingsTableReachesTheStarter(t *testing.T) {
	if len(optionViews) != len(launch.Settings) {
		t.Fatalf("%d views for %d rows", len(optionViews), len(launch.Settings))
	}
	for _, st := range launch.Settings {
		v := viewOf(st.Key)
		edits := st.Kind == launch.Number || st.Kind == launch.Text || st.Kind == launch.Arguments
		if edits != (v.edit != fNone) || edits == (v.act != aNone) || len(v.values) != len(st.Options) {
			t.Errorf("%s: view controls do not fit the kind", st.Key)
		}
		if (st.Excludes != "") != (v.clash != "") || (st.Kind == launch.Number || st.Kind == launch.Arguments) != (v.refusal != "") {
			t.Errorf("%s: view refusals do not fit the row", st.Key)
		}
		if got := defaultSettings(); got.option(st) != st.Load("") {
			t.Errorf("%s: default %q, row %q", st.Key, got.option(st), st.Load(""))
		}
		accepted, refused := launchSamples(st)
		for value, want := range accepted {
			s := defaultSettings()
			s.setOption(st, value)
			f := mustParse(t, "")
			s.store(f)
			if line := strings.TrimSpace(st.Key+" = "+value) + "\n"; !strings.Contains(string(f.Bytes()), line) {
				t.Errorf("%s=%q: not written:\n%s", st.Key, value, f.Bytes())
			}
			if again := loadSettings(f); again.option(st) != st.Load(value) {
				t.Errorf("%s=%q: read back %q", st.Key, value, again.option(st))
			}
			args, err := s.launchArgs(gameLaunch{Root: "root", ModsDir: "mods"})
			if err != nil || !reflect.DeepEqual(args, append([]string{"-assets", "root"}, want...)) {
				t.Errorf("%s=%q: arguments %q %v, want %q", st.Key, value, args, err, want)
			}
		}
		for _, value := range refused {
			s := defaultSettings()
			s.setOption(st, value)
			args, err := s.launchArgs(gameLaunch{Root: "root", ModsDir: "mods"})
			if err == nil {
				t.Errorf("%s=%q: passed as %q", st.Key, value, args)
				continue
			}
			if st.Kind == launch.Number && !strings.Contains(err.Error(), strconv.Quote(value)) {
				t.Errorf("%s=%q: refusal %q does not name the value", st.Key, value, err)
			}
			if v.bounds && !strings.Contains(err.Error(), fmt.Sprintf("from %d to %d", st.Min, st.Max)) {
				t.Errorf("%s=%q: refusal %q does not name the bounds", st.Key, value, err)
			}
		}
	}
}

// TestLaunchSettingsBoundsInCaptions holds the captions that write a bound
// as text to the table's bounds, and shows a bounded label in both languages.
func TestLaunchSettingsBoundsInCaptions(t *testing.T) {
	vol := launch.Volume
	for lang, want := range map[string][]string{
		"en": {fmt.Sprintf("%d-%d", vol.Min, vol.Max), fmt.Sprintf("from %d to %d", vol.Min, vol.Max)},
		"ru": {fmt.Sprintf("%d-%d", vol.Min, vol.Max), fmt.Sprintf("от %d до %d", vol.Min, vol.Max)},
	} {
		if !strings.HasSuffix(caption(lang, cVolume), want[0]) || !strings.Contains(caption(lang, cBadVolume, "x"), want[1]) {
			t.Errorf("%s volume captions do not name the bounds %v", lang, want)
		}
	}
	shop := launch.ShopStockMultiplier
	for _, language := range []string{"english", "russian"} {
		a := rigWithLanguage(t, language).app()
		openSettings(t, a)
		want := fmt.Sprintf("(%d..%d, ROM1)", shop.Min, shop.Max)
		found := false
		for _, it := range a.layout() {
			found = found || strings.HasSuffix(it.text, want)
		}
		if !found {
			t.Errorf("%s window shows no label ending %q", language, want)
		}
	}
	s := defaultSettings()
	s.Mission, s.Picker = "3", true
	if _, err := s.launchArgs(gameLaunch{Root: "root", ModsDir: "mods"}); err == nil || err.Error() != caption("en", cPickerMission) {
		t.Errorf("mission with picker: %v", err)
	}
}

// TestDefaultSettingsFileBytes holds the settings file the starter writes for
// the default choices to its bytes before the table.
func TestDefaultSettingsFileBytes(t *testing.T) {
	f := mustParse(t, "")
	defaultSettings().store(f)
	const want = "[starter]\nagainrom =\nclose-on-play = false\nlast-base =\n[mods]\ndir =\nenabled =\naccept-unmarked = false\n" +
		"[options]\nsound = default\nvolume =\nmovies = true\nvideo = normal\nmarkers = false\nchicken = false\nwindow = false\n" +
		"saves =\nmission =\npicker = false\nskill =\nshop-stock-multiplier = 1\nextra =\n"
	if got := string(f.Bytes()); got != want {
		t.Fatalf("default settings file\n%q\nwant\n%q", got, want)
	}
}
