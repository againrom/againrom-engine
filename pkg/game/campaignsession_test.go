package game

import (
	"errors"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// These tests run the campaign lifecycle on a bare CampaignSession and plain
// values. No FrontEnd, install or screen is built.

func TestNewCampaignCandidateIsFreshAndIsolated(t *testing.T) {
	units := &terrain.UnitSet{Bodies: map[string]*terrain.UnitClass{"a": {}}}
	c, err := newCampaignCandidate(Campaign{}, units, 2)
	if err != nil {
		t.Fatalf("newCampaignCandidate: %v", err)
	}
	if c.difficulty != mapload.Difficulty(2) || c.town == nil || !c.fame.Known {
		t.Fatalf("candidate = difficulty %v town %v fame %+v", c.difficulty, c.town, c.fame)
	}
	c.units.Bodies["b"] = nil
	if _, leaked := units.Bodies["b"]; leaked {
		t.Fatal("candidate unit bundle shares its body map with the source")
	}
	if _, err := newCampaignCandidate(Campaign{}, nil, 9); err == nil {
		t.Fatal("difficulty 9 was accepted")
	}
}

func TestAdmitMissionRefusals(t *testing.T) {
	if _, _, err := admitMission(NewTown(Campaign{}), 0, false, 0); err == nil ||
		!strings.Contains(err.Error(), "not a campaign mission number") {
		t.Fatalf("unknown mission: %v", err)
	}
	if _, _, err := admitMission(nil, 10, false, mapload.Difficulty(7)); err == nil {
		t.Fatal("bad difficulty was accepted")
	}
	if _, d, err := admitMission(nil, 10, false, 0); err != nil || d != mapload.DifficultyNormal {
		t.Fatalf("zero difficulty = %v, %v; want Normal", d, err)
	}
}

func TestSessionAdoptThenClear(t *testing.T) {
	var s CampaignSession
	c, err := newCampaignCandidate(Campaign{}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	c.carried = []mapload.PartyMember{{Name: "a"}}
	c.offered = 40
	c.quickSpells = [4]uint32{1, 2, 3, 4}
	s.Shop = &Shop{}
	s.adopt(c)
	if s.Town != c.town || s.Difficulty != 3 || s.Offered != 40 || len(s.Carried) != 1 ||
		s.quickSpells[3] != 4 || s.Shop != nil || !s.fame.Known {
		t.Fatalf("adopt left %+v", s)
	}
	s.live, s.liveMission = &mapWorld{}, 7
	s.clear(Campaign{})
	if s.Town == c.town || s.Carried != nil || s.Offered != 0 || s.Difficulty != 0 ||
		s.live != nil || s.liveMission != 0 || s.fame.Known || s.quickSpells != ([4]uint32{}) {
		t.Fatalf("clear left %+v", s)
	}
}

func TestActivateLiveReturnsOnlyAReplacedWorld(t *testing.T) {
	s := CampaignSession{Town: NewTown(Campaign{})}
	first, second := &mapWorld{}, &mapWorld{}
	if out := s.activateLive(first, 0, nil); out != nil {
		t.Fatalf("first activation replaced %v", out)
	}
	if out := s.activateLive(first, 0, nil); out != nil {
		t.Fatal("re-activating the live map reported it as replaced")
	}
	if out := s.activateLive(second, 0, nil); out != first || s.live != second {
		t.Fatalf("replacement returned %v, live %v", out, s.live)
	}
	if out := s.endLive(); out != second || s.live != nil || s.liveMission != 0 {
		t.Fatalf("endLive returned %v, session %+v", out, s)
	}
}

func TestNextPartyPrefersTheCarriedParty(t *testing.T) {
	var s CampaignSession
	def := []mapload.PartyMember{{Name: "default"}}
	if got := s.nextParty(func() []mapload.PartyMember { return def }); got[0].Name != "default" {
		t.Fatalf("empty carry gave %v", got)
	}
	s.Carried = []mapload.PartyMember{{Name: "carried"}}
	if got := s.nextParty(func() []mapload.PartyMember { t.Fatal("fallback ran"); return nil }); got[0].Name != "carried" {
		t.Fatalf("carry gave %v", got)
	}
}

func TestRecordWinOnASessionAloneAdvancesWithoutTheTown(t *testing.T) {
	campaign := miniCampaign(t, declaring(10, 20))
	s := CampaignSession{Town: NewTown(campaign), Offered: 5}
	arrived := false
	next, msg := s.recordWin(campaign, 10, nil, nil, nil, nil,
		townArrival{arrive: func() { arrived = true }, welcome: func(int) { arrived = true }})
	if next != 20 || arrived || s.Offered != 0 || !s.Town.Done(10) {
		t.Fatalf("next %d arrived %v offered %d done %v (%s)", next, arrived, s.Offered, s.Town.Done(10), msg)
	}
	if want := "mission 10 won - your party carries over; mission 20 follows and opens now"; msg != want {
		t.Fatalf("message %q, want %q", msg, want)
	}
}

// recordingAudio is a mission audio service that is not the production one.
type recordingAudio struct{ attached, released, replies, effects int }

func (a *recordingAudio) attach(*ui.Viewer)                 { a.attached++ }
func (a *recordingAudio) release(*ui.Viewer)                { a.released++ }
func (a *recordingAudio) wireReplies(*mapWorld)             { a.replies++ }
func (a *recordingAudio) wireEffects(*mapWorld, *ui.Viewer) { a.effects++ }

// entryFixture is a mission entry assembled from components: an install over a
// synthetic archive, a bare campaign session, a profile with no stored
// options, and the audio service the test supplies. No FrontEnd is built.
type entryFixture struct {
	campaign Campaign
	ports    missionPorts
}

func newEntryFixture(t *testing.T, audio missionAudio) entryFixture {
	t.Helper()
	r, err := reg.Parse(scenarioFixture())
	if err != nil {
		t.Fatalf("parse the scenario fixture: %v", err)
	}
	campaign := resolved(ReadCampaign(r), nil)
	in := &InstallResources{Campaign: campaign, Archives: &Archives{Containers: missionArchive(t, 10, 20)}, Tiles: &terrain.Tileset{}}
	return entryFixture{campaign: campaign.Value(), ports: missionPorts{
		audio:    audio,
		install:  installMission{in: in},
		profile:  &PersistenceContext{},
		session:  &CampaignSession{},
		art:      func() viewerArt { return viewerArt{} },
		advance:  func(_ int, _ *Mission, advance ui.MapAdvance) ui.MapAdvance { return advance },
		display:  missionDisplay{textSmoothing: true, frameSmoothing: true},
		cityBase: nil,
	}}
}

func (e entryFixture) request() missionRequest {
	return missionRequest{n: 10, party: MissionParty(nil, data.BodyList{}, nil), town: NewTown(e.campaign)}
}

// A different audio service drives the whole mission entry with no change to
// the entry sequence: every attach is wired once and nothing is released.
func TestMissionEntryReachesAudioOnlyThroughThePort(t *testing.T) {
	audio := &recordingAudio{}
	e := newEntryFixture(t, audio)
	if _, err := enterMission(e.request(), e.ports); err != nil {
		t.Fatalf("enterMission: %v", err)
	}
	if audio.attached != 1 || audio.replies != 1 || audio.effects != 1 || audio.released != 0 {
		t.Fatalf("audio calls = %+v, want one attach, replies and effects and no release", *audio)
	}
}

// A successful entry commits the live map to the session it was handed and
// installs the seams the map screen runs through.
func TestMissionEntryCommitsToTheSessionItIsGiven(t *testing.T) {
	audio := &recordingAudio{}
	e := newEntryFixture(t, audio)
	p, err := enterMission(e.request(), e.ports)
	if err != nil {
		t.Fatalf("enterMission: %v", err)
	}
	s := e.ports.session
	if s.live == nil || s.liveMission != 10 || p.viewer == nil || p.tick == nil || p.advance == nil {
		t.Fatalf("live %v mission %d prepared %+v", s.live != nil, s.liveMission, p)
	}
	if s.live.view != p.viewer {
		t.Fatal("the live map is not the prepared viewer's")
	}
}

// An entry that routes its commit to r.activate changes the session only when
// that assignment runs.
func TestMissionEntryDefersTheCommitToActivate(t *testing.T) {
	e := newEntryFixture(t, &recordingAudio{})
	var activate func()
	r := e.request()
	r.activate = &activate
	if _, err := enterMission(r, e.ports); err != nil {
		t.Fatalf("enterMission: %v", err)
	}
	if e.ports.session.live != nil || activate == nil {
		t.Fatal("the session changed before activate ran, or activate was not supplied")
	}
	activate()
	if e.ports.session.live == nil || e.ports.session.liveMission != 10 {
		t.Fatal("activate did not commit the live map")
	}
}

// An entry refused after audio was attached releases it and commits nothing.
func TestRefusedMissionEntryReleasesItsAudio(t *testing.T) {
	audio := &recordingAudio{}
	e := newEntryFixture(t, audio)
	r := e.request()
	refusal := errors.New("refused")
	r.prepare = func(*alm.Map) (*originalFog, error) { return nil, refusal }
	if _, err := enterMission(r, e.ports); !errors.Is(err, refusal) {
		t.Fatalf("enterMission error = %v, want the refusal", err)
	}
	if audio.attached != 1 || audio.released != 1 || audio.replies != 0 {
		t.Fatalf("audio calls = %+v, want one attach and one release", *audio)
	}
	if e.ports.session.live != nil {
		t.Fatal("a refused entry committed a live map")
	}
}

// An entry refused before decode never touches audio.
func TestMissionEntryRefusedByTheCampaignTouchesNoAudio(t *testing.T) {
	audio := &recordingAudio{}
	e := newEntryFixture(t, audio)
	r := e.request()
	r.n = 0
	if _, err := enterMission(r, e.ports); err == nil {
		t.Fatal("unknown mission entered")
	}
	if *audio != (recordingAudio{}) {
		t.Fatalf("audio calls = %+v, want none", *audio)
	}
}

// A fresh entry reads the install only through the install port: a source
// that refuses the decode stops the entry before audio, start or commit.
type refusingInstall struct{ missionInstall }

func (refusingInstall) decodeMission(string) (*MapView, error) { return nil, errors.New("no map") }

func TestMissionEntryStopsWhenTheInstallRefusesTheMap(t *testing.T) {
	audio := &recordingAudio{}
	e := newEntryFixture(t, audio)
	e.ports.install = refusingInstall{e.ports.install}
	if _, err := enterMission(e.request(), e.ports); err == nil || err.Error() != "no map" {
		t.Fatalf("enterMission error = %v, want the install's refusal", err)
	}
	if *audio != (recordingAudio{}) || e.ports.session.live != nil {
		t.Fatalf("audio %+v live %v after a refused decode", *audio, e.ports.session.live != nil)
	}
}
