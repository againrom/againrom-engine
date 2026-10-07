package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"image"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/camera"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// releaseFront opens the lawful install named explicitly by the caller. These
// tests are skipped in the asset-free unit gate and are run again with
// AGAINROM_ASSETS for the release gate; no fixture can silently stand in for
// the rows, save decoder, or art files they are meant to witness.
func releaseFront(t *testing.T) *FrontEnd {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: release integration needs a lawful install")
	}
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", root, err)
	}
	cleanupFrontAudio(t, f)
	// A fixed presentation seed keeps the town wildlife positions, and so every
	// town frame a release test composes, the same from run to run.
	f.AmbientSeed = 1
	return f
}

// This starts through the owner's exact ui.App load path. Save 666 already
// carries the mission-20 message latches as spent, so it must not manufacture
// the old pre-1066 notice sequence merely to let this witness leave the map.
// Its saved Player outcome now presents Victory immediately. The ordinary App
// acknowledgement and automatic town return must carry the same characters.
func TestReleaseSave666ProductionHeadlessScenarioKeepsCharacterOwnership(t *testing.T) {
	save666 := os.Getenv("AGAINROM_SAVE_666")
	if save666 == "" {
		t.Skip("AGAINROM_SAVE_666 is not set")
	}
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "0152-save666.json"))
	if err != nil {
		t.Fatalf("read scenario: %v", err)
	}
	scenario.OriginalSaves = filepath.Dir(save666)
	scenario.Saves = t.TempDir()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("0152-release-headless")
	app.SetSaveSeams(agsSaveSeams(f, SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: scenario.OriginalSaves}, nil))
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	restored := f.HeadlessSnapshot(app.Screen())
	if text, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
		t.Fatalf("save-666 must show saved Victory, not a spent dialogue: %q/%v", text, kind)
	}
	captures := map[string]HeadlessState{"restore_complete": restored}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4000 && !f.townUI.AtTownSquare(); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() || f.Town.Gold() != 1100 {
		t.Fatalf("saved Victory did not return to town once: screen=%v gold=%d", app.Screen(), f.Town.Gold())
	}
	town := f.HeadlessSnapshot(ui.ScreenTown)
	captures["town"] = town
	if _, err := findHeadlessMember(town, "npc:22"); err != nil {
		t.Fatalf("mission-20 town join: %v", err)
	}
	open30 := f.MissionOpener(30)
	if _, _, _, _, _, _, _, _, _, _, err := open30(); err != nil {
		t.Fatalf("open mission 30 after saved mission 20: %v", err)
	}
	state := f.HeadlessSnapshot(ui.ScreenMap)
	if err := assertHeadlessMember(state, HeadlessMemberAssertion{
		ID: "hero", SameCharacterAs: "restore_complete",
	}, captures); err != nil {
		t.Fatal(err)
	}
	if err := assertHeadlessMember(state, HeadlessMemberAssertion{
		ID: "npc:22", SameCharacterAs: "town",
	}, captures); err != nil {
		t.Fatal(err)
	}
	danath, err := findHeadlessMember(state, "hero")
	if err != nil {
		t.Fatal(err)
	}
	reniesta, err := findHeadlessMember(state, "npc:22")
	if err != nil {
		t.Fatal(err)
	}
	if danath.XP == 0 || reniesta.XP == 0 || reflect.DeepEqual(danath.Skills, reniesta.Skills) {
		t.Fatalf("final independent state: Danath xp=%d skills=%v; Reniesta xp=%d skills=%v",
			danath.XP, danath.Skills, reniesta.XP, reniesta.Skills)
	}
	base, spread, ok := f.live.world.WeaponSpellDamage(sim.EntityID(reniesta.Entity))
	if !ok {
		t.Fatal("save-666 scenario ended with no live Reniesta staff spell")
	}
	// HeadlessSnapshot shares the final item-instance boundary with the live
	// popup. The later staff-tooltip hotfix therefore replaces the localized
	// raw damage/effect lines with its dedicated spell block here too.
	wantDamage := fmt.Sprintf("%s %d-%d", ui.AuthoredWords().ItemDamage, base, base+spread)
	physicalZeroLine := ui.AuthoredWords().ItemDamage + " 0-0"
	found, physicalZero := false, false
	for _, piece := range reniesta.Worn {
		if piece.Slot != 1 {
			continue
		}
		for _, line := range piece.Info {
			found = found || line == wantDamage
			physicalZero = physicalZero || line == physicalZeroLine
		}
	}
	if !found || physicalZero {
		t.Fatalf("save-666 headless Reniesta tooltip = %+v, want exact live release %q and no physical 0-0",
			reniesta.Worn, wantDamage)
	}
}

func releaseEntity(t *testing.T, mw *mapWorld, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := mw.entity(id)
	if !ok {
		t.Fatalf("world has no entity %d", id)
	}
	return e
}

func releaseSpeakerActor(t *testing.T, mw *mapWorld, id sim.EntityID) speakerActor {
	t.Helper()
	for _, actor := range mw.speakerActors {
		if actor.id == id {
			return actor
		}
	}
	t.Fatalf("speaker population has no entity %d", id)
	return speakerActor{}
}

func releaseMissionMap(t *testing.T, f *FrontEnd, n int) *alm.Map {
	t.Helper()
	addr, ok := MissionMap(n)
	if !ok {
		t.Fatalf("mission %d has no map address", n)
	}
	b, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	m, err := alm.Open(b)
	if err != nil {
		t.Fatalf("decode %s: %v", addr, err)
	}
	return m
}

type releaseJoinedPayload struct {
	row       string
	class     int32
	figureDir string
	figure    int
	hero      data.Hero
	skillXP   [data.SkillSlots]int32
	worn      [sim.EquipSlots]sim.ItemInstance
}

func assertReleaseJoinedPayload(t *testing.T, ms *Mission, id sim.EntityID, want releaseJoinedPayload) {
	t.Helper()
	member, ok := ms.Start.Roster[id]
	if !ok {
		t.Fatalf("joined roster has no entity %d", id)
	}
	if member.Name != want.row || member.Class != want.class || member.FigureDir != want.figureDir ||
		member.FigureFace != want.figure || member.Hero != want.hero || member.KnownSpells != 0 {
		t.Fatalf("joined roster entity %d = row %q class %d figure %q/%d hero %+v spells %#x; want %q/%d/%q/%d/%+v/0",
			id, member.Name, member.Class, member.FigureDir, member.FigureFace, member.Hero,
			member.KnownSpells, want.row, want.class, want.figureDir, want.figure, want.hero)
	}
	e := releaseEntity(t, &mapWorld{world: ms.World}, id)
	figure := data.FigureDir(want.figureDir)
	wantType := sim.HeroTypeID(figure.Mage(), figure.Female())
	if e.TypeID != wantType || e.Class != want.class || e.Skill != want.hero.Skill ||
		e.SkillXP != want.skillXP || e.KnownSpells != 0 {
		t.Fatalf("joined live entity %d = type %#x class %d skills %v xp %v spells %#x; want %#x/%d/%v/%v/0",
			id, e.TypeID, e.Class, e.Skill, e.SkillXP, e.KnownSpells,
			wantType, want.class, want.hero.Skill, want.skillXP)
	}
	var aggregate int32
	for _, xp := range e.SkillXP {
		aggregate += xp
	}
	var wantAggregate int32
	for _, xp := range want.skillXP {
		wantAggregate += xp
	}
	if aggregate != wantAggregate {
		t.Fatalf("joined live entity %d aggregate xp = %d, want %d", id, aggregate, wantAggregate)
	}
	worn, ok := ms.World.EquippedItems(id)
	if !ok || !reflect.DeepEqual(worn, want.worn) {
		t.Fatalf("joined live entity %d worn instances = %#v, ok=%v; want %#v", id, worn, ok, want.worn)
	}
	carried, ok := ms.World.CarriedItems(id)
	if !ok || len(carried) != 0 {
		t.Fatalf("joined live entity %d carried instances = %#v, ok=%v; want empty", id, carried, ok)
	}
}

func releaseRosterNPC(t *testing.T, ms *Mission, npc int) sim.EntityID {
	t.Helper()
	for id, member := range ms.Start.Roster {
		if member.CompanionNPC == npc {
			return id
		}
	}
	t.Fatalf("mission %d roster has no npc%d", ms.Number, npc)
	return 0
}

// releaseMissionFigurePane composes the exact mission-pane frame expected for
// one resolved figure. It supplies the shipped pane art and fixed mission
// geometry directly, without reading the Viewer's selected-picture state.
func releaseMissionFigurePane(f *FrontEnd, figure *image.RGBA) *image.RGBA {
	const (
		paneSeamW = 16
		paneW     = 160
		paneH     = 242
	)
	panes := f.characterPanes()
	pic := image.NewRGBA(image.Rect(0, 0, paneSeamW+paneW, paneH))
	ui.DrawTownCharacterRegion(pic, ui.TownCharacterView{
		HasSubject: true,
		Figure:     figure,
		FigurePane: panes.Figure,
		StatsPane:  panes.Stats,
		Session:    ui.CharacterPaneMission,
		ScreenH:    ui.MissionFrameH,
		// The fixed 1024x768 mission frame leaves 288 rows below the
		// 242-row pane, so the original's mode-control gate is closed.
		ModeFlag:  true,
		PaneRect:  image.Rect(paneSeamW, 0, paneSeamW+paneW, paneH),
		PackOpen:  true,
		BookOpen:  true,
		CornerArt: f.characterPaneCorners(),
		Font:      f.Font.Value(),
	})
	return pic
}

