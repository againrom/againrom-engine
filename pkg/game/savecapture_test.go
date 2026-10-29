package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
)

// captureGroupEffect runs one mission capture of a LOADed World and returns
// the live Group lists before, inside the build and after it returns.
func captureGroupEffect(t *testing.T, mode saveCaptureMode) (before, during, after string) {
	t.Helper()
	f := loadedGroupFrontForDialog(t, 0, 1)
	before = groupListsForDialog(f)
	hash := f.live.world.Hash()
	if err := f.captureSave(mode, true, func(Snapshot, string) error {
		during = groupListsForDialog(f)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after = groupListsForDialog(f)
	if (after == before) != (f.live.world.Hash() == hash) {
		t.Fatalf("mode %d: lists and hash disagree: %s -> %s", mode, before, after)
	}
	return before, during, after
}

// The mode table as built: a tentative capture builds from the pruned lists
// and leaves the live World as it was; a committed capture prunes the live
// World; a timer capture builds from the pruned lists and leaves the live
// World as a tentative one does (SAV-1251).
func TestSaveCaptureModesOnTheWorld(t *testing.T) {
	check := func(t *testing.T) []string {
		var failures []string
		before, during, after := captureGroupEffect(t, captureTentative)
		if during == before || after != before {
			failures = append(failures, "tentative "+before+" / "+during+" / "+after)
		}
		before, during, after = captureGroupEffect(t, captureCommitted)
		if during == before || after != during {
			failures = append(failures, "committed "+before+" / "+during+" / "+after)
		}
		before, during, after = captureGroupEffect(t, captureTimer)
		if during == before || after != before {
			failures = append(failures, "timer "+before+" / "+during+" / "+after)
		}
		return failures
	}
	if failures := check(t); len(failures) != 0 {
		t.Fatal(failures)
	}
	// Loss controls: each wrong row is seen by the check above.
	for _, c := range []struct {
		mode  saveCaptureMode
		wrong saveCaptureTransform
	}{
		{captureTentative, saveCaptureTransform{pruneGroups: true}},
		{captureCommitted, saveCaptureTransform{pruneGroups: true, restore: true}},
		{captureTimer, saveCaptureTransform{pruneGroups: true}},
		{captureTimer, saveCaptureTransform{}},
	} {
		kept := saveCaptureTransforms[c.mode]
		saveCaptureTransforms[c.mode] = c.wrong
		failures := check(t)
		saveCaptureTransforms[c.mode] = kept
		if len(failures) != 1 {
			t.Fatalf("mode %d row %+v: failures %v, want one", c.mode, c.wrong, failures)
		}
	}
}

// The town half of the table: the town's emptied group is removed by a
// committed capture and put back by a tentative or timer one, whether or not
// a LOAD set the town's groups.
func TestSaveCaptureModesOnTheTown(t *testing.T) {
	party := []mapload.PartyMember{{ID: "hero"}}
	for _, tc := range []struct {
		mode   saveCaptureMode
		pruned bool
	}{{captureTentative, false}, {captureCommitted, true}, {captureTimer, false}} {
		f := &FrontEnd{}
		f.Carried = party
		f.Town = &Town{cityGroups: []cityLiveGroup{
			{Payload: cityConstructedGroup(), Members: []string{"left"}},
			{Payload: cityConstructedGroup(), Members: []string{"hero"}},
		}}
		if err := f.captureSave(tc.mode, false, nil); err != nil {
			t.Fatal(err)
		}
		want := [][]string{{"left"}, {"hero"}}
		if tc.pruned {
			want = want[1:]
		}
		if got := liveGroupIDs(f.Town.cityGroups); !reflect.DeepEqual(got, want) {
			t.Fatalf("mode %d: live groups %v, want %v", tc.mode, got, want)
		}
	}
}

// An unknown mode captures nothing and changes nothing.
func TestSaveCaptureRefusesAnUnknownMode(t *testing.T) {
	f := loadedGroupFrontForDialog(t, 0, 1)
	before := groupListsForDialog(f)
	called := false
	err := f.captureSave(0, true, func(Snapshot, string) error { called = true; return nil })
	if err == nil || called || groupListsForDialog(f) != before {
		t.Fatalf("mode 0: err %v, build %t, lists %s -> %s", err, called, before, groupListsForDialog(f))
	}
}
