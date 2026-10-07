package sav

import (
	"bytes"
	"reflect"
	"testing"
)

func documentCopyOracle(data DocumentData, reindex bool) (DocumentData, []uint16, error) {
	var permutation []uint16
	if reindex {
		permutation = make([]uint16, len(data.Objects)+1)
	}
	d, err := saveDocumentFromDataIndexed(data, permutation)
	if err != nil {
		return DocumentData{}, nil, err
	}
	out, err := saveDocumentToData(d)
	if err != nil {
		return DocumentData{}, nil, err
	}
	if reindex {
		if err := remapNativeActionObjects(&out.State, permutation); err != nil {
			return DocumentData{}, nil, err
		}
	}
	return out, permutation, nil
}

func documentCopyEmptySlices(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			documentCopyEmptySlices(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			documentCopyEmptySlices(v.Field(i))
		}
	case reflect.Array, reflect.Slice:
		if v.Kind() == reflect.Slice && v.IsNil() {
			v.Set(reflect.MakeSlice(v.Type(), 0, 0))
		}
		for i := 0; i < v.Len(); i++ {
			documentCopyEmptySlices(v.Index(i))
		}
	}
}

func poisonDocumentCopy(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			poisonDocumentCopy(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			poisonDocumentCopy(v.Field(i))
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			poisonDocumentCopy(v.Index(i))
		}
	case reflect.String:
		v.SetString("changed")
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		v.SetUint(v.Uint() ^ 7)
	case reflect.Int32:
		v.SetInt(v.Int() ^ 7)
	}
}

func TestDocumentCopyMatchesRecordGraphOracle(t *testing.T) {
	for _, fixture := range []string{"complete", "allocated-empty", "no-objects", "self-cycle", "two-object-cycle", "reversed", "changed-groups", "native-actions", "raw-aliases"} {
		t.Run(fixture, func(t *testing.T) {
			data := documentDataFixture(t)
			switch fixture {
			case "raw-aliases":
				aliasDocumentCopyRaw(&data)
			case "allocated-empty":
				data.Campaign.Children, data.Campaign.Markers = nil, nil
				documentCopyEmptySlices(reflect.ValueOf(&data).Elem())
			case "no-objects":
				data.Objects, data.Players, data.DeadActors = nil, nil, nil
				data.World.Buildings, data.World.Effects, data.World.Sacks = nil, nil, nil
				documentCopyEmptySlices(reflect.ValueOf(&data).Elem())
			case "self-cycle", "two-object-cycle":
				var err error
				data, err = DecodeDocumentData(documentCycleLiteral(t, fixture == "self-cycle"))
				if err != nil {
					t.Fatal(err)
				}
			case "reversed":
				data = reverseDocumentIndices(t, data)
			case "changed-groups":
				_, detached := documentDataGobCopy(t, data)
				actor := *documentRecordByClass(t, &detached, "Human")
				actor.Texts[0].Value = "additional actor"
				data.Objects = append(data.Objects, actor)
				player := documentRecordByClass(t, &data, "Player")
				old := (*documentRefsForField(t, &player.Groups[0], "Actors"))[0]
				*documentRefsForField(t, &player.Groups[0], "Actors") = []uint16{uint16(len(data.Objects)), 0, old, uint16(len(data.Objects))}
				*documentCountForField(t, &player.Groups[0], "Actors") = 4
				*documentCountForField(t, player, "Actors") = 4
			case "native-actions":
				data = reverseDocumentIndices(t, data)
				payload := []byte(`{ "Version":1,"Bindings":[{"ID":41,"Object":2,"Missing":false},{"ID":99,"Object":0,"Missing":true}],"Objects":[{"ID":3,"Object":3}],"Groups":[{"Object":1,"Inline":2}],"PlayerSlots":[{"Object":1}],"AbsentPlayers":[{"Object":1}],"SpellCasters":[{"Object":3,"Caster":2}],"Ownership":[{"Object":0}],"NativeAreas":[{"Object":3}],"NativeDeliveries":[{"Object":3}],"AbsentDiaries":[{"Object":3}],"ArchiveCoordinates":[{"Object":3}],"ActorGroups":[{"Object":2,"Player":1}],"EffectWidths":[{"Object":3,"Actor":2}],"Opaque":{"Object":9,"Value":17}}`)
				if err := SetNativeActions(&data.State, payload); err != nil {
					t.Fatal(err)
				}
			}
			for _, reindex := range []bool{false, true} {
				name := "clone"
				if reindex {
					name = "reindex"
				}
				t.Run(name, func(t *testing.T) {
					before, _ := documentDataGobCopy(t, data)
					want, wantPermutation, wantErr := documentCopyOracle(data, reindex)
					wantRejected := !reindex && (fixture == "reversed" || fixture == "changed-groups" || fixture == "native-actions")
					if (wantErr != nil) != wantRejected {
						t.Fatalf("oracle fixture acceptance differs: %v", wantErr)
					}
					var got DocumentData
					var permutation []uint16
					var err error
					if reindex {
						got, permutation, err = ReindexDocumentData(data)
					} else {
						got, err = CloneDocumentData(data)
					}
					if (err == nil) != (wantErr == nil) || err != nil && err.Error() != wantErr.Error() {
						t.Fatalf("acceptance differs: got %v, want %v", err, wantErr)
					}
					if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(permutation, wantPermutation) {
						t.Fatal("whole DTO or local-index permutation differs from record graph oracle")
					}
					if err == nil {
						gotBytes, err := EncodeDocumentData(got)
						if err != nil {
							t.Fatal(err)
						}
						wantBytes, err := EncodeDocumentData(want)
						if err != nil || !bytes.Equal(gotBytes, wantBytes) {
							t.Fatal("whole SAV bytes differ", err)
						}
						poisonDocumentCopy(reflect.ValueOf(&got).Elem())
					}
					after, _ := documentDataGobCopy(t, data)
					if !bytes.Equal(before, after) {
						t.Fatal("copy, remap or output mutation changed the input")
					}
				})
			}
		})
	}
}

func TestDocumentCopyRejectsNativeActionMutation(t *testing.T) {
	data := documentDataFixture(t)
	if err := SetNativeActions(&data.State, []byte(`{"Version":1,"Bindings":[{"ID":1,"Object":65535,"Missing":false}]}`)); err != nil {
		t.Fatal(err)
	}
	before, _ := documentDataGobCopy(t, data)
	if out, permutation, err := ReindexDocumentData(data); err == nil || !reflect.DeepEqual(out, DocumentData{}) || permutation != nil {
		t.Fatal("identity permutation hid an invalid continuation object", err)
	}
	after, _ := documentDataGobCopy(t, data)
	if !bytes.Equal(before, after) {
		t.Fatal("failed continuation validation changed the source")
	}
}