func TestReleasePersistentJoinProducerPopulationSelectsTenExactRows(t *testing.T) {
	f := releaseFront(t)
	type variant struct {
		name          string
		mission, npc  int
		primaryMage   bool
		primaryFigure data.FigureDir
		row           string
		class         int32
		figure        data.FigureDir
		face          int
		hero          data.Hero
		xp            [data.SkillSlots]int32
		spells        uint32
		worn          int
	}
	variants := []variant{
		{name: "town female primary", mission: 30, npc: 22, primaryFigure: data.FigureDirWomanFighter,
			row: "PC_Fergard", class: 24, figure: data.FigureDirManMage, face: 3,
			hero: data.Hero{Body: 28, Reaction: 20, Mind: 41, Spirit: 32, Skill: [data.SkillSlots]int32{0, 10, 0, 0, 0, 0}},
			xp:   [data.SkillSlots]int32{0, 1593, 0, 0, 0, 0}, spells: 0x00041042, worn: 3},
		{name: "town male primary", mission: 30, npc: 22, primaryFigure: data.FigureDirManFighter,
			row: "PC_Reniesta", class: 24, figure: data.FigureDirWomanMage, face: 1,
			hero: data.Hero{Body: 19, Reaction: 23, Mind: 30, Spirit: 42, Skill: [data.SkillSlots]int32{0, 0, 10, 0, 0, 0}},
			xp:   [data.SkillSlots]int32{0, 0, 1593, 0, 0, 0}, spells: 0x00041042, worn: 3},
		{name: "Brian", mission: 40, npc: 25, primaryFigure: data.FigureDirWomanMage, primaryMage: true,
			row: "PC_Paladin", class: 5, figure: data.FigureDirManFighter, face: 1,
			hero: data.Hero{Body: 41, Reaction: 39, Mind: 25, Spirit: 21, Skill: [data.SkillSlots]int32{0, 25, 3, 0, 0, 1}},
			xp:   [data.SkillSlots]int32{0, 9834, 331, 0, 0, 100}, worn: 7},
		{name: "mission70 female primary", mission: 70, npc: 23, primaryFigure: data.FigureDirWomanFighter,
			row: "PC_Danath_2", class: 3, figure: data.FigureDirManFighter, face: 5,
			hero: data.Hero{Body: 42, Reaction: 38, Mind: 26, Spirit: 18, Skill: [data.SkillSlots]int32{0, 39, 23, 0, 0, 5}},
			xp:   [data.SkillSlots]int32{0, 40144, 7954, 0, 0, 610}, worn: 10},
		{name: "Naira", mission: 70, npc: 23, primaryFigure: data.FigureDirManFighter,
			row: "PC_Naira_2", class: 14, figure: data.FigureDirWomanFighter, face: 1,
			hero: data.Hero{Body: 39, Reaction: 41, Mind: 22, Spirit: 26, Skill: [data.SkillSlots]int32{0, 24, 5, 0, 0, 38}},
			xp:   [data.SkillSlots]int32{0, 8849, 610, 0, 0, 36404}, worn: 9},
		{name: "mission100 male fighter", mission: 100, npc: 24, primaryFigure: data.FigureDirManFighter,
			row: "PC_Fergard_3", class: 24, figure: data.FigureDirManMage, face: 3,
			hero: data.Hero{Body: 30, Reaction: 22, Mind: 45, Spirit: 36, Skill: [data.SkillSlots]int32{0, 60, 40, 29, 16, 33}},
			xp:   [data.SkillSlots]int32{0, 303481, 44259, 14863, 3594, 22225}, spells: 0x05cdb4ee, worn: 7},
		{name: "mission100 female fighter", mission: 100, npc: 24, primaryFigure: data.FigureDirWomanFighter,
			row: "PC_Reniesta_3", class: 24, figure: data.FigureDirWomanMage, face: 1,
			hero: data.Hero{Body: 20, Reaction: 25, Mind: 34, Spirit: 46, Skill: [data.SkillSlots]int32{0, 17, 59, 41, 17, 26}},
			xp:   [data.SkillSlots]int32{0, 4054, 275801, 48785, 4054, 10918}, spells: 0x01ddb7e2, worn: 7},
		{name: "mission100 male mage", mission: 100, npc: 24, primaryMage: true, primaryFigure: data.FigureDirManMage,
			row: "PC_Danath_3", class: 3, figure: data.FigureDirManFighter, face: 5,
			hero: data.Hero{Body: 44, Reaction: 40, Mind: 27, Spirit: 19, Skill: [data.SkillSlots]int32{0, 66, 42, 0, 0, 7}},
			xp:   [data.SkillSlots]int32{0, 538407, 53763, 0, 0, 948}, worn: 8},
		{name: "mission100 female mage", mission: 100, npc: 24, primaryMage: true, primaryFigure: data.FigureDirWomanMage,
			row: "PC_Naira_3", class: 14, figure: data.FigureDirWomanFighter, face: 1,
			hero: data.Hero{Body: 41, Reaction: 43, Mind: 23, Spirit: 27, Skill: [data.SkillSlots]int32{0, 43, 8, 0, 0, 65}},
			xp:   [data.SkillSlots]int32{0, 59240, 1143, 0, 0, 489370}, worn: 8},
		{name: "Rood Glaen", mission: 140, npc: 26, primaryFigure: data.FigureDirWomanFighter,
			row: "PC_Elf", class: 24, figure: data.FigureDirManMage, face: 4,
			hero: data.Hero{Body: 14, Reaction: 33, Mind: 47, Spirit: 44, Skill: [data.SkillSlots]int32{0, 70, 45, 15, 82, 63}},
			xp:   [data.SkillSlots]int32{0, 788746, 71890, 3177, 2477564, 404265}, spells: 0x05fc16ee, worn: 8},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			primary := mapload.PartyMember{ID: "hero", StartingHero: true, PlayerCharacter: true,
				Mage: tc.primaryMage, FigureDir: string(tc.primaryFigure)}
			member, ok := mapload.CampaignNPCMember(f.Table, int32(tc.npc), tc.mission,
				[]mapload.PartyMember{primary})
			if !ok {
				t.Fatalf("mission %d npc%d did not resolve", tc.mission, tc.npc)
			}
			if member.Name != tc.row || member.Class != tc.class || member.FigureDir != string(tc.figure) ||
				member.FigureFace != tc.face || member.Hero != tc.hero || member.KnownSpells != tc.spells ||
				member.CompanionNPC != tc.npc || member.StartingHero || !member.PlayerCharacter {
				t.Fatalf("mission %d npc%d exact row = %+v", tc.mission, tc.npc, member)
			}
			if got := member.Hero.Reward().SkillXP; got != tc.xp {
				t.Fatalf("mission %d npc%d skill xp = %v, want %v", tc.mission, tc.npc, got, tc.xp)
			}
			worn := 0
			for _, item := range member.WornItems {
				if !item.Empty() {
					worn++
				}
			}
			if worn != tc.worn || len(member.CarriedItems) != 0 {
				t.Fatalf("mission %d npc%d item counts = worn %d carried %d, want %d/0",
					tc.mission, tc.npc, worn, len(member.CarriedItems), tc.worn)
			}
		})
	}
}

func TestReleaseCampaignJoinIdentitiesAndMission20ClubmenUseCanonicalRows(t *testing.T) {
	f := releaseFront(t)

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(20)(); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	m20 := releaseMissionMap(t, f, 20)
	addr20, _ := MissionMap(20)
	materialized20, err := StartMissionFrom(m20, addr20, 20, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatalf("materialize mission 20 roster: %v", err)
	}
	f.canonicalizeJoinedRoster(materialized20)
	clubmen := 0
	killedClubman := false
	for i, u := range m20.Units {
		if u.GroupID != 16 || u.DefID != 114 {
			continue
		}
		clubmen++
		eq, ok := f.live.world.Equipped(sim.EntityID(i))
		if !ok || eq[0] == 0 || eq[1] == 0 {
			t.Errorf("mission-20 clubman entity %d equipment = %v, ok=%v; want authored mace and shield", i, eq, ok)
		}
		member, ok := materialized20.Start.Roster[sim.EntityID(i)]
		if !ok || member.Worn[0] == 0 || member.Worn[1] == 0 {
			t.Fatalf("mission-20 clubman entity %d roster = %+v, ok=%v; want authored mace and shield", i, member, ok)
		}
		doll, _ := buildInventorySubject(f.Archives.Containers, &Mission{
			Party: []mapload.PartyMember{member}, World: materialized20.World,
			Start: mapload.Start{IDs: []sim.EntityID{sim.EntityID(i)}},
		})
		if doll.Slots[0] == nil || doll.Slots[1] == nil {
			t.Fatalf("mission-20 clubman entity %d doll weapon/shield nil=%v/%v",
				i, doll.Slots[0] == nil, doll.Slots[1] == nil)
		}
		if !member.SuppressCorpseLoot {
			t.Fatalf("mission-20 clubman entity %d roster lacks corpse-loot suppression", i)
		}
		if !killedClubman {
			id := sim.EntityID(i)
			e := releaseEntity(t, &mapWorld{world: materialized20.World}, id)
			if !e.SuppressCorpseLoot {
				t.Fatalf("mission-20 clubman entity %d lacks live corpse-loot suppression", i)
			}
			for _, sack := range materialized20.World.Sacks() {
				if sack.X == e.X && sack.Y == e.Y {
					t.Fatalf("mission-20 clubman cell (%d,%d) already holds sack %+v", e.X, e.Y, sack)
				}
			}
			releaseTerminalDeath(t, materialized20.World, id)
			for _, sack := range materialized20.World.Sacks() {
				if sack.X == e.X && sack.Y == e.Y {
					t.Fatalf("mission-20 clubman death produced sack %+v", sack)
				}
			}
			if worn, ok := materialized20.World.Equipped(id); !ok || worn != ([sim.EquipSlots]uint16{}) {
				t.Fatalf("mission-20 clubman equipment after death = %v, ok=%v; want deleted", worn, ok)
			}
			if carried, ok := materialized20.World.Carried(id); !ok || len(carried) != 0 {
				t.Fatalf("mission-20 clubman container after death = %v, ok=%v; want deleted", carried, ok)
			}
			killedClubman = true
		}
	}
	if clubmen != 3 {
		t.Fatalf("mission-20 group 16 def#114 count = %d, want 3 clubmen", clubmen)
	}
	if !killedClubman {
		t.Fatal("mission 20 production witness killed no group-16 Clubman")
	}

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(40)(); err != nil {
		t.Fatalf("open mission 40: %v", err)
	}
	m40 := releaseMissionMap(t, f, 40)
	addr40, _ := MissionMap(40)
	materialized40, err := StartMissionFrom(m40, addr40, 40, f.Table, mapload.DifficultyNormal, nil)
	if err != nil {
		t.Fatalf("materialize mission 40 roster: %v", err)
	}
	brianID := sim.EntityID(^uint32(0))
	for i, u := range m40.Units {
		if u.Flags&1 != 0 && u.ClassSubID == 25 {
			brianID = sim.EntityID(i)
			break
		}
	}
	if brianID == sim.EntityID(^uint32(0)) {
		t.Fatal("mission 40 has no npc25 placement")
	}
	brianWornExact := [sim.EquipSlots]sim.ItemInstance{
		0:  {Code: uint16(data.ComposeItemCode(2, 1, 1, 6)), Kind: 2, Price: 4800},
		5:  {Code: uint16(data.ComposeItemCode(2, 6, 1, 10)), Kind: 1, Price: 2700},
		6:  {Code: uint16(data.ComposeItemCode(2, 7, 1, 16)), Kind: 1, Price: 1020},
		7:  {Code: uint16(data.ComposeItemCode(2, 8, 1, 19)), Kind: 1, Price: 4200},
		8:  {Code: uint16(data.ComposeItemCode(2, 9, 1, 22)), Kind: 1, Price: 1800},
		9:  {Code: uint16(data.ComposeItemCode(2, 10, 1, 26)), Kind: 1, Price: 1200},
		11: {Code: uint16(data.ComposeItemCode(2, 12, 1, 30)), Kind: 1, Price: 2400},
	}
	assertReleaseJoinedPayload(t, materialized40, brianID, releaseJoinedPayload{
		row: "PC_Paladin", class: 5, figureDir: string(data.FigureDirManFighter), figure: 1,
		hero: data.Hero{Body: 41, Reaction: 39, Mind: 25, Spirit: 21,
			Skill: [data.SkillSlots]int32{0, 25, 3, 0, 0, 1}},
		skillXP: [data.SkillSlots]int32{0, 9834, 331, 0, 0, 100}, worn: brianWornExact,
	})
	f.canonicalizeJoinedRoster(materialized40)
	wantBrian := f.localizedNPCName(24, "Brian")
	if got := f.live.chars[brianID].Name; got != wantBrian {
		t.Fatalf("mission-40 npc25 live name = %q, want localized Brian %q", got, wantBrian)
	}
	brianWorn, _ := f.live.world.Equipped(brianID)
	if brianWorn[0] == 0 || brianWorn[5] == 0 || brianWorn[11] == 0 {
		t.Fatalf("mission-40 Brian worn set = %v, want his authored weapon and plate outfit", brianWorn)
	}
	brianMember, ok := materialized40.Start.Roster[brianID]
	if !ok {
		t.Fatalf("mission-40 roster has no Brian entity %d", brianID)
	}

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(70)(); err != nil {
		t.Fatalf("open mission 70: %v", err)
	}
	m70 := releaseMissionMap(t, f, 70)
	addr70, _ := MissionMap(70)
	materialized70, err := StartMissionFrom(m70, addr70, 70, f.Table, mapload.DifficultyNormal,
		MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
	if err != nil {
		t.Fatalf("materialize mission 70 roster: %v", err)
	}
	nairaID := sim.EntityID(^uint32(0))
	for i, u := range m70.Units {
		if u.Flags&1 != 0 && u.ClassSubID == 23 {
			nairaID = sim.EntityID(i)
			break
		}
	}
	if nairaID == sim.EntityID(^uint32(0)) {
		t.Fatal("mission 70 has no npc23 placement")
	}
	nairaWornExact := [sim.EquipSlots]sim.ItemInstance{
		0:  {Code: uint16(data.ComposeItemCode(9, 1, 2, 20)), Kind: 2, Effects: []sim.ItemEffect{{Kind: 12, Operand: 22}}, Price: 69538},
		3:  {Code: uint16(data.ComposeItemCode(5, 4, 2, 1)), Kind: 1, Effects: []sim.ItemEffect{{Kind: 23, Operand: 16}}, Price: 29810},
		4:  {Code: uint16(data.ComposeItemCode(5, 5, 2, 2)), Kind: 1, Price: 3200},
		5:  {Code: uint16(data.ComposeItemCode(5, 6, 2, 8)), Kind: 1, Price: 4480},
		6:  {Code: uint16(data.ComposeItemCode(2, 7, 1, 16)), Kind: 1, Price: 1020},
		7:  {Code: uint16(data.ComposeItemCode(5, 8, 2, 18)), Kind: 1, Price: 8000},
		8:  {Code: uint16(data.ComposeItemCode(5, 9, 2, 20)), Kind: 1, Price: 3200},
		9:  {Code: uint16(data.ComposeItemCode(11, 10, 1, 24)), Kind: 1, Price: 16},
		11: {Code: uint16(data.ComposeItemCode(2, 12, 1, 29)), Kind: 1, Effects: []sim.ItemEffect{{Kind: 21, Operand: 19}}, Price: 38895},
	}
	assertReleaseJoinedPayload(t, materialized70, nairaID, releaseJoinedPayload{
		row: "PC_Naira_2", class: 14, figureDir: string(data.FigureDirWomanFighter), figure: 1,
		hero: data.Hero{Body: 39, Reaction: 41, Mind: 22, Spirit: 26,
			Skill: [data.SkillSlots]int32{0, 24, 5, 0, 0, 38}},
		skillXP: [data.SkillSlots]int32{0, 8849, 610, 0, 0, 36404}, worn: nairaWornExact,
	})
	naira := releaseEntity(t, f.live, nairaID)
	if naira.Class != 14 {
		t.Fatalf("mission-70 npc23 class = %d, want PC_Naira's archer class 14", naira.Class)
	}
	wantNaira := f.localizedNPCName(21, "Naira")
	if got := f.live.chars[nairaID].Name; got != wantNaira {
		t.Fatalf("mission-70 npc23 live name = %q, want localized Naira %q", got, wantNaira)
	}
	nairaWorn, _ := f.live.world.Equipped(nairaID)
	if nairaWorn[0] == 0 || nairaWorn[6] == 0 {
		t.Fatalf("mission-70 Naira worn set = %v, want bow and leather mail", nairaWorn)
	}
	nairaFace := composeKey(t, f.live, 23)
	nairaActor := releaseSpeakerActor(t, f.live, nairaID)
	nairaFigure := nairaActor.fig
	nairaFigure.Hero = true
	if nairaFace.fig != nairaFigure || nairaFace.eq != equipmentFromSlots(nairaWorn) {
		t.Fatalf("mission-70 npc23 dialogue = figure %+v equipment %+v, want live Naira %+v/%+v",
			nairaFace.fig, nairaFace.eq, nairaFigure, equipmentFromSlots(nairaWorn))
	}

	// Brian has crossed from mission 40 into the party and is not placed in
	// mission 70. His npc25 dialogue must still resolve the live party actor,
	// including the equipment that crossed the boundary, instead of composing
	// the registry's bare synthetic figure.
	partyWithBrian := append(MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table), brianMember)
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(70, partyWithBrian)(); err != nil {
		t.Fatalf("open mission 70 with carried Brian: %v", err)
	}
	if len(f.live.mission.ids) != 2 {
		t.Fatalf("mission-70 carried party ids = %v, want primary and Brian", f.live.mission.ids)
	}
	carriedBrianID := f.live.mission.ids[1]
	carriedBrianWorn, ok := f.live.world.Equipped(carriedBrianID)
	if !ok || carriedBrianWorn[0] == 0 || carriedBrianWorn[5] == 0 || carriedBrianWorn[11] == 0 {
		t.Fatalf("mission-70 carried Brian worn = %v/%v, want authored weapon and plate outfit", carriedBrianWorn, ok)
	}
	brianFace := composeKey(t, f.live, 25)
	brianDir, brianSheet := memberFigure(f.liveParty[1])
	if brianFace.fig != (figureID{Dir: brianDir, Face: brianSheet, Hero: true}) || brianFace.eq != equipmentFromSlots(carriedBrianWorn) {
		t.Fatalf("mission-70 npc25 dialogue = figure %+v equipment %+v, want live carried Brian %+v/%+v",
			brianFace.fig, brianFace.eq, figureID{Dir: brianDir, Face: brianSheet, Hero: true}, equipmentFromSlots(carriedBrianWorn))
	}

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(100)(); err != nil {
		t.Fatalf("open mission 100: %v", err)
	}
	m100 := releaseMissionMap(t, f, 100)
	mapload.WithdrawBorderPlacements(m100)
	fergardID := sim.EntityID(^uint32(0))
	for i, u := range m100.Units {
		if u.Flags&1 != 0 && u.ClassSubID == 24 {
			fergardID = sim.EntityID(i)
			break
		}
	}
	if fergardID == sim.EntityID(^uint32(0)) {
		t.Fatal("mission 100 has no npc24 placement")
	}
	fergard := releaseEntity(t, f.live, fergardID)
	if fergard.Class != 24 {
		t.Fatalf("mission-100 npc24 class = %d, want PC_Fergard's mage class 24", fergard.Class)
	}
	wantFergard := f.localizedNPCName(22, "Fergard")
	if got := f.live.chars[fergardID].Name; got != wantFergard {
		t.Fatalf("mission-100 npc24 live name = %q, want localized Fergard %q", got, wantFergard)
	}

	femaleParty := f.ChargenParty(ui.ChargenResult{
		Name: "female branch witness", Choices: []int{1, 0, 0}, Stats: []int{20, 20, 20, 20},
	})
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(70, femaleParty)(); err != nil {
		t.Fatalf("open mission 70 with female primary: %v", err)
	}
	for i, u := range m70.Units {
		if u.Flags&1 == 0 || u.ClassSubID != 23 {
			continue
		}
		id := sim.EntityID(i)
		danath := releaseEntity(t, f.live, id)
		if danath.Class != 3 {
			t.Fatalf("mission-70 npc23 beside female primary has class %d, want PC_Danath's swordsman class 3", danath.Class)
		}
		wantDanath := f.localizedNPCName(20, "Danath")
		if got := f.live.chars[id].Name; got != wantDanath {
			t.Fatalf("mission-70 npc23 beside female primary is %q, want localized Danath %q", got, wantDanath)
		}
		if worn, _ := f.live.world.Equipped(id); worn[0] == 0 {
			t.Fatal("mission-70 Danath branch has no authored sword")
		}
		return
	}
	t.Fatal("mission 70 female-primary run has no npc23 placement")
}

