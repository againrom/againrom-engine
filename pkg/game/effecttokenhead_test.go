package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

// actorEffectHeads returns T0C and E0C of every Effect a Unit-family record
// holds in its Effects list.
func actorEffectHeads(t *testing.T, doc sav.DocumentData) [][2]uint32 {
	t.Helper()
	var out [][2]uint32
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" && r.Class != "Humanoid" && r.Class != "Human" {
			continue
		}
		refs, _ := savedObjectRefs(r, "Effects")
		for _, ref := range refs {
			if ref == 0 || int(ref) > len(doc.Objects) || doc.Objects[ref-1].Class != "Effect" {
				continue
			}
			head, errHead := savedStructureValue(&doc.Objects[ref-1], "T0C")
			own, errOwn := savedStructureValue(&doc.Objects[ref-1], "E0C")
			if errHead != nil || errOwn != nil {
				t.Fatal("Effect lacks T0C or E0C", errHead, errOwn)
			}
			out = append(out, [2]uint32{head, own})
		}
	}
	return out
}

// Token::Serialize writes the byte at +0x0c (SAV-TOKEN-034) and
// Effect::Serialize writes the same byte again last (SAV-EFFCHAIN-046), so
// a SAVE writes an actor Effect's T0C equal to its E0C. The cast that attaches
// the Effect sets the value; a cold LOAD and a second SAVE keep it.
func TestActorEffectTokenHeadEqualsItsSpell(t *testing.T) {
	f := currentEffectFront(t, 20)
	_, doc, _ := saveCurrentEffect(t, f)
	heads := actorEffectHeads(t, doc)
	if len(heads) != 1 || heads[0] != [2]uint32{20, 20} {
		t.Fatalf("first SAVE Effect T0C/E0C %v, want one Effect at 20/20", heads)
	}
	raw, _, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	_, again, _ := saveCurrentEffect(t, cold)
	if got := actorEffectHeads(t, again); len(got) != 1 || got[0] != [2]uint32{20, 20} {
		t.Fatalf("SAVE after cold LOAD Effect T0C/E0C %v, want one Effect at 20/20", got)
	}
}
