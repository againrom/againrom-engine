package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
)

// A town SAV writes the first-MapPoint word from the party's position
// (SAV-1331): a loaded town whose word is 0 stands on the selected mission's
// map object and writes 0 again; 1 stands on MapPoint 0 and writes 1.
func TestReleaseCitySaveProjectsCurrentMapPoint(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	reachabilityWalkArrive(t, f, "City point witness")
	f.Town.Won(10)
	f.Town.Won(20)
	f.arriveInTown()
	f.addChapterCompanions(f.Town.Chapter())
	write := func(front *FrontEnd) []byte {
		t.Helper()
		dir := t.TempDir()
		save, _, _ := front.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
		name, err := save(false)
		if err != nil || filepath.Ext(name) != ".sav" {
			t.Fatal("ordinary city SAVE", name, err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	doc, err := sav.DecodeDocumentData(write(f))
	if err != nil || doc.World != nil || doc.Head.Mission != 0 || doc.Campaign.Scalars[4] != 1 {
		t.Fatal("fixture is not a correctly located city", err)
	}
	for _, flag := range []uint32{0, 1} {
		t.Run(fmt.Sprint(flag), func(t *testing.T) {
			doc.Campaign.Scalars[4] = flag
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			g := releaseFront(t)
			opener, town, err := g.RestoreOriginal(raw)
			if err != nil || !town || opener != nil {
				t.Fatal("city LOAD", err)
			}
			before, _, err := g.Snapshot(false)
			if err != nil || before.Campaign.FirstMapPoint != (flag != 0) {
				t.Fatal("input flag was not retained literally", err)
			}
			out, err := sav.DecodeDocumentData(write(g))
			if err != nil || out.World != nil || out.Head.Mission != 0 || out.Campaign.Scalars[4] != flag {
				t.Fatal("city SAVE moved the party", out.Campaign.Scalars[4], err)
			}
			after, _, err := g.Snapshot(false)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("SAVE mutated captured city state", err)
			}
		})
	}
}
