package game

import (
	"encoding/binary"
	"image"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func campaignProjectionAt(c Campaign, mission int) sav.CampaignProjection {
	ch := c.Chapters[mission]
	p := sav.CampaignProjection{
		Main: sav.CampaignRecord{
			Mission: uint32(mission), MapObject: uint32(c.MapObjects[mission]),
			Payment: uint32(ch.Payment), ShopMin: uint32(ch.ShopMin), ShopMax: uint32(ch.ShopMax),
		},
		SelectedMission: uint32(mission), FirstMapPoint: true,
		MercenaryWorking: make([]uint16, 15), MercenaryPristine: make([]uint16, 15),
		MercenaryHired: make([]bool, 15),
	}
	return p
}

func cloneCampaignProjectionForTest(in sav.CampaignProjection) sav.CampaignProjection {
	out := in
	out.Main.AddHero = append([]uint16(nil), in.Main.AddHero...)
	out.Main.EnableMercenary = append([]uint16(nil), in.Main.EnableMercenary...)
	out.Children = append([]sav.CampaignRecord(nil), in.Children...)
	for i := range out.Children {
		out.Children[i].AddHero = append([]uint16(nil), in.Children[i].AddHero...)
		out.Children[i].EnableMercenary = append([]uint16(nil), in.Children[i].EnableMercenary...)
	}
	out.MercenaryWorking = append([]uint16(nil), in.MercenaryWorking...)
	out.MercenaryPristine = append([]uint16(nil), in.MercenaryPristine...)
	out.MercenaryHired = append([]bool(nil), in.MercenaryHired...)
	out.Mercenaries = append([]uint16(nil), in.Mercenaries...)
	out.PermanentMercenaries = append([]uint16(nil), in.PermanentMercenaries...)
	out.InnNPC = append([]uint16(nil), in.InnNPC...)
	out.InnMission = append([]uint16(nil), in.InnMission...)
	out.TCMission = append([]uint16(nil), in.TCMission...)
	out.ShopMission = append([]uint16(nil), in.ShopMission...)
	out.Documents = append([]sav.CampaignDocument(nil), in.Documents...)
	out.Markers = append([]sav.CampaignMarker(nil), in.Markers...)
	return out
}

type originalCampaignWriter struct{ b []byte }

func (w *originalCampaignWriter) u32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *originalCampaignWriter) u16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *originalCampaignWriter) u16s(v []uint16) {
	w.u32(uint32(len(v)))
	for _, n := range v {
		w.u16(n)
	}
}
func (w *originalCampaignWriter) record(r sav.CampaignRecord, child bool) {
	for _, v := range []uint32{r.Mission, r.MapObject, r.Payment, r.ShopMin, r.ShopMax} {
		w.u32(v)
	}
	if r.Announced {
		w.u32(1)
	} else {
		w.u32(0)
	}
	w.u16s(r.AddHero)
	w.u16s(r.EnableMercenary)
	if child {
		w.u32(r.Age)
	}
}

func encodeOriginalCampaignProjection(in sav.CampaignProjection) []byte {
	w := originalCampaignWriter{}
	w.record(in.Main, false)
	w.u32(uint32(len(in.Children)))
	for _, child := range in.Children {
		w.record(child, true)
	}
	w.u32(uint32(len(in.MercenaryWorking)))
	for _, n := range in.MercenaryWorking {
		w.u16(n)
	}
	for _, n := range in.MercenaryPristine {
		w.u16(n)
	}
	w.u32(uint32(len(in.MercenaryHired)))
	for _, hired := range in.MercenaryHired {
		if hired {
			w.u32(1)
		} else {
			w.u32(0)
		}
	}
	for _, values := range [][]uint16{in.Mercenaries, in.PermanentMercenaries, in.InnNPC,
		in.InnMission, in.TCMission, in.ShopMission} {
		w.u16s(values)
	}
	w.u32(uint32(len(in.Documents)))
	for _, d := range in.Documents {
		w.u32(d.Value)
		w.u32(d.Kind)
	}
	w.u32(in.SelectedMission)
	w.u32(0)
	w.u32(in.AutoGetMission)
	w.u32(in.LastMission)
	if in.FirstMapPoint {
		w.u32(1)
	} else {
		w.u32(0)
	}
	w.u32(in.MissionTime)
	w.u32(in.ScoreEvents)
	w.u32(uint32(len(in.Markers)))
	for _, marker := range in.Markers {
		w.u32(marker.Value)
		w.u32(uint32(len(marker.Picture) + 1))
		w.b = append(w.b, marker.Picture...)
		w.b = append(w.b, 0)
		w.u32(marker.Field0)
		w.u32(marker.Field1)
	}
	return w.b
}

