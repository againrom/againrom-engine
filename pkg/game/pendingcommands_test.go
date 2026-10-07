package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func pendingResumeStep(mw *mapWorld) {
	commands := mw.commands()
	mw.pending, mw.pendingIgnored = nil, nil
	sim.Step(mw.world, commands)
}

func TestPendingCommandsSAVKeepsOrderedSource(t *testing.T) {
	f := castOrderFront(t)
	f.live.setCadence(62000, true)
	f.live.enqueue(1, 14, 10)
	f.live.strike(1, 2)
	f.live.enqueue(1, 16, 12)
	queue := append([]sim.Command(nil), f.live.pending...)
	before, hash := marshalWorld(t, f.live.world), f.live.world.Hash()
	raw, _, _ := saveCurrentEffect(t, f)
	if !bytes.Equal(before, marshalWorld(t, f.live.world)) || hash != f.live.world.Hash() || !reflect.DeepEqual(queue, f.live.pending) {
		t.Fatal("SAVE changed World or consumed pending commands")
	}
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "before first resumed tick")
	if !reflect.DeepEqual(queue, cold.live.pending) {
		t.Fatalf("cold LOAD lost ordered commands: source %+v cold %+v", queue, cold.live.pending)
	}
}

func TestPendingCommandsResumeAndLossControls(t *testing.T) {
	f := midStrikeCastFront(t)
	f.live.pending = []sim.Command{sim.MoveTo(1, sim.CellPoint{X: 16, Y: 12}), sim.Attack(1, 2)}
	f.live.commanded[1] = true
	before, actions := marshalWorld(t, f.live.world), f.live.world.Actions()
	raw, doc, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	if !bytes.Equal(before, marshalWorld(t, cold.live.world)) || !reflect.DeepEqual(actions, cold.live.world.Actions()) {
		t.Fatal("paused queued replacement changed old physical strike clocks at LOAD")
	}
	loss := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) { a.Pending.Commands = nil })
	swap := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) {
		a.Pending.Commands[0], a.Pending.Commands[1] = a.Pending.Commands[1], a.Pending.Commands[0]
	})
	for _, control := range []*FrontEnd{loss, swap} {
		if !bytes.Equal(before, marshalWorld(t, control.live.world)) {
			t.Fatal("queue-only control changed World before resume")
		}
	}
	for tick := range 80 {
		pendingResumeStep(f.live)
		pendingResumeStep(cold.live)
		pendingResumeStep(loss.live)
		pendingResumeStep(swap.live)
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, "queued first action and subsequent ticks")
		if tick == 0 {
			if len(cold.live.pending) != 0 || cold.live.world.Hash() == loss.live.world.Hash() || cold.live.world.Hash() == swap.live.world.Hash() {
				t.Fatal("missing/swap control did not change the first resumed action")
			}
			_, _, again := saveCurrentEffect(t, cold)
			if again.Pending == nil || len(again.Pending.Commands) != 0 {
				t.Fatal("second SAVE retained an already drained queue")
			}
		}
	}
}

func TestPendingCommandsOperandLossControls(t *testing.T) {
	for _, control := range []struct {
		name    string
		command sim.Command
		change  func(*currentPendingCommand)
	}{
		{"cell", sim.MoveTo(1, sim.CellPoint{X: 16, Y: 12}), func(c *currentPendingCommand) { c.Command.X = 10 }},
		{"issuer", sim.Attack(1, 2), func(c *currentPendingCommand) {
			c.Command.Entity, c.Issuer.ID, c.Issuer.Object = 2, 2, c.Target.Object
		}},
		{"target", sim.Attack(1, 2), func(c *currentPendingCommand) {
			c.Command.X, c.Target.ID, c.Target.Object = 1, 1, c.Issuer.Object
		}},
		{"spell", sim.Cast(1, 2, 6), func(c *currentPendingCommand) { c.Command.Y = 3 }},
		{"item", sim.UseScroll(1, 0, 2), func(c *currentPendingCommand) { c.Command.Spell = 99 }},
		{"group", sim.GroupMoveTo(1, sim.CellPoint{X: 16, Y: 12}, 19), func(c *currentPendingCommand) { c.Command.Group = 20 }},
	} {
		t.Run(control.name, func(t *testing.T) {
			f := midStrikeScrollFront(t)
			f.live.pending = []sim.Command{control.command}
			if control.name == "group" {
				f.live.pending = append(f.live.pending, sim.GroupMoveTo(2, sim.CellPoint{X: 16, Y: 12}, 19))
			}
			before := marshalWorld(t, f.live.world)
			_, doc, _ := saveCurrentEffect(t, f)
			altered := control.command
			cold := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) {
				control.change(&a.Pending.Commands[0])
				altered = a.Pending.Commands[0].Command
			})
			if !bytes.Equal(before, marshalWorld(t, cold.live.world)) || len(cold.live.pending) != len(f.live.pending) || cold.live.pending[0] != altered || !slices.Equal(cold.live.pending[1:], f.live.pending[1:]) || altered == control.command {
				t.Fatal("operand-only control changed pre-resume World or lost its exact operand")
			}
			pendingResumeStep(f.live)
			pendingResumeStep(cold.live)
			if control.name == "item" {
				for remaining := 160; remaining > 0 && f.live.world.Hash() == cold.live.world.Hash(); remaining-- {
					pendingResumeStep(f.live)
					pendingResumeStep(cold.live)
				}
			}
			if f.live.world.Hash() == cold.live.world.Hash() {
				t.Fatal("operand-only loss did not change the resumed action")
			}
		})
	}
}

