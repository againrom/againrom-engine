package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// rawCellBindings1084 deliberately walks the installed ALM bytes without the
// production type-9 decoder or a research probe. The literal expected rows
// below independently pin which words (not the item-tail interpretation) feed
// the runtime binding. No install bytes are committed.
func rawCellBindings1084(t *testing.T, raw []byte) []sim.CellTail {
	t.Helper()
	u32 := binary.LittleEndian.Uint32
	u16 := binary.LittleEndian.Uint16
	if len(raw) < 20 {
		t.Fatal("short ALM")
	}
	var section []byte
	off := 20
	for k := uint32(0); k < u32(raw[12:16]); k++ {
		if off+20 > len(raw) {
			t.Fatal("short record header")
		}
		h := raw[off : off+20]
		size := int(u32(h[8:12]))
		if size > len(raw)-off-20 || u32(h[4:8]) != 20 {
			t.Fatal("invalid record span")
		}
		if u32(h[12:16]) == 9 {
			section = raw[off+20 : off+20+size]
		}
		off += 20 + size
	}
	if off != len(raw) || len(section) < 4 || u32(section[:4]) != 11 {
		t.Fatal("M10 type-9 census differs from 11")
	}
	var out []sim.CellTail
	off = 4
	for k := 0; k < 11; k++ {
		if off+26 > len(section) {
			t.Fatal("short type-9 head")
		}
		h := section[off:]
		n := int(u32(h[22:26]))
		if n > (len(h)-26)/6 {
			t.Fatal("short type-9 tail")
		}
		x, y := u32(h[4:8]), u32(h[8:12])
		if (x != 0 || y != 0) && u16(h[12:14]) < 4 {
			if n < 2 {
				t.Fatal("cell has no source pair")
			}
			out = append(out, sim.CellTail{X: int32(byte(x)), Y: int32(byte(y)),
				Bytes: [6]byte{h[18], h[20], h[26], h[28], h[32], h[34]}})
		}
		off += 26 + n*6
	}
	if off != len(section) {
		t.Fatal("type-9 trailing bytes")
	}
	return out
}

func entryPointer1084(t *testing.T, app *ui.App, x, y int) {
	t.Helper()
	for i := 0; i < 16 && app.HeadlessNoticeOpen(); i++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	for py := 100; py < 560; py += 4 {
		for px := 160; px < 750; px += 4 {
			cx, cy, err := app.HeadlessDropCell(px, py)
			if err == nil && cx == x && cy == y {
				if err := app.HeadlessPointer("press", px, py); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessPointer("release", px, py); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
	}
	t.Fatalf("cell (%d,%d) has no visible ground pointer witness", x, y)
}

func entryStep1084(app *ui.App) error {
	for i := 0; i < 16 && app.HeadlessNoticeOpen(); i++ {
		if err := app.HeadlessKey("enter"); err != nil {
			return err
		}
	}
	return app.HeadlessStep()
}

func checkEntryNative1084(t *testing.T, f *FrontEnd, stage string) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	disk, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(disk)
	if err != nil {
		t.Fatal(err)
	}
	restored := releaseFront(t)
	opener, town, err := restored.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatalf("%s Restore: %v town=%v", stage, err, town)
	}
	if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
		t.Fatal(err)
	}
	form, _ := restored.live.world.MarshalBinary()
	if !bytes.Equal(form, snapshot.World) || restored.live.world.Hash() != f.live.world.Hash() {
		t.Fatalf("%s native restore changed canonical state", stage)
	}
	// Compare the next ordinary simulation tick without changing the live App.
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	a, b := sim.StepReported(&control, nil), sim.StepReported(restored.live.world, nil)
	if control.Hash() != restored.live.world.Hash() || !reflect.DeepEqual(a, b) {
		t.Fatalf("%s pending/effect continuation changed across native restore", stage)
	}
	t.Logf("native %s: tails=%d pending=%d next events=%d", stage,
		len(f.live.world.CellTails()), len(f.live.world.ScriptCasts()), len(a.ScriptCasts))
}

func TestReleaseM10CellEntry1084UsesInstalledBindingPointerLightningAndNativeSave(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(10)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatal(err)
	}
	bindings := rawCellBindings1084(t, raw)
	want := []sim.CellTail{
		{X: 22, Y: 64, Bytes: [6]byte{13, 1, 23, 65, 22, 64}},
		{X: 21, Y: 63, Bytes: [6]byte{13, 1, 19, 61, 21, 63}},
	}
	if !reflect.DeepEqual(bindings, want) {
		t.Fatalf("raw installed cell bindings = %+v, want %+v", bindings, want)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := m.CellBindings()
	if err != nil || len(decoded) != 2 || decoded[0].SourceX != 23 || decoded[1].SourceY != 61 {
		t.Fatalf("typed binding = %+v, %v", decoded, err)
	}
	for i, binding := range want {
		t.Run([]string{"22_64", "21_63"}[i], func(t *testing.T) {
			entryApp1084(t, releaseFront(t), binding, want)
		})
	}
}

