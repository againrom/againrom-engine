package game

import (
	"testing"

	"againrom/pkg/ui"
)

// TestScenarioSourcePrefixIsTheArchiveAddress holds the editor's scenario
// source prefix equal to the scenario archive's address prefix, so a scenario
// map's source is the address the archive reads.
func TestScenarioSourcePrefixIsTheArchiveAddress(t *testing.T) {
	if got := ui.MapSourceKey(ui.SourceScenario, ""); got != scenarioPrefix {
		t.Fatalf("scenario source prefix %q, archive address prefix %q", got, scenarioPrefix)
	}
	if src := ui.ParseMapSource(scenarioPrefix + "10.alm"); src.Kind != ui.SourceScenario || src.Number != 10 || !src.HasNumber {
		t.Fatalf("ParseMapSource(%q) = %+v", scenarioPrefix+"10.alm", src)
	}
	if n := sourceMission(scenarioPrefix + "10.alm"); n != 10 {
		t.Fatalf("sourceMission = %d, want 10", n)
	}
}
