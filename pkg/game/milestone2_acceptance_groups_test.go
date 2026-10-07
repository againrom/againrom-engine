//go:build sessioncorpusaudit

package game

import "testing"

func TestMilestone2Groups(t *testing.T) {
	files, groups, members, normalized, words, paths, actors, patrol, mismatches := 0, 0, 0, 0, 0, 0, 0, 0, 0
	var population groups1155Population
	var refused, unreadable, projectionUnavailable []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		want, err := groups1155Expected(mf.f)
		if err != nil {
			unreadable = append(unreadable, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		groups += len(want.groups)
		actors += len(want.actors)
		for _, g := range want.groups {
			members += len(g.members)
			normalized += len(g.current)
			words += len(g.words)
			paths += len(g.path)
		}
		for _, a := range want.actors {
			patrol += len(a.patrol)
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
		// This is intentionally BEFORE Snapshot: it must still contain the
		// original sequence A,null,B,A, not the normalized current null,B,A.
		d := groups1155DocumentDifferences(want, join, ms.savedDocument.Document)
		l, p := groups1155LiveDifferences(want, join, ms.savedDocument, ms.World)
		for _, difference := range append(d, l...) {
			mismatches++
			t.Error(difference)
		}
		population.bound += p.bound
		population.unbound += p.unbound
		population.nulls += p.nulls
		population.orders += p.orders
		for _, reason := range p.unavailable {
			population.unavailable = append(population.unavailable, mf.rel+": "+reason)
		}
		for _, reason := range p.issues {
			population.issues = append(population.issues, mf.rel+": "+reason)
		}
		projected, err := snapshotSavedDocument(ms)
		if err != nil {
			projectionUnavailable = append(projectionUnavailable, milestone2ResumeRefusal{mf.rel, err})
			t.Error("current Group Document Snapshot refused:", err)
		} else if reason := projected.GroupBindings.Unavailable; reason != "" {
			population.unavailable = append(population.unavailable, mf.rel+": current Document projection unavailable: "+reason)
			t.Error("current Group Document projection unavailable:", reason)
		}
	})
	for _, r := range unreadable {
		t.Logf("groups: raw Group document unreadable %s: %v", r.rel, r.err)
	}
	milestone2LogRefusals(t, "groups", refused)
	for _, r := range projectionUnavailable {
		t.Logf("groups: Snapshot refused %s: %v", r.rel, r.err)
	}
	for _, reason := range population.unavailable {
		t.Logf("groups: binding/projection unavailable %s", reason)
	}
	for _, reason := range population.issues {
		t.Logf("groups: continuation limit %s", reason)
	}
	t.Logf("groups: raw world population %d inline Groups, %d member slots -> %d normalized slots, %d G20 words, %d AI-owned Path words; %d distinct member actors, %d active actor Patrol words", groups, members, normalized, words, paths, actors, patrol)
	t.Logf("groups: %d files resumed, %d raw-Document/live mismatches, %d raw-unreadable, %d import refusals; %d bound members, %d unmaterialized, %d nulls, %d compared live orders", files, mismatches, len(unreadable), len(refused), population.bound, population.unbound, population.nulls, population.orders)
	t.Logf("groups: %d named binding/projection gaps, %d Snapshot refusals, %d named continuation issues; raw AI80/order148 retention and live AI76/order144 exclude replaced pointer identities", len(population.unavailable), len(projectionUnavailable), len(population.issues))
	if files == 0 || groups == 0 || actors == 0 {
		t.Fatal("no raw Group/member population compared")
	}
}
