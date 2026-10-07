package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func itemScrollFront1115(t *testing.T) *FrontEnd {
	t.Helper()
	f := cellStateFront(t)
	f.SetDeterministicFrames(true)
	p := make([]int32, 19)
	p[1], p[2], p[4], p[6], p[8], p[16], p[17] = 99, 1, 1, 32, 1, 1, 1
	f.Table.Spells = dbCollection{{}, {name: "literal scroll spell", params: p}}
	return f
}

func itemScrollOpen1115(t *testing.T, count uint32, shared bool) (*FrontEnd, *ui.App) {
	t.Helper()
	f := itemScrollFront1115(t)
	doc := sackObjectsLiteral1115(t, f)
	effect := literalSavedEffectRecord(0x810001)
	newGroupSetValue1115(t, &effect, "E3C", 41)
	newGroupSetValue1115(t, &effect, "E40", 1|40<<16)
	doc.Objects = append(doc.Objects, effect)
	effectIndex := uint16(len(doc.Objects))
	refs := []uint16{effectIndex}
	if shared {
		refs = append(refs, effectIndex)
	}
	item := literalItemRecord1115("Item", 0x0e01, count, 0x820001, refs, 0)
	newGroupSetValue1115(t, &item, "F44", 4)
	doc.Objects = append(doc.Objects, item)
	itemIndex := uint16(len(doc.Objects))
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class == "Unit" && actorProjectionValue(t, *r, "Identity") == newGroupA {
			literalSavedObjectRefs(t, r, "Inventory", []uint16{itemIndex}, true)
		}
	}
	if shared {
		doc.Objects = append(doc.Objects, literalItemRecord1115("Item", 0x0e02, 1, 0x820002, []uint16{effectIndex}, 0))
		literalSavedObjectRefs(t, &doc.Objects[doc.World.Sacks[0]-1], "Contents", []uint16{uint16(len(doc.Objects))}, true)
	}
	var err error
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("literal source scroll import", town, err)
	}
	app := f.App("source scroll current owner")
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return f, app
}

func itemScrollBinding1115(t *testing.T, rows []SnapshotSAVObjectBinding, id sim.SavedObjectID) SnapshotSAVObjectBinding {
	t.Helper()
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("missing scroll binding %d", id)
	return SnapshotSAVObjectBinding{}
}

func currentItemMenuCheckpoint(t *testing.T, f *FrontEnd, app *ui.App, fronts ...func(*testing.T) *FrontEnd) (*FrontEnd, *ui.App) {
	t.Helper()
	before := snapshotCurrentObjects(t, f)
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatal("ordinary scroll menu SAVE", entries, err, app.Screen(), app.HeadlessMessage())
	}
	name := entries[0].Name
	if !reflect.DeepEqual(before, snapshotCurrentObjects(t, f)) {
		t.Fatal("ordinary scroll SAVE mutated its live owner")
	}
	front := itemScrollFront1115
	if len(fronts) != 0 {
		front = fronts[0]
	}
	fresh := front(t)
	freshApp := fresh.App("fresh reserved scroll native LOAD")
	save, list, load = fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(save, list, load)
	groundAppLoad(t, freshApp, list, localOriginalSaveToken(name))
	got := snapshotCurrentObjects(t, fresh)
	itemMutationSame1115(t, before, got)
	if f.live.world.Hash() != fresh.live.world.Hash() {
		t.Fatal("ordinary SAV changed the exact current World")
	}
	return fresh, freshApp
}

