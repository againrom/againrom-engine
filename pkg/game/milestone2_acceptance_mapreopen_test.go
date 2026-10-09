//go:build sessioncorpusaudit

package game

import (
	"strconv"
	"strings"
	"testing"
)

// TestMilestone2MapReopen is the independent acceptance instrument for the
// "map reopen" row of pipeline/SAV-COMPLETION.md's owner-milestone-2 table.
//
// Fields compared (SAV-HEAD-025): the head's Mission dword (world+0x80) and
// MapName CString (world+0x28), read here by this file's own layout walk --
// u32 CounterA, u32 CounterB, one length-prefixed CString at body offset 8,
// eleven reserved u32, then Mission, then Difficulty -- never through
// sav.File.Head, which is pkg/formats/sav's own decode of the identical
// bytes (campaign.go readHead). Difficulty is out of this row's stated scope
// (SAV-COMPLETION.md names only "rebuilt from the ALM by mission number" and
// "saved map name string unused") and is not compared here.
//
// SAV-HEAD-025 also states the ORIGINAL's own reopen mechanism: Mission is a
// mode switch, and when nonzero the load arm reopens MapName with a
// "Scenario\" prefix. This engine's importer (originalsave.go
// RestoreOriginal) instead looks up the map from Mission's numeric value
// alone, through its own mission-to-map table (OriginalSaveResume.MapName
// carries the string only for a load-menu label -- originalsave.go:1655).
//
// PARTY-SESSION-008
func TestMilestone2MapReopen(t *testing.T) {
	fileCount, liveChecked := 0, 0
	stemMismatches, numberMismatches, addressMismatches := 0, 0, 0
	betweenMission := 0
	var refused []milestone2ResumeRefusal

	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		b := mf.body
		if len(b) < 9 {
			t.Fatalf("body is %d bytes, too short for a head", len(b))
		}
		n := int(b[8])
		if n == 0xff || 9+n+11*4+16 > len(b) {
			t.Fatalf("map name length byte %#02x makes an %d-byte head layout unreadable", b[8], len(b))
		}
		mapName := string(b[9 : 9+n])
		p := 9 + n + 11*4 // past the eleven reserved dwords (SAV-HEAD-025)
		mission := rawU32(b, p)
		fileCount++

		if mission == 0 {
			// SAV-HEAD-025: Mission 0 is the between-mission mode switch; the
			// original itself does not reopen a map from this record. Nothing
			// to compare against live resumed state, which this engine also
			// never builds for a between-mission save (mf.present is false
			// here in every corpus member observed).
			betweenMission++
			return
		}

		// File-internal cross-check, no live state needed: SAV-HEAD-025 says
		// Mission is itself parsed out of the map file name (SESS-LOAD-009:
		// "server+0x80 == 0 ... atoi's the part before the dot into
		// server+0x80" -- see this file's own package doc comment above for
		// what the same routine does instead once Mission is already
		// nonzero, which is this test's own live-checked case). The corpus's
		// own map names are
		// "<mission>.alm", so the numeric stem must equal the Mission dword on
		// every legitimately-authored save; a mismatch here is a malformed or
		// hand-edited save, not an importer defect, but is counted rather than
		// assumed away.
		stem := strings.TrimSuffix(strings.ToLower(mapName), ".alm")
		if stemVal, err := strconv.Atoi(stem); err != nil || uint32(stemVal) != mission {
			stemMismatches++
			t.Errorf("map name %q's numeric stem does not agree with Mission %d", mapName, mission)
		}

		if !mf.present {
			return
		}
		ms, _, err := loadOriginalMission(fe, mf.raw)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		liveChecked++
		if uint32(ms.Number) != mission {
			numberMismatches++
			t.Errorf("live Mission.Number %d, file Mission %d", ms.Number, mission)
		}
		// The original's own reopen re-derives the map's own name from the
		// stored MapName string; this engine's mission-to-map table starts
		// from Mission's numeric value alone. Confirming the two name the
		// same map file is what shows the two mechanisms reach the same
		// result although the importer never reads mapName above for
		// selection -- the row's own stated gap.
		wantSuffix := "/" + strings.ToLower(mapName)
		if !strings.HasSuffix(strings.ToLower(ms.Address), wantSuffix) {
			addressMismatches++
			t.Errorf("live Address %q does not name file map %q", ms.Address, mapName)
		}
	})

	t.Logf("map reopen: %d file(s) with a readable head (%d between-mission), %d resumed and live-checked, %d refused",
		fileCount, betweenMission, liveChecked, len(refused))
	milestone2LogRefusals(t, "map reopen", refused)
	t.Logf("map reopen mismatches: %d name/Mission stem, %d live Mission.Number, %d live Address", stemMismatches, numberMismatches, addressMismatches)
	if stemMismatches != 0 || numberMismatches != 0 || addressMismatches != 0 {
		t.Fatalf("%d stem, %d number, %d address mismatch(es)", stemMismatches, numberMismatches, addressMismatches)
	}
}
