package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cheatReleaseApp(t *testing.T, mage bool, chicken ...bool) (*FrontEnd, *ui.App, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.SetTipsOff(true)
	if len(chicken) != 0 {
		f.SetChickenAtMissionStart(chicken[0])
	}
	a := f.App("cheat witness")
	a.SetCutscenes(nil)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(10, helpWitnessParty(mage))); err != nil {
		t.Fatal(err)
	}
	helpCloseNotices(t, f, a)
	f.live.stopped = true
	return f, a, f.live.mission.ids[0]
}

func cheatChat(t *testing.T, a *ui.App, command string) {
	t.Helper()
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessType(command, false); err != nil {
		t.Fatal(err)
	}
	if line, open := a.HeadlessChatState(); !open || line != command {
		t.Fatalf("chat draft=(%q,%v), want (%q,true)", line, open, command)
	}
	if pic, err := a.HeadlessChatFrame(); err != nil || pic == nil {
		t.Fatalf("installed chat frame: %v", err)
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if _, open := a.HeadlessChatState(); open {
		t.Fatal("chat submission retained edit control")
	}
}

func cheatNoticeWant(t *testing.T, f *FrontEnd, code int) {
	t.Helper()
	row := 221 + 2*(code-5)
	name := EncodeInstallText(f.live.cheatPlayerName(sim.SelfSlot), f.live.view.TextSelector())
	want := rawTextLine(t, f, MainTextPath, row) + name + rawTextLine(t, f, MainTextPath, row+1)
	lines := f.live.view.MessageLines()
	if len(lines) == 0 || lines[len(lines)-1].Text != want {
		t.Fatalf("notice %d = %v, want installed %q", code, lines, want)
	}
}

func cheatEnable(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	cheatChat(t, a, "#Chicken")
	if f.live.cheats.privilege[sim.SelfSlot] != 255 {
		t.Fatal("Chicken did not set the privilege byte to 255")
	}
	cheatNoticeWant(t, f, 5)
}

func cheatCold(t *testing.T, f *FrontEnd) (*FrontEnd, sim.EntityID) {
	t.Helper()
	cold, _ := cheatColdApp(t, f)
	return cold, cold.live.mission.ids[0]
}

func cheatColdApp(t *testing.T, f *FrontEnd) (*FrontEnd, *ui.App) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	cold.SetTipsOff(true)
	_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("ordinary cheat LOAD: town=%t error=%v", town, err)
	}
	a := cold.App("cheat reload witness")
	a.SetCutscenes(nil)
	a.Layout(1024, 768)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	cold.live.stopped = true
	helpCloseNotices(t, cold, a)
	if cold.live.cheats.privilege[sim.SelfSlot] != 0 {
		t.Fatal("ordinary cold LOAD restored unsaved privilege")
	}
	return cold, a
}

func cheatMagicItem(t *testing.T, f *FrontEnd) (string, uint16) {
	t.Helper()
	if f.Table.MagicItems != nil {
		for row := 1; row < f.Table.MagicItems.Len() && row <= 255; row++ {
			name := f.Table.MagicItems.EntryName(row)
			if name == "" || strings.ContainsAny(name, "\r\n") || len(name)+10 > 256 {
				continue
			}
			item := mapload.ItemInstanceFromCode(uint16(0x0e00|row), f.Table)
			if item.ValidateWeight() == nil {
				return name, item.Code
			}
		}
	}
	t.Fatal("installed magic-item table has no bounded valid exact name")
	return "", 0
}

func cheatItemCount(t *testing.T, w *sim.World, id sim.EntityID, code uint16) int {
	t.Helper()
	items, ok := w.Carried(id)
	if !ok {
		t.Fatal("hero has no carried inventory")
	}
	n := 0
	for _, item := range items {
		if item == code {
			n++
		}
	}
	return n
}

func TestReleaseCheatCreateUsesAppAndOrdinarySAV(t *testing.T) {
	for _, kind := range []string{"gold", "item", "decayingHero"} {
		t.Run(kind, func(t *testing.T) {
			f, a, id := cheatReleaseApp(t, false)
			purse := f.live.world.Purse(sim.SelfSlot)
			cheatChat(t, a, "#create 37 Gold")
			if f.live.world.Purse(sim.SelfSlot) != purse {
				t.Fatal("create bypassed the privilege gate")
			}
			cheatNoticeWant(t, f, 6)
			cheatEnable(t, f, a)
			if kind == "decayingHero" {
				f.live.world.CheatKillPlayer(sim.SelfSlot)
				sim.Step(f.live.world, nil)
				fallen, present := f.live.world.Entity(id)
				if !present || fallen.Decay == 0 {
					t.Fatal("ordinary death did not retain a decaying hero control")
				}
				cheatChat(t, a, "#create 37 Gold")
				cheatNoticeWant(t, f, 6)
				if f.live.world.Purse(sim.SelfSlot) != purse {
					t.Fatal("create credited the purse of a decaying hero")
				}
				return
			}
			if kind == "gold" {
				cheatChat(t, a, "#create 37 Gold")
				cheatNoticeWant(t, f, 7)
				if got := f.live.world.Purse(sim.SelfSlot); got != purse+37 {
					t.Fatalf("create gold=%d, want %d", got, purse+37)
				}
				store := SaveStore{Dir: t.TempDir()}
				name, _ := deadPatrolSave(t, f, store)
				cold := deadPatrolLoad(t, store, name)
				if got := cold.live.world.Purse(sim.SelfSlot); got != purse+37 {
					t.Fatalf("cold gold=%d, want %d", got, purse+37)
				}
			} else {
				name, code := cheatMagicItem(t, f)
				before := cheatItemCount(t, f.live.world, id, code)
				cheatChat(t, a, "#create 2 "+name)
				cheatNoticeWant(t, f, 7)
				if got := cheatItemCount(t, f.live.world, id, code); got != before+2 {
					t.Fatalf("created item count=%d, want %d", got, before+2)
				}
				cold, coldID := cheatCold(t, f)
				if got := cheatItemCount(t, cold.live.world, coldID, code); got != before+2 {
					t.Fatalf("cold item count=%d, want %d", got, before+2)
				}
			}
		})
	}
}

