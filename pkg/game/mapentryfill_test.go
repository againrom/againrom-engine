package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// Both fills enter through the one entry body; each answers with its own
// admission and its own advance.
func TestMapEntryReadsItsFillWhicheverFilledIt(t *testing.T) {
	address, ok := MissionMap(10)
	if !ok {
		t.Fatal("no map for mission 10")
	}
	for _, tc := range []struct {
		name string
		fill func(admitted, advanced *int) mapEntry
	}{
		{"campaign", func(admitted, advanced *int) mapEntry { return wrapFill(campaignEntry(), admitted, advanced) }},
		{"session", func(admitted, advanced *int) mapEntry {
			s := &mapSession{entry: MapEntry{Source: address}, end: func(*Mission) string { return "" }}
			f := sessionEntry(s)
			f.loose = false
			return wrapFill(f, admitted, advanced)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEntryFixture(t, &recordingAudio{})
			var admitted, advanced int
			r := e.request()
			r.fill = tc.fill(&admitted, &advanced)
			p, err := enterMission(r, e.ports)
			if err != nil {
				t.Fatalf("enterMission: %v", err)
			}
			if admitted != 1 || advanced != 1 || p.advance == nil {
				t.Fatalf("admitted %d advanced %d, want one each", admitted, advanced)
			}
		})
	}
}

// wrapFill counts the calls the entry body makes into f.
func wrapFill(f mapEntry, admitted, advanced *int) mapEntry {
	admit, advance := f.admit, f.advance
	f.admit = func(rules campaignService, r missionRequest) (string, mapload.Difficulty, error) {
		*admitted++
		return admit(rules, r)
	}
	f.advance = func(p missionPorts, n int, ms *Mission, d ui.MapAdvance) ui.MapAdvance {
		*advanced++
		return advance(p, n, ms, d)
	}
	return f
}

// A session's leave commits through the session's own end and carries nothing
// to the campaign.
func TestSessionEntryCommitsThroughItsOwnEnd(t *testing.T) {
	ended := 0
	s := &mapSession{end: func(*Mission) string { ended++; return "written" }}
	driver := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
		return ui.NoticeToMapList, "", nil
	}
	ports := missionPorts{session: &CampaignSession{}}
	advance := sessionEntry(s).advance(ports, 0, &Mission{}, driver)
	dest, msg, _ := advance(ui.NoticeLeave)
	if dest != ui.NoticeToCharacters || msg != "written" || ended != 1 {
		t.Fatalf("dest %v msg %q ended %d", dest, msg, ended)
	}
	if ports.session.live != nil {
		t.Fatal("the session entry advanced to the campaign successor")
	}
}
