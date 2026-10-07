package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestNativeCompanionGrantDefaultsAndAdmission(t *testing.T) {
	c := Campaign{Main: []int{20, 30, 40}, Offered: []int{30, 40}, Chapters: map[int]Chapter{
		20: {Mission: 20}, 30: {Mission: 30, AddHero: []int{22}}, 40: {Mission: 40, AddHero: []int{25}},
	}}
	legacy := Snapshot{Open: true, Won: []int{20}}
	back := restoreTown(c, legacy)
	if got := back.takeAddHeroes(30); len(got) != 0 {
		t.Fatal("old reached-town default reissued its companion", got)
	}
	if got := back.takeAddHeroes(40); !reflect.DeepEqual(got, []int{25}) {
		t.Fatal("old default consumed a future chapter", got)
	}
	legacy.Open = false
	if got := restoreTown(c, legacy).takeAddHeroes(30); !reflect.DeepEqual(got, []int{22}) {
		t.Fatal("unopened old town lost its grant", got)
	}
	legacy.Open, legacy.HeroGrantState = true, true
	if got := restoreTown(c, legacy).takeAddHeroes(30); !reflect.DeepEqual(got, []int{22}) {
		t.Fatal("explicit current pending state was treated as legacy", got)
	}
	for _, bad := range []Snapshot{
		{ConsumedHeroGrants: []int{30}},
		{HeroGrantState: true, ConsumedHeroGrants: []int{-1}},
		{HeroGrantState: true, ConsumedHeroGrants: []int{20}},
		{HeroGrantState: true, ConsumedHeroGrants: []int{30, 30}},
		{HeroGrantState: true, ConsumedHeroGrants: []int{40, 30}},
	} {
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
		before := f.Town
		if _, _, err := f.Restore(bad); err == nil || !strings.Contains(err.Error(), "companion grant") {
			t.Fatal("malformed grant state not refused", bad, err)
		}
		if f.Town != before || f.Town.Gold() != initialPlayerPurse {
			t.Fatal("refused grant state changed the running town")
		}
	}
	current := Snapshot{HeroGrantState: true, CampaignState: true, ConsumedHeroGrants: []int{30}}
	if err := validateSnapshotHeroGrants(c, current); err != nil {
		t.Fatal("current campaign rejected independent consumed-grant history", err)
	}
	if got := restoreTown(c, current).takeAddHeroes(30); len(got) != 0 {
		t.Fatal("current campaign reissued a consumed grant", got)
	}
	// Map insertion order must not change persisted grant bytes.
	var prior []byte
	for _, order := range [][]int{{40, 30}, {30, 40}} {
		town := NewTown(c)
		for _, n := range order {
			town.takeAddHeroes(n)
		}
		var snapshot Snapshot
		snapshotTown(town, &snapshot)
		raw, err := EncodeSave(snapshot, "grant order")
		if err != nil {
			t.Fatal(err)
		}
		if prior != nil && !bytes.Equal(prior, raw) {
			t.Fatal("nondeterministic native grant bytes")
		}
		prior = raw
	}
}

func TestNativeCompanionGrantIsConsumedAndPersisted(t *testing.T) {
	c := Campaign{Main: []int{30, 40}, Chapters: map[int]Chapter{30: {Mission: 30, AddHero: []int{22}}}}
	town := NewTown(c)
	if got := town.takeAddHeroes(30); !reflect.DeepEqual(got, []int{22}) {
		t.Fatal("first grant", got)
	}
	if got := town.takeAddHeroes(30); len(got) != 0 {
		t.Fatal("consumed native grant was offered again", got)
	}
	var snapshot Snapshot
	snapshotTown(town, &snapshot)
	raw, err := EncodeSave(snapshot, "consumed grant")
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := restoreTown(c, snapshot).takeAddHeroes(30); len(got) != 0 {
		t.Fatal("AGS restored a consumed grant", got)
	}
	if got := NewTown(c).takeAddHeroes(30); !reflect.DeepEqual(got, []int{22}) {
		t.Fatal("consumption mutated the immutable registry", got)
	}
}

func nativeGrantFreshReturn(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Grant return", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("grant controlled completion")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	if next, line, err := f.LiveCompleteCampaign(); err != nil || next != 0 {
		t.Fatal(next, line, err)
	}
	if len(f.Carried) != 2 || f.Carried[1].CompanionNPC != 22 {
		t.Fatal("missing actual granted companion")
	}
	return f
}

// Read Main.AddHero's raw counted array; do not use either campaign projector.
func nativeGrantRawCount(t *testing.T, raw []byte) uint32 {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil || len(f.TailRest) < 28 {
		t.Fatal("campaign tail", err)
	}
	return binary.LittleEndian.Uint32(f.TailRest[24:28])
}

func TestReleaseNativeConsumedCompanionGrant(t *testing.T) {
	for _, absent := range []bool{false, true} {
		t.Run(map[bool]string{false: "present", true: "already-granted-member-absent"}[absent], func(t *testing.T) {
			f := nativeGrantFreshReturn(t)
			if absent {
				f.Carried = f.Carried[:1]
			} // Controlled roster loss, not a claimed dismissal UI.
			want := len(f.Carried)
			snapshot, label, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			rawAGS, err := EncodeSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, _, err = DecodeSave(rawAGS)
			if err != nil {
				t.Fatal(err)
			}
			g := releaseFront(t)
			if _, town, err := g.Restore(snapshot); err != nil || !town {
				t.Fatal("native reload", town, err)
			}
			g.addChapterCompanions(g.Town.Chapter())
			if len(g.Carried) != want {
				t.Fatal("grant repeated after AGS LOAD", len(g.Carried), want)
			}
			for cycle := 0; cycle < 2; cycle++ {
				dir := t.TempDir()
				name, raw := townReturnSave1168(t, g, dir)
				if count := nativeGrantRawCount(t, raw); count != 0 {
					t.Fatal("SAV carries consumed grant", count)
				}
				g = townReturnLoad1168(t, dir, name)
				if len(g.Carried) != want {
					t.Fatal("SAV LOAD changed party", len(g.Carried), want)
				}
			}
		})
	}
}
