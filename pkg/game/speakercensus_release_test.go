package game

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// oldMissionCast is the mission cast as it was built before the one
// constructor: the world's candidates and the player direction of the first
// marked member.
func oldMissionCast(party []mapload.PartyMember, ids []sim.EntityID) speakerCast {
	c := speakerCast{actors: missionSpeakers(nil, nil, nil, party, ids, nil),
		alive: func(sim.EntityID) bool { return true }}
	for _, m := range party {
		if primaryPlayerHero(m) {
			c.playerDir, _ = memberFigure(m)
			c.hasPlayer = true
			break
		}
	}
	return c
}

// oldTavernPick is the tavern talk's pick before the one constructor: the
// first live party member or stock unit the record admits, with no
// companion first.
func oldTavernPick(party []mapload.PartyMember, stock []tavernStockActor, rec data.NPCFace) int {
	c := speakerCast{alive: func(sim.EntityID) bool { return true }}
	for i, p := range party {
		dir, face := memberFigure(p)
		c.actors = append(c.actors, speakerActor{
			id: sim.EntityID(i), me: primaryPlayerHero(p), hero: p.PlayerCharacter && p.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: p.Class,
		})
		if primaryPlayerHero(p) && !c.hasPlayer {
			c.playerDir, c.hasPlayer = dir, true
		}
	}
	for k, a := range stock {
		a.actor.id = sim.EntityID(len(party) + k)
		c.actors = append(c.actors, a.actor)
	}
	if a, ok := c.resolve(rec); ok {
		return int(a.id)
	}
	return -1
}

// oldVoicePick is the town voice's pick before the one constructor: the
// first carried member whose figure the record admits.
func oldVoicePick(party []mapload.PartyMember, rec data.NPCFace) int {
	cast := speakerCast{}
	for _, member := range party {
		if primaryPlayerHero(member) {
			cast.playerDir, _ = memberFigure(member)
			cast.hasPlayer = true
		}
	}
	for i, member := range party {
		dir, face := memberFigure(member)
		actor := speakerActor{me: primaryPlayerHero(member), hero: member.PlayerCharacter && member.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: member.Class}
		if cast.matches(rec, actor) {
			return i
		}
	}
	return -1
}

// speakerCensusParty is the starting hero h with every persistent companion
// the install's table can build for mission m, each by the production
// constructor.
func speakerCensusParty(f *FrontEnd, h speakerSexHero, m int) []mapload.PartyMember {
	party := f.ChargenParty(ui.ChargenResult{Name: "Hero", Choices: []int{h.choiceSex, h.choiceClass, 0}})
	if len(party) != 1 || !party[0].StartingHero {
		return nil
	}
	for _, j := range speakerSexJoins {
		if m < j.mission {
			continue
		}
		if member, ok := mapload.CampaignNPCMember(f.Table, j.npc, j.mission, party); ok {
			party = append(party, member)
		}
	}
	return party
}

var censusTownText = regexp.MustCompile(`^main/text/inn/(?:npc/npc\d+m\d+|mercenary/npc\d+)\.txt$`)

// censusLine is one speaker-bearing part and the pick a path makes for it.
type censusLine struct {
	where string
	npc   int
}

