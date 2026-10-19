package game

import (
	"strings"
	"testing"

	"againrom/pkg/sim"
)

func TestSaveCensusKeepsCurrentMissionSightAndUnmodifiedTownSight(t *testing.T) {
	for _, tc := range []struct {
		name string
		scan uint8
		word uint16
	}{
		{"Light", 7, 1955},
		{"Darkness", 3, 931},
		{"Unmodified", 5, 1443},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entity := sim.Entity{Mind: 20, Reaction: 21, ScanRange: tc.scan}
			sheet := liveHeroSheet(entity)
			sheet.stats[1], sheet.stats[2] = 21, 20
			sheet.sight = tc.word
			if bad := censusMismatch(sheet, entity, false); len(bad) != 0 {
				t.Fatal(bad)
			}
			sheet.sight = 1443
			if bad := censusMismatch(sheet, entity, true); len(bad) != 0 {
				t.Fatal(bad)
			}
			if tc.scan != 5 {
				bad := censusMismatch(sheet, entity, false)
				if len(bad) != 1 || !strings.HasPrefix(bad[0], "Sight ") {
					t.Fatalf("unmodified mission sight must fail: %v", bad)
				}
				sheet.sight = tc.word
				bad = censusMismatch(sheet, entity, true)
				if len(bad) != 1 || !strings.HasPrefix(bad[0], "Sight ") {
					t.Fatalf("effect-modified town sight must fail: %v", bad)
				}
			}
		})
	}
}
