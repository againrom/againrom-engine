//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2Players(t *testing.T) {
	files, slots, identities, diaries, elements, entries, tailBytes, mismatches := 0, 0, 0, 0, 0, 0, 0, 0
	var live players1154LivePopulation
	var refused, unreadable []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		want, err := players1154Expected(mf.f)
		if err != nil {
			unreadable = append(unreadable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		slots += len(want.roots)
		identities += len(want.players)
		tailBytes += 32 * len(want.players)
		for _, d := range want.diaries {
			if d.location.Off >= 0 {
				diaries++
				elements += len(d.dwords)/4 + len(d.words)/2
				entries += len(d.entries())
			}
		}
		join, err := players1154Origins(mf.raw)
		if err != nil {
			unreadable = append(unreadable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, fe.Difficulty, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		d := players1154DocumentDifferences(want, join, ms.savedDocument.Document)
		l, p := players1154LiveDifferences(want, join, ms.savedDocument, ms.World)
		for _, difference := range append(d, l...) {
			mismatches++
			t.Error(difference)
		}
		live.players += p.players
		live.purses += p.purses
		live.unavailable += p.unavailable
		live.playerDiaries += p.playerDiaries
		live.actorDiaries += p.actorDiaries
		live.entries += p.entries
		live.manaOwners += p.manaOwners
		for _, reason := range p.excluded {
			live.excluded = append(live.excluded, mf.rel+": "+reason)
		}
	})
	for _, r := range unreadable {
		t.Logf("players: unreadable raw Player/Diary document %s: %v", r.rel, r.err)
	}
	milestone2LogRefusals(t, "players", refused)
	for _, reason := range live.excluded {
		t.Logf("players: supported-subset exclusion %s", reason)
	}
	t.Logf("players: raw world population %d root slots, %d distinct Player archive identities, %d tail bytes, %d owned Diary subjects, %d array elements, %d nondefault entry pairs", slots, identities, tailBytes, diaries, elements, entries)
	t.Logf("players: %d world files resumed; retained Player/Diary and live comparisons %d mismatches; %d raw-unreadable, %d import refusals", files, mismatches, len(unreadable), len(refused))
	t.Logf("players: live subset %d exact Player identities/Slots, %d purses, %d explicitly unavailable purses; %d first-nonnull-root Player Diaries, %d living PartyWalk Human Diaries, %d nondefault entries; PRaw32 and other Diaries remain complete-Document retention", live.players, live.purses, live.unavailable, live.playerDiaries, live.actorDiaries, live.entries)
	t.Logf("players: F58 live actor-basis subset %d unambiguous Player owners, %d named exclusions", live.manaOwners, len(live.excluded))
	if files == 0 || identities == 0 || diaries == 0 {
		t.Fatal("no Player/Diary population compared")
	}
}