func TestPendingCommandsTypedDomainsAndActorZero(t *testing.T) {
	commands := []sim.Command{
		sim.Attack(0, 91), sim.Cast(0, 91, 6), sim.CastAt(0, 3, sim.CellPoint{X: 91, Y: 0}),
		sim.GroupDefend(0, 91, 7), sim.AttackStructure(0, 91), sim.UseStructure(0, 91),
		sim.UseScroll(0, 91, 91), sim.UseScrollAt(0, 91, sim.CellPoint{X: 91, Y: 0}),
		sim.EquipDisplacing(0, 91, 2, 3), sim.Unequip(0, 2), sim.DropCarried(0, 91, sim.CellPoint{X: 91, Y: 0}),
		sim.DropWorn(0, 2, sim.CellPoint{X: 91, Y: 0}), sim.ReadBook(0, 91), sim.UsePotion(0, 91),
		sim.GroupMoveTo(0, sim.CellPoint{X: 91, Y: 0}, 17), sim.GroupMoveTo(91, sim.CellPoint{X: 91, Y: 0}, 17),
		sim.GroupStance(0, sim.OrderGuard, 17), sim.GroupRetreat(0, 91, 17), sim.Autocast(0, 91),
		sim.SetPlayerParameter(91, sim.PlayerParameterFormation, 2),
	}
	r := SnapshotResidue{PendingQueue: &SnapshotPendingQueue{Commands: commands, Ignored: make([]bool, len(commands))}, Commanded: []uint32{91, 0}}
	q, err := projectCurrentPending(r, map[sim.EntityID]uint16{0: 3, 91: 7}, map[sim.EntityID]uint16{91: 8})
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range q.Commands {
		if p.Command != commands[i] {
			t.Fatal("projection normalized ordered command operands")
		}
		if p.Issuer != nil && (p.Issuer.Missing || p.Issuer.Object != map[sim.EntityID]uint16{0: 3, 91: 7}[commands[i].Entity]) {
			t.Fatal("actor zero lost its exact binding")
		}
		if p.Target != nil && p.Target.Object != map[bool]uint16{false: 7, true: 8}[p.Target.Structure] {
			t.Fatal("actor/structure namespaces were conflated")
		}
	}
	f := castOrderFront(t)
	q.Commands = q.Commands[:4]
	q.Commanded = nil
	before := marshalWorld(t, f.live.world)
	if err := f.live.restoreCurrentPending(q, map[uint16]sim.EntityID{3: 0, 7: 2}); err != nil {
		t.Fatal(err)
	}
	if f.live.pending[0].Entity != 0 || f.live.pending[0].X != 2 || f.live.pending[1].Y != 6 || f.live.pending[2].X != 91 || f.live.pending[3].Group != 7 || !bytes.Equal(before, marshalWorld(t, f.live.world)) {
		t.Fatal("typed remap changed scalar operands or World")
	}
}

