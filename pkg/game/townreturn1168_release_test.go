package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// This raw oracle does not call File.Campaign or either current-state
// projector. Framing is fixed by SAV-CAMPAIGN-076..085.
func townReturnRaw1168(t *testing.T, raw []byte) string {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	b := f.TailRest
	u32 := func() uint32 {
		if len(b) < 4 {
			t.Fatal("truncated raw campaign")
		}
		n := binary.LittleEndian.Uint32(b)
		b = b[4:]
		return n
	}
	array := func() []int {
		n := int(u32())
		if n > len(b)/2 {
			t.Fatal("raw array exceeds campaign")
		}
		out := make([]int, n)
		for i := range out {
			out[i] = int(binary.LittleEndian.Uint16(b[2*i:]))
		}
		b = b[2*n:]
		return out
	}
	var announced []int
	record := func(child bool) int {
		mission := int(u32())
		for i := 0; i < 4; i++ {
			u32()
		}
		if u32() != 0 {
			announced = append(announced, mission)
		}
		array()
		array()
		if child {
			u32()
		}
		return mission
	}
	main := record(false)
	var children []int
	for n := u32(); n > 0; n-- {
		children = append(children, record(true))
	}
	for n := u32(); n > 0; n-- {
		u32()
	} // paired u16 pools
	for n := u32(); n > 0; n-- {
		u32()
	} // u32 hire flags
	array()
	array() // current and permanent mercenary lists
	npc, inn, school, shop := array(), array(), array(), array()
	var documents [][2]int
	for n := u32(); n > 0; n-- {
		documents = append(documents, [2]int{int(u32()), int(u32())})
	}
	selected := u32()
	if len(f.Players) != 1 || f.Players[0].Participant != 0 {
		t.Fatal("unexpected player graph")
	}
	if f.Head.Mission != 0 {
		t.Fatal("settled town wrote a mission header")
	}
	return fmt.Sprintf("main=%d selected=%d children=%v announced=%v npc=%v inn=%v school=%v shop=%v docs=%v money=%d",
		main, selected, children, announced, npc, inn, school, shop, documents, f.Players[0].Money)
}

func townReturnLive1168(t *testing.T, f *FrontEnd) string {
	t.Helper()
	main, selected := f.Town.Chapter(), f.Town.selectedMission()
	var children []int
	if f.Town.progress == nil {
		selected = main // The existing generated-city selection policy.
		for _, n := range candidateSides(f.Town.ChapterData()) {
			if !f.Town.Done(n) {
				children = append(children, n)
			}
		}
	} else {
		for _, c := range f.Town.progress.children {
			children = append(children, c.mission)
		}
	}
	var npc, inn, school, shop []int
	for _, o := range f.Town.Offers(TownTavern) {
		npc = append(npc, o.NPC)
		inn = append(inn, o.Mission)
	}
	for _, o := range f.Town.Offers(TownSchool) {
		school = append(school, o.Mission)
	}
	for _, o := range f.Town.Offers(TownShop) {
		shop = append(shop, o.Mission)
	}
	var documents [][2]int
	for _, d := range f.Town.documents {
		documents = append(documents, [2]int{d.Value, d.Kind})
	}
	return fmt.Sprintf("main=%d selected=%d children=%v announced=%v npc=%v inn=%v school=%v shop=%v docs=%v money=%d",
		main, selected, children, f.Town.Available(), npc, inn, school, shop, documents, f.Town.Gold())
}

func townReturnExpect1168(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("campaign changed:\n got %s\nwant %s", got, want)
	}
}

// Drive the production town row dispatch. Hearing an NPC to the end inside the
// tavern commits nothing: the inn queues the mission and registers it when the
// player leaves. No test writes Town.Take/taken.
func townReturnHear1168(t *testing.T, f *FrontEnd) int {
	t.Helper()
	screen := f.TownScreen().(*townScreen)
	if screen.room == roomSquare {
		for i, d := range townDoors {
			if d.room == roomTavern {
				screen.Choose(i)
				break
			}
		}
	}
	if screen.room != roomTavern {
		t.Fatal("not in tavern", screen.room)
	}
	screen.Choose(0)
	if screen.room != roomTalk || len(screen.townLines()) == 0 {
		t.Fatal("installed NPC conversation missing")
	}
	mission := screen.offer.Mission
	if slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("mission %d was at the gates before its conversation was heard: %v", mission, f.Town.Available())
	}
	for n := 0; screen.room == roomTalk && n < 64; n++ {
		if action := screen.Choose(0); action.Msg != "" {
			t.Fatalf("page %d of the conversation posted %q", n+1, action.Msg)
		}
	}
	if screen.room != roomTavern {
		t.Fatal("the conversation did not end in the tavern", screen.room)
	}
	if slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("mission %d reached the gates inside the tavern: %v", mission, f.Town.Available())
	}
	return mission
}