func TestItemObjects1115OversizedCountKeepsNativeSaveAndExplicitGraphCoverage(t *testing.T) {
	f, app := openCurrentItemFixtureApp(t, false, func(doc *sav.DocumentData) {
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if !savedItemClass(r.Class) {
				continue
			}
			newGroupSetValue1115(t, r, "F4A", 0)
			switch actorProjectionValue(t, *r, "Identity") {
			case 0x420001:
				newGroupSetValue1115(t, r, "F42", 65535)
			case 0x410001:
				newGroupSetValue1115(t, r, "F42", 1)
			case 0x410002:
				newGroupSetValue1115(t, r, "F40", 0x0e07)
				newGroupSetValue1115(t, r, "F42", 1)
			}
		}
	})
	w := f.live.world
	original := itemObjectByKey(t, w.SavedObjects(), 0x420001)
	if original.Value.Count != 65535 {
		t.Fatal("source word boundary was not retained")
	}
	if err := w.TakeSack(original.Owner.Entity, 15, 16); err != nil {
		t.Fatal(err)
	}
	now := itemObjectByKey(t, w.SavedObjects(), 0x420001)
	if now.ID != original.ID || now.Value.Count != 65536 {
		t.Fatal("actual pickup did not retain native uint32 merge count", now)
	}
	state := snapshotCurrentObjects(t, f).SavedDocument
	binding := itemScrollBinding1115(t, state.Objects.Items, original.ID)
	if binding.ObjectIndex == 0 || binding.Unavailable != "" || itemObjectValue1115(t, state.Document.Objects[binding.ObjectIndex-1], "F42") != 65535 {
		t.Fatal("wide current Item lacks its single ordinary word record", binding)
	}
	doc, actions := currentRootSAVDocument(t, f)
	found := false
	for _, row := range actions.Ownership {
		if row.ID == original.ID {
			found = row.Object != 0 && row.Item == nil && row.CountLift != nil && *row.CountLift == (currentItemCountLift{Wire: 65535, Lift: 1}) && itemObjectValue1115(t, doc.Objects[row.Object-1], "F42") == 65535
		}
	}
	if !found {
		t.Fatal("wide current Item lost its exact ordinary-bound count lift")
	}
	for _, a := range state.Actors {
		if a.EntityID == original.Owner.Entity {
			refs, _ := savedObjectRefs(&state.Document.Objects[a.ObjectIndex-1], "Inventory")
			if len(refs) != 3 || !slices.Contains(refs, binding.ObjectIndex) {
				t.Fatal("wide current Item lost its inventory edge", refs)
			}
		}
	}
	fresh, freshApp := currentItemMenuCheckpoint(t, f, app, cellStateFront)
	if itemObjectByKey(t, fresh.live.world.SavedObjects(), 0x420001).Value.Count != 65536 {
		t.Fatal("native LOAD narrowed the full count")
	}
	for range 20 {
		f.live.tick()
		fresh.live.tick()
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("oversized item changed native continuation")
		}
	}
	currentItemMenuCheckpoint(t, fresh, freshApp, cellStateFront)
}

