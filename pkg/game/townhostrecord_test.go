package game

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/town"
)

// hostRecordDescriptions are the four shipped town descriptions the host
// reads its modes from, and the defaults a screen with none falls back to.
func hostRecordDescriptions() map[string]*town.Description {
	out := map[string]*town.Description{"defaults": ROM1TownDescription()}
	for _, name := range []string{"rom1", "rom2-town1", "rom2-town2", "rom2-town3"} {
		if d := townDescriptions[name]; d != nil {
			out[name] = d
		}
	}
	return out
}

// hostRecordFields prints every host field the code can consult, in field
// order, as text: a mode keeps its JSON text whatever type holds it.
func hostRecordFields(h town.HostSpec) string {
	v := reflect.ValueOf(h)
	var parts []string
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		if name == "Cite" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", name, v.Field(i).Interface()))
	}
	return strings.Join(parts, " ")
}

// hostRecordSites is, for every hostRules() call site that picks a branch on
// a mode, the branch it takes as a named boolean.
var hostRecordSites = []struct {
	site string
	take func(h town.HostSpec) bool
}{
	{"missionshop.go Shop!=chapter", func(h town.HostSpec) bool { return h.Shop != "chapter" }},
	{"roompage.go Pages==entry", func(h town.HostSpec) bool { return h.Pages == "entry" }},
	{"roomsound.go AudioRoom==requested", func(h town.HostSpec) bool { return h.AudioRoom == "requested" }},
	{"secondtown.go Pages==retained", func(h town.HostSpec) bool { return h.Pages == "retained" }},
	{"shopbinding.go Shop==bank", func(h town.HostSpec) bool { return h.Shop == "bank" }},
	{"townhost.go TrackSource==campaign", func(h town.HostSpec) bool { return h.TrackSource == "campaign" }},
	{"townhost.go Talk==building", func(h town.HostSpec) bool { return h.Talk == "building" }},
	{"townhost.go Rows==chapter-hints", func(h town.HostSpec) bool { return h.Rows == "chapter-hints" }},
	{"townhost.go Header==location", func(h town.HostSpec) bool { return h.Header == "location" }},
	{"townhost.go Footer==services", func(h town.HostSpec) bool { return h.Footer == "services" }},
	{"townscenehost.go Art==manifest", func(h town.HostSpec) bool { return h.Art == "manifest" }},
	{"townscenehost.go Talk==section", func(h town.HostSpec) bool { return h.Talk == "section" }},
	{"towntalk.go townTalkRules[Talk]", func(h town.HostSpec) bool {
		_, ok := townTalkRules[h.Talk]
		return ok
	}},
}

// hostRecordScreen is a town screen over one description and no install.
func hostRecordScreen(d *town.Description, second bool) *townScreen {
	town := &Town{}
	if second {
		town.second = newSecondCampaign()
	}
	s := &townScreen{shopController: shopController{sess: &CampaignSession{Town: town}, in: &InstallResources{}},
		defaults: d, art: fakeTownArt{}, tags: secondEventTags{}}
	if second {
		s.adapter = destinationTownCampaign{s}
	}
	return s
}

func hostRecordDump(t *testing.T) string {
	t.Helper()
	descs := hostRecordDescriptions()
	var names []string
	for n := range descs {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		d := descs[n]
		s := hostRecordScreen(d, d.Host.Talk == "section")
		h := s.hostRules()
		fmt.Fprintf(&b, "%s: %s\n", n, hostRecordFields(h))
		for _, site := range hostRecordSites {
			fmt.Fprintf(&b, "  %s: %v\n", site.site, site.take(h))
		}
		p := s.talkProfile()
		fmt.Fprintf(&b, "  talkProfile trim=%v heroNPC=%d\n", p.trim, p.speech.heroNPC)
		fmt.Fprintf(&b, "  StartsInTown=%v TownClickSounds=%v Header=%q Footer=%q\n", s.StartsInTown(), s.TownClickSounds(), s.Header(), s.Footer())
	}
	// A description with no host section reads the defaults.
	bare := hostRecordScreen(&town.Description{}, false)
	fmt.Fprintf(&b, "no-host: %s\n", hostRecordFields(bare.hostRules()))
	return b.String()
}

