package main

import (
	"flag"
	"io"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/game"
	launchset "againrom/pkg/launch"
)

// The shipped defaults the tests name, from the launch table.
var (
	defaultVolume  = launchset.Volume.GameInt()
	defaultMarkers = launchset.Markers.GameBool()
)

// launchValue is the option a flag of the table sets in o, as text; -skill
// reads as the resolved slot.
func launchValue(t *testing.T, o options, flagName string) string {
	t.Helper()
	if flagName == launchset.Skill.Flag {
		return strconv.Itoa(int(o.skill))
	}
	var skill string
	switch p := launchFlags(&o, &skill)[flagName].target.(type) {
	case *bool:
		return strconv.FormatBool(*p)
	case *int:
		return strconv.Itoa(*p)
	case *string:
		return *p
	}
	t.Fatalf("flag -%s sets no option", flagName)
	return ""
}

// TestLaunchSettingsTableReachesTheGame walks launch.Settings: every game flag
// of a row is defined with the row's default, the arguments the starter
// builds for each accepted value parse to that value, and a value one past
// each bound is refused by the starter's build and by the game.
func TestLaunchSettingsTableReachesTheGame(t *testing.T) {
	var o options
	var skill string
	fs := flag.NewFlagSet("walk", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bound := registerLaunchFlags(fs, &o, &skill)
	defined := 0
	fs.VisitAll(func(*flag.Flag) { defined++ })
	if defined != len(bound) {
		t.Fatalf("%d flags defined from the table, %d bound", defined, len(bound))
	}
	if launchset.Volume.Max != audio.MasterUnit || launchset.Volume.GameInt() != audio.MasterUnit {
		t.Fatalf("volume bound %d and default %d, mixer unit %d", launchset.Volume.Max, launchset.Volume.GameInt(), audio.MasterUnit)
	}
	control, err := parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	skillName := strings.ToLower(data.SkillName(data.SkillGeneral + 1))
	for _, st := range launchset.Settings {
		check := func(args []string, flagName, want string) {
			t.Helper()
			got, err := parse(args)
			if err != nil {
				t.Errorf("%s %q: refused: %v", st.Key, args, err)
				return
			}
			if v := launchValue(t, got, flagName); v != want {
				t.Errorf("%s %q: -%s is %s, want %s", st.Key, args, flagName, v, want)
			}
		}
		switch st.Kind {
		case launchset.Switch, launchset.Override:
			if v := launchValue(t, control, st.Flag); v != strconv.FormatBool(st.GameBool()) {
				t.Errorf("%s: game default %s", st.Key, v)
			}
			var values []string
			if st.Kind == launchset.Switch {
				values = []string{"true", "false"}
			}
			for _, o := range st.Options {
				values = append(values, o.Value)
			}
			for _, v := range values {
				args, err := st.Args(v)
				want := strconv.FormatBool(st.GameBool())
				if st.Kind == launchset.Switch {
					want = v
				} else if i, _ := st.Index(v); i > 0 {
					want = strconv.FormatBool(st.Options[i].On)
				}
				if err != nil {
					t.Fatal(err)
				}
				check(args, st.Flag, want)
			}
		case launchset.Choice:
			for i, c := range st.Options {
				args, err := st.Args(c.Value)
				if err != nil {
					t.Fatal(err)
				}
				for j, other := range st.Options[1:] {
					check(args, other.Flag, strconv.FormatBool(i == j+1))
				}
			}
		case launchset.Number:
			if v := launchValue(t, control, st.Flag); v != strconv.Itoa(st.GameInt()) {
				t.Errorf("%s: game default %s", st.Key, v)
			}
			edges := []int{st.Min}
			if st.Max > 0 {
				edges = append(edges, st.Max)
			}
			for _, n := range edges {
				args, err := st.Args(strconv.Itoa(n))
				if err != nil {
					t.Fatalf("%s %d: starter refused: %v", st.Key, n, err)
				}
				check(args, st.Flag, strconv.Itoa(n))
			}
			past := []int{st.Min - 1}
			if st.Max > 0 {
				past = append(past, st.Max+1)
			}
			for _, n := range past {
				if args, err := st.Args(strconv.Itoa(n)); err == nil {
					t.Errorf("%s %d: starter passed %q", st.Key, n, args)
				}
				got, err := parse([]string{"-" + st.Flag, strconv.Itoa(n)})
				if st.Read == launchset.ReadAtStart {
					if _, ok := game.MissionMap(got.startMission(nil)); err != nil || ok {
						t.Errorf("%s %d: parse %v, or the game would start it", st.Key, n, err)
					}
					continue
				}
				if err == nil {
					t.Errorf("%s %d: game accepted it", st.Key, n)
				}
			}
		case launchset.Text:
			value, want := "some-dir", "some-dir"
			if st.Flag == launchset.Skill.Flag {
				slot, _ := skillSlot(skillName)
				value, want = skillName, strconv.Itoa(int(slot))
			}
			args, err := st.Args(value)
			if err != nil {
				t.Fatal(err)
			}
			check(args, st.Flag, want)
		}
		if st.Excludes != "" {
			x, _ := launchset.ByKey(st.Excludes)
			value := func(s launchset.Setting) string {
				switch s.Key {
				case st.Key:
					return strconv.Itoa(st.Min)
				case x.Key:
					return "true"
				}
				return s.Load("")
			}
			if args, err := launchset.Build(value); err == nil {
				t.Errorf("%s with %s: starter passed %q", st.Key, x.Key, args)
			}
			if _, err := parse([]string{"-" + x.Flag, "-" + st.Flag, strconv.Itoa(st.Min)}); err == nil {
				t.Errorf("%s with %s: game accepted both", st.Key, x.Key)
			}
		}
	}
}