func TestItemObjects1115ScrollSessionNativeContinuation(t *testing.T) {
	for _, count := range []uint32{1, 2} {
		for _, shared := range []bool{false, true} {
			for _, cancel := range []bool{false, true} {
				t.Run(fmt.Sprintf("count=%d/shared=%t/cancel=%t", count, shared, cancel), func(t *testing.T) {
					f, app := itemScrollOpen1115(t, count, shared)
					initial := snapshotCurrentObjects(t, f)
					initialBytes, err := json.Marshal(initial)
					if err != nil {
						t.Fatal(err)
					}
					before := f.live.world.SavedObjects()
					original := itemObjectByKey(t, before, 0x820001)
					actor := original.Owner.Entity
					for tick := 0; tick < 20 && f.live.world.ActorMotionActive(actor); tick++ {
						f.live.tick()
					}
					f.live.pending = append(f.live.pending, sim.Command{Kind: sim.KindUseScroll, Entity: actor, X: int32(actor)})
					f.live.tick()
					casts := f.live.world.ScrollCasts()
					if len(casts) != 1 || casts[0].Item.ObjectID == 0 || casts[0].Caster != actor {
						t.Fatal("actual scroll command did not reserve a source item", casts)
					}
					reserved := casts[0].Item.ObjectID
					current := f.live.world.SavedObjects()
					var row sim.SavedItemObject
					for _, item := range current.Items {
						if item.ID == reserved {
							row = item
						}
					}
					if !current.HasRoot(reserved, sim.SavedObjectOwner{Kind: sim.SavedOwnerSession, SessionHandle: casts[0].Reservation}) || !sim.StackStateEqual(row.Value, sim.StackItem(casts[0].Item, 1)) || count == 1 && reserved != original.ID || count == 2 && reserved == original.ID {
						t.Fatal("reserved exact identity/value/count boundary differs", row)
					}
					mid := snapshotCurrentObjects(t, f)
					binding := itemScrollBinding1115(t, mid.SavedDocument.Objects.Items, reserved)
					if binding.ObjectIndex != 0 || binding.Unavailable != "" {
						t.Fatal("active scroll falsely retained an inventory archive edge", binding)
					}
					for _, id := range row.Effects {
						b := itemScrollBinding1115(t, mid.SavedDocument.Objects.Effects, id)
						wantDetached := count == 2 || !shared
						if wantDetached != (b.ObjectIndex == 0) || b.Unavailable != "" {
							t.Fatal("shared/exclusive scroll child graph coverage differs", b)
						}
					}
					if err := current.ValidateExternal([]sim.SavedExternalItem{{Handle: casts[0].Reservation, ID: reserved, Value: sim.StackItem(casts[0].Item, 1)}}); err != nil {
						t.Fatal(err)
					}
					fresh, freshApp := currentItemMenuCheckpoint(t, f, app)
					if !reflect.DeepEqual(current, fresh.live.world.SavedObjects()) || !reflect.DeepEqual(casts, fresh.live.world.ScrollCasts()) {
						t.Fatal("native save substituted metadata for the live reservation")
					}
					if cancel {
						command := sim.Command{Kind: sim.KindGroupStance, Entity: actor, X: sim.OrderStandGround}
						f.live.pending, fresh.live.pending = append(f.live.pending, command), append(fresh.live.pending, command)
					}
					for tick := 0; tick < 50; tick++ {
						f.live.tick()
						fresh.live.tick()
						if f.live.world.Hash() != fresh.live.world.Hash() {
							t.Fatal("scroll reservation continuation differs", tick)
						}
						if len(f.live.world.ScrollCasts()) == 0 {
							break
						}
					}
					if count == 1 && shared && !cancel {
						graph := f.live.world.SavedObjects()
						disposed, _ := graph.Item(reserved)
						if !disposed.Retired {
							t.Fatal("completed scroll retained its consumed Item")
						}
						for _, effect := range graph.Effects {
							if slices.Contains(disposed.Effects, effect.ID) && effect.Retired {
								t.Fatal("scroll consumption retired a child still referenced by another Item")
							}
						}
					}
					if len(f.live.world.ScrollCasts()) != 0 {
						t.Fatal("scroll never completed or refunded", f.live.world.ScrollCasts())
					}
					stock, _ := f.live.world.CarriedStacks(actor)
					quantity := uint32(0)
					for _, item := range stock {
						quantity += item.Count
					}
					wantCount := count - 1
					if cancel {
						wantCount = count
					}
					if quantity != wantCount {
						t.Fatal("scroll did not consume/refund exactly one unit", quantity, wantCount)
					}
					after := snapshotCurrentObjects(t, fresh)
					b := itemScrollBinding1115(t, after.SavedDocument.Objects.Items, reserved)
					if cancel && count == 1 {
						if b.ObjectIndex == 0 || b.Unavailable != "" || len(stock) != 1 || stock[0].ObjectID != original.ID {
							t.Fatal("refund did not reproject the same source identity", b, stock)
						}
					} else if b.ObjectIndex != 0 || b.Unavailable != "" {
						t.Fatal("disposed or merged scroll kept live session coverage", b)
					}
					currentItemMenuCheckpoint(t, fresh, freshApp)
					unchanged, err := json.Marshal(initial)
					if err != nil || !bytes.Equal(initialBytes, unchanged) || itemScrollBinding1115(t, initial.SavedDocument.Objects.Items, original.ID).ObjectIndex == 0 {
						t.Fatal("later projection mutated an earlier snapshot")
					}
				})
			}
		}
	}
}