func originalSaveWithCampaign(t *testing.T, mission uint32, projection sav.CampaignProjection) []byte {
	t.Helper()
	saved := savedFile(mission, tenActors())
	tail := int(binary.LittleEndian.Uint32(saved[4:])) + 0x100
	if tail > len(saved) {
		t.Fatalf("synthetic original tail starts at %d past %d bytes", tail, len(saved))
	}
	out := append([]byte(nil), saved[:tail]...)
	out = append(out, synth.Reg(0, nil)...)
	return append(out, encodeOriginalCampaignProjection(projection)...)
}

func restoredCampaignFixture() (Campaign, sav.CampaignProjection) {
	c := Campaign{
		Main: []int{30, 40, 50, 60}, Side: []int{41, 51}, Offered: []int{30, 40, 50, 51, 60},
		MapObjects: map[int]int{41: 4, 50: 5, 51: 6, 60: 7, 61: 8, 62: 9},
		Chapters: map[int]Chapter{
			30: {Mission: 30, AddHero: []int{22}, Inn: []int{30}},
			40: {Mission: 40, Inn: []int{40}},
			50: {Mission: 50, InnNPC: []int{1, 2, 3}, Inn: []int{0, 50, 51}, ShopMin: 100, ShopMax: 1000},
			60: {Mission: 60, InnNPC: []int{4, 5}, Inn: []int{60, 61}, Shop: []int{62},
				Mercenaries: []int{4}, EnableMercenary: []int{5}, Payment: 900},
			41: {Mission: 41}, 51: {Mission: 51},
			61: {Mission: 61, Payment: 100}, 62: {Mission: 62, Payment: 200},
		},
	}
	p := sav.CampaignProjection{
		Main: sav.CampaignRecord{Mission: 50, MapObject: 5, Payment: 700, ShopMin: 100,
			ShopMax: 1000, EnableMercenary: []uint16{3}},
		Children: []sav.CampaignRecord{
			{Mission: 41, MapObject: 4, Payment: 70, Announced: true, EnableMercenary: []uint16{10}, Age: 1},
			{Mission: 51, MapObject: 6, Payment: 80},
		},
		Mercenaries: []uint16{3}, PermanentMercenaries: []uint16{1, 2},
		InnNPC: []uint16{1, 2, 3}, InnMission: []uint16{0, 50, 51},
		Documents:       []sav.CampaignDocument{{Value: 1, Kind: DocumentText}},
		SelectedMission: 41, AutoGetMission: 50, LastMission: 40, MissionTime: 99,
		Markers: []sav.CampaignMarker{{Value: 41, Picture: "mark.bmp", Field0: 7, Field1: 8}},
	}
	for i := 0; i < 15; i++ {
		p.MercenaryWorking = append(p.MercenaryWorking, uint16(i+1))
		p.MercenaryPristine = append(p.MercenaryPristine, uint16(i+11))
		p.MercenaryHired = append(p.MercenaryHired, i == 2)
	}
	return c, p
}