// Every installed mission line, tavern talk line and town voice leaf, for
// each starting hero and each stage of the companions' joining, is resolved by
// the production cast and by the casts as they were built before the one
// constructor. The mission line must not change. A tavern talk or voice pick
// may change only to the companion who joined as the line's speaker, which is
// the order the mission line already uses. The log lists every difference.
func TestReleaseSpeakerCastCensusOneResolver(t *testing.T) {
	f := releaseFront(t)
	faces := f.NPCFaces
	game := f.Archives.Game()
	src := f.Archives.Containers
	var missionLines, missionSame, tavernLines, tavernSame, voiceLines, voiceSame int
	var diffs []string
	fail := func(format string, a ...any) { t.Errorf(format, a...) }

	// Town text parts naming a speaker, by address.
	type townPart struct {
		address string
		part    int
		npc     int
	}
	var townParts []townPart
	for _, e := range src.Entries() {
		low := strings.ToLower(e.Address)
		if !censusTownText.MatchString(low) {
			continue
		}
		payload, err := src.ReadFile(e.Address)
		if err != nil {
			continue
		}
		for n := 1; n < 64; n++ {
			if speaker, named := EventPartSpeaker(payload, n, EventAudience{}); named {
				townParts = append(townParts, townPart{e.Address, n, speaker})
			}
		}
	}
	sort.Slice(townParts, func(i, j int) bool {
		if townParts[i].address != townParts[j].address {
			return townParts[i].address < townParts[j].address
		}
		return townParts[i].part < townParts[j].part
	})
	for _, h := range speakerSexHeroes {
		for m := 1; m <= 200; m++ {
			party := speakerCensusParty(f, h, m)
			if party == nil {
				continue
			}
			ids := make([]sim.EntityID, len(party))
			for i := range ids {
				ids[i] = sim.EntityID(i + 1)
			}
			who := h.name
			stage := m
			old := oldMissionCast(party, ids)
			cast := newSpeakerCast(party, &speakerWorld{actors: missionSpeakers(nil, nil, nil, party, ids, nil),
				alive: func(sim.EntityID) bool { return true }})
			aud := partyAudience(party, faces)
			for e := 1; e <= 99; e++ {
				payload, ok := ReadEventTextFor(src, game, stage, e)
				if !ok {
					continue
				}
				for n := 1; n < 64; n++ {
					if _, shown := dialoguePart(payload, n, aud); !shown {
						continue
					}
					speaker, named := EventPartSpeaker(payload, n, aud)
					if !named {
						continue
					}
					rec, known := faces[int32(speaker)]
					if !known {
						continue
					}
					missionLines++
					a, aok := old.resolveNPC(int32(speaker), rec)
					b, bok := cast.resolveNPC(int32(speaker), rec)
					if aok != bok || a.id != b.id {
						diffs = append(diffs, fmt.Sprintf("mission line %s m%d e%02d part %d npc%d: before %v/%d, after %v/%d", who, stage, e, n, speaker, aok, a.id, bok, b.id))
						continue
					}
					missionSame++
				}
			}

		}
	}
	f.Carried = nil
	stages := []int{1}
	for _, j := range speakerSexJoins {
		stages = append(stages, j.mission)
	}
	for _, h := range speakerSexHeroes {
		for _, stage := range stages {
			party := speakerCensusParty(f, h, stage)
			if party == nil {
				t.Skipf("the install builds no generator party for %s", h.name)
			}
			f.Carried = party
			// The second game's town is its own screen: it has no tavern
			// talk and no hero voice leaf, so only the mission lines apply.
			s, first := f.TownScreen().(*townScreen)
			if !first {
				continue
			}
			stock := s.tavernStockActors()
			who := fmt.Sprintf("%s stage %d", h.name, stage)

			// Tavern talk and voice over the installed town text parts.
			for _, p := range townParts {
				rec, known := faces[int32(p.npc)]
				if !known {
					continue
				}
				tavernLines++
				before := oldTavernPick(party, stock, rec)
				after := -1
				if a, ok := tavernTalkCast(party, stock).resolveNPC(int32(p.npc), rec); ok {
					after = int(a.id)
				}
				if before == after {
					tavernSame++
				} else {
					diffs = append(diffs, fmt.Sprintf("tavern talk %s %s part %d npc%d: before %d, after %d", who, p.address, p.part, p.npc, before, after))
					if after < 0 || after >= len(party) || party[after].CompanionNPC != p.npc {
						fail("tavern talk %s %s part %d npc%d: the new pick %d is not the companion of that speaker", who, p.address, p.part, p.npc, after)
					}
				}
				if !strings.HasPrefix(strings.ToLower(p.address), "main/text/inn/npc/") || rec.Kind != data.NPCNoPicture {
					continue
				}
				voiceLines++
				vb := oldVoicePick(party, rec)
				va := -1
				if a, ok := newSpeakerCast(party, nil).resolveNPC(int32(p.npc), rec); ok {
					va = int(a.id)
				}
				if vb == va {
					voiceSame++
				} else {
					diffs = append(diffs, fmt.Sprintf("voice %s %s part %d npc%d: before %d, after %d", who, p.address, p.part, p.npc, vb, va))
					if va < 0 || va >= len(party) || party[va].CompanionNPC != p.npc {
						fail("voice %s %s part %d npc%d: the new pick %d is not the companion of that speaker", who, p.address, p.part, p.npc, va)
					}
				}
			}
		}
	}
	for _, d := range diffs {
		if strings.HasPrefix(d, "mission line") {
			fail("%s", d)
		}
		t.Log(d)
	}
	t.Logf("mission lines %d, same %d; tavern talk lines %d, same %d; voice leaves %d, same %d; differences %d",
		missionLines, missionSame, tavernLines, tavernSame, voiceLines, voiceSame, len(diffs))
	if missionLines == 0 {
		t.Skip("the install holds no speaker record: no cast is built")
	}
	if len(diffs) == 0 {
		t.Error("no tavern talk or voice pick changed; the loss control for the companion-first order found nothing")
	}
}