// This is the viewer-side witness for the ownership-changing tick. Each map
// actor is selected while still foreign. A controlled one-node mission program
// then performs the ordinary GiveUnit step, and the pane composed by the
// retained production selection is compared pixel-for-pixel with an expected
// pane built from the exact joined row and its complete live equipment.
func TestReleaseJoinedHeroHandoverTickComposesExactPanePopulation(t *testing.T) {
	type joinFigureCase struct {
		name         string
		mission, npc int
		primarySex   int
		primaryClass int
		figure       figureID
		naira        bool
		rescue       bool
	}
	cases := []joinFigureCase{
		{name: "fixed Brian", mission: 40, npc: 25,
			figure: figureID{Dir: data.FigureDirManFighter, Face: 1, Hero: true}},
		{name: "npc23 Danath", mission: 70, npc: 23, primarySex: 1,
			figure: figureID{Dir: data.FigureDirManFighter, Face: 5, Hero: true}},
		{name: "npc23 Naira", mission: 70, npc: 23, naira: true,
			figure: figureID{Dir: data.FigureDirWomanFighter, Face: 1, Hero: true}},
		{name: "npc24 Fergard", mission: 100, npc: 24, rescue: true,
			figure: figureID{Dir: data.FigureDirManMage, Face: 3, Hero: true}},
		{name: "npc24 Reniesta", mission: 100, npc: 24, primarySex: 1, rescue: true,
			figure: figureID{Dir: data.FigureDirWomanMage, Face: 1, Hero: true}},
		{name: "npc24 Danath", mission: 100, npc: 24, primaryClass: 1, rescue: true,
			figure: figureID{Dir: data.FigureDirManFighter, Face: 5, Hero: true}},
		{name: "npc24 Naira", mission: 100, npc: 24, primarySex: 1, primaryClass: 1, naira: true, rescue: true,
			figure: figureID{Dir: data.FigureDirWomanFighter, Face: 1, Hero: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{
				Name:    tc.name + " primary",
				Choices: []int{tc.primarySex, tc.primaryClass, 0},
				Stats:   []int{25, 25, 25, 25},
			})
			app := f.App("1046-joined-figure-population")
			if err := app.OpenMission(f.MissionOpenerWith(tc.mission, party)); err != nil {
				t.Fatalf("open mission %d: %v", tc.mission, err)
			}
			joinID := releaseRosterNPC(t, f.live.mission.state, tc.npc)
			before := releaseEntity(t, f.live, joinID)
			if before.Owner == sim.SelfSlot {
				t.Fatalf("mission %d npc%d is already owned before the controlled handover", tc.mission, tc.npc)
			}
			f.live.view.Camera().CenterOn(
				float64(before.X*camera.CellSize+camera.CellSize/2),
				float64(before.Y*camera.CellSize+camera.CellSize/2))
			if err := app.HeadlessSelectEntity(uint32(joinID)); err != nil {
				t.Fatalf("preselect mission %d npc%d before handover: %v", tc.mission, tc.npc, err)
			}
			if selected, ok := f.live.view.SelectedUnit(); !ok || selected != uint32(joinID) {
				t.Fatalf("pre-handover selection = %d/%v, want npc%d/%d", selected, ok, tc.npc, joinID)
			}
			if got := releaseEntity(t, f.live, joinID).Owner; got == sim.SelfSlot {
				t.Fatalf("mission %d npc%d joined during preselection, before the controlled handover", tc.mission, tc.npc)
			}

			joinScript := joinedLiveScript(t, joinID)
			if tc.rescue {
				if before.HP != 0 || before.Decay != sim.DecayFallen {
					t.Fatalf("mission %d npc%d rescue setup = HP %d decay %d, want authored 0/fallen",
						tc.mission, tc.npc, before.HP, before.Decay)
				}
				joinScript = joinedLiveRescueScript(t, joinID)
			}
			entryWorn, _ := f.live.world.EquippedItems(joinID)
			controlled, err := sim.NewControlledScriptWorld(f.live.world, joinScript)
			if err != nil {
				t.Fatalf("controlled npc%d handover: %v", tc.npc, err)
			}
			f.live.world = controlled
			f.live.mission.state.World = controlled
			joinedAt := -1
			for tick := 0; tick < 32; tick++ {
				f.live.tick()
				if releaseEntity(t, f.live, joinID).Owner == sim.SelfSlot {
					joinedAt = tick + 1
					break
				}
			}

			if joinedAt < 0 {
				t.Fatalf("controlled npc%d handover did not fire within 32 ticks", tc.npc)
			}
			if tc.rescue {
				rescued := releaseEntity(t, f.live, joinID)
				if rescued.HP != 1 || rescued.Decay != sim.DecayNone || rescued.Dwell != 0 {
					t.Fatalf("same-tick npc%d rescue = HP %d decay %d dwell %d, want 1/none/0",
						tc.npc, rescued.HP, rescued.Decay, rescued.Dwell)
				}
			}
			if selected, ok := f.live.view.SelectedUnit(); !ok || selected != uint32(joinID) {
				t.Fatalf("same-tick npc%d selection = %d/%v, want retained", tc.npc, selected, ok)
			}
			joined, ok := f.live.mission.state.Start.Roster[joinID]
			if !ok {
				t.Fatalf("mission %d npc%d has no roster row", tc.mission, tc.npc)
			}
			if got := (figureID{Dir: data.FigureDir(joined.FigureDir), Face: joined.FigureFace, Hero: joined.PlayerCharacter && !joined.Hired()}); got != tc.figure {
				t.Fatalf("mission %d npc%d roster figure = %+v, want %+v", tc.mission, tc.npc, got, tc.figure)
			}
			if got := f.live.figures[joinID]; got != tc.figure {
				t.Fatalf("same-tick npc%d installed figure = %+v, want %+v", tc.npc, got, tc.figure)
			}
			liveItems, ok := f.live.world.EquippedItems(joinID)
			if !ok || !reflect.DeepEqual(liveItems, entryWorn) {
				t.Fatalf("same-tick npc%d worn instances = %#v/%v, want roster %#v",
					tc.npc, liveItems, ok, entryWorn)
			}
			liveSlots, ok := f.live.world.Equipped(joinID)
			if !ok {
				t.Fatalf("same-tick npc%d has no live equipment", tc.npc)
			}
			fullEquipment := equipmentFromSlots(liveSlots)
			if fullEquipment == (data.Equipment{}) {
				t.Fatalf("same-tick npc%d fixture has no equipment to preserve", tc.npc)
			}
			exactFigure, _ := composeUnitFigure(f.live.archive(), fullEquipment, tc.figure)
			if exactFigure == nil {
				t.Fatalf("same-tick npc%d exact joined figure did not compose", tc.npc)
			}
			pane, statistics, err := app.HeadlessCharacterPane()
			if err != nil || pane == nil || statistics {
				t.Fatalf("same-tick npc%d character pane = %v statistics %v err %v",
					tc.npc, pane != nil, statistics, err)
			}
			wantPane := releaseMissionFigurePane(f, exactFigure)
			if pane.Bounds() != wantPane.Bounds() || !bytes.Equal(pane.Pix, wantPane.Pix) {
				t.Fatalf("same-tick npc%d pane is not the exact fully equipped joined figure", tc.npc)
			}

			if tc.naira {
				danath, _ := composeUnitFigure(f.live.archive(), fullEquipment,
					figureID{Dir: data.FigureDirManFighter, Face: 5, Hero: true})
				naked, _ := composeUnitFigure(f.live.archive(), data.Equipment{}, tc.figure)
				if danath == nil || naked == nil {
					t.Fatalf("same-tick npc%d Naira alternatives did not compose", tc.npc)
				}
				if bytes.Equal(pane.Pix, releaseMissionFigurePane(f, danath).Pix) {
					t.Fatalf("same-tick npc%d Naira pane is indistinguishable from Danath", tc.npc)
				}
				if bytes.Equal(pane.Pix, releaseMissionFigurePane(f, naked).Pix) {
					t.Fatalf("same-tick npc%d Naira pane is indistinguishable from her naked figure", tc.npc)
				}
			}
		})
	}
}

