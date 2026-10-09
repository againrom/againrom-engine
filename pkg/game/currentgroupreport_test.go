package game

import (
	"againrom/pkg/sim"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCurrentGroupReportFollowsRestoredWorld(t *testing.T) {
	wantIssues := []string{
		"Group 71: member archive 0 is unmaterialized",
		"Group 71: primary dispatch has unmaterialized members",
	}
	for _, path := range []string{"low level", "front end"} {
		t.Run(path, func(t *testing.T) {
			f, snapshot, before := partialCurrentGraph(t)
			wantGroups, _, _ := before.SavedGroups()
			hash := before.Hash()
			raw, err := f.ExportCurrentSave(snapshot, "current Group report")
			if err != nil {
				t.Fatal(err)
			}
			var after *sim.World
			var report string
			if path == "low level" {
				mission, resumed, err := loadOriginalMission(f, raw)
				if err != nil {
					t.Fatal(err)
				}
				after, report = mission.World, resumed.String()
				if resumed.GroupsRestored != 1 || !reflect.DeepEqual(resumed.GroupIssues, wantIssues) {
					t.Errorf("returned report uses provisional Groups: count=%d issues=%v", resumed.GroupsRestored, resumed.GroupIssues)
				}
			} else {
				report = captureCurrentGroupReport(t, func() {
					open, town, err := f.RestoreOriginal(raw)
					if err != nil || town {
						t.Fatal(town, err)
					}
					if err := f.App("current Group report").OpenMission(open); err != nil {
						t.Fatal(err)
					}
				})
				after = f.live.world
			}
			groups, _, _ := after.SavedGroups()
			if !reflect.DeepEqual(groups, wantGroups) || before.Hash() != hash {
				t.Fatalf("reporting changed current Group or source World: before=%x after=%x", hash, after.Hash())
			}
			wantLine := fmt.Sprintf("saved Groups 1 RESTORED; bounded continuation issues: %v", wantIssues)
			for _, line := range strings.Split(report, "\n") {
				if strings.Contains(line, "saved Groups ") {
					t.Logf("before=%x after=%x report=%s", hash, after.Hash(), strings.TrimSpace(line))
				}
			}
			if !strings.Contains(report, wantLine) {
				t.Errorf("published report omitted the final Group count or genuine unresolved warnings; want %q", wantLine)
			}
		})
	}
}

func captureCurrentGroupReport(t *testing.T, run func()) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resume-stderr.txt")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	func() {
		os.Stderr = file
		defer func() {
			os.Stderr = previous
			if err := file.Close(); err != nil {
				t.Error(err)
			}
		}()
		run()
	}()
	output, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