func TestCompletingRestoredMainRebuildsTheNextRegistryRecord(t *testing.T) {
	c, projection := restoredCampaignFixture()
	c.Side = append(c.Side, 61, 62)
	c.TransitionRewards = map[int]int{50: 500}
	projection.Main = sav.CampaignRecord{Mission: 50, MapObject: 5, EnableMercenary: []uint16{3}}
	projection.SelectedMission = 50
	projection.Children[0].Age = 1 // expires on the main-load age pass
	projection.Children[1].Age = 0 // remains at age 1
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(c, progress)
	if side, ok := town.Won(50); side || !ok {
		t.Fatalf("Won(50) = side %v, ok %v", side, ok)
	}
	if town.Chapter() != 60 || town.selectedMission() != 60 {
		t.Fatalf("advanced chapter/selected = %d/%d, want 60/60", town.Chapter(), town.selectedMission())
	}
	if progress.main.mapObject != 7 || town.ChapterData().Payment != 900 ||
		!reflect.DeepEqual(town.ChapterData().Inn, []int{60, 61}) {
		t.Fatalf("next main record was not rebuilt from registry: %+v", town.ChapterData())
	}
	if progress.record(41) != nil || progress.record(51) == nil || progress.record(51).age != 1 ||
		progress.record(61) == nil || progress.record(62) == nil {
		t.Fatalf("aged/appended children = %+v", progress.children)
	}
	if !reflect.DeepEqual(progress.permanent, []int{1, 2, 3}) || !reflect.DeepEqual(progress.mercenaries, []int{4}) {
		t.Fatalf("unlocks/shelf = %v/%v", progress.permanent, progress.mercenaries)
	}
	if town.Gold() != initialPlayerPurse+500 {
		t.Fatalf("restored transition reward = %d, want %d", town.Gold(), initialPlayerPurse+500)
	}
}

func restoredTownFixture(t *testing.T) (Campaign, *Town) {
	t.Helper()
	c, projection := restoredCampaignFixture()
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	return c, newTownFromCampaignProgress(c, progress)
}

func TestRestoredCampaignReplacesFreshDefaults(t *testing.T) {
	_, town := restoredTownFixture(t)
	if town.Chapter() != 50 || town.selectedMission() != 41 || !town.Open() {
		t.Fatalf("restored chapter=%d selected=%d open=%v, want 50/41/true",
			town.Chapter(), town.selectedMission(), town.Open())
	}
	if got := town.Available(); !reflect.DeepEqual(got, []int{41}) {
		t.Fatalf("available = %v, want announced child 41", got)
	}
	offers := town.Offers(TownTavern)
	want := []TownOffer{{Index: 0, Mission: 0, NPC: 1}, {Index: 1, Mission: 50, NPC: 2}, {Index: 2, Mission: 51, NPC: 3}}
	if !reflect.DeepEqual(offers, want) {
		t.Fatalf("inn offers = %+v, want %+v", offers, want)
	}
	if town.Done(30) != true || town.Done(40) != true || town.canOpenMission(40) {
		t.Fatalf("lower progress reopened: done30=%v done40=%v canOpen40=%v",
			town.Done(30), town.Done(40), town.canOpenMission(40))
	}
	if containsMission(town.Available(), 30) || containsMission(town.Available(), 40) {
		t.Fatalf("fresh defaults leaked into restored availability: %v", town.Available())
	}
}

func TestRestoredAcceptanceConsumesCandidateBeforeAnnouncement(t *testing.T) {
	_, town := restoredTownFixture(t)
	if _, ok := town.Take(TownTavern, 0); ok {
		t.Fatal("zero inn sentinel was accepted")
	}
	if m, ok := town.Take(TownTavern, 1); !ok || m != 50 {
		t.Fatalf("Take = %d,%v, want 50,true", m, ok)
	}
	if got := town.progress.innMission; !reflect.DeepEqual(got, []int{0, 51}) {
		t.Fatalf("persisted inn candidates = %v, want [0 51]", got)
	}
	if got := town.progress.innNPC; !reflect.DeepEqual(got, []int{1, 3}) {
		t.Fatalf("persisted inn NPCs = %v, want [1 3]", got)
	}
	if !town.progress.main.announced || !containsMission(town.Available(), 50) {
		t.Fatalf("accepted main not announced: latch=%v available=%v",
			town.progress.main.announced, town.Available())
	}
}