func entryApp1084(t *testing.T, f *FrontEnd, binding sim.CellTail, want []sim.CellTail) {
	t.Helper()
	x, y := int(binding.X), int(binding.Y)
	from := image.Pt(int(binding.Bytes[2]), int(binding.Bytes[3]))
	f.SetDeterministicFrames(true)
	app := f.App("1084-cell-entry")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	id := live.mission.ids[0]
	if got := live.world.CellTails(); !reflect.DeepEqual(got, []sim.CellTail{want[1], want[0]}) {
		t.Fatalf("production mission lost cell bindings: %+v", got)
	}
	// Fixture setup only: place the existing campaign hero next to the cell.
	// The entry itself is a real pointer-issued order, not a forced teleport.
	if err := live.world.HeadlessPlace(id, 21, 64); err != nil {
		t.Fatal(err)
	}
	live.push()
	live.view.Camera().CenterOn(22*32, 64*32)
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	checkEntryNative1084(t, f, "before-entry")
	before, _ := live.entity(id)
	entryPointer1084(t, app, x, y)
	if len(live.pending) == 0 || live.pending[len(live.pending)-1].X != int32(x) || live.pending[len(live.pending)-1].Y != int32(y) {
		t.Fatalf("pointer did not queue the authored cell: %+v", live.pending)
	}
	pending := false
	for tick := 0; tick < 100; tick++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
		for _, c := range live.world.ScriptCasts() {
			if c.Target == id && c.Spell == 13 {
				if c.FromX != int32(from.X) || c.FromY != int32(from.Y) || c.Power != 1 || !c.AtUnit {
					t.Fatalf("cell requested %+v", c)
				}
				pending = true
			}
		}
		if pending {
			break
		}
		if _, _, open := f.LiveNotice(); open {
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !pending {
		e, _ := live.entity(id)
		text, kind, open := f.LiveNotice()
		t.Fatalf("pointer route did not attach to hazard: tick=%d screen=%s notice=%q/%v/%v pos=(%d,%d) turn=%d", live.world.Tick(), app.Screen(), text, kind, open, e.X, e.Y, e.TurnRemaining)
	}
	checkEntryNative1084(t, f, "requested")
	for tick := 0; tick < 8 && len(live.world.ScriptCasts()) > 0; tick++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
	}
	launched, _ := live.entity(id)
	if len(live.world.ScriptCasts()) != 0 || live.world.PendingSpellDeliveries() != 1 || launched.HP != before.HP || launched.SpellFXSpell == 13 {
		t.Fatal("Lightning release must prepare one child without applying damage")
	}
	found := false
	for _, p := range flightRecords(live) {
		if p.Picture != 34 {
			continue
		}
		for _, b := range live.recordPaths(p, 0) {
			if b.from == from && b.to == image.Pt(x, y) {
				if p.ActionSegments != 4 || len(live.pathDraws(b)) == 0 {
					t.Fatalf("direct Lightning has no installed-art path: %+v", p)
				}
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("no source-to-target Lightning presentation: %+v", flightRecords(live))
	}
	checkEntryNative1084(t, f, "in-flight")
	launchTick := live.world.Tick()
	for frame := 0; frame < 32 && live.world.PendingSpellDeliveries() != 0; frame++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
		e, _ := live.entity(id)
		if live.world.Tick() < launchTick+11 && e.HP < before.HP {
			t.Fatal("Lightning damaged before its transport handed off")
		}
		if live.world.Tick() == launchTick+10 {
			pending := live.world.NativeSpellDeliverySaveStates()
			if len(pending) != 1 || !pending[0].Released {
				t.Fatal("Lightning lost its pending Point at handoff")
			}
		}
	}
	after, _ := live.entity(id)
	if live.world.Tick() != launchTick+11 || live.world.PendingSpellDeliveries() != 0 || after.HP >= before.HP || after.SpellFXSpell != 13 {
		t.Fatalf("Lightning delivery: elapsed%d hp%d->%d effect%d", live.world.Tick()-launchTick, before.HP, after.HP, after.SpellFXSpell)
	}
	checkEntryNative1084(t, f, "after-effect")
	for i := 0; i < 20; i++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
		if len(live.world.ScriptCasts()) != 0 {
			t.Fatal("standing on a hazard requested another cast")
		}
	}
	standing, _ := live.entity(id)
	if standing.HP < after.HP {
		t.Fatalf("standing lost HP %d -> %d", after.HP, standing.HP)
	}
	// Leave through a nearby non-hazard, then reenter with another pointer
	// order. No setup teleport is used for either half of this repeat control.
	live.view.Camera().CenterOn(22*32, 64*32)
	entryPointer1084(t, app, 21, 64)
	for tick := 0; tick < 150; tick++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
		if len(live.world.ScriptCasts()) != 0 {
			t.Fatal("nearby (21,64) requested a cell cast")
		}
		e, _ := live.entity(id)
		if e.X == 21 && e.Y == 64 && e.Transit == 0 {
			break
		}
		if tick == 149 {
			t.Fatal("pointer leave did not reach the nearby cell")
		}
	}
	live.view.Camera().CenterOn(22*32, 64*32)
	entryPointer1084(t, app, x, y)
	repeated := false
	for tick := 0; tick < 150; tick++ {
		if err := entryStep1084(app); err != nil {
			t.Fatal(err)
		}
		if len(live.world.ScriptCasts()) != 0 {
			repeated = true
			break
		}
	}
	if !repeated {
		t.Fatal("pointer reentry failed to request another cast")
	}
	t.Logf("M10 pointer entry (%d,%d): source %v, spell 13, power 1, HP %d -> %d; nearby/standing/reentry and native before/during/after matched", x, y, from, before.HP, after.HP)
}