func TestPendingCommandsMissingEndpointDoesNotMintOrReplay(t *testing.T) {
	f := castOrderFront(t)
	f.live.pending = []sim.Command{sim.Attack(1, 9000), sim.UseStructure(1, 9000), sim.MoveTo(9000, sim.CellPoint{X: 14, Y: 10})}
	before := marshalWorld(t, f.live.world)
	raw, _, a := saveCurrentEffect(t, f)
	if a.Pending == nil || !a.Pending.Commands[0].Target.Missing || !a.Pending.Commands[1].Target.Structure || !a.Pending.Commands[2].Issuer.Missing {
		t.Fatal("missing endpoint policy was not explicit")
	}
	cold := openCurrentEffectSave(t, f, raw)
	if !bytes.Equal(before, marshalWorld(t, cold.live.world)) || !slices.Equal(cold.live.pendingIgnored, []bool{true, true, true}) {
		t.Fatal("missing queue endpoint mutated World or ID floors")
	}
	_, _, second := saveCurrentEffect(t, cold)
	for _, c := range second.Pending.Commands {
		if !c.Ignored {
			t.Fatal("second SAVE reactivated an ignored command")
		}
	}
	pendingResumeStep(f.live)
	pendingResumeStep(cold.live)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "missing endpoint resume")
	if len(cold.live.pending) != 0 || len(cold.live.pendingIgnored) != 0 {
		t.Fatal("ignored commands did not drain once")
	}
}

func TestPendingCommandsRemintReindexAndMalformedAtomicLOAD(t *testing.T) {
	f := castOrderFront(t)
	f.live.pending = []sim.Command{sim.Cast(1, 2, 6), sim.GroupMoveTo(2, sim.CellPoint{X: 15, Y: 10}, 3)}
	_, doc, _ := saveCurrentEffect(t, f)
	reindexed, _, err := sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	reindexed, err = sav.RemintDocumentKeys(reindexed)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(reindexed)
	if err != nil {
		t.Fatal(err)
	}
	cold := openCurrentEffectSave(t, f, raw)
	if !reflect.DeepEqual(f.live.pending, cold.live.pending) {
		t.Fatal("remint/reindex lost ordered queue")
	}
	for _, change := range []func(*currentPendingQueue){
		func(q *currentPendingQueue) { q.Commands[0].Command.Kind = 255 },
		func(q *currentPendingQueue) { q.Commands[0].Issuer = nil },
		func(q *currentPendingQueue) { q.Commands[0].Target.Structure = true },
		func(q *currentPendingQueue) { q.Commands[0].Issuer.Object = 65535 },
		func(q *currentPendingQueue) { q.Commands = make([]currentPendingCommand, 65537) },
	} {
		a, err := readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		change(a.Pending)
		payload, _ := json.Marshal(a)
		bad, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		sav.SetNativeActions(&bad.State, payload)
		encoded, err := sav.EncodeDocumentData(bad)
		if err != nil {
			continue
		}
		live, before, queue := f.live, marshalWorld(t, f.live.world), slices.Clone(f.live.pending)
		open, _, err := f.RestoreOriginal(encoded)
		if err == nil {
			err = f.App("malformed queue LOAD").OpenMission(open)
		}
		if err == nil || f.live != live || !bytes.Equal(before, marshalWorld(t, f.live.world)) || !reflect.DeepEqual(queue, f.live.pending) {
			t.Fatal("malformed queue LOAD changed live session", err)
		}
	}
}

func TestPendingCommandsConsumablesHaveOneResumeDebit(t *testing.T) {
	for _, command := range []sim.Command{sim.Cast(1, 2, 6), sim.UseScroll(1, 0, 2)} {
		f := midStrikeScrollFront(t)
		f.live.pending = []sim.Command{command}
		before := marshalWorld(t, f.live.world)
		raw, _, _ := saveCurrentEffect(t, f)
		cold := openCurrentEffectSave(t, f, raw)
		if !bytes.Equal(before, marshalWorld(t, f.live.world)) || !bytes.Equal(before, marshalWorld(t, cold.live.world)) {
			t.Fatal("paused SAVE/LOAD spent a resource")
		}
		for range 160 {
			pendingResumeStep(f.live)
			pendingResumeStep(cold.live)
			assertCurrentWorldEqual(t, f.live.world, cold.live.world, "one resumed consumable")
		}
		if command.Kind == sim.KindUseScroll {
			items, _ := cold.live.world.CarriedItems(1)
			if len(items) != 1 || len(cold.live.world.ScrollCasts()) != 0 {
				t.Fatal("scroll did not consume once and complete")
			}
		}
	}
}

func TestPendingCommandsOldAbsentAndNewEmptyQueue(t *testing.T) {
	f := castOrderFront(t)
	_, doc, _ := saveCurrentEffect(t, f)
	for _, present := range []bool{false, true} {
		cold := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) {
			a.Options = []PendingGameOption{{Option: ui.GameOptionFormation, Value: 2}}
			if !present {
				a.Pending = nil
			}
		})
		want := 1
		if present {
			want = 0
		}
		if len(cold.live.pending) != want || cold.live.view.SaveApplication().PlayerPaused {
			t.Fatal("old absent/new empty queue policy failed", present, cold.live.pending)
		}
	}
}