func TestRestoredOneShotGrantsDocumentsAndSelectionMutateProjection(t *testing.T) {
	c, projection := restoredCampaignFixture()
	projection.Main.AddHero = []uint16{22}
	c.Chapters[41] = Chapter{Mission: 41, TextDocuments: []int{2}}
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(c, progress)
	if got := town.takeAddHeroes(50); !reflect.DeepEqual(got, []int{22}) {
		t.Fatalf("AddHero grant = %v, want [22]", got)
	}
	if got := town.takeAddHeroes(50); len(got) != 0 || len(progress.projection().Main.AddHero) != 0 {
		t.Fatalf("AddHero was not consumed once: second=%v projection=%v", got, progress.projection().Main.AddHero)
	}
	town.CollectDocuments(41)
	if got := town.Documents(); !reflect.DeepEqual(got, []Document{{Value: 1, Kind: DocumentText}, {Value: 2, Kind: DocumentText}}) {
		t.Fatalf("restored document append = %v", got)
	}
	town.selectMission(51)
	if progress.projection().SelectedMission != 51 {
		t.Fatalf("selected mission = %d, want 51", progress.projection().SelectedMission)
	}
	// DIV-904
	if got := progress.selectedMarkers(); !reflect.DeepEqual(got, []int{41}) {
		t.Fatalf("selected marker history = %v, want the loaded marker's own mission 41", got)
	}
}

func TestSelectedMarkersDedupesAndIgnoresNonPositiveValues(t *testing.T) {
	c, projection := restoredCampaignFixture()
	projection.Markers = []sav.CampaignMarker{
		{Value: 41, Picture: "mark.bmp"},
		{Value: 0, Picture: "mark.bmp"},
		{Value: 51, Picture: "mark.bmp"},
		{Value: 41, Picture: "mark.bmp"},
	}
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	if got := progress.selectedMarkers(); !reflect.DeepEqual(got, []int{41, 51}) {
		t.Fatalf("selected markers = %v, want [41 51]", got)
	}
}

func TestCompletingRestoredSideReturnsToMainWithoutLowerReplay(t *testing.T) {
	c, town := restoredTownFixture(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town}}
	next, line := f.FinishMissionWithRoster(41, nil, nil, nil, nil)
	if next != 0 || f.Offered != 50 || f.Town.Chapter() != 50 || f.Town.selectedMission() != 50 {
		t.Fatalf("completion next=%d offered=%d chapter=%d selected=%d; want 0/50/50/50",
			next, f.Offered, f.Town.Chapter(), f.Town.selectedMission())
	}
	if !strings.Contains(line, "restored main mission 50 remains current") {
		t.Fatalf("completion line = %q", line)
	}
	if f.Town.progress.record(41) != nil || f.Town.progress.record(51) == nil {
		t.Fatalf("children after completion = %+v", f.Town.progress.children)
	}
	if !f.Town.MercenaryEnabled(10) {
		t.Fatal("child EnableMercenary was not drained into permanent unlocks")
	}
	if f.Town.canOpenMission(40) || containsMission(f.Town.Available(), 30) || containsMission(f.Town.Available(), 40) {
		t.Fatalf("lower mission reopened after side completion: available=%v", f.Town.Available())
	}
}

func TestRestoredCampaignRoundTripsThroughAgainromEnvelope(t *testing.T) {
	c, town := restoredTownFixture(t)
	front := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town, Offered: 41}}
	snapshot, _, err := front.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSave(snapshot, "campaign projection")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.CampaignState || !reflect.DeepEqual(decoded.Campaign, snapshot.Campaign) {
		t.Fatalf("campaign envelope round trip = state %v %+v, want %+v",
			decoded.CampaignState, decoded.Campaign, snapshot.Campaign)
	}
	// CampaignProjection is authoritative over the legacy duplicate fields.
	// A decoder must not union fresh/lower defaults back into restored state.
	decoded.Available = append(decoded.Available, 30, 40)
	decoded.Documents = append(decoded.Documents, SnapshotDocument{Value: 999, Kind: DocumentText})
	decoded.MercenaryPool[3] = 999
	back := restoreTown(c, decoded)
	if back == nil || back.Chapter() != 50 || back.selectedMission() != 41 || !reflect.DeepEqual(back.Available(), []int{41}) {
		t.Fatalf("restored envelope town = %#v", back)
	}
	if back.MercenaryPool(3) != 3 || !reflect.DeepEqual(back.Documents(), []Document{{Value: 1, Kind: DocumentText}}) {
		t.Fatalf("legacy duplicate fields overrode campaign projection: pool=%d documents=%v",
			back.MercenaryPool(3), back.Documents())
	}
}

