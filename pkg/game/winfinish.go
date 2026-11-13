package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

// A won mission is finished by one table of rows, in the shape of the second
// game's departure rows. A row set is a campaign model's answer to a win: a
// guard, its rows, the route a successor takes and the return to town. The
// first row of the set whose guard holds finishes the win: its writes run in
// order, a write that refuses ends the win at -1, and the first exit whose
// gate holds answers the next place. The first game is the reduced second
// game: its row set reads the campaign registry for the successor, the
// payment and the AutoGetMission travel, and a guest chain is one more row.

// winInput is one won mission the finisher completes. seam is that the win
// came through the mission's acknowledged advance, won that the advance's
// one win predicate admits it; the public entries come through no seam.
type winInput struct {
	n      int
	party  []mapload.PartyMember
	w      *sim.World
	ids    []sim.EntityID
	roster map[sim.EntityID]mapload.PartyMember
	seam   bool
	won    bool
}

// winState is what a row's writes leave for its exits: the identity of the
// won mission, and for a guest link the guest who joins and the next link.
type winState struct {
	id     missionIdentity
	member *mapload.PartyMember
	joiner mod.Mission
	next   mod.Mission
}

// winWrite is one write of a row; a non-empty answer is the refusal.
type winWrite func(f *FrontEnd, in winInput, s *winState) (refused string)

// winExit is one exit of a row: the place answers when the gate holds; a nil
// gate always holds.
type winExit struct {
	gate  func(f *FrontEnd, in winInput, s *winState) bool
	place func(f *FrontEnd, in winInput, s *winState) (next int, line string)
}

// winRow is one case of what follows a win; a nil guard always holds.
type winRow struct {
	guard  func(id missionIdentity) bool
	writes []winWrite
	exits  []winExit
}

// winRowSet is the rows of one campaign model; a nil guard always holds.
type winRowSet struct {
	guard func(f *FrontEnd, seam bool) bool
	rows  []winRow
	route func(f *FrontEnd, n, successor int) winRoute
	home  func(f *FrontEnd, n int)
}

// winRowSets is the one declaration of what follows a win, in the order the
// sets are tried. The bank set holds when the campaign service answers that
// the win takes it; every other win, the second game's direct entries
// included, takes the chapter set.
func winRowSets() []winRowSet {
	return []winRowSet{
		{guard: bankWin,
			rows:  []winRow{{writes: []winWrite{bankFinish, carryWin, bankComplete}, exits: []winExit{{place: bankPlace}}}},
			route: bankRoute, home: bankHome},
		{
			rows: []winRow{
				{guard: func(id missionIdentity) bool { return id.guest && id.record.assigned() },
					writes: []winWrite{carryWin, guestJoiner}, exits: []winExit{{place: guestHome}}},
				{guard: func(id missionIdentity) bool { return id.guest },
					writes: []winWrite{carryWin, guestJoiner},
					exits:  []winExit{{gate: guestFollows, place: guestNext}, {place: guestHome}}},
				{exits: []winExit{{place: chapterPlace}}},
			},
			route: chapterRoute, home: chapterHome},
	}
}

// winRows is the row set a win on this front end takes.
func (f *FrontEnd) winRows(seam bool) winRowSet {
	sets := winRowSets()
	for _, set := range sets {
		if set.guard == nil || set.guard(f, seam) {
			return set
		}
	}
	return sets[len(sets)-1]
}

// finishWin is the one finisher of a won mission. It answers a positive
// mission to open, zero beside the sentence that states what follows, or -1
// beside the refusal.
func (f *FrontEnd) finishWin(in winInput) (int, string) {
	set := f.winRows(in.seam)
	id := f.missionIdentity(in.n)
	for _, row := range set.rows {
		if row.guard != nil && !row.guard(id) {
			continue
		}
		s := &winState{id: id}
		for _, write := range row.writes {
			if refused := write(f, in, s); refused != "" {
				return -1, refused
			}
		}
		for _, e := range row.exits {
			if e.gate == nil || e.gate(f, in, s) {
				return e.place(f, in, s)
			}
		}
		break
	}
	return 0, ""
}

// carryWin carries the party and purse home through the carry every won
// mission uses.
func carryWin(f *FrontEnd, in winInput, _ *winState) string {
	return f.CampaignSession.carryMissionHome(f.townInstall(), in.n, in.party, in.w, in.ids, in.roster)
}

// bankWin is the campaign service's answer whether the win takes the bank set.
func bankWin(f *FrontEnd, seam bool) bool { return f.campaign().bankWin(f, seam) }

// bankFinish completes the mission in the scenario bank.
func bankFinish(f *FrontEnd, in winInput, _ *winState) string {
	if err := f.Town.second.finish(in.n, in.w, in.won); err != nil {
		return err.Error()
	}
	return ""
}

// bankComplete stores the bank's case, leaves the location, sets the
// completion movie the departure output selects, and offers the first
// mission the bank makes available.
func bankComplete(f *FrontEnd, in winInput, _ *winState) string {
	town := f.Town
	// The output reads the bank before the case stores (R2-ENGINE-148).
	output := town.second.complete(in.w, in.party, in.ids)
	f.CampaignSession.leftLocation(f.townInstall())
	if live := f.live; live != nil && live.view != nil {
		movie := ""
		if a := f.Archives; a != nil && a.Containers != nil {
			movie = secondGameCompletionDirectory(a.Containers, InstallTextCode(a.Containers, a.Game().Edition()), output)
		}
		live.view.SetCompletionCutscene(movie)
	}
	firstCompletion := !town.won[in.n]
	town.won[in.n] = true
	if firstCompletion {
		f.CampaignSession.completeFame(f.Campaign.Value(), in.n, in.party, in.w, in.ids, in.roster)
	}
	f.Offered = 0
	for _, l := range town.second.available {
		if l.kind == placeMission {
			f.Offered = l.id
			break
		}
	}
	return ""
}