// joinedLiveRescueScript reproduces mission 100's required ordering in one
// controlled script pass: instant 34 selector 6 writes health 1, then GiveUnit
// hands the now-living actor over. The release witness therefore keeps testing
// the same-tick figure handover without pretending the shipped npc24 placement
// starts alive.
func joinedLiveRescueScript(t *testing.T, id sim.EntityID) *sim.Script {
	t.Helper()
	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{1}},
		},
		[]sim.ScriptInstant{
			{Op: sim.ScriptInstantProperty, Unit: id, HasUnit: true, Args: [10]int32{6, 1}},
			{Op: sim.ScriptInstantGiveUnit, Unit: id, HasUnit: true,
				Player: sim.SelfSlot, HasPlayer: true},
		},
		[]sim.ScriptTrigger{{
			Pairs:    [3]sim.ScriptPair{{Left: 0, Right: 1, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, 1, sim.ScriptNone, sim.ScriptNone}, Once: true,
		}},
	)
	if err != nil {
		t.Fatalf("NewScript rescue handover: %v", err)
	}
	return s
}

func TestReleaseCampaignJoinRoutesAreImmediateInteractiveAndPersistent(t *testing.T) {
	cases := []struct {
		name, row     string
		mission, next int
		npc           int
		moveX, moveY  int
		unequip       int
		nameIndex     int
		skillXP       [data.SkillSlots]int32
		witnessSlot   int
		witnessCode   data.ItemCode
		witnessPrice  int32
	}{
		{name: "Brian", row: "PC_Paladin", mission: 40, next: 70, npc: 25,
			moveX: 76, moveY: 108, unequip: 5, nameIndex: 24,
			skillXP:     [data.SkillSlots]int32{0, 9834, 331, 0, 0, 100},
			witnessSlot: 5, witnessCode: data.ComposeItemCode(2, 6, 1, 10), witnessPrice: 2700},
		{name: "Naira", row: "PC_Naira_2", mission: 70, next: 100, npc: 23,
			moveX: 38, moveY: 108, unequip: 3, nameIndex: 21,
			skillXP:     [data.SkillSlots]int32{0, 8849, 610, 0, 0, 36404},
			witnessSlot: 3, witnessCode: data.ComposeItemCode(5, 4, 2, 1), witnessPrice: 29810},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			app := f.App("1046-joined-hero-route")
			if err := app.OpenMission(f.MissionOpener(tc.mission)); err != nil {
				t.Fatalf("open mission %d: %v", tc.mission, err)
			}
			if f.live == nil || f.live.mission == nil || f.live.mission.state == nil || len(f.live.mission.ids) != 1 {
				t.Fatalf("mission %d entry party = %#v", tc.mission, f.live)
			}
			joinID := releaseRosterNPC(t, f.live.mission.state, tc.npc)
			entryWorn, _ := f.live.world.EquippedItems(joinID)
			primaryID := f.live.mission.ids[0]
			// Select the mission-local actor through the production pointer path
			// while it still belongs to its original owner. The point-selection
			// rule admits a foreign actor, and the retained selection is the state
			// that exposed the first-post-handover figure gap.
			beforeJoin := releaseEntity(t, f.live, joinID)
			f.live.view.Camera().CenterOn(
				float64(beforeJoin.X*camera.CellSize+camera.CellSize/2),
				float64(beforeJoin.Y*camera.CellSize+camera.CellSize/2))
			if err := app.HeadlessSelectEntity(uint32(joinID)); err != nil {
				t.Fatalf("preselect mission %d npc%d before handover: %v", tc.mission, tc.npc, err)
			}
			if selected, ok := f.live.view.SelectedUnit(); !ok || selected != uint32(joinID) {
				t.Fatalf("pre-handover selection = %d/%v, want npc%d/%d", selected, ok, tc.npc, joinID)
			}
			if !f.live.invSubjectSet || f.live.invSubject.ID != uint32(primaryID) {
				t.Fatalf("pre-handover inventory subject = set %v id %d, want primary %d",
					f.live.invSubjectSet, f.live.invSubject.ID, primaryID)
			}
			f.live.enqueue(uint32(primaryID), tc.moveX, tc.moveY)
			joinedAt := -1
			for tick := 0; tick < 8000; tick++ {
				f.live.tick()
				e := releaseEntity(t, f.live, joinID)
				if e.Owner == sim.SelfSlot {
					joinedAt = tick + 1
					break
				}
			}
			if joinedAt < 0 {
				t.Fatalf("mission %d production route did not hand npc%d over within 8000 ticks", tc.mission, tc.npc)
			}
			if len(f.live.mission.party) != 2 || len(f.live.mission.ids) != 2 || f.live.mission.ids[1] != joinID {
				t.Fatalf("same-tick joined party = %#v ids %v, want primary then npc%d/%d",
					f.live.mission.party, f.live.mission.ids, tc.npc, joinID)
			}
			joined := f.live.mission.party[1]
			joinedEntity := releaseEntity(t, f.live, joinID)
			joinedXP := joinedEntity.SkillXP
			wantName := f.localizedNPCName(tc.nameIndex, tc.name)
			if joined.CompanionNPC != tc.npc || joined.StartingHero || !joined.PlayerCharacter ||
				joined.Carry == nil || joined.Carry.SkillXP != joinedXP || joined.Hero.Skill != joinedEntity.Skill ||
				joined.Name != wantName {
				t.Fatalf("same-tick npc%d member = %+v", tc.npc, joined)
			}
			for i, xp := range joinedXP {
				if xp < tc.skillXP[i] {
					t.Fatalf("same-tick npc%d skill-xp slot %d = %d, below source %d", tc.npc, i, xp, tc.skillXP[i])
				}
			}
			if !f.live.isGuarded(joinID) {
				t.Fatalf("same-tick npc%d is not an ordinary guarded player character", tc.npc)
			}
			if selected, ok := f.live.view.SelectedUnit(); !ok || selected != uint32(joinID) {
				t.Fatalf("same-tick npc%d selection = %d/%v, want retained", tc.npc, selected, ok)
			}
			if !f.live.invSubjectSet || f.live.invSubject.ID != uint32(primaryID) {
				t.Fatalf("same-tick npc%d inventory subject = set %v id %d, want retained primary %d",
					tc.npc, f.live.invSubjectSet, f.live.invSubject.ID, primaryID)
			}
			char, ok := f.live.chars[joinID]
			if !ok || !char.Known || char.Name != wantName {
				t.Fatalf("same-tick npc%d character = %+v, present %v; want localized hero %q",
					tc.npc, char, ok, wantName)
			}
			drawn := false
			for _, draw := range f.live.entityDraws() {
				if draw.ID != uint32(joinID) {
					continue
				}
				drawn = true
				if !draw.PlayerCharacter || draw.Name != wantName {
					t.Fatalf("same-tick npc%d draw = player %v name %q, want player hero %q",
						tc.npc, draw.PlayerCharacter, draw.Name, wantName)
				}
			}
			if !drawn {
				t.Fatalf("same-tick npc%d has no map draw", tc.npc)
			}
			bodyKey := data.HeroBodyKey(joined.BodyDir, data.HeroBody(joined.Body))
			if f.live.units == nil || f.live.units.Bodies[bodyKey] == nil || f.live.art[joinID] != f.live.units.Bodies[bodyKey] {
				t.Fatalf("same-tick npc%d body art = %q/%p, want installed %q/%p",
					tc.npc, releaseArtName(f.live, joinID), f.live.art[joinID], bodyKey, f.live.units.Bodies[bodyKey])
			}
			wantFigureDir, wantFigureFace := memberFigure(joined)
			gotFigure, ok := f.live.figures[joinID]
			if !ok || gotFigure != (figureID{Dir: wantFigureDir, Face: wantFigureFace, Hero: true}) {
				t.Fatalf("same-tick npc%d figure identity = %+v/%v, want %s/%d",
					tc.npc, gotFigure, ok, wantFigureDir, wantFigureFace)
			}
			liveItems, ok := f.live.world.EquippedItems(joinID)
			rosterItems := entryWorn
			if !ok || !reflect.DeepEqual(liveItems, rosterItems) {
				t.Fatalf("same-tick npc%d full live equipment = %#v/%v, want roster %#v",
					tc.npc, liveItems, ok, rosterItems)
			}
			liveSlots, ok := f.live.world.Equipped(joinID)
			if !ok {
				t.Fatalf("same-tick npc%d has no live equipment projection", tc.npc)
			}
			figureKey := figureCacheKey{fig: gotFigure, eq: equipmentFromSlots(liveSlots)}
			composed, composedAtPush := f.live.figurePics[figureKey]
			if !composedAtPush || composed == nil {
				t.Fatalf("same-tick npc%d selected portrait did not compose its full figure", tc.npc)
			}
			if portrait := f.live.unitPicture(joinID, joinedEntity.Class); portrait != composed {
				t.Fatalf("same-tick npc%d portrait = %p, want composed figure %p", tc.npc, portrait, composed)
			}
			doll, ok := f.LiveDoll(uint32(joinID))
			if !ok || !doll.FullDrawn || doll.Equipment != liveSlots || doll.Full != sha256.Sum256(composed.Pix) {
				t.Fatalf("same-tick npc%d doll = ok %v drawn %v equipment %v digest %x; want %v/%x",
					tc.npc, ok, doll.FullDrawn, doll.Equipment, doll.Full, liveSlots, sha256.Sum256(composed.Pix))
			}
			pane, statistics, err := app.HeadlessCharacterPane()
			if err != nil || pane == nil || statistics {
				t.Fatalf("same-tick npc%d composed character pane = %v statistics %v err %v",
					tc.npc, pane != nil, statistics, err)
			}

			// The next ordinary application frame performs the production
			// selection-to-inventory refresh. No direct subject switch is used.
			if err := app.HeadlessStep(); err != nil {
				t.Fatalf("refresh joined npc%d through production frame: %v", tc.npc, err)
			}
			if !f.live.invSubjectSet || f.live.invSubject.ID != uint32(joinID) {
				t.Fatalf("npc%d inventory subject after production refresh = set %v id %d",
					tc.npc, f.live.invSubjectSet, f.live.invSubject.ID)
			}
			before, ok := f.live.world.EquippedItems(joinID)
			if !ok || data.ItemCode(before[tc.witnessSlot].Code) != tc.witnessCode ||
				before[tc.witnessSlot].Price != tc.witnessPrice {
				t.Fatalf("npc%d witness slot %d = %+v, ok=%v; want %s price %d",
					tc.npc, tc.witnessSlot+1, before[tc.witnessSlot], ok, tc.witnessCode.Name(), tc.witnessPrice)
			}
			item := before[tc.witnessSlot].Clone()
			f.live.enqueueUnequip(tc.unequip)
			f.live.tick()
			afterOff, _ := f.live.world.EquippedItems(joinID)
			if !afterOff[tc.witnessSlot].Empty() {
				t.Fatalf("npc%d reversible unequip left slot %d = %+v", tc.npc, tc.witnessSlot+1, afterOff[tc.witnessSlot])
			}
			stacks, _ := f.live.world.CarriedStacks(joinID)
			packIndex := -1
			for i, stack := range stacks {
				if sim.ItemEqual(stack.Instance(), item) && stack.Price == item.Price {
					packIndex = i
					break
				}
			}
			if packIndex < 0 {
				t.Fatalf("npc%d pack after unequip = %+v, missing %+v", tc.npc, stacks, item)
			}
			f.live.enqueueEquip(packIndex)
			f.live.tick()
			afterOn, _ := f.live.world.EquippedItems(joinID)
			if !reflect.DeepEqual(afterOn[tc.witnessSlot], item) {
				t.Fatalf("npc%d reversible equip restored %+v, want %+v", tc.npc, afterOn[tc.witnessSlot], item)
			}

			snap, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatalf("snapshot joined npc%d: %v", tc.npc, err)
			}
			if len(snap.Party) != 1 {
				t.Fatalf("joined npc%d save mint prefix = %#v, want only entry hero", tc.npc, snap.Party)
			}
			encoded, err := EncodeSave(snap, label)
			if err != nil {
				t.Fatalf("encode joined npc%d: %v", tc.npc, err)
			}
			decoded, _, err := DecodeSave(encoded)
			if err != nil {
				t.Fatalf("decode joined npc%d: %v", tc.npc, err)
			}
			restored := releaseFront(t)
			opener, town, err := restored.Restore(decoded)
			if err != nil || town || opener == nil {
				t.Fatalf("prepare joined npc%d restore = opener %v town %v err %v", tc.npc, opener != nil, town, err)
			}
			if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
				t.Fatalf("open joined npc%d restore: %v", tc.npc, err)
			}
			if len(restored.live.mission.party) != 2 || len(restored.live.mission.ids) != 2 {
				t.Fatalf("restored npc%d party = %#v ids %v", tc.npc, restored.live.mission.party, restored.live.mission.ids)
			}
			restoredID := restored.live.mission.ids[1]
			restoredMember := restored.live.mission.party[1]
			restoredEntity := releaseEntity(t, restored.live, restoredID)
			restoredWorn, _ := restored.live.world.EquippedItems(restoredID)
			if restoredMember.CompanionNPC != tc.npc || restoredMember.Carry == nil ||
				restoredMember.Carry.SkillXP != joinedXP || restoredEntity.SkillXP != joinedXP ||
				!reflect.DeepEqual(restoredWorn[tc.witnessSlot], item) {
				t.Fatalf("restored npc%d = member %+v entity xp %v slot %+v",
					tc.npc, restoredMember, restoredEntity.SkillXP, restoredWorn[tc.witnessSlot])
			}

			carried := mapload.CarryRoster(restored.live.mission.party, restored.live.world,
				restored.live.mission.ids, restored.live.mission.state.Start.Roster)
			if _, _, _, _, _, _, _, _, _, _, err := restored.MissionOpenerWith(tc.next, carried)(); err != nil {
				t.Fatalf("open mission %d after joined npc%d: %v", tc.next, tc.npc, err)
			}
			count, carriedAt := 0, -1
			for i, member := range restored.live.mission.party {
				if member.CompanionNPC == tc.npc {
					count++
					carriedAt = i
				}
			}
			if count != 1 || carriedAt < 0 || carriedAt >= len(restored.live.mission.ids) {
				t.Fatalf("mission %d npc%d continuity count/index = %d/%d party %#v",
					tc.next, tc.npc, count, carriedAt, restored.live.mission.party)
			}
			carriedID := restored.live.mission.ids[carriedAt]
			carriedEntity := releaseEntity(t, restored.live, carriedID)
			carriedWorn, _ := restored.live.world.EquippedItems(carriedID)
			// CarryRoster releases world-local handles; the new mission binds
			// the same complete item value into its own registry.
			carriedItem := carriedWorn[tc.witnessSlot].Clone()
			if carriedItem.ObjectID == 0 {
				t.Fatal("next mission did not bind the carried item")
			}
			carriedItem.ObjectID, item.ObjectID = 0, 0
			if carriedEntity.SkillXP != joinedXP || !reflect.DeepEqual(carriedItem, item) {
				t.Fatalf("mission %d carried npc%d = xp %v slot %+v", tc.next, tc.npc,
					carriedEntity.SkillXP, carriedWorn[tc.witnessSlot])
			}
		})
	}
}