// townReturnLeaveTavern leaves the tavern by the production Exit route, which
// registers what its conversations queued.
func townReturnLeaveTavern(t *testing.T, f *FrontEnd, mission int) {
	t.Helper()
	screen := f.TownScreen().(*townScreen)
	if !screen.Back() || screen.room != roomSquare {
		t.Fatal("the tavern did not close", screen.room)
	}
	if !slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("leaving the tavern did not put mission %d on the available list %v", mission, f.Town.Available())
	}
}

func townReturnSave1168(t *testing.T, f *FrontEnd, dir string) (string, []byte) {
	t.Helper()
	store := SaveStore{Dir: dir}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("ordinary SAVE did not write SAV", name, err)
	}
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	return name, raw
}

func townReturnLoad1168(t *testing.T, dir, name string) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	_, _, load := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, nil)
	if IsOriginal(name) {
		name = localOriginalSaveToken(name)
	}
	if _, town, err := load(name); err != nil || !town {
		t.Fatal("fresh town LOAD", town, err)
	}
	return f
}

func townReturnImported1168(t *testing.T) (*FrontEnd, string) {
	t.Helper()
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	private := filepath.Join(t.TempDir(), "source.sav")
	if err := WriteConvertedSave(private, raw, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	_, _, load := agsSaveSeams(f, SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(private)}, nil)
	if _, town, err := load("source.sav"); err != nil || !town {
		t.Fatal(town, err)
	}
	t.Cleanup(func() {
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(raw, after) {
			t.Fatal("preserved original source changed")
		}
	})
	return f, private
}