func TestMalformedCampaignSnapshotDoesNotMutateFrontEnd(t *testing.T) {
	c, town := restoredTownFixture(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town, Offered: 77}}
	bad := Snapshot{CampaignState: true}
	_, _, err := f.Restore(bad)
	if err == nil || !strings.Contains(err.Error(), "main mission") {
		t.Fatalf("Restore malformed campaign err = %v", err)
	}
	if f.Town != town || f.Offered != 77 {
		t.Fatal("malformed campaign snapshot partially changed the front end")
	}
}

func TestEmptyOriginalCampaignMarkerPictureIsRejectedAtomically(t *testing.T) {
	c, projection := restoredCampaignFixture()
	projection.Markers[0].Picture = ""
	old := NewTown(c)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: old, Offered: 77}}

	_, _, err := f.RestoreOriginal(originalSaveWithCampaign(t, 41, projection))
	if err == nil || !strings.Contains(err.Error(), "marker 0 picture is empty") {
		t.Fatalf("RestoreOriginal empty marker picture err = %v", err)
	}
	if f.Town != old || f.Offered != 77 {
		t.Fatal("empty marker picture partially changed the front end")
	}
}

func TestRestoredMainGuardRunsBeforeMapIO(t *testing.T) {
	_, town := restoredTownFixture(t)
	f := &FrontEnd{CampaignSession: CampaignSession{Town: town}}
	_, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(40)()
	if err == nil || !strings.Contains(err.Error(), "below restored main progress 50") {
		t.Fatalf("open lower mission err = %v", err)
	}
	if fresh := NewTown(f.Campaign.Value()); !fresh.canOpenMission(40) {
		t.Fatal("fresh campaign inherited the restored lower-main guard")
	}
}

func TestEveryRestoredPreTownMainStaysClosedUntilTheCampaignBoundary(t *testing.T) {
	c := townCampaign(t)
	for _, mission := range []int{10, 20} {
		t.Run(string(rune('0'+mission/10)), func(t *testing.T) {
			progress, err := campaignProgressFromSAV(c, campaignProjectionAt(c, mission))
			if err != nil {
				t.Fatal(err)
			}
			if town := newTownFromCampaignProgress(c, progress); town.Open() {
				t.Fatalf("restored main %d opened the town before mission 30", mission)
			}
		})
	}

	progress, err := campaignProgressFromSAV(c, campaignProjectionAt(c, 20))
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: newTownFromCampaignProgress(c, progress)}}
	if _, _, err := f.Snapshot(false); err == nil || !strings.Contains(err.Error(), "no game to save") {
		t.Fatalf("pre-town off-map save err = %v", err)
	}
	dest, _, _ := f.continuity(20, continuityMission(t, 20, sim.ScriptInstantLose), menuAdvance)()
	if dest != ui.NoticeToMenu || f.Town.Open() {
		t.Fatalf("pre-town defeat destination/open = %v/%v, want menu/closed", dest, f.Town.Open())
	}
	next, _ := f.FinishMissionWithRoster(20, nil, nil, nil, nil)
	if next != 0 || !f.Town.Open() || f.Town.Chapter() != 30 || f.Offered != 30 {
		t.Fatalf("boundary completion next/open/chapter/offered = %d/%v/%d/%d, want 0/true/30/30",
			next, f.Town.Open(), f.Town.Chapter(), f.Offered)
	}
	if screen := f.TownScreen().(*townScreen); !screen.AtTownSquare() {
		t.Fatal("campaign boundary did not enter the production town screen")
	}
}

func TestCampaignProjectionRejectsEveryDeadCandidateAtomically(t *testing.T) {
	c, base := restoredCampaignFixture()
	c.Side = append(c.Side, 52)
	c.Chapters[52] = Chapter{Mission: 52}
	cases := []struct {
		name   string
		mutate func(*sav.CampaignProjection)
	}{
		{"inn", func(p *sav.CampaignProjection) {
			p.InnNPC = append(p.InnNPC, 9)
			p.InnMission = append(p.InnMission, 52)
		}},
		{"school", func(p *sav.CampaignProjection) { p.TCMission = append(p.TCMission, 52) }},
		{"shop", func(p *sav.CampaignProjection) { p.ShopMission = append(p.ShopMission, 52) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projection := cloneCampaignProjectionForTest(base)
			tc.mutate(&projection)
			old := NewTown(c)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: old, Offered: 77}}
			bad := Snapshot{CampaignState: true, Campaign: projection, Mission: 41}
			if _, _, err := f.Restore(bad); err == nil || !strings.Contains(err.Error(), "no live main or retained-child record") {
				t.Fatalf("dead %s candidate err = %v", tc.name, err)
			}
			if f.Town != old || f.Offered != 77 {
				t.Fatalf("dead %s candidate partially installed", tc.name)
			}
		})
	}
}