func releaseArtName(mw *mapWorld, id sim.EntityID) string {
	if mw == nil || mw.art[id] == nil {
		return ""
	}
	return mw.art[id].Name
}

// TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity exercises
// the player-facing generator route against the lawful install: a non-default
// identity enters mission 10, survives our save reader in a fresh front end,
// and is the same person handed to the campaign successor.
func TestReleaseGeneratedCharacterLaunchSaveLoadAndCampaignContinuity(t *testing.T) {
	f := releaseFront(t)
	result := ui.ChargenResult{Name: "T2 Hero", Choices: []int{1, 1, 3}, Stats: []int{31, 27, 24, 29}}
	generated := f.ChargenParty(result)
	if len(generated) != 1 {
		t.Fatalf("ChargenParty length = %d, want 1", len(generated))
	}
	want := generated[0]
	identity := func(member mapload.PartyMember) string {
		weapon := ""
		if member.Weapon != nil {
			weapon = member.Weapon.Name
		}
		return fmt.Sprintf("%q mage=%v figure=%q spread=%d/%d/%d/%d skills=%v weapon=%q worn=%v",
			member.Name, member.Mage, member.FigureDir, member.Hero.Body, member.Hero.Reaction,
			member.Hero.Mind, member.Hero.Spirit, member.Hero.Skill, weapon, member.Worn)
	}
	assertIdentity := func(stage string, got mapload.PartyMember) {
		if identity(got) != identity(want) {
			t.Fatalf("%s identity = %s, want %s", stage, identity(got), identity(want))
		}
	}

	app := f.App("generated-character-sheet")
	if err := app.OpenMission(f.MissionOpenerWith(10, generated)); err != nil {
		t.Fatalf("launch generated mission: %v", err)
	}
	if len(f.liveParty) != 1 {
		t.Fatalf("launched party length = %d, want 1", len(f.liveParty))
	}
	assertIdentity("launch", f.liveParty[0])
	// Read the actual first gameplay sheet through the Viewer's selection
	// adapter only after the mission accepts the generated party. This is not a
	// second direct party projection: changing Viewer selection/panel routing
	// changes this oracle with the screen it represents.
	preview := f.ChargenPreview(result)
	if card := ui.RenderCharacterPanel(ui.AuthoredPanelLayout(), f.ChargenAssets.Presentation.Font, preview.Subject); card.Bounds().Size() != image.Pt(300, 286) {
		t.Fatalf("generated native production Card bounds=%v, want 300x286", card.Bounds().Size())
	}
	if err := app.HeadlessSelectEntity(uint32(f.live.mission.ids[0])); err != nil {
		t.Fatalf("select first generated entity through production input: %v", err)
	}
	// This comparison is about the selected character, not the new hover
	// subject. Leaving the map restores the ordinary selection presentation.
	if err := app.HeadlessPointer("hover", -1, -1); err != nil {
		t.Fatalf("leave hover before reading selected character sheet: %v", err)
	}
	statement, ok := f.live.view.PanelStatement()
	if !ok {
		t.Fatal("launched Viewer has no first character-sheet subject")
	}
	got := ui.PanelStatement(ui.CompactPanelLayout(nil), preview.Subject)
	if len(got) != 17 || len(statement) != 17 {
		t.Fatalf("generator/lived sheet line count = %d/%d, want 17 (CompactPanelLayout's own row count, WEIGHT included since story 1025)",
			len(got), len(statement))
	}
	for i := range got {
		if got[i] != statement[i] {
			t.Fatalf("generator Card line %d = %q, launched first-character sheet line = %q", i, got[i], statement[i])
		}
	}

	dir := t.TempDir()
	at := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	save, _, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return at })
	name, err := save(true)
	if err != nil {
		t.Fatalf("save generated mission: %v", err)
	}
	restored := releaseFront(t)
	_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	open, town, err := load(localOriginalSavePrefix + name)
	if err != nil || town || open == nil {
		t.Fatalf("load generated mission = opener %v town %v err %v", open != nil, town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatalf("relaunch generated save: %v", err)
	}
	if len(restored.liveParty) != 1 {
		t.Fatalf("reloaded party length = %d, want 1", len(restored.liveParty))
	}
	assertIdentity("save/load", restored.liveParty[0])

	next, _ := restored.FinishMission(10, restored.liveParty, restored.live.world, restored.live.mission.ids)
	if next <= 0 {
		t.Fatalf("mission 10 supplied no campaign successor")
	}
	if _, _, _, _, _, _, _, _, _, _, err := restored.MissionOpener(next)(); err != nil {
		t.Fatalf("launch campaign successor %d: %v", next, err)
	}
	if len(restored.liveParty) != 1 {
		t.Fatalf("successor party length = %d, want generated hero alone", len(restored.liveParty))
	}
	assertIdentity("campaign successor", restored.liveParty[0])
}

