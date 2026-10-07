//go:build sessioncorpusaudit

package game

import (
	"fmt"
	"testing"
)

func TestMilestone2SpellEffects(t *testing.T) {
	files, roots, objects, fields, typed, refs, aliases, empty, mismatches, nonemptyResumed := 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
	classes := map[string]int{"SpellEffect": 0, "PointEffect": 0, "AreaEffect": 0, "SpellTransport": 0, "Effect": 0, "Effect_DirectDamage": 0}
	var refused []milestone2ResumeRefusal
	var nonempty []string
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		want, err := spell1152Expected(mf.f)
		if err != nil {
			mismatches++
			t.Errorf("independent SpellEffect read: %v", err)
			return
		}
		// Count the source before attempting LOAD. Refused files therefore
		// stay in the authored population, while the compared count is explicit.
		roots += len(want.roots)
		objects += len(want.nodes)
		if len(want.roots) == 0 {
			empty++
		}
		f, v, r, a, c := want.population()
		if len(want.roots) > 0 {
			nonempty = append(nonempty, fmt.Sprintf("%s: %d roots, %d objects, classes %v", mf.rel, len(want.roots), len(want.nodes), c))
		}
		fields, typed, refs, aliases = fields+f, typed+v, refs+r, aliases+a
		for class, count := range c {
			classes[class] += count
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, fe.Difficulty, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		files++
		if len(want.roots) > 0 {
			nonemptyResumed++
		}
		for _, difference := range want.typedDifferences(ms.World.SavedSpellEffects()) {
			mismatches++
			t.Error("typed: " + difference)
		}
		for _, difference := range want.documentDifferences(ms.savedDocument) {
			mismatches++
			t.Error("retained Document: " + difference)
		}
	})
	milestone2LogRefusals(t, "spell effects", refused)
	for _, source := range nonempty {
		t.Log("spell effects: nonempty source " + source)
	}
	t.Logf("spell effects: %d world files resumed (%d nonempty); raw population %d roots, %d distinct objects, %d reference slots, %d alias edges, %d empty lists; %d mismatches, %d refused", files, nonemptyResumed, roots, objects, refs, aliases, empty, mismatches, len(refused))
	t.Logf("spell effects: raw fields %d bytes = %d typed member bytes + %d Token bytes retained only in Document; class population %v; counts describe discovered sources without assuming original-runtime provenance", fields, typed, fields-typed, classes)
	if nonemptyResumed == 0 {
		t.Fatal("no nonempty SpellEffect population compared")
	}
}