const hostRecordWant = `defaults: CurrentHookRoom=true Shop=chapter TrackSource=description RequireArt=false Rows=chapter-hints Header=chapter HeaderPrefix= Footer=party Talk=building Pages=retained Art=pages AudioRoom=building Leave=room-map Speech=true SilentSquare=true ClickSounds=true DialogueSpeech=true CacheScreen=true EagerSquare=true StartsInTown=false
  missionshop.go Shop!=chapter: false
  roompage.go Pages==entry: false
  roomsound.go AudioRoom==requested: false
  secondtown.go Pages==retained: true
  shopbinding.go Shop==bank: false
  townhost.go TrackSource==campaign: false
  townhost.go Talk==building: true
  townhost.go Rows==chapter-hints: true
  townhost.go Header==location: false
  townhost.go Footer==services: false
  townscenehost.go Art==manifest: false
  townscenehost.go Talk==section: false
  towntalk.go townTalkRules[Talk]: true
  talkProfile trim=false heroNPC=0
  StartsInTown=false TownClickSounds=true Header="the town square - the campaign offers nothing further" Footer=["gold 0    party 1    carried 1 items, 0 worn" "available []    done 0"]
rom1: CurrentHookRoom=true Shop=chapter TrackSource=description RequireArt=false Rows=chapter-hints Header=chapter HeaderPrefix= Footer=party Talk=building Pages=retained Art=pages AudioRoom=building Leave=room-map Speech=true SilentSquare=true ClickSounds=true DialogueSpeech=true CacheScreen=true EagerSquare=true StartsInTown=false
  missionshop.go Shop!=chapter: false
  roompage.go Pages==entry: false
  roomsound.go AudioRoom==requested: false
  secondtown.go Pages==retained: true
  shopbinding.go Shop==bank: false
  townhost.go TrackSource==campaign: false
  townhost.go Talk==building: true
  townhost.go Rows==chapter-hints: true
  townhost.go Header==location: false
  townhost.go Footer==services: false
  townscenehost.go Art==manifest: false
  townscenehost.go Talk==section: false
  towntalk.go townTalkRules[Talk]: true
  talkProfile trim=false heroNPC=0
  StartsInTown=false TownClickSounds=true Header="the town square - the campaign offers nothing further" Footer=["gold 0    party 1    carried 1 items, 0 worn" "available []    done 0"]
rom2-town1: CurrentHookRoom=false Shop=bank TrackSource=campaign RequireArt=true Rows=hotspots Header=location HeaderPrefix=ROM2 campaign Footer=services Talk=section Pages=entry Art=manifest AudioRoom=requested Leave=destinations Speech=false SilentSquare=false ClickSounds=false DialogueSpeech=true CacheScreen=false EagerSquare=false StartsInTown=true
  missionshop.go Shop!=chapter: true
  roompage.go Pages==entry: true
  roomsound.go AudioRoom==requested: true
  secondtown.go Pages==retained: false
  shopbinding.go Shop==bank: true
  townhost.go TrackSource==campaign: true
  townhost.go Talk==building: false
  townhost.go Rows==chapter-hints: false
  townhost.go Header==location: true
  townhost.go Footer==services: true
  townscenehost.go Art==manifest: true
  townscenehost.go Talk==section: true
  towntalk.go townTalkRules[Talk]: true
  talkProfile trim=true heroNPC=21
  StartsInTown=true TownClickSounds=false Header="ROM2 campaign: town 1" Footer=[]
rom2-town2: CurrentHookRoom=false Shop=bank TrackSource=campaign RequireArt=true Rows=hotspots Header=location HeaderPrefix=ROM2 campaign Footer=services Talk=section Pages=entry Art=manifest AudioRoom=requested Leave=destinations Speech=false SilentSquare=false ClickSounds=false DialogueSpeech=true CacheScreen=false EagerSquare=false StartsInTown=true
  missionshop.go Shop!=chapter: true
  roompage.go Pages==entry: true
  roomsound.go AudioRoom==requested: true
  secondtown.go Pages==retained: false
  shopbinding.go Shop==bank: true
  townhost.go TrackSource==campaign: true
  townhost.go Talk==building: false
  townhost.go Rows==chapter-hints: false
  townhost.go Header==location: true
  townhost.go Footer==services: true
  townscenehost.go Art==manifest: true
  townscenehost.go Talk==section: true
  towntalk.go townTalkRules[Talk]: true
  talkProfile trim=true heroNPC=21
  StartsInTown=true TownClickSounds=false Header="ROM2 campaign: town 1" Footer=[]
rom2-town3: CurrentHookRoom=false Shop=bank TrackSource=campaign RequireArt=true Rows=hotspots Header=location HeaderPrefix=ROM2 campaign Footer=services Talk=section Pages=entry Art=manifest AudioRoom=requested Leave=destinations Speech=false SilentSquare=false ClickSounds=false DialogueSpeech=true CacheScreen=false EagerSquare=false StartsInTown=true
  missionshop.go Shop!=chapter: true
  roompage.go Pages==entry: true
  roomsound.go AudioRoom==requested: true
  secondtown.go Pages==retained: false
  shopbinding.go Shop==bank: true
  townhost.go TrackSource==campaign: true
  townhost.go Talk==building: false
  townhost.go Rows==chapter-hints: false
  townhost.go Header==location: true
  townhost.go Footer==services: true
  townscenehost.go Art==manifest: true
  townscenehost.go Talk==section: true
  towntalk.go townTalkRules[Talk]: true
  talkProfile trim=true heroNPC=21
  StartsInTown=true TownClickSounds=false Header="ROM2 campaign: town 1" Footer=[]
no-host: CurrentHookRoom=true Shop=chapter TrackSource=description RequireArt=false Rows=chapter-hints Header=chapter HeaderPrefix= Footer=party Talk=building Pages=retained Art=pages AudioRoom=building Leave=room-map Speech=true SilentSquare=true ClickSounds=true DialogueSpeech=true CacheScreen=true EagerSquare=true StartsInTown=false
`