// TestReleaseLoadCandidateKeepsTheLiveMissionUntilCommit is 1008 AC-9 over the
// lawful install named by AGAINROM_ASSETS. It uses the same App controls and
// SaveSeams as the windowed game. The default gate skips it; closure runs it
// once on each shipped language root.
func TestReleaseLoadCandidateKeepsTheLiveMissionUntilCommit(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1008-release-load")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	valid := snap
	stale := snap
	stale.World = append([]byte(nil), stale.World...)
	// Version 65 is now the immediately previous, losslessly migratable form.
	// Use the unwritten version 51 hole so this fixture continues to exercise a
	// candidate the loader must refuse rather than a valid old save.
	stale.World[0] = 51

	store := SaveStore{Dir: t.TempDir()}
	var hostile bytes.Buffer
	if err := gob.NewEncoder(&hostile).Encode(stale); err != nil {
		t.Fatal(err)
	}
	staleBytes := historicalEnvelope(t, hostile.Bytes(), "stale candidate")
	if _, err := store.Write(time.Date(2026, 8, 16, 17, 0, 0, 0, time.UTC), staleBytes); err != nil {
		t.Fatal(err)
	}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	oldLive, oldTown, oldHash := f.live, f.Town, f.live.world.Hash()

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("stale candidate"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || app.HeadlessGameplayScreen() != ui.ScreenMap ||
		f.live != oldLive || f.Town != oldTown || f.live.world.Hash() != oldHash {
		t.Fatalf("stale refusal = screen %v over %v live %v town %v hash %016x",
			app.Screen(), app.HeadlessGameplayScreen(), f.live == oldLive, f.Town == oldTown, f.live.world.Hash())
	}
	if !strings.Contains(strings.ToLower(app.HeadlessMessage()), "too old") {
		t.Fatalf("stale refusal = %q", app.HeadlessMessage())
	}

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	validBytes, err := EncodeSave(valid, "valid candidate")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Write(time.Date(2026, 8, 16, 17, 0, 1, 0, time.UTC), validBytes); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("valid candidate"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.live == nil || f.live == oldLive || f.live.world.Hash() != oldHash {
		t.Fatalf("valid commit = screen %v new live %v hash %016x, want map/true/%016x",
			app.Screen(), f.live != oldLive, f.live.world.Hash(), oldHash)
	}
}

// This is the real owner-facing prologue route: mission 20 finishes, its live
// participant purse is carried to town, the town transition pays the disclosed
// authored reward, npc22 speaks through the tavern seam, and mission 30 starts
// with the two independently identified characters.
func TestReleaseReniestaTavernJoinAndMapKeepBothCharacters(t *testing.T) {
	f := releaseFront(t)
	if got := f.Town.Gold(); got != initialPlayerPurse {
		t.Fatalf("fresh campaign purse = %d, want %d", got, initialPlayerPurse)
	}
	if got := f.Campaign.Value().Chapters[20].Payment; got != 0 {
		t.Fatalf("shipped mission-20 Payment = %d, want absent/zero", got)
	}
	if got := f.Campaign.Value().TransitionRewards[20]; got != firstTownTransitionReward {
		t.Fatalf("authored mission-20 transition reward = %d, want %d", got, firstTownTransitionReward)
	}

	open20 := f.MissionOpener(20)
	if _, _, _, _, _, _, _, _, _, _, err := open20(); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	if len(f.liveParty) != 1 {
		t.Fatalf("mission-20 party = %d, want primary hero alone", len(f.liveParty))
	}
	primaryBefore := f.liveParty[0]
	primaryID := f.live.mission.ids[0]
	primaryEntityBefore := releaseEntity(t, f.live, primaryID)
	primaryEquipmentBefore, _ := f.live.world.Equipped(primaryID)
	npc22Name := f.localizedNPCName(23, "Reniesta")
	if npc22Name == "" || f.Table.Humans.EntryName(29) != "PC_Reniesta" {
		t.Fatalf("lawful npc22 identity = localized %q Humans[29] %q, want a localized display name and PC_Reniesta template",
			npc22Name, f.Table.Humans.EntryName(29))
	}
	_, completion := f.FinishMission(20, f.liveParty, f.live.world, f.live.mission.ids)
	if !strings.Contains(completion, "campaign-transition reward +500 gold") {
		t.Fatalf("completion %q does not expose the authored +500 reward", completion)
	}
	if got := f.Town.Gold(); got != initialPlayerPurse+firstTownTransitionReward {
		t.Fatalf("town purse after mission 20 = %d, want %d", got, initialPlayerPurse+firstTownTransitionReward)
	}
	if len(f.Carried) != 2 {
		t.Fatalf("carried party after mission 20 = %d, want Danath and Reniesta", len(f.Carried))
	}
	if f.Carried[0].Name != primaryBefore.Name || f.Carried[1].Name != npc22Name || f.Carried[1].CompanionNPC != 22 {
		t.Fatalf("joined identities = %q/%q npc%d, want %q/%q npc22",
			f.Carried[0].Name, f.Carried[1].Name, f.Carried[1].CompanionNPC, primaryBefore.Name, npc22Name)
	}
	reniesta := f.Carried[1]
	if reniesta.FigureDir != string(data.FigureDirWomanMage) || !reniesta.Mage {
		t.Fatalf("Reniesta figure/class = %q mage=%v, want female mage", reniesta.FigureDir, reniesta.Mage)
	}
	if reniesta.Worn[0] == 0 || reniesta.Worn[6] == 0 || reniesta.Worn[7] == 0 {
		t.Fatalf("Reniesta starting worn set = %v, want staff, dress, and cloak", reniesta.Worn)
	}
	if reniesta.Weapon == nil || reniesta.Weapon.Defence != 0 {
		t.Fatalf("Reniesta staff = %+v, want a weapon with zero physical defence", reniesta.Weapon)
	}
	dress, err := data.ArmorFromCode(data.ItemCode(reniesta.Worn[6]),
		f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil {
		t.Fatalf("resolve Reniesta dress: %v", err)
	}
	cloak, err := data.ArmorFromCode(data.ItemCode(reniesta.Worn[7]),
		f.Table.Shapes, f.Table.Materials, f.Table.Armors)
	if err != nil {
		t.Fatalf("resolve Reniesta cloak: %v", err)
	}
	if dress.Defence != 11 || dress.Absorption != 0 || cloak.Defence != 4 || cloak.Absorption != 0 {
		t.Fatalf("Reniesta armour dress=%+v cloak=%+v, want defence/absorption 11/0 and 4/0", dress, cloak)
	}
	// The screenshot's DEFENSE 9 was not a lawful bare-mage value: the staff
	// had been given a stale physical contribution. The current staff is zero,
	// Reaction 23 contributes 7, the dress adds 11, and the cloak adds 4. The
	// shipped no-per-slot fold therefore makes the fully dressed sheet 22.
	bareMember := reniesta
	bareMember.Carry = nil
	bareMember.Worn = [sim.EquipSlots]uint16{}
	bareMember.WornItems = [sim.EquipSlots]sim.ItemInstance{}
	dressMember := bareMember
	dressMember.Worn[0], dressMember.Worn[6] = reniesta.Worn[0], reniesta.Worn[6]
	bareDerived, _, _ := mapload.PartySpawnWithTable(bareMember, f.Table)
	dressDerived, _, _ := mapload.PartySpawnWithTable(dressMember, f.Table)
	fullDerived, _, _ := mapload.PartySpawnWithTable(reniesta, f.Table)
	if bareDerived.Combat.Defence != 7 || dressDerived.Combat.Defence != 18 || fullDerived.Combat.Defence != 22 {
		t.Fatalf("Reniesta defence bare/dress/full = %d/%d/%d, want 7/18/22",
			bareDerived.Combat.Defence, dressDerived.Combat.Defence, fullDerived.Combat.Defence)
	}
	if dressDerived.Combat.Defence-bareDerived.Combat.Defence != dress.Defence ||
		fullDerived.Combat.Defence-dressDerived.Combat.Defence != cloak.Defence {
		t.Fatalf("Reniesta armour deltas dress/cloak = %d/%d, want %d/%d",
			dressDerived.Combat.Defence-bareDerived.Combat.Defence,
			fullDerived.Combat.Defence-dressDerived.Combat.Defence, dress.Defence, cloak.Defence)
	}
	full, unread := composeInventorySubject(f.Archives.Containers, 22,
		equipmentFromSlots(reniesta.Worn), data.FigureDir(reniesta.FigureDir), reniesta.FigureFace)
	naked, _ := composeInventorySubject(f.Archives.Containers, 22,
		data.Equipment{}, data.FigureDir(reniesta.FigureDir), reniesta.FigureFace)
	if full.Figure == nil || naked.Figure == nil || len(unread) != 0 {
		t.Fatalf("real Reniesta figure full=%v naked=%v unread=%v", full.Figure != nil, naked.Figure != nil, unread)
	}
	if sha256.Sum256(full.Figure.Pix) == sha256.Sum256(naked.Figure.Pix) {
		t.Fatal("Reniesta's real staff/dress/cloak figure is pixel-identical to her naked figure")
	}

	// Drive the same square -> tavern -> npc22 callbacks the UI invokes.
	town := f.TownScreen().(*townScreen)
	town.atSquare()
	town.Choose(0)
	offers := f.Town.Offers(TownTavern)
	row := -1
	for i, offer := range offers {
		if offer.NPC == 22 {
			row = i
			break
		}
	}
	if row < 0 {
		t.Fatalf("real chapter tavern offers %+v, want npc22", offers)
	}
	town.Choose(row)
	if town.room != roomTalk || town.npc != 22 {
		t.Fatalf("tavern choice opened room=%v npc=%d, want npc22 dialogue", town.room, town.npc)
	}
	if pic, ok := town.TownDialogue(); !ok || pic == nil {
		t.Fatal("real npc22 tavern dialogue has no rendered modal")
	}
	spoken := town.resolver.playerFigures[22]
	if spoken == nil || sha256.Sum256(spoken.Pix) != sha256.Sum256(full.Figure.Pix) {
		t.Fatal("npc22 tavern speaker is not Reniesta's equipped figure")
	}

	partyAtGate := append([]mapload.PartyMember(nil), f.Carried...)
	open30 := f.MissionOpener(30)
	_, pace, _, _, _, _, _, _, _, _, err := open30()
	if err != nil {
		t.Fatalf("open mission 30: %v", err)
	}
	if len(f.liveParty) != len(partyAtGate) {
		t.Fatalf("opening mission 30 changed party size from %d to %d", len(partyAtGate), len(f.liveParty))
	}
	for i, before := range partyAtGate {
		actual := f.liveParty[i]
		if actual.ID != before.ID || actual.CompanionNPC != before.CompanionNPC {
			t.Fatalf("mission-30 party member %d changed identity: before %q/%d, after %q/%d",
				i, before.ID, before.CompanionNPC, actual.ID, actual.CompanionNPC)
		}
		after := actual
		if before.Carry == nil && actual.Carry != nil {
			// A freshly joined companion's Carry is a materialized projection; verify first.
			carry := actual.Carry
			if carry.SkillXP != before.Hero.Reward().SkillXP {
				t.Fatalf("mission-30 member %d materialized carry XP %v, want the city hero projection %v",
					i, carry.SkillXP, before.Hero.Reward().SkillXP)
			}
			items := mapload.MemberCarriedItems(before, f.Table)
			if len(carry.Items) != len(items) || len(carry.ItemInstances) != len(items) {
				t.Fatalf("mission-30 member %d materialized carry item counts %d/%d, want %d",
					i, len(carry.Items), len(carry.ItemInstances), len(items))
			}
			for item := range items {
				if carry.Items[item] != items[item].Code || !reflect.DeepEqual(carry.ItemInstances[item], items[item]) {
					t.Fatalf("mission-30 member %d materialized carry item %d differs from the city member", i, item)
				}
			}
			if carry.Equipped != before.Worn ||
				!reflect.DeepEqual(carry.EquippedItems, mapload.MemberItemEquipment(before, f.Table)) {
				t.Fatalf("mission-30 member %d materialized carry equipment differs from the city member", i)
			}
			if carry.LiveLoad != nil || len(carry.OrderedStacks) != 0 {
				t.Fatalf("mission-30 member %d materialized unowned live carry state: load=%v stacks=%d",
					i, carry.LiveLoad != nil, len(carry.OrderedStacks))
			}
			after.Carry = nil
		}
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("opening mission 30 changed party member %d beyond the verified Carry projection", i)
		}
	}
	if len(f.live.mission.ids) != 2 {
		t.Fatalf("mission-30 ids = %v, want two", f.live.mission.ids)
	}
	danathID, reniestaID := f.live.mission.ids[0], f.live.mission.ids[1]
	danathEntity := releaseEntity(t, f.live, danathID)
	reniestaEntity := releaseEntity(t, f.live, reniestaID)
	if reniestaEntity.Defence != fullDerived.Combat.Defence || reniestaEntity.Absorption != fullDerived.Combat.Absorption {
		t.Fatalf("Reniesta live combat defence/absorption = %d/%d, want full loadout %d/%d",
			reniestaEntity.Defence, reniestaEntity.Absorption,
			fullDerived.Combat.Defence, fullDerived.Combat.Absorption)
	}
	if danathEntity.Skill != primaryEntityBefore.Skill || danathEntity.SkillXP != primaryEntityBefore.SkillXP {
		t.Fatalf("primary state crossed identity at map start: skill/xp %v/%v, want %v/%v",
			danathEntity.Skill, danathEntity.SkillXP, primaryEntityBefore.Skill, primaryEntityBefore.SkillXP)
	}
	if reniestaEntity.Skill != reniesta.Hero.Skill || reniestaEntity.Skill == danathEntity.Skill || reniestaEntity.SkillXP == danathEntity.SkillXP {
		t.Fatalf("Reniesta state = skill/xp %v/%v; Danath = %v/%v",
			reniestaEntity.Skill, reniestaEntity.SkillXP, danathEntity.Skill, danathEntity.SkillXP)
	}
	danathEquipment, _ := f.live.world.Equipped(danathID)
	reniestaEquipment, _ := f.live.world.Equipped(reniestaID)
	if danathEquipment != primaryEquipmentBefore || reniestaEquipment != reniesta.Worn {
		t.Fatalf("map-start equipment Danath=%v Reniesta=%v, want %v/%v",
			danathEquipment, reniestaEquipment, primaryEquipmentBefore, reniesta.Worn)
	}
	if releaseArtName(f.live, danathID) == "" || releaseArtName(f.live, reniestaID) == "" {
		t.Fatalf("map-start art Danath=%q Reniesta=%q", releaseArtName(f.live, danathID), releaseArtName(f.live, reniestaID))
	}
	var reniestaDraw *ui.MapEntity
	for _, draw := range f.live.entityDraws() {
		if draw.ID == uint32(reniestaID) {
			copy := draw
			reniestaDraw = &copy
			break
		}
	}
	if reniestaDraw == nil {
		t.Fatalf("mission-30 draw stream has no Reniesta entity %d", reniestaID)
	}
	sheet := strings.Join(ui.PanelStatement(ui.AuthoredPanelLayout(), ui.PanelSubject{
		ID: reniestaDraw.ID, Name: reniestaDraw.Name,
		HP: reniestaDraw.HP, MaxHP: reniestaDraw.MaxHP,
		Mana: reniestaDraw.Mana, MaxMana: reniestaDraw.MaxMana,
		Cell: reniestaDraw.Cell, Selected: 1,
		Combat: reniestaDraw.Combat, Char: reniestaDraw.Char, Speed: reniestaDraw.Speed,
	}), "\n")
	if !strings.Contains(sheet, "WATER 10") || strings.Contains(sheet, "AXE 10") {
		t.Fatalf("Reniesta live panel skill mapping is wrong:\n%s", sheet)
	}
	if !strings.Contains(sheet, "DEFENSE 22") {
		t.Fatalf("Reniesta live panel did not fold dress and cloak defence:\n%s", sheet)
	}

	// The worn-cell tooltip must name the exact interval the selected live
	// entity would release through the staff. This crosses the production
	// presenter and the headless observation surface; neither may fall back to
	// the staff row's unused physical 0-0.
	f.live.switchInventorySubject(uint32(reniestaID))
	spellBase, spellSpread, ok := f.live.world.WeaponSpellDamage(reniestaID)
	if !ok {
		t.Fatal("Reniesta's equipped Wood Staff has no live weapon-spell interval")
	}
	wantStaffDamage := fmt.Sprintf("%s %d-%d", f.Words.ItemDamage, spellBase, spellBase+spellSpread)
	hasLine := func(lines []string, want string) bool {
		for _, line := range lines {
			if line == want {
				return true
			}
		}
		return false
	}
	staffInfo := f.live.invSubject.SlotInfo[0]
	magicLine := roomCaptionRawLine(t, roomCaptionRaw(t, f, "main/text/main.txt"), 189)
	var reniestaSpell sim.SpellRule
	reniestaSpellFound := false
	for _, entity := range f.live.world.Entities() {
		if entity.ID == reniestaID {
			reniestaSpell, reniestaSpellFound = f.live.world.Spell(uint32(entity.WeaponSpell))
			break
		}
	}
	reniestaSpellName := f.Table.Spells.EntryName(int(reniestaSpell.ID))
	if int(reniestaSpell.ID) < len(f.Words.ItemSpellNames) && f.Words.ItemSpellNames[reniestaSpell.ID] != "" {
		reniestaSpellName = f.Words.ItemSpellNames[reniestaSpell.ID]
	}
	wantStaffCast := fmt.Sprintf("%s %s", f.Words.ItemCasts, reniestaSpellName)
	wantStaffRange := fmt.Sprintf("%s %d", f.Words.ItemRange, reniestaSpell.MaxRange)
	if !reniestaSpellFound || !hasLine(staffInfo, magicLine) || !hasLine(staffInfo, wantStaffCast) ||
		!hasLine(staffInfo, wantStaffDamage) || !hasLine(staffInfo, wantStaffRange) ||
		hasLine(staffInfo, f.Words.ItemDamage+" 0-0") {
		t.Fatalf("Reniesta Wood Staff tooltip = %v, want Magic/%q/%q/%q and no physical 0-0",
			staffInfo, wantStaffCast, wantStaffDamage, wantStaffRange)
	}
	authoredWords := ui.AuthoredWords()
	headlessCast := fmt.Sprintf("%s %s", authoredWords.ItemCasts, f.Table.Spells.EntryName(int(reniestaSpell.ID)))
	headlessDamage := fmt.Sprintf("%s %d-%d", authoredWords.ItemDamage, spellBase, spellBase+spellSpread)
	headlessRange := fmt.Sprintf("%s %d", authoredWords.ItemRange, reniestaSpell.MaxRange)
	headless := f.HeadlessSnapshot(ui.ScreenMap)
	headlessReniesta, err := findHeadlessMember(headless, "npc:22")
	if err != nil {
		t.Fatal(err)
	}
	if len(headlessReniesta.Worn) == 0 || !hasLine(headlessReniesta.Worn[0].Info, "MAGIC:") ||
		!hasLine(headlessReniesta.Worn[0].Info, headlessCast) ||
		!hasLine(headlessReniesta.Worn[0].Info, headlessDamage) ||
		!hasLine(headlessReniesta.Worn[0].Info, headlessRange) ||
		hasLine(headlessReniesta.Worn[0].Info, authoredWords.ItemDamage+" 0-0") {
		t.Fatalf("headless Reniesta worn snapshot = %+v, want Magic/%q/%q/%q",
			headlessReniesta.Worn, headlessCast, headlessDamage, headlessRange)
	}

	// A non-spell weapon stays on the physical item interval. The magic branch
	// is an override for weapon-borne spells, not a new interpretation of all
	// weapon tooltips.
	f.live.switchInventorySubject(uint32(danathID))
	if _, _, ok := f.live.world.WeaponSpellDamage(danathID); ok {
		t.Fatal("Danath's ordinary weapon unexpectedly resolved a live weapon spell")
	}
	danathWeapon, err := data.WeaponFromCode(data.ItemCode(danathEquipment[0]),
		f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
	if err != nil {
		t.Fatalf("resolve Danath weapon: %v", err)
	}
	wantPhysical := fmt.Sprintf("#%s: %d-%d", f.Words.ItemStats[43], danathWeapon.DamageBase,
		danathWeapon.DamageBase+danathWeapon.DamageSpread)
	if got := f.live.invSubject.SlotInfo[0]; !hasLine(got, wantPhysical) {
		t.Fatalf("Danath weapon tooltip = %v, want physical interval %q", got, wantPhysical)
	}

	beforeSelection, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal before selection: %v", err)
	}
	for _, id := range []sim.EntityID{reniestaID, danathID} {
		f.live.switchInventorySubject(uint32(id))
		f.live.refreshPack()
		f.live.refreshEquipment()
		f.live.refreshAppearance()
		if f.live.invSubject.Figure == nil {
			t.Fatalf("selected party id %d has no inventory doll", id)
		}
	}
	if got := inventoryGold(f.live.invSubject); got != uint32(initialPlayerPurse+firstTownTransitionReward) {
		t.Fatalf("primary inventory gold = %d, want %d", got, initialPlayerPurse+firstTownTransitionReward)
	}
	f.live.switchInventorySubject(uint32(reniestaID))
	if got := inventoryGold(f.live.invSubject); got != 0 {
		t.Fatalf("Reniesta inventory exposes campaign gold %d, want primary-only", got)
	}
	if pace != nil {
		pace()
	}
	afterSelection, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal after selection: %v", err)
	}
	if !bytes.Equal(beforeSelection, afterSelection) {
		t.Fatal("tavern join, map start, selection, or first presentation frame changed simulation state")
	}
}

// This release-data witness keeps the two selection boundaries distinct. A
// temporary member that is actually in the mission party gets its own composed
// figure, worn set, and pack; a map actor outside that parallel party/id list
// supplies only its own picture (or current frame) and cannot replace the
// party's inventory subject.
func TestReleaseTemporaryPartyAndNonPartyKeepTheirOwnPresentation(t *testing.T) {
	f := releaseFront(t)
	open20 := f.MissionOpener(20)
	if _, _, _, _, _, _, _, _, _, _, err := open20(); err != nil {
		t.Fatalf("open mission 20: %v", err)
	}
	f.FinishMission(20, f.liveParty, f.live.world, f.live.mission.ids)
	if len(f.Carried) != 2 {
		t.Fatalf("party after mission 20 = %d, want primary plus npc22", len(f.Carried))
	}

	temporary := f.Carried[1]
	temporary.Name += " (temporary selection witness)"
	temporary.PlayerCharacter = false
	temporary.StartingHero = false
	temporary.CompanionNPC = 0
	// AND THE MARKER IS Temporary, NOT MercenaryType (owner). It is a real
	// field with real consumers, and each time one arrived it changed what this
	// fixture was building.
	//
	// Temporary is the field this test's own doc names, and it carries no
	// consumer that could take it somewhere else: PartyMember documents it as
	// membership metadata, saying only whether campaign continuity keeps the
	// membership. The two selection boundaries this witness keeps distinct are
	// IN THE MISSION PARTY versus NOT IN IT, which is what it always was;
	// nothing here was ever about being hired.
	temporary.Temporary = true
	temporary.MercenaryType = 0
	temporary.Carry = nil
	temporary.Saved = nil
	temporary.Carried = []uint16{temporary.Worn[0]}
	f.Carried = append(f.Carried, temporary)

	open30 := f.MissionOpener(30)
	if _, _, _, _, _, _, _, _, _, _, err := open30(); err != nil {
		t.Fatalf("open mission 30 with temporary party member: %v", err)
	}
	if len(f.live.mission.ids) != 3 {
		t.Fatalf("mission party ids = %v, want three", f.live.mission.ids)
	}
	temporaryID := f.live.mission.ids[2]
	before, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal before temporary selection: %v", err)
	}
	f.live.switchInventorySubject(uint32(temporaryID))
	if f.live.invSubject.ID != uint32(temporaryID) || f.live.invSubject.Figure == nil {
		t.Fatalf("temporary party subject = id%d figure=%v, want its own composed figure",
			f.live.invSubject.ID, f.live.invSubject.Figure != nil)
	}
	if got := f.live.currentEquipment(); got != equipmentFromSlots(temporary.Worn) {
		t.Fatalf("temporary party worn = %v, want %v", got, temporary.Worn)
	}
	if got := f.live.invCodes; len(got) != 1 || got[0].Code != temporary.Worn[0] || got[0].Count != 1 {
		t.Fatalf("temporary party pack = %v, want its own item %#04x", got, temporary.Worn[0])
	}
	if got := inventoryGold(f.live.invSubject); got != 0 {
		t.Fatalf("temporary party subject displays campaign gold %d", got)
	}
	afterTemporary, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal after temporary selection: %v", err)
	}
	if !bytes.Equal(before, afterTemporary) {
		t.Fatal("temporary party selection changed simulation state")
	}

	partyIDs := make(map[sim.EntityID]bool, len(f.live.mission.ids))
	for _, id := range f.live.mission.ids {
		partyIDs[id] = true
	}
	nonPartyID := sim.EntityID(0)
	nonPartyHasPicture := false
	for _, draw := range f.live.entityDraws() {
		id := sim.EntityID(draw.ID)
		if partyIDs[id] {
			continue
		}
		e := releaseEntity(t, f.live, id)
		pic := f.live.unitPicture(id, e.Class)
		if pic != nil || draw.Frame != nil {
			nonPartyID, nonPartyHasPicture = id, true
			break
		}
	}
	if !nonPartyHasPicture {
		t.Fatal("real mission 30 has no non-party actor with a portrait/figure or current-frame fallback")
	}
	f.live.switchInventorySubject(uint32(nonPartyID))
	if f.live.invSubject.ID != uint32(temporaryID) {
		t.Fatalf("non-party selection replaced inventory subject with id %d, want temporary party id %d retained",
			f.live.invSubject.ID, temporaryID)
	}
	afterNonParty, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal after non-party selection: %v", err)
	}
	if !bytes.Equal(before, afterNonParty) {
		t.Fatal("non-party presentation lookup changed simulation state")
	}
}

