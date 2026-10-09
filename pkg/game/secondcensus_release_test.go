package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseSecondGameSupportCensus pins the campaign support census on each
// second-game root. A slice that closes a gap moves these numbers.
func TestReleaseSecondGameSupportCensus(t *testing.T) {
	archives := secondGameRoot(t)
	census, err := SecondGameCensus(archives)
	if err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("AGAINROM_ROM2_CENSUS_OUT"); dir != "" {
		name := "census-" + filepath.Base(archives.Root) + ".tsv"
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		werr := WriteSecondCensus(f, census)
		if err := f.Close(); werr != nil || err != nil {
			t.Fatal(werr, err)
		}
	}
	placed, triggers := 4212, 715
	if LanguageSelector(archives.Containers) == 1 {
		placed, triggers = 4234, 714
	}
	got := SecondTotals(census)
	want := SecondCensusTotals{
		Maps: 46, Started: 46, Ran: 46, Lost: 1,
		Placements: placed, Withdrawn: 9, AuthoredHP: 296, BornFallen: 4, NoSpellMaps: 5,
		Events: 425, NoReasonText: 2,
		Checks: 986, Instants: 1334, Triggers: triggers, SharedNodes: 1146,
		FixtureMaps: 32, EntryMaps: 3, WinMaps: 3, Ready: 3,
		Exits: 15, GatedExits: 6, EngineExits: 4,
	}
	want.Omitted[CauseFixture] = SecondOmission{Nodes: 189, Triggers: 192, ActionTriggers: 8}
	if got != want {
		t.Errorf("census totals\n got %+v\nwant %+v", got, want)
	}
	for _, c := range census {
		blockers := c.Blockers()
		switch c.Mission {
		case 10, 20, 21:
			if len(blockers) != 0 {
				t.Errorf("mission %d blockers %v, want none", c.Mission, blockers)
			}
		case 53, 87, 92, 102:
			if !slices.Equal(c.NoSpellRule, []uint16{7}) {
				t.Errorf("mission %d spells without a rule %v, want Blizzard alone", c.Mission, c.NoSpellRule)
			}
		case 101:
			if c.Run.Outcome != sim.OutcomeLost || c.Run.Reason != 5 || c.Run.DecidedAt != 32 || c.BornFallen != 1 {
				t.Errorf("mission 101 run %+v with %d fallen placements, want lost at tick 32 for reason 5 beside one", c.Run, c.BornFallen)
			}
		case 110:
			if !slices.Equal(c.NoSpellRule, []uint16{7}) {
				t.Errorf("mission 110 spells without a rule %v, want Blizzard alone", c.NoSpellRule)
			}
			if got := censusExits(c.Departure); !slices.Equal(got, []string{"movie 5 if 779!=0:no", "movie 4 if 779=0:no"}) || !slices.Contains(c.BankSlots, 779) {
				t.Errorf("mission 110 exits %v with bank slots %v, want both outputs behind the slot its script writes", got, c.BankSlots)
			}
		case 40, 96:
			want := map[int][]int32{40: {7}, 96: {5}}[c.Mission]
			if !slices.Equal(c.NoReasonText, want) {
				t.Errorf("mission %d reasons without text %v, want %v", c.Mission, c.NoReasonText, want)
			}
		}
	}
}
