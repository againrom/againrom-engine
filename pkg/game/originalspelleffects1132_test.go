package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// sameSpellEffectGraphForAudit compares a file's own decoded SpellEffect list
// against the corresponding live state, walking both graphs in parallel and
// requiring that two sites sharing one object in the file share one Go
// pointer in the live state too. It is independent of spellEffectConverter:
// it never calls it, on TestCellRecordCorpusAudit1131's own stated reason for
// reconstructing its own comparison value.
//
// Untagged (unlike the sessioncorpusaudit-gated corpus audit that first
// needed it) so the ordinary release-witness tests in
// originalspelleffects1132_release_test.go can call it too.
func sameSpellEffectGraphForAudit(file []sav.SpellEffect, live []sim.SavedSpellEffect) (string, bool) {
	if len(file) != len(live) {
		return fmt.Sprintf("count %d vs %d", len(file), len(live)), false
	}
	seenEffect := map[*sav.Effect]*sim.SavedEffect{}
	seenSpellEffect := map[*sav.SpellEffect]*sim.SavedSpellEffect{}
	var walkOne func(a sav.SpellEffect, b sim.SavedSpellEffect, path string) (string, bool)
	walkEffect := func(a *sav.Effect, b *sim.SavedEffect, path string) (string, bool) {
		if (a == nil) != (b == nil) {
			return path + ": reference presence differs", false
		}
		if a == nil {
			return "", true
		}
		if want, ok := seenEffect[a]; ok {
			if want != b {
				return path + ": shared Effect identity not preserved", false
			}
			return "", true
		}
		seenEffect[a] = b
		if a.Class != b.Class || a.E3C != b.E3C || a.E3D != b.E3D || a.E40 != b.E40 ||
			a.E0C != b.E0C || a.DirectDamage != b.DirectDamage {
			return fmt.Sprintf("%s: file %+v vs live %+v", path, a, b), false
		}
		return "", true
	}
	walkSpellEffect := func(a *sav.SpellEffect, b *sim.SavedSpellEffect, path string) (string, bool) {
		if (a == nil) != (b == nil) {
			return path + ": reference presence differs", false
		}
		if a == nil {
			return "", true
		}
		if want, ok := seenSpellEffect[a]; ok {
			if want != b {
				return path + ": shared SpellEffect identity not preserved", false
			}
			return "", true
		}
		seenSpellEffect[a] = b
		return walkOne(*a, *b, path)
	}
	walkOne = func(a sav.SpellEffect, b sim.SavedSpellEffect, path string) (string, bool) {
		if a.Class != b.Class || a.SE40 != b.SE40 || a.SE41 != b.SE41 || a.PE44 != b.PE44 ||
			a.AE48 != b.AE48 || a.AE4C != b.AE4C || a.ST4C != b.ST4C {
			return fmt.Sprintf("%s: file %+v vs live %+v", path, a, b), false
		}
		if msg, ok := walkEffect(a.PE48, b.PE48, path+".PE48"); !ok {
			return msg, false
		}
		if msg, ok := walkEffect(a.AE44, b.AE44, path+".AE44"); !ok {
			return msg, false
		}
		if msg, ok := walkSpellEffect(a.ST44, b.ST44, path+".ST44"); !ok {
			return msg, false
		}
		if msg, ok := walkSpellEffect(a.ST48, b.ST48, path+".ST48"); !ok {
			return msg, false
		}
		return "", true
	}
	for i := range file {
		if msg, ok := walkOne(file[i], live[i], fmt.Sprintf("[%d]", i)); !ok {
			return msg, false
		}
	}
	return "", true
}