// The owner's two live Naira saves exercise original-save decode, restored sex,
// real art lookup, mission start, and the first presentation frame. The save
// directory is explicit because lawful installs are never test fixtures.
func TestReleaseOriginalNairaSavesKeepFemaleAppearance(t *testing.T) {
	saves := os.Getenv("AGAINROM_ORIGINAL_SAVES")
	if saves == "" {
		t.Skip("no AGAINROM_ORIGINAL_SAVES: Naira save integration not requested")
	}
	for _, name := range []string{"game0013.sav", "game0014.sav"} {
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			payload, err := os.ReadFile(filepath.Join(saves, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			label, err := OriginalSaveLabel(payload, 0)
			if err != nil || !strings.Contains(strings.ToLower(label), "naira") {
				t.Fatalf("OriginalSaveLabel(%s) = %q, %v; want Naira", name, label, err)
			}
			open, town, err := f.RestoreOriginal(payload)
			if err != nil || town || open == nil {
				t.Fatalf("RestoreOriginal(%s) = town %v, opener %v, err %v", name, town, open != nil, err)
			}
			_, pace, _, _, _, _, _, _, _, _, err := open()
			if err != nil {
				t.Fatalf("open %s: %v", name, err)
			}
			if len(f.liveParty) != 1 {
				t.Fatalf("%s party = %d, want one Naira", name, len(f.liveParty))
			}
			p := f.liveParty[0]
			if p.Name != "Naira" || p.FigureDir != string(data.FigureDirWomanFighter) || p.FigureFace != 1 {
				t.Fatalf("%s identity/figure = %q %q face%d, want Naira ffighter face1", name, p.Name, p.FigureDir, p.FigureFace)
			}
			figure, unread := composeInventorySubject(f.Archives.Containers, 1,
				equipmentFromSlots(p.Worn), data.FigureDir(p.FigureDir), p.FigureFace)
			if figure.Figure == nil || len(unread) != 0 {
				t.Fatalf("%s Naira doll nil=%v unread=%v", name, figure.Figure == nil, unread)
			}
			id := f.live.mission.ids[0]
			artBefore := releaseArtName(f.live, id)
			stateBefore, _ := f.live.world.MarshalBinary()
			pace()
			stateAfter, _ := f.live.world.MarshalBinary()
			if artBefore == "" || releaseArtName(f.live, id) != artBefore {
				t.Fatalf("%s first frame changed art %q -> %q", name, artBefore, releaseArtName(f.live, id))
			}
			if !bytes.Equal(stateBefore, stateAfter) {
				t.Fatalf("%s first presentation frame changed simulation state", name)
			}
		})
	}
}

