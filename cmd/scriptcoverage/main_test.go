package main

import (
	"os"
	"testing"
)

func TestTheClaimBackedOperationDenominatorHasFiftyTwoUniqueRows(t *testing.T) {
	keys := operationKeys()
	if len(keys) != 52 {
		t.Fatalf("operation denominator has %d rows, want 52", len(keys))
	}
	seen := map[operationKey]bool{}
	for _, key := range keys {
		if seen[key] {
			t.Fatalf("operation denominator repeats %+v", key)
		}
		seen[key] = true
		if claimOracle[key] == "" {
			t.Fatalf("operation denominator row %+v has no active claim id", key)
		}
	}
}

// TestReleaseCampaignScriptExecutionClosure is the durable lawful-install gate.
// pipeline/check-release-tests.sh discovers it from the AGAINROM_ASSETS skip,
// then runs it once against each preserved root. No asset byte or derived map is
// written to the repository.
func TestReleaseCampaignScriptExecutionClosure(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the 28-map script execution matrix needs a lawful install")
	}
	result, maps, err := loadCoverage(root)
	if err != nil {
		t.Fatalf("load production campaign: %v", err)
	}
	if err := validateDenominator(result); err != nil {
		t.Fatal(err)
	}
	if err := runControlled(result, maps); err != nil {
		t.Fatalf("controlled production seam: %v", err)
	}
	if err := runSynthetic(result, maps); err != nil {
		t.Fatalf("synthetic production seam: %v", err)
	}
	if err := runNatural(result, maps, 4000); err != nil {
		t.Fatalf("ordinary campaign drives: %v", err)
	}
	if result.Natural <= 147 {
		t.Fatalf("ordinary input strategy observed %d exact instant nodes; the standing baseline already observed 147", result.Natural)
	}
	if len(result.Operations) != 52 {
		t.Fatalf("matrix closed %d operation rows, want 52", len(result.Operations))
	}
	for _, row := range result.Operations {
		if row.Disposition != "PASS" || !row.Effect {
			t.Errorf("%s:%d has no claim-backed effect oracle", row.Family, row.Opcode)
		}
		if row.Family != "check-build" && row.Family != "instant-build" && !row.Dispatched {
			t.Errorf("%s:%d has no production dispatch witness", row.Family, row.Opcode)
		}
		if row.Family != "check-build" && row.Family != "instant-build" && row.Natural == 0 && !row.Controlled {
			t.Errorf("%s:%d is neither naturally observed nor controlled", row.Family, row.Opcode)
		}
	}
	for _, row := range result.Nodes {
		if row.Reachable && claimOracle[operationKey{row.Family, row.Opcode}] != "" {
			if row.Family == "check-build" || row.Family == "instant-build" {
				if row.Disposition != "PASS" || !row.Effect || row.Dispatched {
					t.Errorf("builder node is not closed at the builder boundary: %+v", row)
				}
			} else if row.Family != "trigger" {
				if !row.Controlled || !row.Dispatched || !row.Effect || row.Disposition != "PASS" {
					t.Errorf("reachable exact runtime node has no controlled effect closure: %+v", row)
				}
			}
		}
		if row.Family == "check" && row.Accepted && !row.Natural {
			t.Errorf("%s check node %d was accepted but not observed in an ordinary pass", row.Map, row.Node)
		}
		if row.Family == "trigger" && row.Accepted && !row.Natural {
			t.Errorf("%s trigger node %d was accepted but has no ordinary decision record", row.Map, row.Node)
		}
	}
	wantSynthetic := map[string]bool{
		syntheticItemTest12: true,
		syntheticDropAll:    true, syntheticGuard: true, syntheticRoam: true, syntheticDwell: true,
		syntheticAttackVeto: true, syntheticDefendRange: true, syntheticFollowRange: true,
	}
	wantSyntheticCount := map[string]int{
		syntheticItemTest12: 1,
		syntheticDropAll:    3, syntheticGuard: 1, syntheticRoam: 1, syntheticDwell: 1,
		syntheticAttackVeto: 1, syntheticDefendRange: 1, syntheticFollowRange: 1,
	}
	seenSynthetic := map[string]bool{}
	seenSyntheticCount := map[string]int{}
	for _, row := range result.Synthetic {
		seenSynthetic[row.Case] = true
		seenSyntheticCount[row.Case]++
		if row.Case == syntheticRoam {
			if !row.Dispatched || row.Divergence != "" {
				t.Errorf("unexpected Roam disposition: %+v", row)
			}
		}
		if row.Disposition != "PASS" || !row.Effect {
			t.Errorf("synthetic row has no effect/no-op oracle: %+v", row)
		}
	}
	for name := range wantSynthetic {
		if !seenSynthetic[name] {
			t.Errorf("synthetic section has no %q row", name)
		}
		if seenSyntheticCount[name] != wantSyntheticCount[name] {
			t.Errorf("synthetic section has %d %q rows, want %d", seenSyntheticCount[name], name, wantSyntheticCount[name])
		}
	}
	if len(result.Synthetic) != 10 {
		t.Errorf("synthetic section has %d rows, want 10", len(result.Synthetic))
	}
	t.Logf("%s: maps=%d checks=%d instants=%d triggers=%d rows=%d synthetic=%d natural_exact_instants=%d",
		result.Root, result.Maps, result.Checks, result.Instants, result.Triggers, len(result.Operations), len(result.Synthetic), result.Natural)
}