// The host's modes and the branch each call site takes, for every shipped
// description and the defaults, recorded before the modes became typed.
func TestTownHostModesRecord(t *testing.T) {
	got := hostRecordDump(t)
	if got != hostRecordWant {
		t.Fatalf("host modes changed:\n%s", got)
	}
}

const hostRecordTalk = "<part=1 npc=31>\r\nOne\r\n<part=2>\r\nTwo\r\n<part=3>\r\nThree\r\n"

func hostRecordCursor(c *talkCursor) string {
	if c == nil {
		return "nil"
	}
	return fmt.Sprintf("part=%d key=%q bytes=%d", c.part, c.key, len(c.payload))
}

// The talk cursor's position across one page turn and one town re-entry,
// in each game's campaign.
func TestTownTalkCursorRecord(t *testing.T) {
	var b strings.Builder
	f, app, s := visibleOfferApp(t, 30)
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(hostRecordTalk)}})
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&b, "first open: %s open=%v\n", hostRecordCursor(s.campaignTalk()), s.talkOpen())
	s.turnTownTalk()
	fmt.Fprintf(&b, "first turned: %s open=%v\n", hostRecordCursor(s.campaignTalk()), s.talkOpen())
	again := f.TownScreen().(*townScreen)
	fmt.Fprintf(&b, "first re-entered: same=%v %s open=%v\n", again == s, hostRecordCursor(again.campaignTalk()), again.talkOpen())

	sess := &CampaignSession{Town: &Town{second: newSecondCampaign()}}
	mk := func() *townScreen {
		s := &townScreen{shopController: shopController{sess: sess, in: &InstallResources{NPCFaces: map[int32]data.NPCFace{31: {}}}},
			art: fakeTownArt{}, tags: secondEventTags{}}
		s.adapter = destinationTownCampaign{s}
		s.defaults = townDescriptions["rom2-town1"]
		return s
	}
	two := mk()
	two.setTownDialogue("sec", []byte(hostRecordTalk))
	fmt.Fprintf(&b, "second open: %s open=%v\n", hostRecordCursor(two.campaignTalk()), two.talkOpen())
	two.turnTownTalk()
	fmt.Fprintf(&b, "second turned: %s open=%v\n", hostRecordCursor(two.campaignTalk()), two.talkOpen())
	rebuilt := mk()
	fmt.Fprintf(&b, "second re-entered: %s open=%v\n", hostRecordCursor(rebuilt.campaignTalk()), rebuilt.talkOpen())
	g := &FrontEnd{CampaignSession: *sess}
	fmt.Fprintf(&b, "second save: %v\n", secondCampaignRules{}.townSavePoint(g))
	rebuilt.endTalk()
	fmt.Fprintf(&b, "second ended: %s open=%v\n", hostRecordCursor(rebuilt.campaignTalk()), rebuilt.talkOpen())
	if got := b.String(); got != hostRecordTalkWant {
		t.Fatalf("talk cursor changed:\n%s", got)
	}
}

const hostRecordTalkWant = `first open: part=1 key="" bytes=54 open=true
first turned: part=2 key="" bytes=54 open=true
first re-entered: same=true part=2 key="" bytes=54 open=true
second open: part=1 key="sec" bytes=54 open=true
second turned: part=2 key="sec" bytes=54 open=true
second re-entered: part=2 key="sec" bytes=54 open=true
second save: saving is not available for this game yet
second ended: part=0 key="" bytes=0 open=false
`