// The owner's save-666 is a lawful mid-mission file with seven spent trigger
// latches and a populated directional diplomacy matrix. This drives the real
// Load Game path up to the first frame and compares the canonical world with
// the original save accessors before any gameplay tick can rewrite either.
func TestReleaseOriginalSave666RestoresTriggerLatchesAndDiplomacyBeforeFirstTick(t *testing.T) {
	path := os.Getenv("AGAINROM_SAVE_666")
	if path == "" {
		t.Skip("no AGAINROM_SAVE_666: original session integration not requested")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read save 666: %v", err)
	}
	sf, err := sav.Open(payload)
	if err != nil {
		t.Fatalf("decode save 666: %v", err)
	}
	expected, err := sf.SessionState()
	if err != nil {
		t.Fatalf("save 666 session: %v", err)
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(payload)
	if err != nil || town || open == nil {
		t.Fatalf("RestoreOriginal = town %v, opener %v, err %v", town, open != nil, err)
	}
	_, tick, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatalf("open save 666: %v", err)
	}
	w := f.live.world
	savedTick := rawSavedSubTick1112(t, payload)
	if w.Tick() != savedTick {
		t.Fatalf("the original session first became observable at tick %d, want saved tick %d", w.Tick(), savedTick)
	}

	set := 0
	for i, want := range expected.TriggerLatches {
		if want != 0 {
			set++
		}
		if got := w.ScriptLatched(int32(i)); got != (want != 0) {
			t.Fatalf("trigger latch %d = %v, save carries %d", i, got, want)
		}
	}
	if set == 0 {
		t.Fatal("save 666 carries no spent trigger latch, so it cannot witness continuity")
	}

	// Slot zero names no roster entry in sim and therefore has no semantic
	// accessor. The 49x49 playable population is compared byte-for-byte here;
	// the sim transactional test separately proves the full 2500-byte copy is
	// canonical and survives caller mutation.
	nonzero := 0
	relations := w.Relations()
	for from := 1; from < 50; from++ {
		for to := 1; to < 50; to++ {
			want := expected.Diplomacy[from*50+to]
			if want != 0 {
				nonzero++
			}
			if got := relations.Byte(uint32(from), uint32(to)); got != want {
				t.Fatalf("diplomacy[%d][%d] = %#x, save carries %#x", from, to, got, want)
			}
		}
	}
	if nonzero == 0 {
		t.Fatal("save 666 carries no nonzero playable diplomacy cell, so it cannot witness continuity")
	}

	// NewAnnouncer already exists here. Every saved latch that owns a message
	// raise must be seeded as seen; otherwise the first script pass would report
	// an event that happened before the save.
	seeded := 0
	for _, raise := range f.live.mission.state.Raises {
		if raise.Latch < 0 || int(raise.Latch) >= len(expected.TriggerLatches) ||
			expected.TriggerLatches[raise.Latch] == 0 {
			continue
		}
		seeded++
		if !f.live.mission.ann.prev[raise.Latch] {
			t.Fatalf("announcer did not seed saved message latch %d", raise.Latch)
		}
	}
	if seeded == 0 {
		t.Fatal("save 666 has no saved message latch, so it cannot witness announcer ordering")
	}
	// This is the observable half of the ordering assertion. Seeding an
	// internal array is insufficient evidence unless the real map tick then
	// stays silent where the pre-1066 load repeated the already-fired message.
	for i := 0; i < 64; i++ {
		tick()
	}
	if text, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
		t.Fatalf("saved Victory must not replay old dialogue: %q/%v", text, kind)
	}
	t.Logf("save 666 restored %d spent latches, %d nonzero playable diplomacy cells; "+
		"%d saved message latches seeded at saved tick %d; no replay through %d direct driver ticks",
		set, nonzero, seeded, savedTick, w.Tick()-savedTick)
}

func TestReleaseABetweenMissionSaveStartsAtTheMapsOwnDrop(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: between-mission placement integration not requested")
	}
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: between-mission placement integration not requested")
	}
	var paths []string
	if err := filepath.WalkDir(corpus, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip := corpusDirSkip(d); skip != nil {
			return skip
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".sav") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", corpus, err)
	}
	sort.Strings(paths)
	town := 0
	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		sf, err := sav.Open(payload)
		if err != nil || sf.Head.Mission != 0 {
			continue
		}
		town++
		name := filepath.Base(filepath.Dir(path)) + "/" + filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			chars, err := sf.Party()
			if err != nil || len(chars) == 0 {
				t.Skipf("%s decodes no characters: %v", name, err)
			}
			refused := make(map[[2]int32]string, len(chars))
			for _, c := range chars {
				refused[[2]int32{int32(c.Col()), int32(c.Row())}] = c.Name
			}

			f := releaseFront(t)
			open, isTown, err := f.RestoreOriginal(payload)
			if err != nil || !isTown || open != nil {
				t.Fatalf("RestoreOriginal = town %v, opener %v, err %v", isTown, open != nil, err)
			}
			n := f.Town.selectedMission()
			if _, ok := MissionMap(n); !ok {
				t.Skipf("the restored campaign selects mission %d, which is not a campaign map", n)
			}
			ms, err := StartMission(f.Archives.Containers, n, f.Table, mapload.DifficultyNormal, f.Carried)
			if err != nil {
				t.Fatalf("mission %d: %v", n, err)
			}
			t.Logf("%s: %d characters, mission %d, drop %v (authorised %d, fallback %v)",
				name, len(chars), n, ms.Start.Drop, ms.Start.Authorised, ms.Start.Fallback)
			for i, c := range ms.Start.Cells {
				if who, ok := refused[[2]int32{c.X, c.Y}]; ok {
					t.Errorf("member %d (%q) stands at %v, the cell the save recorded in the mission he left",
						i, who, c)
				}
				dx, dy := c.X-ms.Start.Drop.X, c.Y-ms.Start.Drop.Y
				if dx*dx+dy*dy > 100 {
					t.Errorf("member %d stands at %v, %d cells from the map's own drop %v",
						i, c, dx*dx+dy*dy, ms.Start.Drop)
				}
			}
		})
	}
	t.Logf("between-mission saves in %s: %d of %d .sav files", corpus, town, len(paths))
	if town == 0 {
		t.Fatalf("no between-mission save in %s: the corpus holds %d .sav files and none has mission 0",
			corpus, len(paths))
	}
}

// The named owner save is a separate required regression from the two Naira
// saves. It guards the exact original-load -> map-start -> first-frame boundary
// that previously replaced the restored body with a freshly derived one.
func TestReleaseOriginalSave666KeepsItsRestoredAppearance(t *testing.T) {
	path := os.Getenv("AGAINROM_SAVE_666")
	if path == "" {
		t.Skip("no AGAINROM_SAVE_666: save-666 integration not requested")
	}
	f := releaseFront(t)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read save 666: %v", err)
	}
	label, err := OriginalSaveLabel(payload, 0)
	if err != nil || !strings.Contains(label, "666") {
		t.Fatalf("OriginalSaveLabel = %q, %v; want label 666", label, err)
	}
	open, town, err := f.RestoreOriginal(payload)
	if err != nil || town || open == nil {
		t.Fatalf("RestoreOriginal = town %v, opener %v, err %v", town, open != nil, err)
	}
	_, pace, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		t.Fatalf("open save 666: %v", err)
	}
	if len(f.liveParty) != 1 || len(f.live.mission.ids) != 1 {
		t.Fatalf("save 666 party/ids = %d/%d, want 1/1", len(f.liveParty), len(f.live.mission.ids))
	}
	id := f.live.mission.ids[0]
	memberBefore := f.liveParty[0]
	entityBefore := releaseEntity(t, f.live, id)
	equipmentBefore, _ := f.live.world.Equipped(id)
	artBefore := releaseArtName(f.live, id)
	stateBefore, _ := f.live.world.MarshalBinary()
	pace()
	stateAfter, _ := f.live.world.MarshalBinary()
	if !reflect.DeepEqual(f.liveParty[0], memberBefore) || releaseEntity(t, f.live, id) != entityBefore {
		t.Fatal("save 666 first frame changed restored party or entity identity/state")
	}
	equipmentAfter, _ := f.live.world.Equipped(id)
	if equipmentAfter != equipmentBefore || artBefore == "" || releaseArtName(f.live, id) != artBefore {
		t.Fatalf("save 666 first frame equipment/art = %v/%q, want %v/%q",
			equipmentAfter, releaseArtName(f.live, id), equipmentBefore, artBefore)
	}
	if !bytes.Equal(stateBefore, stateAfter) {
		t.Fatal("save 666 first presentation frame changed simulation byte state")
	}
	// The four excluded mission-20 allies remain ordinary map actors. Selecting
	// one must have a picture source, but it must not gain a party inventory or
	// replace the restored primary's subject.
	nonPartyID := sim.EntityID(0)
	nonPartyFound := false
	for _, draw := range f.live.entityDraws() {
		candidate := sim.EntityID(draw.ID)
		if candidate == id {
			continue
		}
		e := releaseEntity(t, f.live, candidate)
		if f.live.unitPicture(candidate, e.Class) != nil || draw.Frame != nil {
			nonPartyID, nonPartyFound = candidate, true
			break
		}
	}
	if !nonPartyFound {
		t.Fatal("save 666 has no non-party actor with its own picture/current-frame presentation")
	}
	f.live.switchInventorySubject(uint32(nonPartyID))
	if f.live.invSubject.ID != uint32(id) {
		t.Fatalf("save-666 non-party actor %d replaced primary inventory subject with %d",
			nonPartyID, f.live.invSubject.ID)
	}
	stateAfterNonParty, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal after save-666 non-party presentation: %v", err)
	}
	if !bytes.Equal(stateBefore, stateAfterNonParty) {
		t.Fatal("save-666 non-party presentation lookup changed simulation byte state")
	}
}

func corpusDirSkip(d fs.DirEntry) error {
	if d.IsDir() && strings.HasPrefix(d.Name(), "exp") && strings.Contains(d.Name(), "-") {
		return filepath.SkipDir
	}
	return nil
}
