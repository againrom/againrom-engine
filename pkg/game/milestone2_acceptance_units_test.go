//go:build sessioncorpusaudit

package game

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// unitHiredInputs pins source files whose hired party members the engine
// presents under its own convention: the mercenary backing type in U148 for a
// hired Human whose raw record holds 0, and a manifest Name that is empty for a
// standard hire name or the hired type's name for a nameless Unit. Each such
// difference is dropped from the comparison only for a hired party member and
// only when the raw record is nameless or carries the standard hire name; the
// count of dropped differences is pinned.
var unitHiredInputs = map[string]int{
	"2026-10-02/projectiles-original-en/game0022.sav": 29,
	"2026-10-02/projectiles-original-en/game0023.sav": 29,
	"2026-10-02/projectiles-original-en/game0024.sav": 29,
}

var unitD8Inputs = map[string]struct {
	sha                                                    string
	rawCapacity, rawOwnWeight, liveCapacity, liveOwnWeight int
}{
	"2026-09-27/oldsaves7/game0005.sav": {"715d7f9d20b90a966ab6e9f14fb8fa4cea90097a1a41f2488ed8d5d38da414a4", 109, 17, 106, 17},
	"2026-09-27/oldsaves7/game0006.sav": {"b652cb4c5745b6dfa1a4cec12ea3be1dbb5991f44bbde97716755d6a5b8197ba", 109, 4, 83, 4},
}

type unitD8Counts struct {
	rawCapacity, rawOwnWeight   int
	liveCapacity, liveOwnWeight int
}

func unitD8Values(class string, values map[string]uint32, live bool) (map[string]uint32, bool, bool) {
	if class != "Unit" || !live {
		return values, false, false
	}
	normalized := maps.Clone(values)
	capacity, ownWeight := false, false
	if normalized["Capacity"] == 0 {
		normalized["Capacity"] = 300
		capacity = true
	}
	if normalized["U8E"] == 0 && normalized["U90"] != 0 {
		normalized["U8E"] = normalized["U90"]
		ownWeight = true
	}
	return normalized, capacity, ownWeight
}