func TestActiveCampaignMissionMustBeTheSelectedLiveRecordOnBothRestorePaths(t *testing.T) {
	c, projection := restoredCampaignFixture()
	c.Side = append(c.Side, 52)
	c.Chapters[52] = Chapter{Mission: 52}
	cases := []struct {
		name    string
		mission int
		want    string
	}{
		{"missing record", 52, "no live main or retained-child record"},
		{"not selected", 51, "does not match selected mission 41"},
	}
	for _, tc := range cases {
		t.Run("againrom "+tc.name, func(t *testing.T) {
			old := NewTown(c)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: old, Offered: 77}}
			bad := Snapshot{CampaignState: true, Campaign: cloneCampaignProjectionForTest(projection), Mission: tc.mission}
			if _, _, err := f.Restore(bad); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("active snapshot err = %v, want %q", err, tc.want)
			}
			if f.Town != old || f.Offered != 77 {
				t.Fatal("active snapshot mismatch partially installed")
			}
		})
		t.Run("original "+tc.name, func(t *testing.T) {
			old := NewTown(c)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: old, Offered: 77}}
			saved := originalSaveWithCampaign(t, uint32(tc.mission), projection)
			if _, _, err := f.RestoreOriginal(saved); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("active original err = %v, want %q", err, tc.want)
			}
			if f.Town != old || f.Offered != 77 {
				t.Fatal("active original mismatch partially installed")
			}
		})
	}
}

func TestPreTownCampaignCannotEnterThroughATownOnlySave(t *testing.T) {
	c := townCampaign(t)
	projection := campaignProjectionAt(c, 20)
	for _, tc := range []struct {
		name string
		load func(*FrontEnd) error
	}{
		{"againrom", func(f *FrontEnd) error {
			_, _, err := f.Restore(Snapshot{CampaignState: true, Campaign: projection})
			return err
		}},
		{"original", func(f *FrontEnd) error {
			_, _, err := f.RestoreOriginal(originalSaveWithCampaign(t, 0, projection))
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := NewTown(c)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: old, Offered: 77}}
			if err := tc.load(f); err == nil || !strings.Contains(err.Error(), "before campaign town begins") {
				t.Fatalf("pre-town town-only load err = %v", err)
			}
			if f.Town != old || f.Offered != 77 {
				t.Fatal("pre-town town-only refusal partially installed")
			}
		})
	}
}

func restoredWorldMapFront(t *testing.T, first, importedMarker bool) *FrontEnd {
	t.Helper()
	c := townCampaign(t)
	projection := campaignProjectionAt(c, 30)
	projection.Main.Announced = true
	projection.FirstMapPoint = first
	if importedMarker {
		projection.Markers = []sav.CampaignMarker{{Value: 30, Picture: "boss", Field0: 7, Field1: 8}}
	}
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil)}, CampaignSession: CampaignSession{Town: newTownFromCampaignProgress(c, progress)}}
}

func seedWorldMapMarker(f *FrontEnd) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, 1, 1))
	f.worldMapCache.Value().markers[30] = pic
	f.worldMapCache.Value().markerTried[30] = true
	return pic
}

func TestRestoredMapPointAndImportedMarkerReachTheProductionWorldMap(t *testing.T) {
	for _, tc := range []struct {
		name  string
		first bool
		want  image.Point
	}{
		{"first point", true, image.Pt(2, 3)},
		{"selected point", false, image.Pt(32, 23)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := restoredWorldMapFront(t, tc.first, false)
			s := f.TownScreen().(*townScreen)
			s.Choose(3)
			if got := s.WorldMapView().Position; got != tc.want {
				t.Fatalf("world-map entry position = %v, want %v", got, tc.want)
			}
		})
	}

	f := restoredWorldMapFront(t, false, true)
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	want := seedWorldMapMarker(f)
	view := s.WorldMapView()
	if len(view.Missions) != 1 || view.Missions[0].Marker != want {
		t.Fatalf("imported marker cache did not cross production view: %+v", view.Missions)
	}
}

