package modrt

import (
	"fmt"
	"testing"

	"againrom/pkg/mod"
)

// A mod whose numbered map files all replace nothing marks no save; one that
// replaces a mission keeps its mark. Fixtures only, on both games' bases.
func TestAModWhoseMapsReplaceNothingIsStateFree(t *testing.T) {
	for _, base := range []string{"rom1-en", "rom2-en"} {
		for name, c := range map[string]struct {
			files    map[string]string
			campaign []int
			free     bool
		}{
			"q5-99":      {map[string]string{"99.alm": "x"}, []int{20}, true},
			"f-root20":   {map[string]string{"20.alm": "x"}, nil, true},
			"replacing":  {map[string]string{"20.alm": "x"}, []int{20}, false},
			"one of two": {map[string]string{"20.alm": "x", "maps/99.alm": "x"}, []int{20}, false},
			"rows":       {map[string]string{"main.star": missionsMain, "data/missions.toml": replacingSrc, "m/en/map.alm": "x"}, nil, false},
			"manifest":   {map[string]string{}, nil, true},
		} {
			t.Run(base+" "+name, func(t *testing.T) {
				res, err := Load(makeMods(t, map[string]map[string]string{"m": c.files}), base, nil, Options{})
				if err != nil {
					t.Fatal(err)
				}
				admitted, _, err := res.Missions.Listed(func(n int) error {
					for _, h := range c.campaign {
						if h == n {
							return nil
						}
					}
					return fmt.Errorf("number %d is no mission of this campaign", n)
				})
				if err != nil {
					t.Fatal(err)
				}
				set := res.Set.Settled(admitted)
				if got := set.Mods[0].StateFree; got != c.free {
					t.Errorf("state-free %v, want %v", got, c.free)
				}
				if marked := len(set.Marked().Mods) == 1; marked == c.free {
					t.Errorf("marked %v with state-free %v", marked, c.free)
				}
			})
		}
	}
}

// Before the install is asked, a mod of map files alone is not yet state-free.
func TestAModOfMapFilesIsSettledOnlyByTheInstall(t *testing.T) {
	res, err := Load(makeMods(t, map[string]map[string]string{"m": {"99.alm": "x"}}), "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if e := res.Set.Mods[0]; e.StateFree || !e.MapsOnly {
		t.Fatalf("%+v", e)
	}
	var none mod.MissionData
	if !res.Set.Settled(none).Mods[0].StateFree {
		t.Fatal("not settled")
	}
}