func bankPlace(f *FrontEnd, _ winInput, _ *winState) (int, string) {
	return f.Town.second.enteredWithoutMap(), ""
}

// bankRoute is the chapter route, one terminal test included, with the map in
// the list's place: a win that names no successor travels on.
func bankRoute(f *FrontEnd, n, successor int) winRoute {
	if route := chapterRoute(f, n, successor); route != winList {
		return route
	}
	return winTown
}

func bankHome(f *FrontEnd, _ int) { releaseWorldAudio(f.runtimeAudio(), f.endLive()) }

// guestJoiner records the guest who joins the town as she ended the won
// link, refusing a win whose joining companion did not come home.
func guestJoiner(f *FrontEnd, in winInput, s *winState) string {
	g := s.id.record
	s.member = g.member
	joiner, joins := g.joiner()
	s.joiner = joiner
	if !joins {
		return ""
	}
	is := func(p mapload.PartyMember) bool { return p.Membership().Matches(mapload.CompanionMember, joiner.Guest) }
	if i := slices.IndexFunc(f.Carried, is); i >= 0 {
		c := mapload.CloneParty([]mapload.PartyMember{f.Carried[i]})[0]
		s.member = &c
	} else if slices.ContainsFunc(g.party(), is) {
		return fmt.Sprintf("mission %d won - companion %d did not come home", in.n, joiner.Guest)
	}
	return ""
}

// guestFollows is the chain's AfterWin exit: a link follows the won one for
// the waiting party's starting hero.
func guestFollows(f *FrontEnd, in winInput, s *winState) bool {
	g := s.id.record
	next, ok := tableMissions(f.Table).AfterWin(in.n, mapload.StartingHeroKind(mapload.HeroHold{Party: g.heroParty(f.Carried)}))
	s.next = next
	return ok
}

// guestNext opens the next link with no town between, each member it names
// again as she ended the won link.
func guestNext(f *FrontEnd, in winInput, s *winState) (int, string) {
	g, next := s.id.record, s.next
	town, _ := f.townInstall().campaign.NextMission(s.id.back)
	start, err := f.guestParty(next, town.Mission, g.heroParty(f.Carried), f.Carried)
	if err != nil {
		return -1, fmt.Sprintf("mission %d won - mod %q: %v", in.n, next.Mod, err)
	}
	f.Town.guest = &guestMission{mission: next, stash: g.stash, start: start[0], rest: start[1:],
		member: s.member, chain: tableMissions(f.Table).Chain(next), hero: g.hero}
	return next.Number, fmt.Sprintf("mission %d won - the guest party goes on to mission %d", in.n, next.Number)
}

// guestHome brings the waiting party back through the one town LOAD of the
// stash, which records the guest who joins beside it so the town's grant
// takes her as she is now; nothing else of the chain carries back. Only a
// campaign mission's win is then the campaign's.
func guestHome(f *FrontEnd, in winInput, s *winState) (int, string) {
	g, member := s.id.record, s.member
	campaign := g.mission.Campaign()
	var back *guestMission
	if campaign {
		back = &guestMission{mission: g.mission, start: g.start, member: member}
	}
	if member != nil {
		start := *member
		if s.joiner.Number == g.mission.Number {
			start = g.start
		}
		back = &guestMission{mission: s.joiner, start: start, member: member}
	}
	if _, _, _, err := f.restoreOriginal(g.stash, back); err != nil {
		return -1, fmt.Sprintf("mission %d won - the waiting party did not come back: %v", in.n, err)
	}
	if campaign {
		if member == nil {
			f.Town.guest = f.Town.guest.previous
		}
		in.party, in.w, in.ids, in.roster = f.Carried, nil, nil, nil
		return chapterPlace(f, in, s)
	}
	if member == nil {
		return 0, fmt.Sprintf("mission %d won - the party goes on to the town", in.n)
	}
	return 0, fmt.Sprintf("mission %d won - %s returns; the party goes on to the town", in.n, member.Name)
}

// chapterPlace is the campaign registry's answer to a won chapter mission:
// the party carried home, the win and its payment recorded, the successor,
// the AutoGetMission travel, or the town, where a guest chain may start.
func chapterPlace(f *FrontEnd, in winInput, s *winState) (int, string) {
	w := in.w
	return f.CampaignSession.finishMission(f.townInstall(), in.n, in.party, w, in.ids, in.roster,
		townArrival{carried: s.id.guest, arrive: f.arriveInTown, welcome: f.addChapterCompanions,
			guest: func(won int) (int, string) { return f.startGuestMission(won, w, nil, nil, nil) }})
}

func chapterRoute(f *FrontEnd, n, successor int) winRoute {
	return f.CampaignSession.routeAfterWin(f.Campaign.Value(), n, successor)
}

// chapterHome travels home on the world map from the mission the identity
// returns to.
func chapterHome(f *FrontEnd, n int) {
	f.TownScreen().(*townScreen).beginWorldMapReturn(f.missionIdentity(n).back)
}