func townReturnCycle1168(t *testing.T, f *FrontEnd, private, before, after string) {
	t.Helper()
	townReturnExpect1168(t, townReturnLive1168(t, f), before)
	if f.Offered != f.Town.Chapter() {
		t.Fatal("missing return advisory", f.Offered)
	}
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	// A serialized AGS baseline keeps the advisory. SAV normalization must
	// not erase it on Snapshot or mutate either the source or live state.
	ags, err := EncodeSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	baseline, _, err := DecodeSave(ags)
	if err != nil || baseline.Offered != f.Offered {
		t.Fatal("AGS advisory lost", err)
	}
	export := f.ExportNativeCitySave
	if f.originalCity != nil {
		export = f.ExportOriginalSave
	}
	control := s
	control.Offered = 0
	with, err := export(s, label)
	if err != nil {
		t.Fatal(err)
	}
	without, err := export(control, label)
	if err != nil || bytes.Equal(with, without) {
		t.Fatal("current advisory was not carried independently in SAV", err)
	}
	for _, offered := range []int{-1, f.Town.Chapter() + 1, 65535} {
		custom := s
		custom.Offered = offered
		out, err := export(custom, label)
		if err != nil || bytes.Equal(out, without) {
			t.Fatal("current advisory was not carried in SAV", offered, err)
		}
		dir := t.TempDir()
		name := fmt.Sprintf("advisory-%d.sav", offered)
		if err := os.WriteFile(filepath.Join(dir, name), out, 0600); err != nil {
			t.Fatal(err)
		}
		loaded := townReturnLoad1168(t, dir, name)
		if loaded.Offered != offered {
			t.Fatal("SAV lost current advisory", loaded.Offered, offered)
		}
	}
	travel := s
	travel.WorldMapReturn = &SnapshotMapReturn{Mission: f.Town.Chapter() - 10, Shown: 1}
	if _, err := export(travel, label); err == nil {
		t.Fatal("pending world-map return became a third save point")
	}
	current, _, err := f.Snapshot(false)
	if err != nil || !reflect.DeepEqual(current, s) {
		t.Fatal("export mutated live source", err)
	}
	dir := t.TempDir()
	first, raw := townReturnSave1168(t, f, dir)
	townReturnExpect1168(t, townReturnRaw1168(t, raw), before)
	if private != "" {
		if err := os.Remove(private); err != nil {
			t.Fatal(err)
		}
	}
	g := townReturnLoad1168(t, dir, first)
	if g.Offered != s.Offered || g.originalCity == nil {
		t.Fatal("SAV LOAD did not carry current advisory and bind source writer", g.Offered)
	}
	townReturnExpect1168(t, townReturnLive1168(t, g), before)
	unchanged, unchangedRaw := townReturnSave1168(t, g, dir)
	townReturnExpect1168(t, townReturnRaw1168(t, unchangedRaw), before)
	g = townReturnLoad1168(t, dir, unchanged)
	mission := townReturnHear1168(t, g)
	townReturnExpect1168(t, townReturnLive1168(t, g), before)
	townReturnLeaveTavern(t, g, mission)
	townReturnExpect1168(t, townReturnLive1168(t, g), after)
	second, raw2 := townReturnSave1168(t, g, dir)
	townReturnExpect1168(t, townReturnRaw1168(t, raw2), after)
	a, _ := sav.Open(unchangedRaw)
	b, _ := sav.Open(raw2)
	if bytes.Equal(a.Body, b.Body) && bytes.Equal(a.Store, b.Store) {
		t.Fatal("the mission registered on leaving the tavern was not encoded")
	}
	h := townReturnLoad1168(t, dir, second)
	townReturnExpect1168(t, townReturnLive1168(t, h), after)
	gold := h.Town.Gold()
	h.Town.Won(h.Town.Chapter() - 10)
	if h.Town.Gold() != gold {
		t.Fatal("completed prior main paid a second time")
	}
	if out := os.Getenv("AGAINROM_1168_ARTIFACTS"); out != "" {
		for suffix, payload := range map[string][]byte{"baseline.ags": ags, "first.sav": raw, "second.sav": raw2} {
			if err := WriteConvertedSave(filepath.Join(out, strings.ReplaceAll(t.Name(), "/", "-")+"-"+suffix), payload, os.Getenv("AGAINROM_ASSETS")); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("ordinary SAVE => %s; source-free LOAD/current advisory; unchanged SAVE and a conversation heard inside the inn preserve state; leaving the inn => %s; source-bound second SAVE/LOAD keeps actor graph, store and payment", before, after)
}

func TestReleaseTownReturnCurrentCampaign1168(t *testing.T) {
	t.Run("fresh-mission20", func(t *testing.T) {
		f := releaseFront(t)
		party := f.ChargenParty(ui.ChargenResult{Name: "Town return", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
		app := f.App("1168 controlled completion")
		if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := f.LiveCompleteCampaign(); err != nil {
			t.Fatal(err)
		}
		if err := app.OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
			t.Fatal(err)
		}
		// The real opener supplies World/party/IDs/roster. This calls the
		// completion boundary, not a claimed played victory trigger.
		next, line, err := f.LiveCompleteCampaign()
		if err != nil || next != 0 {
			t.Fatal(next, line, err)
		}
		t.Log(line)
		// Mission 10's own section grants three text documents on entry
		// (REG-SCN-097/MISSION-DOC-021), which the original fixture never
		// reached because it skipped straight to mission 20. Chapters 20 and
		// 30 grant none, so this is entirely mission 10's own AddTextDocument
		// keys and not a route artifact from anywhere else on the way to town.
		townReturnCycle1168(t, f, "",
			"main=30 selected=30 children=[31] announced=[] npc=[22] inn=[30] school=[] shop=[31] docs=[[1 1] [2 1] [3 1]] money=600",
			"main=30 selected=30 children=[31] announced=[30] npc=[] inn=[] school=[] shop=[31] docs=[[1 1] [2 1] [3 1]] money=600")
	})
	t.Run("imported-campaign-only", func(t *testing.T) {
		f, private := townReturnImported1168(t)
		// Campaign-only endpoint, not a full imported mission return: the
		// source actor graph deliberately remains unchanged.
		if _, accepted := f.Town.Won(30); !accepted {
			t.Fatal("main30 not completed")
		}
		f.Offered = 40
		f.arriveInTown()
		townReturnCycle1168(t, f, private,
			"main=40 selected=40 children=[41] announced=[] npc=[22 90] inn=[40 41] school=[] shop=[] docs=[[1 1] [2 1] [3 1]] money=1683",
			"main=40 selected=40 children=[41] announced=[40] npc=[90] inn=[41] school=[] shop=[] docs=[[1 1] [2 1] [3 1]] money=1683")
	})
}

func TestReleaseTownReturnImportedActorReturn1168(t *testing.T) {
	f, private := townReturnImported1168(t)
	app := f.App("1168 imported controlled completion")
	if err := app.OpenMission(f.MissionOpenerWith(30, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.ExportOriginalSave(s, label)
	if err != nil {
		t.Fatal("current same-graph return refused", err)
	}
	dropped := 0
	for _, binding := range f.originalCity.bindings {
		expected, err := binding.expectedMember()
		if err != nil || binding.returned == nil {
			t.Fatal(err)
		}
		for _, member := range s.Party {
			if member.ID == binding.partyID && expected.OriginalHuman != nil && expected.Saved != nil && member.OriginalHuman == nil && member.Saved == nil {
				dropped++
			}
		}
	}
	if dropped != 2 {
		t.Fatal("expected two exact CarryRoster provenance drops", dropped)
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := agsSaveSeams(f, store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("current actor return must write SAV", name, err)
	}
	if err := os.Remove(private); err != nil {
		t.Fatal(err)
	}
	g := townReturnLoad1168(t, store.Dir, name)
	townReturnExpect1168(t, townReturnLive1168(t, g), "main=40 selected=40 children=[41] announced=[] npc=[22 90] inn=[40 41] school=[] shop=[] docs=[[1 1] [2 1] [3 1]] money=1683")
	if g.Offered != f.Offered {
		t.Fatal("SAV advisory did not survive", g.Offered, f.Offered)
	}
	for i := range g.Carried {
		if !reflect.DeepEqual(g.Carried[i].Carry.LiveLoad, f.Carried[i].Carry.LiveLoad) {
			t.Fatal("cold SAV lost current actor values")
		}
	}
	t.Log("current imported mission30 return: immutable source baseline, separate returned Human values, ordinary SAV, source-free LOAD and unchanged campaign40")
}