func TestReleaseCheatCreateLocalizedAliasUsesAppAndOrdinarySAV(t *testing.T) {
	f, a, id := cheatReleaseApp(t, false)
	ru := f.live.view.TextSelector() == text.SelectorConverting
	name := ""
	code := uint16(0)
	for candidate := 1; candidate <= 65535; candidate++ {
		raw, found := f.Table.Names.NameFor(data.ItemCode(candidate))
		if !found || raw == "" || strings.ContainsAny(raw, "\r\n") || ru && utf8.ValidString(raw) {
			continue
		}
		item, supported := mapload.CheatItem(raw, f.Table)
		if !supported || item.Code != uint16(candidate) || item.ValidateWeight() != nil {
			continue
		}
		decoded := raw
		if ru {
			var err error
			decoded, err = charmap.CodePage866.NewDecoder().String(raw)
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(decoded)+10 <= 256 {
			name, code = decoded, uint16(candidate)
			break
		}
	}
	if name == "" {
		t.Fatal("installed item aliases have no bounded supported name in the required alphabet")
	}
	cheatEnable(t, f, a)
	before := cheatItemCount(t, f.live.world, id, code)
	cheatChat(t, a, "#create 2 "+name)
	cheatNoticeWant(t, f, 7)
	if got := cheatItemCount(t, f.live.world, id, code); got != before+2 {
		t.Fatalf("typed installed alias %q created %d units of %#x, want %d", name, got, code, before+2)
	}
	cold, coldID := cheatCold(t, f)
	if got := cheatItemCount(t, cold.live.world, coldID, code); got != before+2 {
		t.Fatalf("cold installed alias %q has %d units of %#x, want %d", name, got, code, before+2)
	}
}

func cheatGodWords(t *testing.T, e sim.Entity) {
	t.Helper()
	modifier := e.SourceNow().Modifier
	if e.ActorLoad.Source.Class == 0 {
		modifier = e.NativeBasis.Modifier
	}
	cheatGodModifier(t, modifier[:])
}

func cheatGodModifier(t *testing.T, modifier []byte) {
	t.Helper()
	if len(modifier) != 64 {
		t.Fatalf("modifier bytes=%d, want 64", len(modifier))
	}
	for i := 0; i < 6; i++ {
		if word := binary.LittleEndian.Uint16(modifier[46+2*i:]); word != 100 || modifier[58+i] != 100 {
			t.Fatalf("god slot %d protection=%d damage-kind=%d", i, word, modifier[58+i])
		}
	}
}

func TestReleaseCheatModifyUsesAppAndOrdinarySAV(t *testing.T) {
	for _, command := range []string{"#modify self +god", "#modify army +god", "#modify self +spell 2", "#modify self +spells", "#modify army +spell 2", "#modify army +spells"} {
		t.Run(command, func(t *testing.T) {
			f, a, id := cheatReleaseApp(t, true)
			before := releaseEntity(t, f.live, id)
			cheatChat(t, a, command)
			cheatNoticeWant(t, f, 7)
			if f.live.cheats.privilege[sim.SelfSlot] != 0 {
				t.Fatal("modify manufactured privilege")
			}
			live := releaseEntity(t, f.live, id)
			if strings.HasSuffix(command, "+god") {
				cheatGodWords(t, live)
				raw, doc, actions := saveCurrentEffect(t, f)
				object := uint16(0)
				for _, row := range actions.Bindings {
					if row.ID == id && !row.Structure && !row.Missing {
						object = row.Object
					}
				}
				if object == 0 || int(object) > len(doc.Objects) {
					t.Fatal("god hero has no ordinary SAV actor binding")
				}
				block, err := savedActorRaw(&doc.Objects[object-1], "UD4", 64)
				if err != nil {
					t.Fatal(err)
				}
				cheatGodModifier(t, block)
				cold, coldID := coldMission(t, raw)
				loaded := releaseEntity(t, cold.live, coldID)
				cheatGodWords(t, loaded)
			} else {
				cold, coldID := cheatCold(t, f)
				loaded := releaseEntity(t, cold.live, coldID)
				want := before.KnownSpells | 1<<2
				if strings.HasSuffix(command, "+spells") {
					want = before.KnownSpells | (1<<29 - 2)
				}
				if strings.HasPrefix(command, "#modify army ") {
					want = before.KnownSpells
				}
				if live.KnownSpells != want || loaded.KnownSpells != want {
					t.Fatalf("spell membership live=%#x cold=%#x want=%#x", live.KnownSpells, loaded.KnownSpells, want)
				}
			}
		})
	}
}

func cheatSummonName(t *testing.T, f *FrontEnd, hero bool) (string, sim.Entity) {
	t.Helper()
	collection := f.Table.Units
	if hero {
		collection = f.Table.Humans
	}
	if collection != nil {
		for row := 1; row < collection.Len() && row <= 255; row++ {
			name := collection.EntryName(row)
			if name == "" || strings.ContainsAny(name, "\r\n") || len(name)+15 > 256 {
				continue
			}
			e, _, _, err := mapload.CheatActor(name, hero, f.Table, mapload.DifficultyNormal)
			if err == nil && e.MaxHP > 0 && (!hero || e.Humanoid) {
				return name, e
			}
		}
	}
	t.Fatal("installed table contains no bounded constructible exact actor name")
	return "", sim.Entity{}
}

func cheatNearbyMovePoint(t *testing.T, f *FrontEnd, a *ui.App, actor sim.Entity) (int, int) {
	t.Helper()
	raw, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var destination sim.CellPoint
	found := false
	for _, offset := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
		var scratch sim.World
		if err := scratch.UnmarshalBinary(raw); err != nil {
			t.Fatal(err)
		}
		mapload.BindSourceDerive(&scratch)
		at := sim.CellPoint{X: actor.X + offset[0], Y: actor.Y + offset[1]}
		sim.Step(&scratch, []sim.Command{sim.GroupMoveTo(actor.ID, at, 1)})
		for range 31 {
			sim.Step(&scratch, nil)
		}
		moved, present := scratch.Entity(actor.ID)
		if present && (moved.X != actor.X || moved.Y != actor.Y) {
			destination, found = at, true
			break
		}
		t.Logf("neighbor (%d,%d): position=(%d,%d) HP=%d state=%d target=%t (%d,%d) pending=%+v route=%v", at.X, at.Y, moved.X, moved.Y, moved.HP, moved.ActorState, moved.HasTarget, moved.TargetX, moved.TargetY, moved.PendingOrder, scratch.Route(actor.ID))
	}
	if !found {
		t.Fatalf("cold summoned actor did not move toward any of eight neighboring cells within 32 ticks: actor=%+v", actor)
	}
	before := f.live.world.Hash()
	inspectionCentre(f.live, int(destination.X), int(destination.Y))
	for y := 100; y < 560; y += 4 {
		for x := 160; x < 740; x += 4 {
			cx, cy, err := a.HeadlessDropCell(x, y)
			if err != nil || int32(cx) != destination.X || int32(cy) != destination.Y {
				continue
			}
			if err := a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			if cursor, present := a.HeadlessMapCursor(); !present || cursor != "move" {
				continue
			}
			if f.live.world.Hash() != before {
				t.Fatal("move-point search changed the stopped cold world")
			}
			t.Logf("cold summoned move target=(%d,%d), source=(%d,%d)", cx, cy, actor.X, actor.Y)
			return x, y
		}
	}
	t.Fatalf("no ordinary move cursor for reachable neighboring cell (%d,%d)", destination.X, destination.Y)
	return 0, 0
}

