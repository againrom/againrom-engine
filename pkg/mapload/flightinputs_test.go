package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

func TestWorldConstructorsBindInstalledFlightInputs(t *testing.T) {
	table := &Table{FlightInputs: &sim.FlightInputs{Pictures: map[int32]sim.FlightPicture{1007: {Present: true, Phases: 11}}}}
	m := &alm.Map{Width: 40, Height: 40}
	for _, tc := range []struct {
		name  string
		build func() (*sim.World, error)
	}{
		{"map", func() (*sim.World, error) { return FromALMWith(m, table, DifficultyNormal) }},
		{"mission", func() (*sim.World, error) {
			w, _, err := StartMission(m, table, DifficultyNormal, nil)
			return w, err
		}},
		{"scripted", func() (*sim.World, error) {
			w, _, err := StartMissionScripted(m, table, DifficultyNormal, nil, nil)
			return w, err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := tc.build()
			if err != nil {
				t.Fatal(err)
			}
			if !w.HasFlightInputs() {
				t.Fatal("World reached its first tick without installed flight inputs")
			}
		})
	}
}
