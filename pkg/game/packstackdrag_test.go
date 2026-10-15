package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type packDragRig struct {
	f    *FrontEnd
	app  *ui.App
	hero sim.EntityID
}

func packDragSelect(t *testing.T, f *FrontEnd, app *ui.App, hero sim.EntityID) packDragRig {
	t.Helper()
	app.Layout(1024, 768)
	for range 32 {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
	e, ok := f.live.world.Entity(hero)
	if !ok {
		t.Fatal("inventory actor absent", hero)
	}
	f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
	if !f.live.invSubjectSet || f.live.invSubject.ID != uint32(hero) || !slices.Contains(app.HeadlessSelection(), uint32(hero)) {
		if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
			t.Fatal(err)
		}
	}
	if !f.live.view.SaveApplication().InventoryOpen {
		if err := app.HeadlessKey("i"); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.invSubject.ID != uint32(hero) {
		t.Fatal("pointer inventory is bound to another actor")
	}
	return packDragRig{f: f, app: app, hero: hero}
}

func packDragFixture(t *testing.T, sourceCount uint32) packDragRig {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	party := f.ChargenParty(ui.ChargenResult{Name: "Pack drag", Choices: []int{0, 0, 4}, Stats: []int{31, 27, 24, 29}})
	bow := mapload.ItemInstanceFromCode(0x8114, f.Table)
	other := mapload.ItemInstanceFromCode(0x8115, f.Table)
	stacks := []sim.ItemStack{sim.StackItem(bow, 6), sim.StackItem(bow, 2), sim.StackItem(bow, sourceCount), sim.StackItem(other, 2)}
	party[0].Carry = &mapload.Carry{OrderedStacks: stacks}
	for _, stack := range stacks {
		for range stack.Count {
			party[0].Carry.ItemInstances = append(party[0].Carry.ItemInstances, stack.Instance())
			party[0].Carry.Items = append(party[0].Carry.Items, stack.Code)
		}
	}
	app := f.App("pack stack drag")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	r := packDragSelect(t, f, app, f.live.mission.ids[0])
	got := r.pack(t)
	if len(got) != len(stacks) {
		t.Fatalf("ordered fixture cells = %+v, want four separate stacks", got)
	}
	for i, stack := range got {
		if stack.ObjectID == 0 || stack.Code != stacks[i].Code || stack.Count != stacks[i].Count {
			t.Fatalf("fixture cell %d = %+v, want bound %#x x%d", i, stack, stacks[i].Code, stacks[i].Count)
		}
	}
	return r
}

func (r packDragRig) pack(t *testing.T) []sim.ItemStack {
	t.Helper()
	stacks, ok := r.f.live.world.CarriedStacks(r.hero)
	if !ok {
		t.Fatal("inventory actor has no pack")
	}
	return stacks
}

func (r packDragRig) drag(t *testing.T, source, target int, whole bool) {
	t.Helper()
	sx, sy, err := r.app.HeadlessPackCellPoint(source)
	if err != nil {
		t.Fatal(err)
	}
	var dx, dy int
	if target < 0 {
		dx, dy, err = r.app.HeadlessGroundPoint()
	} else {
		dx, dy, err = r.app.HeadlessPackCellPoint(target)
	}
	if err != nil {
		t.Fatal(err)
	}
	prefix := ""
	if whole {
		prefix = "shift-"
	}
	for _, edge := range []struct {
		name string
		x, y int
	}{{"press", sx, sy}, {"move", dx, dy}, {"release", dx, dy}} {
		if source == target && edge.name == "move" {
			edge.x += 6
		}
		if err := r.app.HeadlessPointer(prefix+edge.name, edge.x, edge.y); err != nil {
			t.Fatal(err)
		}
	}
}

func (r packDragRig) drain(t *testing.T) {
	t.Helper()
	for range 3 {
		if err := stepConsumableApp(r.app); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.f.live.pending) != 0 {
		t.Fatalf("production command drain left %+v", r.f.live.pending)
	}
}

func packDragRequire(t *testing.T, label string, got, want []sim.ItemStack) {
	t.Helper()
	if !slices.EqualFunc(got, want, sim.StackStateEqual) {
		t.Fatalf("%s pack\n got %+v\nwant %+v", label, got, want)
	}
}

func packDragOutput(t *testing.T) string {
	t.Helper()
	root := os.Getenv("AGAINROM_PACK_STACK_WITNESS_DIR")
	if root == "" {
		return t.TempDir()
	}
	dir := filepath.Join(root, filepath.Base(os.Getenv("AGAINROM_ASSETS")), strings.ReplaceAll(t.Name(), "/", "-"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func (r packDragRig) cold(t *testing.T, label string) packDragRig {
	t.Helper()
	path := saveCorpseMission(t, r.f, filepath.Join(packDragOutput(t), label))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, stack := range r.pack(t) {
		row, ok := r.f.live.world.SavedObjects().Item(stack.ObjectID)
		if !ok {
			t.Fatal("live carried object is absent from its registry", stack.ObjectID)
		}
		found := 0
		for i := range doc.Objects {
			object := &doc.Objects[i]
			if !savedItemClass(object.Class) {
				continue
			}
			identity, err := savedStructureValue(object, "Identity")
			if err != nil || identity != row.Token.Identity {
				continue
			}
			found++
			code, codeErr := savedStructureValue(object, "F40")
			count, countErr := savedStructureValue(object, "F42")
			if codeErr != nil || countErr != nil || code != uint32(stack.Code) || count != stack.Count {
				t.Fatalf("ordinary SAV object %#x code/count %#x/%d, want %#x/%d", identity, code, count, stack.Code, stack.Count)
			}
		}
		if found != 1 {
			t.Fatalf("ordinary SAV has %d records for carried object %#x, want one", found, row.Token.Identity)
		}
	}
	g := releaseFront(t)
	g.SetDeterministicFrames(true)
	g.Options = OptionsStore{}
	open, town, err := g.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("cold SAV LOAD: town=%v err=%v", town, err)
	}
	app := g.App("pack stack drag cold LOAD")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var hero sim.EntityID
	for i, member := range r.f.live.mission.party {
		if r.f.live.mission.ids[i] != r.hero {
			continue
		}
		for j, restored := range g.live.mission.party {
			if restored.ID == member.ID {
				hero = g.live.mission.ids[j]
			}
		}
	}
	if hero == 0 {
		t.Fatal("cold SAV lost the inventory actor's party identity")
	}
	t.Logf("%s: ordinary SAV %s SHA256=%x bytes=%d", label, path, sha256.Sum256(raw), len(raw))
	return packDragSelect(t, g, app, hero)
}

func TestReleasePackStackDragGestures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target int
		whole  bool
		single bool
	}{
		{"merge-one", 1, false, false},
		{"merge-whole", 1, true, false},
		{"floor-one", -1, false, false},
		{"floor-whole", -1, true, false},
		{"unequal-partial", 3, false, false},
		{"unequal-whole", 3, true, false},
		{"unequal-singleton", 3, false, true},
		{"self", 2, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := uint32(3)
			if tc.single {
				count = 1
			}
			r := packDragFixture(t, count)
			before := r.pack(t)
			want := slices.Clone(before)
			switch {
			case tc.target == 1:
				take := uint32(1)
				if tc.whole {
					take = count
				}
				want[1].Count += take
				want[2].Count -= take
				if want[2].Count == 0 {
					want = slices.Delete(want, 2, 3)
				}
			case tc.target == 3 && (tc.whole || tc.single):
				want[2], want[3] = want[3], want[2]
			case tc.target < 0:
				want[2].Count--
				if tc.whole {
					want = slices.Delete(want, 2, 3)
				}
			}
			r.drag(t, 2, tc.target, tc.whole)
			r.drain(t)
			packDragRequire(t, "after pointer/drain", r.pack(t), want)
			ground := func(r packDragRig) uint32 {
				var total uint32
				for _, sack := range r.f.live.world.Sacks() {
					for _, item := range sack.ItemInstances {
						if item.Code == before[2].Code {
							total++
							if item.ObjectID == 0 || tc.whole && item.ObjectID != before[2].ObjectID || !tc.whole && item.ObjectID == before[2].ObjectID {
								t.Fatalf("floor object %d, want preserved whole identity or new one-item identity from %d", item.ObjectID, before[2].ObjectID)
							}
						}
					}
				}
				return total
			}
			wantFloor := uint32(0)
			if tc.target < 0 {
				wantFloor = 1
				if tc.whole {
					wantFloor = count
				}
			}
			if got := ground(r); got != wantFloor {
				t.Fatalf("floor quantity %d, want %d", got, wantFloor)
			}
			var carried uint32
			for _, stack := range r.pack(t) {
				if stack.Code == before[2].Code {
					carried += stack.Count
				}
			}
			if carried+ground(r) != 8+count {
				t.Fatal("pointer gesture changed total quantity", carried, ground(r), 8+count)
			}
			cold := r.cold(t, "after-gesture")
			packDragRequire(t, "cold LOAD", cold.pack(t), want)
			if got := ground(cold); got != wantFloor {
				t.Fatalf("cold LOAD floor quantity %d, want %d", got, wantFloor)
			}
			// The first equal cell is an independent destination after LOAD.
			next := slices.Clone(want)
			next[0].Count++
			next[1].Count--
			cold.drag(t, 1, 0, false)
			cold.drain(t)
			packDragRequire(t, "next pointer action after LOAD", cold.pack(t), next)
			t.Logf("source=%d destination=%d whole=%v: exact cells/identities, quantities, SAV/cold LOAD and next drag", 2, tc.target, tc.whole)
		})
	}
}