func TestPendingCommandsActorZeroThroughOrdinarySAV(t *testing.T) {
	f, snapshot, source := currentPlayerSlotFixture(t, 1)
	actors := source.Entities()
	for i := range actors {
		actors[i].Capacity = data.UnitCapacity()
	}
	var err error
	source, err = sim.NewWorld(31, source.Bounds(), sim.ModeCanonical, nil, actors)
	if err != nil {
		t.Fatal(err)
	}
	for owner := uint32(0); owner <= 1; owner++ {
		source.SetPurse(owner, 0xf1234500+owner)
	}
	snapshot.World = marshalWorld(t, source)
	commands := []sim.Command{sim.MoveTo(0, sim.CellPoint{X: 12, Y: 11}), sim.Attack(1, 0), sim.GroupDefend(1, 0, 19)}
	snapshot.Residue.PendingQueue = &SnapshotPendingQueue{Commands: commands, Ignored: make([]bool, len(commands))}
	snapshot.Residue.Commanded = []uint32{0, 1}
	raw, err := f.ExportCurrentSave(snapshot, "actor zero queued")
	if err != nil {
		t.Fatal(err)
	}
	cold := cellStateFront(t)
	cold.Campaign, cold.Table = f.Campaign, f.Table
	open, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("actor zero cold queue").OpenMission(open)
	}
	if err != nil || town {
		t.Fatal(err, town)
	}
	if !slices.Equal(cold.live.pending, commands) {
		t.Fatal("actor zero was treated as no issuer/target", cold.live.pending)
	}
	assertCurrentWorldEqual(t, source, cold.live.world, "actor zero before resume")
	sim.Step(source, commands)
	pendingResumeStep(cold.live)
	assertCurrentWorldEqual(t, source, cold.live.world, "actor zero first resumed action")
}

func TestPendingCommandsPotionSlotOrderAndSingleDebit(t *testing.T) {
	f := castOrderFront(t)
	actors := f.live.world.Entities()
	actors[0].HP, actors[0].HealthRegenPeriod, actors[0].ManaRegenPeriod = 10, 0, 0
	first := sim.PlainItem(0xe07)
	first.Kind, first.Price, first.Effects = 3, 100, []sim.ItemEffect{{Kind: 6, Operand: 30}}
	second := first.Clone()
	second.Effects[0].Operand = 50
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, f.live.world.Relations(), nil, []sim.Stock{{ID: 1, ItemInstances: []sim.ItemInstance{first, second}}}, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	f.live.pending = []sim.Command{sim.UsePotion(1, 0), sim.UsePotion(1, 1)}
	before := marshalWorld(t, w)
	raw, doc, _ := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	swap := midStrikeSAVActionsControl(t, f, doc, func(a *currentActionData) {
		a.Pending.Commands[0], a.Pending.Commands[1] = a.Pending.Commands[1], a.Pending.Commands[0]
	})
	if !bytes.Equal(before, marshalWorld(t, w)) || !bytes.Equal(before, marshalWorld(t, cold.live.world)) || !bytes.Equal(before, marshalWorld(t, swap.live.world)) {
		t.Fatal("paused potion SAVE or slot-order control changed source pools")
	}
	pendingResumeStep(f.live)
	pendingResumeStep(cold.live)
	pendingResumeStep(swap.live)
	assertCurrentWorldEqual(t, w, cold.live.world, "potion ordered resume")
	items, _ := cold.live.world.CarriedItems(1)
	if castOrderHP(cold.live.world, 1) != 40 || len(items) != 1 || items[0].Effects[0].Operand != 50 || castOrderHP(swap.live.world, 1) != 90 {
		t.Fatal("potion slot order/debit loss control failed", castOrderHP(cold.live.world, 1), castOrderHP(swap.live.world, 1), items)
	}
	for range 24 {
		pendingResumeStep(f.live)
		pendingResumeStep(cold.live)
		assertCurrentWorldEqual(t, w, cold.live.world, "potion no second debit")
	}
	items, _ = cold.live.world.CarriedItems(1)
	if len(items) != 1 || items[0].Effects[0].Operand != 50 {
		t.Fatal("resumed potion duplicated a debit or effect")
	}
}