func TestReleaseCheatSummonUsesAppAndOrdinarySAV(t *testing.T) {
	for _, hero := range []bool{false, true} {
		t.Run(strconv.FormatBool(hero), func(t *testing.T) {
			f, a, originalID := cheatReleaseApp(t, false)
			original, originalKnown := f.live.chars[originalID]
			if !originalKnown || !original.Known || original.Band != ui.CharacterBandPerson {
				t.Fatalf("original hero character unavailable: id=%d character=%+v", originalID, original)
			}
			name, template := cheatSummonName(t, f, hero)
			cheatEnable(t, f, a)
			before := f.live.world.Entities()
			command := "#summon 1 " + name
			if hero {
				command = "#summon hero " + name
			}
			cheatChat(t, a, command)
			entities := f.live.world.Entities()
			if len(entities) != len(before)+1 {
				t.Fatalf("summon actor count=%d, want %d; failure=%v", len(entities), len(before)+1, f.live.cheats.failure)
			}
			spawn := entities[len(entities)-1]
			if spawn.Owner != sim.SelfSlot || spawn.TypeID != template.TypeID || spawn.Humanoid != template.Humanoid || spawn.OffMap {
				t.Fatalf("summon current actor=%+v, template type=%d", spawn, template.TypeID)
			}
			if err := a.HeadlessSelectEntity(uint32(spawn.ID)); err != nil {
				t.Fatal(err)
			}
			if panel, ok := f.live.view.InspectionPanel(); !ok || panel.ID != uint32(spawn.ID) || panel.Name != name {
				t.Fatalf("live summoned actor card=%+v, want name %q", panel, name)
			}
			livePicture := f.live.inspectionUnitPicture(uint32(spawn.ID))
			if spawn.Humanoid {
				typ, face := spawn.SourceBinding.TypeID, spawn.SourceBinding.Face
				mage, female, faceIndex := typ == 0x17 || typ == 0x18, face&0x80 != 0, int(face&0x7f)
				if typ >= 0x20 && typ < 0x40 {
					axes := typ - 0x21
					mage, female, faceIndex = axes&2 != 0, axes&1 != 0, int(face)
				}
				figure := figureID{Dir: data.FigureDirFor(mage, female), Face: faceIndex,
					Horse: data.FigureHasHorse(spawn.TypeID), Hero: data.FigureIsHero(spawn.TypeID)}
				if got, ok := f.live.figures[spawn.ID]; !ok || got != figure {
					t.Fatalf("live summoned Human figure=%+v, found=%t, want source figure=%+v", got, ok, figure)
				}
				want, _ := composeUnitFigure(f.live.archive(), f.live.equipmentOf(spawn.ID), figure)
				if livePicture == nil || want == nil || !imagesEqual(livePicture, want) {
					t.Fatal("live summoned Human lost its source/equipment inspection picture")
				}
				frame, statistics, err := a.HeadlessCharacterPane()
				if err != nil || statistics || frame == nil {
					t.Fatalf("live summoned Human figure pane: statistics=%t error=%v", statistics, err)
				}
				checkInspectionFigure(t, frame, want, "live summoned Human")
			}
			cold, coldApp := cheatColdApp(t, f)
			coldOriginalID := cold.live.mission.ids[0]
			coldOriginal, coldOriginalKnown := cold.live.chars[coldOriginalID]
			if !coldOriginalKnown || coldOriginal.Known != original.Known || coldOriginal.Band != original.Band || coldOriginal.Mage != original.Mage || coldOriginal.Name != original.Name {
				t.Fatalf("cold LOAD changed original hero presentation: id=%d character=%+v, live id=%d character=%+v", coldOriginalID, coldOriginal, originalID, original)
			}
			found := false
			var loaded sim.Entity
			for _, e := range cold.live.world.Entities() {
				if e.SourceBinding.Identity == spawn.SourceBinding.Identity {
					found = true
					loaded = e
					if e.Owner != spawn.Owner || e.TypeID != spawn.TypeID || e.HP != spawn.HP || e.X != spawn.X || e.Y != spawn.Y {
						t.Fatalf("cold summoned actor=%+v, live=%+v", e, spawn)
					}
					if e.MaxHP != spawn.MaxHP || e.Mana != spawn.Mana || e.MaxMana != spawn.MaxMana || e.Reaction != spawn.Reaction || e.Mind != spawn.Mind || e.Spirit != spawn.Spirit || e.ToHit != spawn.ToHit || e.Defence != spawn.Defence || e.Absorption != spawn.Absorption || e.DamageBase != spawn.DamageBase || e.DamageSpread != spawn.DamageSpread || e.Protection != spawn.Protection || e.Resistance != spawn.Resistance || e.KnownSpells != spawn.KnownSpells {
						t.Fatalf("cold summoned attributes=%+v, live=%+v", e, spawn)
					}
					if e.Speed != spawn.Speed || e.RotationSpeed != spawn.RotationSpeed || e.Domain != spawn.Domain || e.TokenSize != spawn.TokenSize || e.HumanMovement != spawn.HumanMovement || e.Load != spawn.Load || e.Capacity != spawn.Capacity {
						t.Fatalf("cold summoned mover basis=%+v, live=%+v", e, spawn)
					}
				}
			}
			if !found {
				t.Fatal("cold LOAD omitted the summoned actor's ordinary current record")
			}
			if cold.live.actorNames[loaded.ID] != name || !cold.live.chars[loaded.ID].Known || cold.live.chars[loaded.ID].Name != name {
				t.Fatal("cold LOAD lost summoned actor name or character metadata")
			}
			if order, _, found := cold.live.world.FrozenGroupAI(loaded.Owner, loaded.Group); !found || order != 0 {
				t.Fatalf("cold summoned guard group: found=%t order=%d group=%d", found, order, loaded.Group)
			}
			if err := coldApp.HeadlessSelectEntity(uint32(loaded.ID)); err != nil {
				t.Fatal(err)
			}
			if panel, ok := cold.live.view.InspectionPanel(); !ok || panel.ID != uint32(loaded.ID) || panel.Name != name {
				t.Fatalf("cold summoned actor card=%+v, want name %q", panel, name)
			}
			if loaded.Humanoid {
				picture := cold.live.inspectionUnitPicture(uint32(loaded.ID))
				if picture == nil || !imagesEqual(picture, livePicture) || cold.live.figures[loaded.ID] != f.live.figures[spawn.ID] {
					t.Fatal("cold LOAD changed the summoned Human source/equipment picture")
				}
				frame, statistics, err := coldApp.HeadlessCharacterPane()
				if err != nil || statistics || frame == nil {
					t.Fatalf("cold summoned Human figure pane: statistics=%t error=%v", statistics, err)
				}
				checkInspectionFigure(t, frame, livePicture, "cold summoned Human")
			}
			x, y := cheatNearbyMovePoint(t, cold, coldApp, loaded)
			for _, edge := range []string{"press", "release"} {
				if err := coldApp.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if len(cold.live.pending) != 1 || cold.live.pending[0].Kind != sim.KindGroupMoveTo || cold.live.pending[0].Entity != loaded.ID {
				t.Fatalf("cold summoned actor did not accept App move: %v", cold.live.pending)
			}
			queued := cold.live.pending[0]
			cold.live.stopped = false
			for range 32 {
				cold.live.tick()
			}
			moved, present := cold.live.world.Entity(loaded.ID)
			if !present || moved.X == loaded.X && moved.Y == loaded.Y {
				speed, _ := cold.live.world.RateSpeed(loaded.ID)
				fineX, fineY, _ := cold.live.world.ActorFinePosition(loaded.ID)
				t.Fatalf("cold summoned actor did not move: name=%q tick=%d order=%+v speed=%d fine=(%d,%d) route=%v actor=%+v", name, cold.live.world.Tick(), queued, speed, fineX, fineY, cold.live.world.Route(loaded.ID), moved)
			}
		})
	}
}

func cheatForeignOwner(t *testing.T, f *FrontEnd) uint32 {
	t.Helper()
	for _, e := range f.live.world.Entities() {
		if e.Owner > sim.SelfSlot && e.Owner < 50 && e.Alive() {
			return e.Owner
		}
	}
	t.Fatal("mission has no live foreign actor")
	return 0
}

func cheatHPByIdentity(w *sim.World) map[uint32]int32 {
	out := make(map[uint32]int32)
	for _, e := range w.Entities() {
		out[e.SourceBinding.Identity] = e.HP
	}
	return out
}

func TestReleaseCheatKillsUseAppAndOrdinarySAV(t *testing.T) {
	for _, kind := range []string{"named", "all", "all alias", "cheaters"} {
		t.Run(kind, func(t *testing.T) {
			f, a, id := cheatReleaseApp(t, false)
			owner := cheatForeignOwner(t, f)
			cheatEnable(t, f, a)
			before := f.live.world.Entities()
			command := "#kill " + f.live.cheatPlayerName(owner)
			switch kind {
			case "all":
				command = "#killall"
			case "all alias":
				command = "#kill all"
			case "cheaters":
				command = "#kill cheaters"
				f.live.cheats.privilege[owner] = 255
			}
			cheatChat(t, a, command)
			changed := 0
			for _, old := range before {
				want := old.HP
				kill := old.Owner == owner
				if kind == "all" || kind == "all alias" {
					kill = f.live.world.Relations().Hostile(old.Owner, sim.SelfSlot)
				}
				if kill {
					want = -50
					changed++
				}
				if e := releaseEntity(t, f.live, old.ID); e.HP != want {
					t.Fatalf("kill actor %d owner %d HP=%d, want %d", old.ID, old.Owner, e.HP, want)
				}
			}
			if changed == 0 {
				t.Fatal("kill witness selected no affected actor")
			}
			if kind == "cheaters" && (f.live.cheats.privilege[owner] != 0 || f.live.cheats.privilege[sim.SelfSlot] != 255) {
				t.Fatal("kill cheaters failed demotion or demoted the caller")
			}
			if releaseEntity(t, f.live, id).HP == -50 {
				t.Fatal("kill witness killed its own single-player hero")
			}
			liveHP := cheatHPByIdentity(f.live.world)
			cold, _ := cheatCold(t, f)
			if got := cheatHPByIdentity(cold.live.world); !reflect.DeepEqual(got, liveHP) {
				t.Fatalf("cold kill HP differs: live=%v cold=%v", liveHP, got)
			}
		})
	}
}

func TestReleaseCheatPickupAllUsesAppAndOrdinarySAV(t *testing.T) {
	f, a, id := cheatReleaseApp(t, false)
	cheatEnable(t, f, a)
	hero := releaseEntity(t, f.live, id)
	if !f.live.world.SetPurse(sim.SelfSlot, 3000) {
		t.Fatal("setup purse refused")
	}
	pack, _ := f.live.world.CarriedStacks(id)
	if len(pack) == 0 {
		for _, sack := range f.live.world.Sacks() {
			if len(sack.Items) != 0 || len(sack.ItemInstances) != 0 {
				if err := f.live.world.TakeSack(id, sack.X, sack.Y); err != nil {
					t.Fatal(err)
				}
				pack, _ = f.live.world.CarriedStacks(id)
				break
			}
		}
	}
	dropSlot := -1
	for i, stack := range pack {
		if data.ItemCode(stack.Code) != data.QuestDocumentCode {
			dropSlot = i
			break
		}
	}
	if dropSlot < 0 {
		t.Fatal("ordinary pickup setup has no installed droppable carried item")
	}
	sim.Step(f.live.world, []sim.Command{sim.DropCarried(id, sim.ItemSlot(dropSlot), sim.CellPoint{X: hero.X, Y: hero.Y})})
	sim.Step(f.live.world, []sim.Command{sim.DropGold(sim.SelfSlot, 500, sim.CellPoint{X: hero.X, Y: hero.Y})})
	f.live.push()
	sacks := f.live.world.Sacks()
	if len(sacks) == 0 {
		t.Fatal("ordinary gold drop produced no Sack")
	}
	before := f.live.world.Purse(sim.SelfSlot)
	want := before
	wantItems := map[uint16]int{}
	items, _ := f.live.world.Carried(id)
	for _, code := range items {
		wantItems[code]++
	}
	droppedItems := 0
	for _, sack := range sacks {
		want += sack.Gold
		if len(sack.ItemInstances) != 0 {
			for _, item := range sack.ItemInstances {
				wantItems[item.Code]++
				droppedItems++
			}
		} else {
			for _, code := range sack.Items {
				wantItems[code]++
				droppedItems++
			}
		}
	}
	if droppedItems == 0 {
		t.Fatal("ordinary item drop produced no item Sack payload")
	}
	cheatChat(t, a, "#pickup all")
	if len(f.live.world.Sacks()) != 0 || f.live.world.Purse(sim.SelfSlot) != want {
		t.Fatalf("pickup sacks=%v purse=%d, want empty/%d", f.live.world.Sacks(), f.live.world.Purse(sim.SelfSlot), want)
	}
	lines := f.live.view.MessageLines()
	if len(lines) == 0 || lines[len(lines)-1].Text != "All sacks picked up" {
		t.Fatal("pickup omitted the original frame log line")
	}
	cold, coldID := cheatCold(t, f)
	if len(cold.live.world.Sacks()) != 0 || cold.live.world.Purse(sim.SelfSlot) != want {
		t.Fatal("cold LOAD lost Sack removal or purse credits")
	}
	for _, row := range []struct {
		world *sim.World
		id    sim.EntityID
	}{{f.live.world, id}, {cold.live.world, coldID}} {
		got := map[uint16]int{}
		items, _ := row.world.Carried(row.id)
		for _, code := range items {
			got[code]++
		}
		if !reflect.DeepEqual(got, wantItems) {
			t.Fatalf("pickup carried units=%v, want all Sack items %v", got, wantItems)
		}
	}
}

func TestReleaseCheatClientCommandsAndChickenColdReset(t *testing.T) {
	f, a, _ := cheatReleaseApp(t, false)
	hash := f.live.world.Hash()
	cheatChat(t, a, "#chicken")
	if f.live.cheats.privilege[sim.SelfSlot] != 0 {
		t.Fatal("case-insensitive Chicken command was accepted")
	}
	cheatChat(t, a, "#Chicken suffix")
	cheatNoticeWant(t, f, 5)
	if f.live.cheats.privilege[sim.SelfSlot] != 255 || f.live.world.Hash() != hash {
		t.Fatal("Chicken prefix failed or changed saved world state")
	}
	cheatChat(t, a, "#show map")
	cheatNoticeWant(t, f, 7)
	if !f.live.view.FogRevealed() || f.live.world.Hash() != hash {
		t.Fatal("show map failed reveal or modified ordinary saved fog")
	}
	cheatChat(t, a, "#hide map")
	cheatNoticeWant(t, f, 7)
	if f.live.view.FogRevealed() || f.live.world.Hash() != hash {
		t.Fatal("hide map kept reveal or modified ordinary saved fog")
	}
	cheatChat(t, a, "#modify self +knowledge")
	if !f.live.cheats.knowledge || f.live.world.Hash() != hash {
		t.Fatal("knowledge resend did not activate client knowledge or changed Diary state")
	}
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	raw, err := store.Read(name)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	savedFog, present, err := file.Fog()
	if err != nil || !present || savedFog.FirstState != 0x8000 || len(savedFog.Runs) != 1 || int(savedFog.Runs[0]) != len(f.live.fog.explored) {
		t.Fatalf("show-map ordinary Fog record=%+v present=%t error=%v", savedFog, present, err)
	}
	cold := deadPatrolLoad(t, store, name)
	if cold.live.cheats.privilege[sim.SelfSlot] != 0 || cold.live.cheats.knowledge || cold.live.view.FogRevealed() || cold.live.cheats.showMap {
		t.Fatal("cold LOAD retained unsaved client cheat state")
	}
	if len(cold.live.fog.explored) != len(f.live.fog.explored) || slices.Contains(cold.live.fog.explored, byte(0)) {
		t.Fatal("ordinary cold LOAD lost cheat-revealed exploration")
	}
	for _, kind := range []string{"event", "victory"} {
		t.Run(kind, func(t *testing.T) {
			f, a, _ := cheatReleaseApp(t, false)
			if kind == "event" {
				event := -1
				for n := 0; n < 256; n++ {
					if _, ok := ReadEventTextFor(f.live.mission.src, f.Table.Game, 10, n); ok {
						event = n
						break
					}
				}
				if event < 0 {
					t.Fatal("mission 10 has no installed event text in 0..255")
				}
				cheatChat(t, a, fmt.Sprintf("#event %d", event))
				if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeDialogue || f.live.cheats.privilege[sim.SelfSlot] != 0 {
					t.Fatal("event command failed unprivileged installed dialogue")
				}
			} else {
				cheatEnable(t, f, a)
				cheatChat(t, a, "#victory")
				if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
					t.Fatal("victory command did not present the success panel")
				}
			}
			if pic, err := a.HeadlessNoticeFrame(); err != nil || pic == nil {
				t.Fatalf("installed command panel: %v", err)
			}
		})
	}
}

