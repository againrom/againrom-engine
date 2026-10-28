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
		Placements: placed, Withdrawn: 9, AuthoredHP: 296, BornFallen: 4, NoSpellMaps: 0,
		Events: 425, NoReasonText: 2,
		Checks: 986, Instants: 1334, Triggers: triggers, SharedNodes: 1146,
		FixtureMaps: 0, EntryMaps: 45, WinMaps: 46, Ready: 43,
		Exits: 15, GatedExits: 6, EngineExits: 14,
	}
	// The census party carries identities 21, 22 and 23, which bind every
	// hero-band value the 46 maps name (10001, 10002 and 10003).
	if got != want {
		t.Errorf("census totals\n got %+v\nwant %+v", got, want)
	}
	// The one map no published producer reaches on the controller's walks
	// over the four character choices: 52.
	var unreached []int
	for _, c := range census {
		if slices.Contains(c.Blockers(), BlockEntry) {
			unreached = append(unreached, c.Mission)
		}
	}
	if want := []int{52}; !slices.Equal(unreached, want) {
		t.Errorf("maps blocked on entry %v, want %v", unreached, want)
	}
	for _, c := range census {
		blockers := c.Blockers()
		if slices.Contains(blockers, BlockMovie) || slices.Contains(blockers, BlockSave) || slices.Contains(blockers, BlockContinuation) != (c.Mission == 50) {
			t.Errorf("mission %d blockers %v: only mission 50's unresolved record may block continuation", c.Mission, blockers)
		}
		if slices.Contains(blockers, BlockParty) || len(c.UnboundHeroes) != 0 {
			t.Errorf("mission %d blockers %v leave hero-band values %v unbound", c.Mission, blockers, c.UnboundHeroes)
		}
		switch c.Mission {
		case 10, 20, 21, 30, 31, 32, 61, 82, 93:
			if len(blockers) != 0 {
				t.Errorf("mission %d blockers %v, want none", c.Mission, blockers)
			}
		case 53, 87, 92, 102:
			// Blizzard (7) was these maps' one spell without a rule; it now
			// runs the sampled staged program.
			if !slices.Contains(c.Spells, 7) || len(c.NoSpellRule) != 0 {
				t.Errorf("mission %d spells %v, without a rule %v, want Blizzard named and every spell ruled", c.Mission, c.Spells, c.NoSpellRule)
			}
			if len(blockers) != 0 {
				t.Errorf("mission %d blockers %v, want none", c.Mission, blockers)
			}
		case 101:
			if c.Run.Outcome != sim.OutcomeLost || c.Run.Reason != 5 || c.Run.DecidedAt != 32 || c.BornFallen != 1 {
				t.Errorf("mission 101 run %+v with %d fallen placements, want lost at tick 32 for reason 5 beside one", c.Run, c.BornFallen)
			}
		case 110:
			if !slices.Contains(c.Spells, 7) || len(c.NoSpellRule) != 0 {
				t.Errorf("mission 110 spells %v, without a rule %v, want Blizzard named and every spell ruled", c.Spells, c.NoSpellRule)
			}
			if len(blockers) != 0 {
				t.Errorf("mission 110 blockers %v, want none", blockers)
			}
			if got := censusExits(c.Departure); !slices.Equal(got, []string{"movie 5 if 779!=0:yes", "movie 4 if 779=0:yes"}) || !slices.Contains(c.BankSlots, 779) {
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
