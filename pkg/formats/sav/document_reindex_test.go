package sav

import (
	"bytes"
	"reflect"
	"testing"
)

// Reverse only the DTO namespace. Literal scalar keys, bytes, inline records
// and list order keep their meanings; no production reindex helper is used.
func reverseDocumentIndices(t *testing.T, source DocumentData) DocumentData {
	t.Helper()
	_, out := documentDataGobCopy(t, source)
	count := len(out.Objects)
	refs := func(ids []uint16) {
		for i, id := range ids {
			if id != 0 {
				ids[i] = uint16(count + 1 - int(id))
			}
		}
	}
	var record func(*DocumentRecordData)
	record = func(r *DocumentRecordData) {
		for i := range r.RefSlots {
			refs(r.RefSlots[i].Objects)
		}
		for i := range r.Inline {
			record(&r.Inline[i].Record)
		}
		for i := range r.Groups {
			record(&r.Groups[i])
		}
	}
	refs(out.Players)
	refs(out.DeadActors)
	if out.World != nil {
		refs(out.World.Buildings)
		refs(out.World.Effects)
		refs(out.World.Sacks)
	}
	for i := range out.Objects {
		record(&out.Objects[i])
	}
	for i := 0; i < count/2; i++ {
		out.Objects[i], out.Objects[count-1-i] = out.Objects[count-1-i], out.Objects[i]
	}
	return out
}

func TestDocumentReindex1115ExplicitOwnedPermutation(t *testing.T) {
	for _, fixture := range []string{"complete", "self-cycle", "two-object-cycle"} {
		t.Run(fixture, func(t *testing.T) {
			want := documentDataFixture(t)
			if fixture != "complete" {
				var err error
				want, err = DecodeDocumentData(documentCycleLiteral(t, fixture == "self-cycle"))
				if err != nil {
					t.Fatal(err)
				}
			}
			edited := reverseDocumentIndices(t, want)
			before, _ := documentDataGobCopy(t, edited)
			if len(edited.Objects) > 1 {
				if _, err := EncodeDocumentData(edited); err == nil {
					t.Fatal("ordinary encoder repaired unordered input")
				}
				if _, err := CloneDocumentData(edited); err == nil {
					t.Fatal("native adoption repaired unordered input")
				}
			}
			got, permutation, err := ReindexDocumentData(edited)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("explicit reindex changed scalar state, aliases or cycles: %v", err)
			}
			if len(permutation) != len(want.Objects)+1 || permutation[0] != 0 {
				t.Fatal("bad permutation extent or null mapping", permutation)
			}
			for old := 1; old < len(permutation); old++ {
				if permutation[old] != uint16(len(want.Objects)+1-old) {
					t.Fatal("not the independently expected inverse permutation", permutation)
				}
			}
			encoded, err := EncodeDocumentData(got)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeDocumentData(encoded)
			if err != nil || !reflect.DeepEqual(again, want) {
				t.Fatal("reindexed whole container differs", err)
			}
			if len(got.Label) != 0 {
				got.Label[0] ^= 1
			}
			got.World.Session.Diplomacy[0][0] ^= 1
			var mutate func(*DocumentRecordData)
			mutate = func(r *DocumentRecordData) {
				for i := range r.Values {
					r.Values[i].Value ^= 1
				}
				for i := range r.Raw {
					clear(r.Raw[i].Bytes)
				}
				for i := range r.RefSlots {
					clear(r.RefSlots[i].Objects)
				}
				for i := range r.Inline {
					mutate(&r.Inline[i].Record)
				}
				for i := range r.Groups {
					mutate(&r.Groups[i])
				}
			}
			for i := range got.Objects {
				mutate(&got.Objects[i])
			}
			after, _ := documentDataGobCopy(t, edited)
			if !bytes.Equal(before, after) {
				t.Fatal("reindex output aliases or mutates its input")
			}
		})
	}
}

func TestDocumentReindex1115ChangedInlineMembership(t *testing.T) {
	d := documentDataFixture(t)
	_, duplicate := documentDataGobCopy(t, d)
	actor := *documentRecordByClass(t, &duplicate, "Human")
	actor.Texts[0].Value = "second actor"
	d.Objects = append(d.Objects, actor)
	newActor := uint16(len(d.Objects))
	p := documentRecordByClass(t, &d, "Player")
	oldActor := (*documentRefsForField(t, &p.Groups[0], "Actors"))[0]
	*documentRefsForField(t, &p.Groups[0], "Actors") = []uint16{newActor, 0, oldActor, newActor}
	*documentCountForField(t, &p.Groups[0], "Actors") = 4
	*documentCountForField(t, p, "Actors") = 4
	requireCurrentActionReindex(t, d, oldActor, newActor)
	// This is a format-only graph test. Two records intentionally share a wire
	// identity; it must never be used to join their distinct DTO identities.
	before, _ := documentDataGobCopy(t, d)
	got, permutation, err := ReindexDocumentData(d)
	if err != nil {
		t.Fatal(err)
	}
	if permutation[1] != 1 || permutation[newActor] != 2 || permutation[oldActor] != 3 {
		t.Fatal("membership did not control first encounter", permutation)
	}
	for old := int(oldActor) + 1; old < int(newActor); old++ {
		if permutation[old] != uint16(old+1) {
			t.Fatal("later object binding did not move", old, permutation)
		}
	}
	p = documentRecordByClass(t, &got, "Player")
	if refs := *documentRefsForField(t, &p.Groups[0], "Actors"); !reflect.DeepEqual(refs, []uint16{2, 0, 3, 2}) {
		t.Fatal("membership order/null/alias changed", refs)
	}
	if !reflect.DeepEqual(got.DeadActors, []uint16{3, 3}) || got.Objects[1].Texts[0].Value != "second actor" || got.Objects[2].Texts[0].Value == "second actor" {
		t.Fatal("cross-root aliases or distinct actors conflated")
	}
	if _, err := EncodeDocumentData(got); err != nil {
		t.Fatal(err)
	}
	after, _ := documentDataGobCopy(t, d)
	if !bytes.Equal(before, after) {
		t.Fatal("membership input was mutated")
	}
}
