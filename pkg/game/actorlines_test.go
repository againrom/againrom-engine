package game

import (
	"reflect"
	"testing"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// actorLineWords is the word set of one root's language: EN the authored
// words, RU the RU install's main.txt[85..89] and raise line 130 and the
// engine's Russian words, in the map font's CP866 bytes.
func actorLineWords(ru bool) ui.Words {
	if !ru {
		return (&InstallWords{Language: "english"}).Words()
	}
	w := (&InstallWords{Language: "russian", Selector: text.SelectorConverting}).Words()
	enc := func(s string) string { return EncodeInstallText(s, text.SelectorConverting) }
	w.PickedUp, w.PickedUpNow, w.PickedUpPieces = enc("Вы подняли:"), enc("теперь"), enc("шт")
	w.PickedUpGold, w.PickedUpGoldUnit = enc("Вы подняли"), enc("монет")
	w.SkillRaised[0] = enc("Повышен навык владения мечом")
	return w
}

// actorLineShown is a line as the player reads it.
func actorLineShown(t *testing.T, ru bool, s string) string {
	t.Helper()
	if !ru {
		return s
	}
	shown, err := charmap.CodePage866.NewDecoder().String(s)
	if err != nil {
		t.Fatal(err)
	}
	return shown
}

// Every row of the owner's table, EN and RU: a skill raise, a pickup of one
// and of a count, a sack's gold, a grant of one and of a count, and granted
// gold. The control beside each is the same line naming nobody, which is the
// line this build stated before the character was named.
func TestActorLinesStateTheOwnersTableInBothLanguages(t *testing.T) {
	for _, c := range []struct {
		ru   bool
		name string
		want [7]string
		bare [7]string
	}{
		{false, "Gord", [7]string{
			"Blade skill improved: 11 (Gord)",
			"Picked up Sword (Gord)",
			"Picked up Sword (now 2 pieces, Gord)",
			"Picked up 78 gold (Gord)",
			"Received Sword (Gord)",
			"Received Sword (now 2 pieces, Gord)",
			"Received 78 gold",
		}, [7]string{
			"Blade skill improved: 11",
			"Picked up Sword",
			"Picked up Sword (now 2 pieces)",
			"Picked up 78 gold",
			"Received Sword",
			"Received Sword (now 2 pieces)",
			"Received 78 gold",
		}},
		{true, "Горд", [7]string{
			"Повышен навык владения мечом: 11 (Горд)",
			"Вы подняли: Меч (Горд)",
			"Вы подняли: Меч (теперь 2 шт, Горд)",
			"Вы подняли 78 монет (Горд)",
			"Получено: Меч (Горд)",
			"Получено: Меч (теперь 2 шт, Горд)",
			"Получено: 78 монет",
		}, [7]string{
			"Повышен навык владения мечом: 11",
			"Вы подняли: Меч",
			"Вы подняли: Меч (теперь 2 шт)",
			"Вы подняли 78 монет",
			"Получено: Меч",
			"Получено: Меч (теперь 2 шт)",
			"Получено: 78 монет",
		}},
	} {
		words := actorLineWords(c.ru)
		item := "Sword"
		if c.ru {
			item = EncodeInstallText("Меч", text.SelectorConverting)
		}
		lines := func(name string) [7]string {
			var last, current [data.SkillSlots]int32
			last[data.SkillBlade], current[data.SkillBlade] = 10, 11
			raise := skillRiseRows(&words, false, last, current, name)
			if len(raise) != 1 {
				t.Fatalf("ru=%v: raise rows %q, want one", c.ru, raise)
			}
			return [7]string{
				raise[0],
				pickupItemLine(&words, item, 1, name),
				pickupItemLine(&words, item, 2, name),
				pickupGoldLine(&words, 78, name),
				grantItemLine(&words, item, 1, name),
				grantItemLine(&words, item, 2, name),
				grantGoldLine(&words, 78),
			}
		}
		name := c.name
		if c.ru {
			name = EncodeInstallText(name, text.SelectorConverting)
		}
		for i, got := range lines(name) {
			if shown := actorLineShown(t, c.ru, got); shown != c.want[i] {
				t.Errorf("ru=%v row %d = %q, want %q", c.ru, i, shown, c.want[i])
			}
		}
		for i, got := range lines("") {
			if shown := actorLineShown(t, c.ru, got); shown != c.bare[i] {
				t.Errorf("ru=%v control row %d = %q, want %q", c.ru, i, shown, c.bare[i])
			}
		}
	}
}

// namedGrabWorld is grabWorld with the party's hero named: the name the hero's
// card shows is the player's.
func namedGrabWorld(t *testing.T, w *sim.World, ids []sim.EntityID, names []string) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w, Start: mapload.Start{IDs: ids}}
	for i, name := range names {
		ms.Party = append(ms.Party, mapload.PartyMember{PlayerCharacter: true, StartingHero: i == 0, Name: name})
	}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	mw.invParty.table = eqDefsTable(t)
	return mw
}