func TestReleaseCheatAltConsoleThroughApp(t *testing.T) {
	f, a, _ := cheatReleaseApp(t, false)
	if err := a.HeadlessKey("alt-d"); err != nil {
		t.Fatal(err)
	}
	if f.live.cheats.turnTrace {
		t.Fatal("Alt+D bypassed privilege")
	}
	cheatEnable(t, f, a)
	for _, letter := range []byte{'D', 'H', 'I', 'Q', 'T', 'U'} {
		t.Run(string(letter), func(t *testing.T) {
			f.live.view.ClearMessages()
			hash := f.live.world.Hash()
			if err := a.HeadlessKey("alt-" + strings.ToLower(string(letter))); err != nil {
				t.Fatal(err)
			}
			lines := f.live.view.MessageLines()
			if len(lines) == 0 {
				t.Fatalf("Alt+%c produced no console reply", letter)
			}
			switch letter {
			case 'D':
				if !f.live.cheats.turnTrace || lines[0].Text != "Turn tracing turned on." {
					t.Fatal("Alt+D did not toggle turn tracing and report its new state")
				}
			case 'T':
				if !f.live.cheats.scriptTrace || lines[0].Text != "Script tracing turned on." {
					t.Fatal("Alt+T did not toggle script tracing and report its new state")
				}
			case 'Q':
				if !f.live.cheats.safe || lines[0].Text != "Safe mode turned on." || f.live.world.Hash() == hash {
					t.Fatal("Alt+Q did not toggle deterministic AI admission state")
				}
			case 'H':
				prefixes := []string{"<Alt-h>", "<Alt-q>", "<Alt-t>", "<Alt-i>", "<Alt-d>", "<Alt-u>"}
				if len(lines) != len(prefixes) {
					t.Fatalf("console help has %d lines, want %d", len(lines), len(prefixes))
				}
				for i, prefix := range prefixes {
					if !strings.HasPrefix(lines[i].Text, prefix) {
						t.Fatalf("help line %d=%q, want prefix %q", i, lines[i].Text, prefix)
					}
				}
			case 'I':
				if lines[0].Text != "Last Turn Statistics:" || lines[len(lines)-1].Text != "Average Turn Statistics:" {
					t.Fatal("Alt+I omitted the two original statistics block labels")
				}
			case 'U':
				if lines[0].Text != "Mission units stats:" || len(lines) < 2 {
					t.Fatal("Alt+U omitted the original mission-unit label or group counts")
				}
			}
			if letter != 'Q' && f.live.world.Hash() != hash {
				t.Fatalf("readout/trace Alt+%c changed canonical world state", letter)
			}
		})
	}
	for letter := byte('B'); letter <= 'Y'; letter++ {
		if slices.Contains([]byte{'D', 'H', 'I', 'Q', 'T', 'U', 'S'}, letter) {
			continue
		}
		t.Run("inert "+string(letter), func(t *testing.T) {
			f.live.view.ClearMessages()
			before := f.live.cheats
			hash := f.live.world.Hash()
			if err := a.HeadlessKey("alt-" + strings.ToLower(string(letter))); err != nil {
				t.Fatal(err)
			}
			if f.live.cheats != before || f.live.world.Hash() != hash || len(f.live.view.MessageLines()) != 0 {
				t.Fatalf("decoded inert Alt+%c acted", letter)
			}
		})
	}
}