func TestReleasePackStackDragPendingSAVEAndCancellation(t *testing.T) {
	r := packDragFixture(t, 3)
	before := r.pack(t)
	if err := r.app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if !r.f.live.stopped {
		t.Fatal("player pause did not stop inventory command application")
	}
	sx, sy, err := r.app.HeadlessPackCellPoint(2)
	if err != nil {
		t.Fatal(err)
	}
	dx, dy, err := r.app.HeadlessPackCellPoint(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []struct {
		name string
		x, y int
	}{{"shift-press", sx, sy}, {"shift-move", dx, dy}} {
		if err := r.app.HeadlessPointer(edge.name, edge.x, edge.y); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := r.app.HeadlessPointer("release", dx, dy); err != nil {
		t.Fatal(err)
	}
	packDragRequire(t, "cancelled drag", r.pack(t), before)
	if len(r.f.live.pending) != 0 {
		t.Fatal("cancelled drag left an executable inventory command", r.f.live.pending)
	}
	r = packDragFixture(t, 3)
	before = r.pack(t)
	if err := r.app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	if !r.f.live.stopped {
		t.Fatal("fresh pending-queue witness did not enter player pause")
	}
	r.drag(t, 2, 1, true)
	if err := stepConsumableApp(r.app); err != nil {
		t.Fatal(err)
	}
	if len(r.f.live.pending) != 1 || r.f.live.pending[0] != sim.RearrangeCarried(r.hero, 2, 1, true) {
		t.Fatalf("paused pointer queue = %+v, want one exact rearrangement", r.f.live.pending)
	}
	packDragRequire(t, "queued before SAVE", r.pack(t), before)
	cold := r.cold(t, "pending-gesture")
	packDragRequire(t, "pending cold LOAD", cold.pack(t), before)
	if len(cold.f.live.pending) != 1 || cold.f.live.pending[0] != sim.RearrangeCarried(cold.hero, 2, 1, true) {
		t.Fatalf("cold pending queue = %+v, want exact source, destination and quantity", cold.f.live.pending)
	}
	if err := cold.app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	cold.drain(t)
	want := slices.Clone(before)
	want[1].Count += want[2].Count
	want = slices.Delete(want, 2, 3)
	packDragRequire(t, "resumed queued drag", cold.pack(t), want)
}

func TestReleasePackStackDragOwnerDoubleSpear(t *testing.T) {
	input := os.Getenv("AGAINROM_PACK_STACK_OWNER_SAVE")
	if input == "" {
		t.Skip("AGAINROM_PACK_STACK_OWNER_SAVE required")
	}
	if os.Getenv("AGAINROM_PACK_STACK_WITNESS_DIR") == "" {
		t.Fatal("AGAINROM_PACK_STACK_WITNESS_DIR must name the owned output directory")
	}
	raw, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "c77b69888b8d41391c8db3358645a9d1bb79580a32bae9c5075218f298c7de9c" {
		t.Fatalf("owner double-spear SAV SHA256=%s", got)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("owner double-spear LOAD: town=%v err=%v", town, err)
	}
	app := f.App("owner double spear")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	var hero sim.EntityID
	source, target := -1, -1
	for _, id := range f.live.mission.ids {
		pack, _ := f.live.world.CarriedStacks(id)
		for i, a := range pack {
			for j := i + 1; j < len(pack); j++ {
				b := pack[j]
				if a.SourceEquipment.Class == sim.SourceWeapon && a.Code == 0x112f && b.Code == a.Code && a.Count == 3 && b.Count == 1 {
					hero, target, source = id, i, j
				}
			}
		}
		var cells []string
		for _, stack := range pack {
			cells = append(cells, fmt.Sprintf("%#x x%d object%d", stack.Code, stack.Count, stack.ObjectID))
		}
		t.Logf("owner actor %d pack: %s", id, strings.Join(cells, "; "))
	}
	if source < 0 {
		t.Fatal("owner SAV has no two weapon cells with one code and total quantity four")
	}
	r := packDragSelect(t, f, app, hero)
	before := r.pack(t)
	want := slices.Clone(before)
	want[target].Count = 4
	want = slices.Delete(want, source, source+1)
	r.drag(t, source, target, true)
	r.drain(t)
	packDragRequire(t, "owner merged spear", r.pack(t), want)
	cold := r.cold(t, "owner-merged")
	packDragRequire(t, "owner spear cold LOAD", cold.pack(t), want)
	index := slices.IndexFunc(want, func(stack sim.ItemStack) bool { return stack.ObjectID == before[target].ObjectID })
	if index < 0 || want[index].Count != 4 {
		t.Fatal("merged owner destination identity absent")
	}
	cold.drag(t, index, -1, false)
	cold.drain(t)
	want[index].Count = 3
	packDragRequire(t, "owner next single floor drop after LOAD", cold.pack(t), want)
	if after, err := os.ReadFile(input); err != nil || sha256.Sum256(after) != sha256.Sum256(raw) {
		t.Fatal("owner input changed", err)
	}
	t.Logf("owner input SHA256=%x: %#x merged into destination object %d x4; SAV/cold LOAD and next single drop x3", sha256.Sum256(raw), before[target].Code, before[target].ObjectID)
}
