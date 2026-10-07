package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestCitySaveNoticeNamesEachApproximation is a direct, fixture-free check
// on cityExportNotice: each DIVERGENCES.md row this story added must appear
// in the disclosed notice exactly when its own writer condition holds, and
// the notice must stay empty whenever none of them do or the export is a
// mission-path one (out of scope, no notice).
func TestCitySaveNoticeNamesEachApproximation(t *testing.T) {
	knownFame := &SnapshotFame{Known: true, Time: 1, Events: 1}
	unknownFame := &SnapshotFame{Known: false}
	progress := &campaignProgress{campaignRecords: campaignRecords{main: campaignProgressRecord{mission: 30}}}

	t.Run("no approximation stays silent", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{}}}
		city := Snapshot{Fame: knownFame, Offered: 0}
		if got := cityExportNotice(n, city, 0, nil); got != "" {
			t.Fatalf("notice = %q, want empty", got)
		}
	})
	t.Run("unknown score history names DIV-1319", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{}}}
		city := Snapshot{Fame: unknownFame, Offered: 0}
		got := cityExportNotice(n, city, 0, nil)
		if !contains1194(got, "DIV-1319") {
			t.Fatalf("notice = %q, want DIV-1319", got)
		}
	})
	t.Run("orphaned import provenance names DIV-1320 and DIV-1321", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{progress: progress}, originalCity: nil}}
		city := Snapshot{Fame: knownFame, Offered: 30}
		got := cityExportNotice(n, city, 30, nil)
		if !contains1194(got, "DIV-1320") || !contains1194(got, "DIV-1321") {
			t.Fatalf("notice = %q, want DIV-1320 and DIV-1321", got)
		}
	})
	t.Run("retained import provenance stays silent on that row", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{progress: progress}, originalCity: &originalCitySaveState{}}}
		city := Snapshot{Fame: knownFame, Offered: 30}
		got := cityExportNotice(n, city, 30, nil)
		if contains1194(got, "DIV-1320") {
			t.Fatalf("notice = %q, must not claim lost provenance when originalCity is retained", got)
		}
	})
	t.Run("settled advisory mismatch names DIV-1322 from the captured value, not the normalized one", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{progress: progress}, originalCity: &originalCitySaveState{}}}
		city := Snapshot{Fame: knownFame, Offered: 30}
		got := cityExportNotice(n, city, 40, nil)
		if !contains1194(got, "DIV-1322") {
			t.Fatalf("notice = %q, want DIV-1322", got)
		}
	})
	t.Run("captured value equal to chapter stays silent on DIV-1322", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{progress: progress}, originalCity: &originalCitySaveState{}}}
		city := Snapshot{Fame: knownFame, Offered: 30}
		got := cityExportNotice(n, city, 30, nil)
		if contains1194(got, "DIV-1322") {
			t.Fatalf("notice = %q, must not claim an advisory reset when the captured value already matches the chapter", got)
		}
	})
	t.Run("mission-path export carries no notice", func(t *testing.T) {
		n := &FrontEnd{CampaignSession: CampaignSession{Town: &Town{progress: progress}}}
		city := Snapshot{Fame: unknownFame, Offered: 999}
		world := &sim.World{}
		if got := cityExportNotice(n, city, 999, world); got != "" {
			t.Fatalf("mission-path notice = %q, want empty", got)
		}
	})
}

func contains1194(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