func TestReleaseCheatAltConsoleThroughDropGold(t *testing.T) {
	f, a, id := cheatReleaseApp(t, false)
	cheatEnable(t, f, a)
	f.live.world.SetPurse(sim.SelfSlot, 2500)
	f.live.push()
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	if !f.live.view.SaveApplication().InventoryOpen {
		if err := a.HeadlessKey("i"); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	x, y, err := a.HeadlessPackCellPoint(purseCell(t, f))
	if err != nil {
		t.Fatal(err)
	}
	pursePointer(t, a, x, y, "press", "release", "press", "release")
	gold, open := a.HeadlessGold()
	if !open || !gold.Open || gold.Text != "0" {
		t.Fatalf("setup: Drop Gold editor=%+v, open=%t", gold, open)
	}
	f.live.cheats.privilege[sim.SelfSlot] = 0
	f.live.view.ClearMessages()
	before := f.live.world.Hash()
	for _, letter := range []string{"h", "d", "q"} {
		if err := a.HeadlessKey("alt-" + letter); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.cheats.turnTrace || f.live.cheats.safe || f.live.world.Hash() != before || len(f.live.view.MessageLines()) != 0 {
		t.Fatal("Drop Gold console bypassed privilege")
	}
	f.live.cheats.privilege[sim.SelfSlot] = 255
	for _, letter := range []string{"h", "d", "q"} {
		f.live.view.ClearMessages()
		if err := a.HeadlessKey("alt-" + letter); err != nil {
			t.Fatal(err)
		}
		lines := f.live.view.MessageLines()
		switch letter {
		case "h":
			if len(lines) != 6 || lines[0].Text != "<Alt-h> Help" || lines[5].Text != "<Alt-u> Mission units stats" {
				t.Fatal("Drop Gold swallowed console help")
			}
		case "d":
			if !f.live.cheats.turnTrace || len(lines) != 1 || lines[0].Text != "Turn tracing turned on." {
				t.Fatal("Drop Gold swallowed turn tracing")
			}
		case "q":
			if !f.live.cheats.safe || len(lines) != 1 || lines[0].Text != "Safe mode turned on." || f.live.world.Hash() == before {
				t.Fatal("Drop Gold swallowed deterministic Safe mode")
			}
		}
		current, open := a.HeadlessGold()
		if !open || !current.Open || current.Text != gold.Text || current.Caret != gold.Caret || f.live.world.Purse(sim.SelfSlot) != 2500 || len(f.live.pending) != 0 {
			t.Fatalf("Alt+%s changed the Drop Gold draft, purse or order queue", letter)
		}
	}
}

func TestReleaseCheatKnowledgeWithoutPrivilegeKeepsOrdinaryCounts(t *testing.T) {
	f, a, _ := cheatReleaseApp(t, false)
	before := f.live.world.SavedDiaries()
	hash := f.live.world.Hash()
	cheatChat(t, a, "#modify self +knowledge")
	cheatNoticeWant(t, f, 7)
	if f.live.cheats.privilege[sim.SelfSlot] != 0 || f.live.cheats.knowledge || f.live.world.Hash() != hash || !reflect.DeepEqual(f.live.world.SavedDiaries(), before) {
		t.Fatal("unprivileged knowledge resend changed the ordinary Diary counts")
	}
	for _, e := range f.live.world.Entities() {
		if got, want := f.live.cardKnowledge(e), f.live.world.KnowledgeLevel(e); got != want {
			t.Fatalf("unprivileged actor card knowledge=%d, want %d", got, want)
		}
	}
	cold, _ := cheatCold(t, f)
	if !reflect.DeepEqual(cold.live.world.SavedDiaries(), before) || cold.live.cheats.knowledge {
		t.Fatal("ordinary cold LOAD changed Diary counts after the unprivileged resend")
	}
}

func TestReleaseChickenLaunchFlagDoesNotRaiseColdLoadPrivilege(t *testing.T) {
	f, _, _ := cheatReleaseApp(t, false, true)
	if f.live.cheats.privilege[sim.SelfSlot] != 255 {
		t.Fatal("enabled launch flag did not set privilege on fresh mission start")
	}
	store := SaveStore{Dir: t.TempDir()}
	name, _ := deadPatrolSave(t, f, store)
	cold := releaseFront(t)
	cold.SetDeterministicFrames(true)
	cold.SetChickenAtMissionStart(true)
	_, _, load := cold.SaveSeams(store, OriginalStore{}, nil)
	open, town, err := load(localOriginalSaveToken(name))
	if err != nil || town {
		t.Fatalf("cold LOAD=%v town=%v", err, town)
	}
	a := cold.App("enabled flag cold LOAD")
	a.SetCutscenes(nil)
	a.Layout(1024, 768)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if !cold.ChickenAtMissionStart() || cold.live.cheats.privilege[sim.SelfSlot] != 0 {
		t.Fatal("launch flag turned a cold LOAD into a new mission's Chicken command")
	}
}

func TestReleaseCheatParticipantGateRefusesEveryChatCommand(t *testing.T) {
	commands := []string{"#create 37 Gold", "#modify self +god", "#summon hero Unknown", "#killall", "#kill all", "#kill cheaters",
		"#kill 2", "#pickup all", "#show map", "#hide map", "#victory", "#event 1", "#Chicken"}
	for _, command := range commands {
		t.Run(command, func(t *testing.T) {
			f, a, _ := cheatReleaseApp(t, false)
			f.live.mission.state.Map.Meta.Word70 = 4
			f.live.view.ClearMessages()
			before := f.live.cheats
			hash := f.live.world.Hash()
			cheatChat(t, a, command)
			if f.live.world.Hash() != hash || f.live.cheats != before || f.live.view.FogRevealed() || f.live.view.NoticeOpen() || len(f.live.view.MessageLines()) != 0 {
				t.Fatal("multi-participant map accepted the chat command or emitted a cheat notice")
			}
		})
	}
	f, a, _ := cheatReleaseApp(t, false)
	f.live.cheats.privilege[sim.SelfSlot] = 255
	f.live.mission.state.Map.Meta.Word70 = 4
	if err := a.HeadlessKey("alt-d"); err != nil {
		t.Fatal(err)
	}
	if !f.live.cheats.turnTrace {
		t.Fatal("debug console incorrectly reused the typed-command participant gate")
	}
}

func TestReleaseCheatOrdinaryChatAndNamesUseInstalledAlphabet(t *testing.T) {
	for _, source := range []string{"partyFallback", "decodedGroup"} {
		t.Run(source, func(t *testing.T) {
			f, a, id := cheatReleaseApp(t, false)
			playerUTF8, playerBytes := "Player", "Player"
			chatUTF8, chatBytes := "Hello", "Hello"
			if f.live.view.TextSelector() == text.SelectorConverting {
				playerUTF8, playerBytes = "Данас", "\x84\xa0\xad\xa0\xe1"
				chatUTF8, chatBytes = "Привет", "\x8f\xe0\xa8\xa2\xa5\xe2"
			}
			groups := f.live.mission.state.Map.Groups
			if len(groups) < int(sim.SelfSlot) {
				t.Fatal("installed map lacks the local Player group")
			}
			groups[sim.SelfSlot-1].Name = ""
			f.live.mission.party[0].Name = playerBytes
			if source == "decodedGroup" {
				groups[sim.SelfSlot-1].Name = playerUTF8
			}
			f.live.mission.state.Map.Meta.Word70 = 4
			f.live.view.ClearMessages()
			hash := f.live.world.Hash()
			cheatChat(t, a, chatUTF8)
			wantChat := playerBytes + ": " + chatBytes
			lines := f.live.view.MessageLines()
			if len(lines) != 1 || lines[0].Text != wantChat || f.live.world.Hash() != hash || f.live.cheats.privilege[sim.SelfSlot] != 0 {
				t.Fatalf("ordinary participant chat=%v, want install bytes %q without a cheat mutation", lines, wantChat)
			}
			if pic, _, _, ok := f.live.view.MessageLog(); !ok || pic == nil {
				t.Fatal("ordinary installed chat did not draw a mission message")
			}
			f.live.mission.state.Map.Meta.Word70 = 1
			f.live.view.ClearMessages()
			cheatChat(t, a, "#Chicken")
			lines = f.live.view.MessageLines()
			wantFrame := "Player " + playerBytes + " enable cheating."
			wantNotice := rawTextLine(t, f, MainTextPath, 221) + playerBytes + rawTextLine(t, f, MainTextPath, 222)
			if len(lines) != 2 || lines[0].Text != wantFrame || lines[1].Text != wantNotice || f.live.cheats.privilege[sim.SelfSlot] != 255 {
				t.Fatalf("Chicken name projection=%v, want frame %q and notice %q", lines, wantFrame, wantNotice)
			}
			cheatChat(t, a, "#kill "+playerUTF8)
			wantKillNotice := rawTextLine(t, f, MainTextPath, 225) + playerBytes + rawTextLine(t, f, MainTextPath, 226)
			lines = f.live.view.MessageLines()
			if releaseEntity(t, f.live, id).HP != -50 || len(lines) == 0 || lines[len(lines)-1].Text != wantKillNotice {
				t.Fatal("typed current Player name did not target its hero or project the installed notice")
			}
			store := SaveStore{Dir: t.TempDir()}
			name, _ := deadPatrolSave(t, f, store)
			cold := deadPatrolLoad(t, store, name)
			if len(cold.live.mission.ids) == 0 || releaseEntity(t, cold.live, cold.live.mission.ids[0]).HP != -50 {
				t.Fatal("cold LOAD lost the named Player kill's ordinary hero health")
			}
		})
	}
}
