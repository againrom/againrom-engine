package sim

import (
	"fmt"
	"reflect"
	"testing"
)

// A refused replacing attachment on a source actor publishes nothing of its
// removal or apply.
func TestSourceEffectRefusedReplacementLeavesWorldUnchanged(t *testing.T) {
	for _, refuse := range []int{2, 3} {
		t.Run(fmt.Sprintf("refuse derive %d", refuse), func(t *testing.T) {
			w := originalAttachmentWorld(t)
			rule := SpellRule{ID: 13}
			if !w.attachEffect(w.entities[0].ID, ^EntityID(0), rule, EffectSpeed, 3, 40, EffectDuration) {
				t.Fatal("first attachment refused")
			}
			if len(w.attached) != 1 {
				t.Fatalf("attached %d effects, want 1", len(w.attached))
			}
			calls := 0
			w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
				calls++
				if calls == refuse {
					return s, fmt.Errorf("refused derive %d", calls)
				}
				return s, nil
			})
			hash, actor := w.Hash(), w.entities[0]
			attached := append([]attachedEffect(nil), w.attached...)
			objects := w.SavedObjects()
			if w.attachEffect(w.entities[0].ID, ^EntityID(0), rule, EffectSpeed, 5, 40, EffectDuration) {
				t.Fatal("replacement with a refused derive was accepted")
			}
			if calls < refuse {
				t.Fatalf("derive reached %d times, the refusal is at %d", calls, refuse)
			}
			if w.entities[0] != actor {
				t.Fatal("refused replacement changed the actor")
			}
			if !reflect.DeepEqual(w.attached, attached) {
				t.Fatal("refused replacement changed the attached effects")
			}
			if !reflect.DeepEqual(w.SavedObjects(), objects) {
				t.Fatal("refused replacement changed the saved objects")
			}
			if w.Hash() != hash {
				t.Fatal("refused replacement changed the World hash")
			}
		})
	}
}