func TestMilestone2UnitScalars(t *testing.T) {
	worlds, cities, rawRecords, worldRecords, cityRecords, live, rawOnly, bytesRead, mismatches := 0, 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[string]int{"Unit": 0, "Human": 0, "Humanoid": 0}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		want, err := unitScalarExpected(mf.f, mf.raw)
		if err != nil {
			mismatches++
			t.Errorf("independent Unit scalar read: %v", err)
			return
		}
		rawRecords += len(want.records)
		for _, r := range want.records {
			classes[r.class]++
			bytesRead += 37 + 19 + 55 + 1 + len(r.name)
		}
		if !mf.present {
			cities++
			cityRecords += len(want.records)
			// The city route has no Mission.SavedDocument/World. This is
			// explicitly a raw-to-complete-codec check, not city App acceptance.
			state, _ := decodeSavedDocument(mf.raw)
			for _, difference := range want.documentDifferences(state) {
				mismatches++
				t.Error("city decode-only Document: " + difference)
			}
			for _, r := range want.records {
				t.Logf("unit scalars: archive %d %s offset %d excluded from live comparison: between-mission document; complete codec fields compared", r.archive, r.class, r.off)
			}
			return
		}
		worldRecords += len(want.records)
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		worlds++
		// This is the original imported state BEFORE Snapshot's current-state
		// projection can conceal a decoder/retention disagreement.
		for _, difference := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("retained Document: " + difference)
		}
		worldExpected := want
		if input, allowed := unitD8Inputs[mf.rel]; allowed {
			if got := fmt.Sprintf("%x", sha256.Sum256(mf.raw)); got != input.sha {
				t.Errorf("D8 source SHA %s, want %s", got, input.sha)
			} else {
				live := map[uint16]bool{}
				for _, actor := range ms.World.Entities() {
					live[actor.SourceBinding.ArchiveIndex] = true
				}
				worldExpected.records = slices.Clone(want.records)
				var counts unitD8Counts
				for i, r := range worldExpected.records {
					if r.class == "Unit" {
						if r.values["Capacity"] == 0 {
							counts.rawCapacity++
						}
						if r.values["U8E"] == 0 && r.values["U90"] != 0 {
							counts.rawOwnWeight++
						}
					}
					values, capChanged, weightChanged := unitD8Values(r.class, r.values, live[r.archive])
					worldExpected.records[i].values = values
					if capChanged {
						counts.liveCapacity++
					}
					if weightChanged {
						counts.liveOwnWeight++
					}
				}
				if counts != (unitD8Counts{input.rawCapacity, input.rawOwnWeight, input.liveCapacity, input.liveOwnWeight}) {
					t.Errorf("D8 source words raw/live %d/%d capacity and %d/%d own weight, want %d/%d and %d/%d",
						counts.rawCapacity, counts.liveCapacity, counts.rawOwnWeight, counts.liveOwnWeight,
						input.rawCapacity, input.liveCapacity, input.rawOwnWeight, input.liveOwnWeight)
				}
				t.Logf("unit scalars: D8 %s: raw Unit zeros capacity %d, own weight %d; live normalized capacity %d, own weight %d; raw Document compared unchanged",
					mf.rel, counts.rawCapacity, counts.rawOwnWeight, counts.liveCapacity, counts.liveOwnWeight)
			}
		}
		differences, excluded, n := worldExpected.worldDifferences(ms.World, ms.ActorManifest, ms)
		if pinnedDrops, pinned := unitHiredInputs[mf.rel]; pinned {
			hired := map[uint16]string{}
			for i, member := range ms.Party {
				if i < len(ms.Start.IDs) && member.Hired() {
					if e, ok := ms.World.Entity(ms.Start.IDs[i]); ok {
						hired[e.SourceBinding.ArchiveIndex] = member.Name
					}
				}
			}
			rawName := map[uint16]string{}
			for _, r := range want.records {
				rawName[r.archive] = r.name
			}
			kept, dropped := differences[:0:0], 0
			for _, difference := range differences {
				var archive uint16
				fmt.Sscanf(difference, "archive %d ", &archive)
				_, isHired := hired[archive]
				_, standard := mercenaryHireTypeFromName(rawName[archive])
				convention := strings.HasSuffix(difference, "live manifest Name differs") ||
					strings.Contains(difference, " U148: World=") && strings.HasSuffix(difference, " raw=0x0")
				if isHired && convention && (rawName[archive] == "" || standard) {
					dropped++
					continue
				}
				kept = append(kept, difference)
			}
			if dropped != pinnedDrops {
				t.Errorf("hired party convention dropped %d differences, pinned %d", dropped, pinnedDrops)
			}
			differences = kept
		}
		live += n
		rawOnly += len(excluded)
		for _, difference := range differences {
			mismatches++
			t.Error("live: " + difference)
		}
		for _, reason := range excluded {
			t.Log("unit scalars: excluded from live scalar basis: " + reason)
		}
	})
	milestone2LogRefusals(t, "unit scalars", refused)
	t.Logf("unit scalars: %d world files resumed, %d city files decode-only; %d raw tagged actors (%d world, %d city), classes %v; %d selected wire bytes; %d live scalar bases, %d world raw-only records; %d mismatches, %d refused", worlds, cities, rawRecords, worldRecords, cityRecords, classes, bytesRead, live, rawOnly, mismatches, len(refused))
	t.Log("unit scalars: every record has 42 scalar values, 24 raw bytes and its CString checked in complete Document; low sight byte retained independently, whole-cell ScanRange checked separately; recipient-mask/control consumers and exact Humanoid runtime acceptance remain Unknown")
	if worlds == 0 || live == 0 || rawRecords == 0 {
		t.Fatal("no Unit scalar acceptance population compared")
	}
	t.Run("D8 normalization sensitivity", func(t *testing.T) {
		raw := map[string]uint32{"Capacity": 0, "U8E": 0, "U90": 7, "Health": 19}
		normalized, capChanged, weightChanged := unitD8Values("Unit", raw, true)
		human, humanCap, humanWeight := unitD8Values("Human", raw, true)
		rawOnly, rawCap, rawWeight := unitD8Values("Unit", raw, false)
		nonzero, nonzeroCap, nonzeroWeight := unitD8Values("Unit", map[string]uint32{"Capacity": 301, "U8E": 5, "U90": 7}, true)
		if !capChanged || !weightChanged || normalized["Capacity"] != 300 || normalized["U8E"] != 7 || normalized["Health"] != 19 ||
			humanCap || humanWeight || human["Capacity"] != 0 || human["U8E"] != 0 || rawCap || rawWeight || rawOnly["Capacity"] != 0 ||
			nonzeroCap || nonzeroWeight || nonzero["Capacity"] != 301 || nonzero["U8E"] != 5 || raw["Capacity"] != 0 {
			t.Fatal("D8 normalization changed a non-Unit, a nonzero word, another field, or the raw oracle")
		}
	})
}
