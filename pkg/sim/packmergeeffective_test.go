package sim

import (
	"fmt"
	"strings"
	"testing"
)

// effectiveMergeWorld is actor 7 holding held and actor 8 holding other, and
// a map Sack at (3,1) holding ground. The world weighs the bow at 3 and
// declares no constructor, as a mission entered from the city does. With
// bind set it carries the bow's constructor as merge operands, which the
// loader binds for a first-game world; with bound set both packs are saved
// objects in one registry, as mission entry binds a party.
func effectiveMergeWorld(t *testing.T, bind, bound bool, held, other []ItemStack, ground ...ItemInstance) *World {
	t.Helper()
	w := mustWorld(t, 25, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20}})
	if len(ground) != 0 {
		if err := w.ReplaceGroundSacks([]Sack{makeSack(3, 1, 0, ground)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.DeclareItemWeights([]ItemWeight{{Code: packMergeBow.Code, Weight: 3}}); err != nil {
		t.Fatal(err)
	}
	if bind {
		if err := w.BindMergeOperands([]ItemWeight{{Code: packMergeBow.Code, Weight: 3, Constructor: packMergeBuilt()}}); err != nil {
			t.Fatal(err)
		}
	}
	w.carried[0], w.carried[1] = held, other
	w.recomputeLoad(0)
	w.recomputeLoad(1)
	if bound {
		bindEffectivePacks(t, w)
	}
	return w
}

// bindEffectivePacks binds both actors' packs as saved objects of one
// registry: actor 7's cells from id 10, actor 8's from id 20.
func bindEffectivePacks(t *testing.T, w *World) {
	t.Helper()
	r := &SavedObjects{Version: 1, NextID: 100}
	var bindings []SavedObjectBinding
	for i := range 2 {
		owner := SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: w.entities[i].ID}
		c := SavedObjectContainer{Owner: owner, Present: true, InsertIndex: uint32(len(w.carried[i])), Accumulator: w.containerWeight(i)}
		for k, value := range w.carried[i] {
			id := SavedObjectID(10 + 10*i + k)
			row := SavedItemObject{ID: id, Origin: SavedObjectOrigin{Kind: SavedObjectOriginal}, Owner: owner, Value: value.Clone(),
				Token: SavedObjectToken{RuntimeID: 0x80000000 + uint32(id), T08: 0x40, T1C: uint32(value.Price), Identity: 0xfedcba98, Reference: 0xfedcba98}}
			row.Value.ObjectID = id
			if value.SourceEquipment.Class != 0 {
				row.Token.T0C = value.SourceEquipment.DefinitionRow
			}
			r.Items = append(r.Items, row)
			c.Items = append(c.Items, id)
			bindings = append(bindings, SavedObjectBinding{ID: id, Owner: owner, Index: uint32(k), Value: row.Value})
		}
		r.Containers = append(r.Containers, c)
	}
	if err := w.ImportSavedObjects(r, nil, bindings...); err != nil {
		t.Fatal(err)
	}
}

// effectiveForms are the three bows of one code: the plain value, the value
// its constructor writes, and a value with Weapon byte 21 set, whose
// operands differ from both.
var effectiveForms = []struct {
	name  string
	value func() ItemInstance
}{
	{"plain", func() ItemInstance { return packMergeBow.Clone() }},
	{"constructed", func() ItemInstance { return packMergeBuiltBow(false) }},
	{"byte-21", packMergeBlockBow},
}

// effectiveRoutes put one unit of incoming into actor 7's pack, which holds
// held, through one production route each, and return actor 7's pack.
var effectiveRoutes = []struct {
	name string
	run  func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World
}{
	{"pickup", func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World {
		w := effectiveMergeWorld(t, bind, bound, []ItemStack{StackItem(held, 1)}, nil, incoming)
		if err := w.TakeSack(7, 3, 1); err != nil {
			t.Fatal(err)
		}
		return w
	}},
	{"drop-and-pickup", func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World {
		w := effectiveMergeWorld(t, bind, bound, []ItemStack{StackItem(held, 1)}, []ItemStack{StackItem(incoming, 1)})
		Step(w, []Command{DropCarried(8, 0, CellPoint{X: 2, Y: 2})})
		if len(w.carried[1]) != 0 || len(w.Sacks()) != 1 {
			t.Fatalf("drop left pack %s and %d sacks", packCells(w.carried[1]), len(w.Sacks()))
		}
		s := w.Sacks()[0]
		if err := w.TakeSack(7, s.X, s.Y); err != nil {
			t.Fatal(err)
		}
		return w
	}},
	{"transfer", func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World {
		w := effectiveMergeWorld(t, bind, bound, []ItemStack{StackItem(held, 1)}, []ItemStack{StackItem(incoming, 1)})
		if err := w.MoveCarried(8, 7, incoming.Code, 1); err != nil {
			t.Fatal(err)
		}
		return w
	}},
	{"unequip", func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World {
		w := effectiveMergeWorld(t, bind, bound, []ItemStack{StackItem(incoming, 1)}, []ItemStack{StackItem(held, 1)})
		Step(w, []Command{Equip(7, 0, 1)})
		if len(w.carried[0]) != 0 {
			t.Fatalf("equip left pack %s", packCells(w.carried[0]))
		}
		if err := w.MoveCarried(8, 7, held.Code, 1); err != nil {
			t.Fatal(err)
		}
		Step(w, []Command{Unequip(7, 1)})
		return w
	}},
	{"grant", func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World {
		w := effectiveMergeWorld(t, bind, bound, []ItemStack{StackItem(held, 1)}, nil)
		if !w.CheatAddItem(7, StackItem(incoming, 1)) {
			t.Fatal("grant refused")
		}
		return w
	}},
}

// effectiveCensus is one line per route, binding, held form and incoming
// form: whether the two units end in one cell of actor 7's pack.
func effectiveCensus(t *testing.T, bind bool) []string {
	var out []string
	for _, route := range effectiveRoutes {
		for _, bound := range []bool{false, true} {
			for _, held := range effectiveForms {
				for _, incoming := range effectiveForms {
					w := route.run(t, bind, bound, held.value(), incoming.value())
					merged := len(w.carried[0]) == 1 && w.carried[0][0].Count == 2
					if bound {
						if err := w.validateSavedObjects(); err != nil {
							t.Fatalf("%s %s<-%s: %v", route.name, held.name, incoming.name, err)
						}
					}
					out = append(out, fmt.Sprintf("%s bound=%v %s<-%s merged=%v", route.name, bound, held.name, incoming.name, merged))
				}
			}
		}
	}
	return out
}

// TestEffectiveMergeCensus: two bows whose effective operands are identical,
// a plain one and one as its constructor writes it, end in one cell on every
// route that puts an item in a pack, bound or unbound, in either order. A
// bow whose Weapon byte 21 differs keeps its own cell from a constructed
// one, and joins a plain one only where the stored-form rule already joined
// it: an unbound incoming plain value (DIV-762). Without merge operands, as a
// second-game world is built, every pair keeps its former outcome.
func TestEffectiveMergeCensus(t *testing.T) {
	got := effectiveCensus(t, true)
	t.Logf("census with merge operands:\n%s", strings.Join(got, "\n"))
	for _, line := range got {
		f := strings.Fields(line)
		pair, merged := f[2], f[3] == "merged=true"
		want := true
		switch pair {
		case "plain<-byte-21", "constructed<-byte-21", "byte-21<-constructed":
			want = false
		case "byte-21<-plain":
			// The stored-form rule lets an unbound plain value take a held
			// Item's blocks; a saved object joins by its operands only. A
			// Sack on the map and a grant hold no saved object.
			want = f[1] == "bound=false" || f[0] == "pickup" || f[0] == "grant"
		}
		if merged != want {
			t.Errorf("%s: want merged=%v", line, want)
		}
	}
}

// checkEffectiveRoute runs route with a plain bow and a constructed bow in
// both orders, unbound and bound, and requires one cell holding both units
// that keeps the destination: its object, form and weight. A bound incoming
// object is retired, the saved pack weighs what the pack weighs, and the
// pack survives a native continuation. A bow with Weapon byte 21 set keeps
// its own cell beside the constructed bow.
func checkEffectiveRoute(t *testing.T, name string, boundIncoming bool) {
	t.Helper()
	var route func(t *testing.T, bind, bound bool, held, incoming ItemInstance) *World
	for _, r := range effectiveRoutes {
		if r.name == name {
			route = r.run
		}
	}
	plain, built := packMergeBow.Clone(), packMergeBuiltBow(false)
	for _, bound := range []bool{false, true} {
		for _, pair := range [][2]ItemInstance{{plain, built}, {built, plain}} {
			held, incoming := pair[0], pair[1]
			label := fmt.Sprintf("bound %v, held weight %v", bound, held.WeightPresent)
			w := route(t, true, bound, held, incoming)
			got := w.carried[0]
			if len(got) != 1 || got[0].Count != 2 || got[0].WeightPresent != held.WeightPresent || got[0].SourceEquipment != held.SourceEquipment {
				t.Fatalf("%s: pack %s; want one cell x2 in the held form", label, packCells(got))
			}
			if w.containerWeight(0) != 2*3 {
				t.Fatalf("%s: pack weight %d, want 6", label, w.containerWeight(0))
			}
			if bound {
				if got[0].ObjectID != 10 && name != "unequip" {
					t.Fatalf("%s: merged cell is object %d, want the held object 10", label, got[0].ObjectID)
				}
				retired := 0
				for _, row := range w.savedObjects.Items {
					if row.Retired && row.Value.Code == packMergeBow.Code {
						retired++
					}
				}
				if boundIncoming && retired != 1 {
					t.Fatalf("%s: %d retired bow objects, want the incoming one", label, retired)
				}
				// A plain cell's weight is not explicit, so only an explicit
				// one binds the saved accumulator.
				if c := w.savedObjects.container(w.savedPackOwner(0)); len(c.Items) != 1 || c.Items[0] != got[0].ObjectID || held.WeightPresent && c.Accumulator != w.containerWeight(0) {
					t.Fatalf("%s: saved pack %+v, pack weight %d", label, c, w.containerWeight(0))
				}
				if err := w.validateSavedObjects(); err != nil {
					t.Fatalf("%s: %v", label, err)
				}
			}
			if cold := reloadOperations(t, w); packCells(cold.carried[0]) != packCells(got) {
				t.Fatalf("%s: native continuation pack %s, want %s", label, packCells(cold.carried[0]), packCells(got))
			}
		}
		w := route(t, true, bound, built, packMergeBlockBow())
		if len(w.carried[0]) != 2 {
			t.Fatalf("bound %v: pack %s; want the byte-21 bow in its own cell", bound, packCells(w.carried[0]))
		}
	}
}

// A bow picked up from a Sack, the Sack a map placed or one a corpse left,
// joins an equal-operand bow of the other form. The world declares no
// constructor, as a mission entered from the city does.
func TestPickupJoinsTheBowOfEqualEffectiveOperands(t *testing.T) {
	checkEffectiveRoute(t, "pickup", false)
}

// A bow another actor dropped is a saved object in a bound world; picked up,
// it joins an equal-operand bow of the other form and its object is retired.
func TestDroppedBowJoinsTheBowOfEqualEffectiveOperands(t *testing.T) {
	checkEffectiveRoute(t, "drop-and-pickup", true)
}

// A bow moved from another pack joins an equal-operand bow of the other form.
func TestMovedBowJoinsTheBowOfEqualEffectiveOperands(t *testing.T) {
	checkEffectiveRoute(t, "transfer", true)
}

// A bow taken off joins the pack's equal-operand bow of the other form.
func TestUnequippedBowJoinsTheBowOfEqualEffectiveOperands(t *testing.T) {
	checkEffectiveRoute(t, "unequip", true)
}

// A granted bow joins the pack's equal-operand bow of the other form.
func TestGrantedBowJoinsTheBowOfEqualEffectiveOperands(t *testing.T) {
	checkEffectiveRoute(t, "grant", false)
}

// Without merge operands, as a second-game world is built, a plain bow and a
// constructed bow given as saved objects keep their cells.
func TestWithoutMergeOperandsStoredFormsStayApart(t *testing.T) {
	w := effectiveMergeWorld(t, false, true, []ItemStack{StackItem(packMergeBow, 1)}, []ItemStack{StackItem(packMergeBuiltBow(false), 1)})
	if err := w.MoveCarried(8, 7, packMergeBow.Code, 1); err != nil {
		t.Fatal(err)
	}
	if len(w.carried[0]) != 2 {
		t.Fatalf("pack %s; want two cells", packCells(w.carried[0]))
	}
}
