package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A placed person's information portrait draws the horse under a rider whose
// own type id lies in the horse band (HERO-DOLL-078), and a scenario-placed
// human's drawn class is his row's own type id (UNIT-APPEAR-030): a Lancer rides
// whatever his roster template carries. The template's class is not the drawn
// class. The equipment law rewrites it to a body class when the mission opens
// and a SAV load replaces it with the class the entity was placed with, and a
// template may carry a body name besides. A person whose row is no rider stays
// unmounted under a template that names a rider's class, and an exact Hero NPC
// placement, which takes the player-character band (ALM-CLS-054), shows the
// hero background and no horse.
func TestPlacedPersonPortraitDrawsHorseOnlyForAHorseTypeID(t *testing.T) {
	const (
		lancerType, lancerFace, lancerServer    = 21, 2, 208
		footmanType, footmanFace, footmanServer = 3, 5, 209
		paladinType, paladinFace, paladinServer = 19, 1, 777
		heroFace                                = 5
		mapSide                                 = 128
	)
	row := func(name string, typeID, face, serverID int32) dbEntry {
		p := humansParams(20, 0, face)
		p[16], p[18], p[24] = typeID, 0, serverID
		return dbEntry{name: name, params: p}
	}
	npcReg, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		{Name: "npc25", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Human"},
			{Name: "DataBinID", Kind: kindInt, Int: paladinServer},
		}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	table := &mapload.Table{
		Humans: dbCollection{{},
			row("Lancer", lancerType, lancerFace, lancerServer),
			row("Footman", footmanType, footmanFace, footmanServer),
			row("Paladin", paladinType, paladinFace, paladinServer)},
		NPC: data.LoadNPCDefs(npcReg),
	}
	const lancer, footman, paladin sim.EntityID = 0, 1, 2
	m := &alm.Map{Width: mapSide, Height: mapSide,
		Tiles: make([]uint16, mapSide*mapSide), Overlay: make([]uint8, mapSide*mapSide),
		Units: []alm.Unit{
			{X: 0x0C80, Y: 0x0C80, ClassID: lancerType, DefID: lancerServer, UnitID: 1, Owner: 2},
			{X: 0x0E80, Y: 0x0C80, ClassID: footmanType, DefID: footmanServer, UnitID: 2, Owner: 2},
			{X: 0x1080, Y: 0x0C80, ClassID: paladinType, ClassSubID: 25, Flags: 1, UnitID: 3, Owner: 2},
		}}
	party := []mapload.PartyMember{{ID: "hero", Name: "Hero", StartingHero: true, PlayerCharacter: true,
		Class: 1, FigureDir: string(data.FigureDirManFighter), FigureFace: heroFace}}

	// Every body sheet paints only pixel (0,0). The horse and the hero
	// background paint all four pixels, so pixel (1,1) shows the horse, the
	// background or nothing.
	body, horse, background := color.RGBA{R: 0xff, A: 0xff}, color.RGBA{G: 0xff, A: 0xff}, color.RGBA{B: 0xff, A: 0xff}
	sheet := invSparseSheet(color.RGBA{R: 0xff}, 0)
	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, lancerFace):  sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, footmanFace): sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, paladinFace): sheet,
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, heroFace):    sheet,
		graphicsPrefix + "infowindow/horse.bmp":                                         synthBMP(2, 2, horse),
		graphicsPrefix + "interface/heroback/backm.256":                                 invSparseSheet(color.RGBA{B: 0xff}, 0, 1, 2, 3),
	}

	// The roster templates a mission carries for the three placements: what the
	// equipment law leaves at the mission's opening, what a SAV load leaves (the
	// class each entity was placed with, the body kept), and a template that
	// names a rider's class and no body at all.
	rowClass := map[sim.EntityID]int32{lancer: lancerType, footman: footmanType, paladin: paladinType}
	for _, tc := range []struct {
		when  string
		class func(sim.EntityID) int32
		body  string
	}{
		{"fresh mission, the equipment law's body class", func(sim.EntityID) int32 { return 13 }, "pikeman_"},
		{"mission loaded from a SAV, the placement class", func(id sim.EntityID) int32 { return rowClass[id] }, "pikeman_"},
		{"a template of a rider's class and no body", func(sim.EntityID) int32 { return paladinType }, ""},
	} {
		world, start, err := mapload.StartMission(m, table, mapload.DifficultyNormal, party)
		if err != nil {
			t.Fatal(err)
		}
		for id, p := range start.Roster {
			p.Class, p.Body = tc.class(id), tc.body
			start.Roster[id] = p
		}
		v, err := ui.NewViewer("placed person horse", terrain.Grid{Width: mapSide, Height: mapSide,
			Tiles: m.Tiles}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		ms := &Mission{Number: 71, Map: m, World: world, Start: start, Party: mapload.OwnParty(party)}
		live := openMission(ms, table, nil, v, src, nil, nil)

		for _, c := range []struct {
			name   string
			id     sim.EntityID
			hero   bool
			behind color.RGBA
		}{
			{"Lancer", lancer, false, horse},
			{"footman", footman, false, color.RGBA{}},
			{"Hero NPC placement on a rider's row", paladin, true, background},
		} {
			e, ok := live.entity(c.id)
			if !ok || data.FigureIsHero(e.TypeID) != c.hero || (!c.hero && e.TypeID != rowClass[c.id]) {
				t.Fatalf("%s: %s discriminator changed: present %t, type id %#x", tc.when, c.name, ok, e.TypeID)
			}
			for _, route := range []struct {
				name string
				pic  *image.RGBA
			}{
				{"hovered", live.inspectionUnitPicture(uint32(c.id))},
				{"selected", live.unitPicture(c.id, e.Class)},
			} {
				if route.pic == nil {
					t.Fatalf("%s: %s has no %s portrait", tc.when, c.name, route.name)
				}
				if got := route.pic.RGBAAt(0, 0); got != body {
					t.Fatalf("%s: %s %s portrait lost its body: %v", tc.when, c.name, route.name, got)
				}
				if got := route.pic.RGBAAt(1, 1); got != c.behind {
					t.Errorf("%s: %s (type id %#x, template class %d, body %q) %s portrait behind the body = %v, want %v",
						tc.when, c.name, e.TypeID, tc.class(c.id), tc.body, route.name, got, c.behind)
				}
			}
		}
	}
}