// grantScript runs instants once on the first script pass.
func grantScript(t *testing.T, instants []sim.ScriptInstant) *sim.Script {
	t.Helper()
	var trigger sim.ScriptTrigger
	trigger.Pairs[0] = sim.ScriptPair{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}
	trigger.Once = true
	for i := range trigger.Instants {
		trigger.Instants[i] = sim.ScriptNone
		if i < len(instants) {
			trigger.Instants[i] = int32(i)
		}
	}
	s, err := sim.NewScript([]sim.ScriptCheck{
		{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
	}, instants, []sim.ScriptTrigger{trigger})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// grantLines runs a script giving hero 7, owned by owner, the instants, and
// answers the lines the map driver posted.
func grantLines(t *testing.T, owner uint32, instants []sim.ScriptInstant) []ui.MessageLine {
	t.Helper()
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: owner, HP: 20, MaxHP: 20}}, grantScript(t, instants), sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{eqNoSlotCode}}})
	if err != nil {
		t.Fatal(err)
	}
	mw := namedGrabWorld(t, w, []sim.EntityID{7}, []string{"Gord"})
	for n := 0; n < 64 && !w.ScriptLatched(0); n++ {
		mw.tick()
	}
	if !w.ScriptLatched(0) {
		t.Fatal("the grant script did not run")
	}
	return mw.view.MessageLines()
}

func addSword() sim.ScriptInstant {
	return sim.ScriptInstant{Op: sim.ScriptInstantAddItem, Unit: 7, HasUnit: true, Item: eqSwordCode, HasItem: true}
}

// Two instant-12 nodes give hero 7 one Sword each: the first states the
// received line with the hero's name, the second the merged count before the
// name (DIV-3766). The control gives the same Items to a hero of another
// player: the Items arrive and no line posts.
func TestScriptItemGrantStatesTheReceivedLineForItsHolder(t *testing.T) {
	want := announced("Received Sword (Gord)", "Received Sword (now 2 pieces, Gord)")
	if got := grantLines(t, sim.SelfSlot, []sim.ScriptInstant{addSword(), addSword()}); !reflect.DeepEqual(got, want) {
		t.Fatalf("the grant posted %+v, want %+v", got, want)
	}
	if got := grantLines(t, 3, []sim.ScriptInstant{addSword(), addSword()}); len(got) != 0 {
		t.Fatalf("a grant to another player's hero posted %+v, want nothing", got)
	}
}

// A script money grant states the received line with no name: the gold goes
// to the shared purse (DIV-3767). The control names another player's purse
// and posts nothing.
func TestScriptMoneyGrantStatesTheReceivedGoldLine(t *testing.T) {
	give := func(player uint32) sim.ScriptInstant {
		return sim.ScriptInstant{Op: sim.ScriptInstantGiveMoney, Args: [10]int32{500}, Player: player, HasPlayer: true}
	}
	if got, want := grantLines(t, sim.SelfSlot, []sim.ScriptInstant{give(sim.SelfSlot)}), announced("Received 500 gold"); !reflect.DeepEqual(got, want) {
		t.Fatalf("the money grant posted %+v, want %+v", got, want)
	}
	if got := grantLines(t, sim.SelfSlot, []sim.ScriptInstant{give(3)}); len(got) != 0 {
		t.Fatalf("a grant to another player's purse posted %+v, want nothing", got)
	}
}

// A sack take by a hired unit names each line's own holder: the Sword the
// unit took names it, the quest document that goes to the primary hero names
// the hero, and the gold line names the taker (DIV-3765). The control is the
// hero's own take, which names the hero on every line.
func TestSackTakeNamesEachLinesOwnHolder(t *testing.T) {
	take := func(taker sim.EntityID) []ui.MessageLine {
		hero, unit := sim.Entity{ID: 7, X: 5, Y: 5, Owner: sim.SelfSlot, HP: 20, MaxHP: 20}, sim.Entity{ID: 8, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 20, MaxHP: 20}
		if taker == 7 {
			hero.X, hero.Y, unit.X, unit.Y = unit.X, unit.Y, hero.X, hero.Y
		}
		w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
			[]sim.Entity{hero, unit}, nil, sim.Relations{},
			[]sim.Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{eqSwordCode, uint16(data.QuestDocumentCode)}}})
		if err != nil {
			t.Fatal(err)
		}
		mw := namedGrabWorld(t, w, []sim.EntityID{7, 8}, []string{"Gord", "Ann"})
		mw.takeSackFor(taker)
		return mw.view.MessageLines()
	}
	doc := itemName(data.QuestDocumentCode, eqDefsTable(t))
	if got, want := take(8), announced("Picked up 40 gold (Ann)", "Picked up Sword (Ann)", "Picked up "+doc+" (Gord)"); !reflect.DeepEqual(got, want) {
		t.Fatalf("the unit's take posted %+v, want %+v", got, want)
	}
	if got, want := take(7), announced("Picked up 40 gold (Gord)", "Picked up Sword (Gord)", "Picked up "+doc+" (Gord)"); !reflect.DeepEqual(got, want) {
		t.Fatalf("the hero's take posted %+v, want %+v", got, want)
	}
}
