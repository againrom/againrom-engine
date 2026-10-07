package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestReleaseCurrentCitySessionOrdinaryCountersWin(t *testing.T) {
	f := currentTown(t, nil, nil)
	f.fame = SnapshotFame{Time: 123, Events: 456}
	f.Town.selectMission(31)
	raw := currentTownSave(t, f)
	for cycle := 0; cycle < 2; cycle++ {
		g := currentTownReload(t, raw)
		if g.fame.Known || g.fame.Time != 123+uint32(cycle) || g.fame.Events != 456+uint32(cycle) || g.Town.selectedMission() != 31 {
			t.Fatal("city current counters, completeness or selection changed", cycle, g.fame, g.Town.selectedMission())
		}
		g.fame.Time++
		g.fame.Events++
		raw = currentTownSave(t, g)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _, err := sav.NativeActions(doc.State)
	if err != nil {
		t.Fatal(err)
	}
	doc.Campaign.Scalars[0], doc.Campaign.Scalars[5], doc.Campaign.Scalars[6] = 30, 999, 888
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 2; cycle++ {
		g := currentTownReload(t, raw)
		if g.fame.Known || g.fame.Time != 999 || g.fame.Events != 888 || g.Town.selectedMission() != 30 {
			t.Fatal("native policy replaced ordinary counters or selection", cycle, g.fame, g.Town.selectedMission())
		}
		if cycle == 0 {
			back, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, _ := sav.NativeActions(back.State)
			if !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary edits changed native policy")
			}
		}
		raw = currentTownSave(t, g)
	}
	f.fame = SnapshotFame{Known: true, Time: 42, Events: 43, Result: &FameResult{Name: string([]byte{0xc4, 0xe0, 0xed}), Score: 17, Recorded: true}}
	g := currentTownReload(t, currentTownSave(t, f))
	if !reflect.DeepEqual(f.fame, g.fame) {
		t.Fatal("terminal result or byte name lost", f.fame, g.fame)
	}
}

func TestReleaseCurrentCityOffersKeepHistoryWithoutRefiltering(t *testing.T) {
	f := currentTown(t, nil, nil)
	c := f.Campaign.Value()
	ch := c.Chapters[30]
	ch.Inn, ch.InnNPC = []int{30, 31, 31, 0}, []int{22, 22, 22, 90}
	c.Chapters[30] = ch
	f.Town = NewTown(c)
	f.Town.Arrive()
	f.Town.takeAddHeroes(30)
	if mission, ok := f.Town.Take(TownTavern, 0); !ok || mission != 30 {
		t.Fatal("fixture did not consume its first offer")
	}
	want := f.Town.Offers(TownTavern)
	raw := currentTownSave(t, f)
	for cycle := 0; cycle < 2; cycle++ {
		g := currentTownReload(t, raw)
		if !reflect.DeepEqual(g.Town.Offers(TownTavern), want) || len(g.Town.taken) != cycle+1 {
			t.Fatal("compacted rows were filtered again or history/indices lost", cycle, g.Town.Offers(TownTavern), want, g.Town.taken)
		}
		if _, ok := g.Town.Take(TownTavern, 3); ok || !reflect.DeepEqual(g.Town.Offers(TownTavern), want) {
			t.Fatal("zero sentinel consumed an offer")
		}
		if cycle == 0 {
			if mission, ok := g.Town.Take(TownTavern, 1); !ok || mission != 31 {
				t.Fatal("next real offer lost after LOAD", mission, ok)
			}
			if _, ok := g.Town.Take(TownTavern, 1); ok {
				t.Fatal("consumed stable index accepted the following duplicate")
			}
			want = want[1:]
		}
		raw = currentTownSave(t, g)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _, _ := sav.NativeActions(doc.State)
	doc.Campaign.Arrays[2], doc.Campaign.Arrays[3] = []uint16{22, 90}, []uint16{30, 0}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	g := currentTownReload(t, raw)
	if offers := g.Town.Offers(TownTavern); len(offers) != 2 || offers[0].Mission != 30 || offers[0].Index != 0 || len(g.Town.taken) != 2 {
		t.Fatal("history or stale ordinal anchors overrode changed ordinary rows", offers, g.Town.taken)
	}
	unchanged, _, _ := sav.NativeActions(doc.State)
	if !bytes.Equal(leaf, unchanged) {
		t.Fatal("offer wire control changed native policy")
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	a.Session.Taken = append(a.Session.Taken, a.Session.Taken[0])
	invalid, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, invalid); err != nil {
		t.Fatal(err)
	}
	bad, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	beforeTown, beforeFame := g.Town, cloneFame(g.fame)
	if _, _, err := g.RestoreOriginal(bad); err == nil || g.Town != beforeTown || !reflect.DeepEqual(g.fame, beforeFame) {
		t.Fatal("malformed history changed the live campaign", err)
	}
}