func TestSelectionAfterZeroMarkerImportRoundTripsToTheProductionWorldMap(t *testing.T) {
	f := restoredWorldMapFront(t, true, false)
	s := f.TownScreen().(*townScreen)
	s.Choose(3)
	seedWorldMapMarker(f)
	card := ui.WorldMapCardRect(0)
	centre := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	s.WorldMapClick(centre)
	if marker := s.WorldMapView().Missions[0].Marker; marker == nil {
		t.Fatal("selection did not expose the marker before saving")
	}

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.CampaignMarkerSelected || len(snapshot.Campaign.Markers) != 1 ||
		snapshot.Campaign.Markers[0].Value != 30 || snapshot.Campaign.Markers[0].Picture != `main\graphics\Global.Map\absent-marker.256` {
		t.Fatalf("selected current marker was not projected: %v/%+v",
			snapshot.CampaignMarkerSelected, snapshot.Campaign.Markers)
	}
	encoded, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	g := &FrontEnd{InstallResources: InstallResources{Campaign: f.Campaign, Archives: &Archives{Containers: worldMapFixtureFS(t)}, Font: resolved(missionFont(), nil)}}
	if open, town, err := g.Restore(decoded); err != nil || open != nil || !town {
		t.Fatalf("round-trip restore = opener %v town %v err %v", open != nil, town, err)
	}
	back := g.TownScreen().(*townScreen)
	back.Choose(3)
	want := seedWorldMapMarker(g)
	view := back.WorldMapView()
	if view.Position != image.Pt(32, 23) || len(view.Missions) != 1 || view.Missions[0].Marker != want {
		t.Fatalf("round-trip world map = position %v missions %+v", view.Position, view.Missions)
	}
}

func TestRestoredSchoolAndShopCandidatesFlowThroughProductionScreens(t *testing.T) {
	c := townCampaign(t)
	projection := campaignProjectionAt(c, 40)
	projection.Children = []sav.CampaignRecord{{Mission: 41, MapObject: uint32(c.MapObjects[41])}}
	projection.TCMission = []uint16{41}
	projection.ShopMission = []uint16{40}
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Font: resolved(missionFont(), nil), Archives: &Archives{Containers: townTextFS(t, []synth.File{{Path: "text/training/npc34m41.txt", Data: []byte("<part=1>The trainer has a job")}, {Path: "text/shop/npc31m40.txt", Data: []byte("<part=1>The merchant has a job")}})}}, CampaignSession: CampaignSession{Town: newTownFromCampaignProgress(c, progress)}}
	s := f.TownScreen().(*townScreen)

	s.Choose(2)
	if s.room != roomTalk || s.dialogueBuilding != TownSchool {
		t.Fatalf("restored school candidate entered room %v/building %v", s.room, s.dialogueBuilding)
	}
	if act := s.AdvanceTownDialogue(); act.Msg != "" {
		t.Fatalf("accepting the restored school candidate posted %q; the original's close posts no line", act.Msg)
	}
	if len(progress.tcMission) != 0 || progress.record(41) == nil || !progress.record(41).announced {
		t.Fatalf("school candidate/latch = %v/%+v", progress.tcMission, progress.record(41))
	}

	s.Back()
	s.Choose(1)
	if s.room != roomTalk || s.dialogueBuilding != TownShop {
		t.Fatalf("restored shop candidate entered room %v/building %v", s.room, s.dialogueBuilding)
	}
	if act := s.AdvanceTownDialogue(); act.Msg != "" {
		t.Fatalf("accepting the restored shop candidate posted %q; the original's close posts no line", act.Msg)
	}
	if len(progress.shopMission) != 0 || !progress.main.announced ||
		!reflect.DeepEqual(f.Town.Available(), []int{40, 41}) {
		t.Fatalf("shop candidate/latch/available = %v/%v/%v",
			progress.shopMission, progress.main.announced, f.Town.Available())
	}
}
