package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestSpeakerPortraitTypeWindow(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 160, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 160; x++ {
			picture.SetRGBA(x, y, color.RGBA{R: uint8(x + 1), G: uint8(y + 1), B: 32, A: 255})
		}
	}
	for _, tc := range []struct {
		name        string
		game        base.Game
		class       int32
		registry    bool
		defaultOnly bool
		want        image.Rectangle
	}{
		{"ogre", base.GameROM2, 66, false, false, image.Rect(10, 44, 82, 140)},
		{"troll", base.GameROM2, 68, false, false, image.Rect(15, 21, 87, 117)},
		{"dragon", base.GameROM2, 71, false, false, image.Rect(7, 49, 79, 145)},
		{"sentinel Y", base.GameROM2, 67, false, false, image.Rect(36, 8, 108, 100)},
		{"below table", base.GameROM2, 63, false, false, image.Rect(36, 8, 108, 100)},
		{"above table", base.GameROM2, 103, false, false, image.Rect(36, 8, 108, 100)},
		{"registry", base.GameROM2, 68, true, false, image.Rect(40, 10, 112, 106)},
		{"first game creature", base.GameROM1, 68, true, false, image.Rect(40, 10, 112, 106)},
		{"first game default", base.GameROM1, 68, true, true, image.Rect(36, 8, 108, 100)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params := make([]int32, 56)
			for i := range params {
				params[i] = -1
			}
			params[29], params[30], params[55] = tc.class, 1, 623
			table := readUnder(tc.game, &mapload.Table{Units: &fixtureCollection{
				names: []string{"", "Creature"}, params: [][]int32{nil, params},
			}})
			if tc.registry {
				table.NPCCatalog = &data.NPCCatalog{Faces: map[int32]data.NPCFace{
					623: {Kind: data.NPCPortrait, Class: tc.class, Face: 1, HasWindow: !tc.defaultOnly, WindowX: 40, WindowY: 10},
				}}
			}
			r := speakerResolver{npcFaces: speakerFaces(table),
				units:     &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{tc.class: {Portrait: "Creature"}}},
				portraits: map[string]*image.RGBA{"graphics/infowindow/Creature.bmp": picture},
			}
			pic, window, ok := r.SpeakerFace(623)
			if !ok || pic != picture {
				t.Fatal("speaker lost its class picture")
			}
			if got := ui.AuthoredDialogueLayout().WithFaceWindow(window).PortraitWindow; got != tc.want {
				t.Errorf("source window = %v, want %v", got, tc.want)
			}
			pane := ui.SpeakerPortrait(pic, window)
			for y := 0; y < 96; y++ {
				for x := 0; x < 96; x++ {
					var want color.RGBA
					if x >= 12 && x < 84 && y < tc.want.Dy() {
						want = picture.RGBAAt(tc.want.Min.X+x-12, tc.want.Min.Y+y)
					}
					if got := pane.RGBAAt(x, y); got != want {
						t.Fatalf("pane (%d,%d) = %v, want %v", x, y, got, want)
					}
				}
			}
		})
	}
}

// The second game's speaker table adds the Humans rows and the player's key
// beside the registry's records, which win; the first game's is the
// registry's alone.
func TestSpeakerFacesByGame(t *testing.T) {
	params := make([]int32, 38)
	for i := range params {
		params[i] = -1
	}
	params[16], params[17], params[18], params[24] = 24, 4, 1, 517
	humans := &fixtureCollection{names: []string{"", "Mage"}, params: [][]int32{nil, params}, raw: [][]byte{nil, nil}}
	registry := map[int32]data.NPCFace{7: {Kind: data.NPCPortrait, Face: 2}, 517: {Kind: data.NPCPortrait, Face: 9}}
	table := func(g base.Game) *mapload.Table {
		return readUnder(g, &mapload.Table{Humans: humans, NPCCatalog: &data.NPCCatalog{Faces: registry}})
	}
	first := speakerFaces(table(base.GameROM1))
	if len(first) != 2 || first[517].Face != 9 {
		t.Fatalf("first game = %+v", first)
	}
	second := speakerFaces(table(base.GameROM2))
	if second[517].Face != 9 || second[7].Face != 2 {
		t.Fatalf("registry records lost: %+v", second)
	}
	if me := second[21]; !me.Tokens.Has(data.NPCTokenMe) {
		t.Fatalf("player key = %+v", me)
	}
	delete(registry, 517)
	if rec := speakerFaces(table(base.GameROM2))[517]; rec.Kind != data.NPCFigure || rec.Face != 4 {
		t.Fatalf("row record = %+v", rec)
	}
}

func TestSpeakerFacesNonHumanPortrait(t *testing.T) {
	human := make([]int32, 38)
	for i := range human {
		human[i] = -1
	}
	human[16], human[17], human[18], human[24] = 24, 4, 1, 517
	humans := &fixtureCollection{names: []string{"", "Mage"}, params: [][]int32{nil, human}}
	units := &fixtureCollection{names: []string{""}, params: [][]int32{nil}}
	for _, key := range []int32{623, 517, 624, 97} {
		p := make([]int32, 56)
		for i := range p {
			p[i] = -1
		}
		p[29], p[30], p[55] = 70, 3, key
		units.names = append(units.names, "Creature")
		units.params = append(units.params, p)
	}
	registry := map[int32]data.NPCFace{624: {Kind: data.NPCPortrait, Class: 64, Face: 1}}
	table := readUnder(base.GameROM2, &mapload.Table{Humans: humans, Units: units,
		NPCCatalog: &data.NPCCatalog{Faces: registry}})
	faces := speakerFaces(table)
	withoutUnits := *table
	withoutUnits.Units = nil
	existing := speakerFaces(&withoutUnits)
	for key, rec := range existing {
		if faces[key] != rec {
			t.Errorf("existing speaker %d changed its record", key)
		}
	}
	want := data.NPCFace{Kind: data.NPCPortrait, Class: 70, HasClass: true, Face: 3,
		Tokens: data.NPCTokens(data.NPCTokenNotHuman) | data.NPCTokens(data.NPCTokenPicture) | data.NPCTokens(data.NPCTokenFace)}
	if got := faces[623]; got != want {
		t.Errorf("creature speaker = %+v, want %+v", got, want)
	}
	if faces[517].Kind != data.NPCFigure || faces[517].Face != 4 || faces[624] != registry[624] {
		t.Fatal("a creature replaced a Humans or registry record")
	}
	fallback, _ := data.FallbackSpeakerFace(97)
	if faces[97] != fallback {
		t.Fatal("a creature replaced an existing stand-in face")
	}
	blue, green := color.RGBA{B: 255, A: 255}, color.RGBA{G: 255, A: 255}
	r := speakerResolver{
		src: missionSource{
			"graphics/infowindow/Troll.bmp":     synthBMP(4, 4, green),
			"graphics/infowindow/Troll3.bmp":    synthBMP(4, 4, blue),
			"graphics/infowindow/Goblin.bmp":    synthBMP(4, 4, green),
			"graphics/equipment/fmage/4.256":    invBaseSheet(),
			"graphics/equipment/fmage/17.256":   invBaseSheet(),
			"graphics/equipment/mfighter/1.256": invBaseSheet(),
		},
		units: &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
			70: {Portrait: "Troll"}, 64: {Portrait: "Goblin"},
		}},
		npcFaces: faces, portraits: map[string]*image.RGBA{},
		cast: speakerCast{actors: []speakerActor{{id: 1, typeID: 3,
			fig: figureID{Dir: data.FigureDirManFighter, Face: 1}}},
			alive: func(sim.EntityID) bool { return true }},
	}
	if pic, _, ok := r.SpeakerFace(623); !ok || pic.RGBAAt(0, 0) != blue {
		t.Error("creature speaker did not draw its class at the row's tier")
	}
	for _, key := range []int{517, 624, 97} {
		before := speakerResolver{src: r.src, units: r.units,
			npcFaces: map[int32]data.NPCFace{int32(key): existing[int32(key)]}, portraits: map[string]*image.RGBA{}, cast: r.cast}
		got, window, ok := r.SpeakerFace(key)
		old, oldWindow, oldOK := before.SpeakerFace(key)
		if !ok || !oldOK || got.Bounds() != old.Bounds() || string(got.Pix) != string(old.Pix) {
			t.Errorf("existing speaker %d lost its picture", key)
		}
		if window != oldWindow || string(ui.SpeakerPortrait(got, window).Pix) != string(ui.SpeakerPortrait(old, oldWindow).Pix) {
			t.Errorf("existing speaker %d moved its pane pixels", key)
		}
	}
	if _, held := CatalogFaces(table)[623]; held {
		t.Error("catalog-only reader acquired a creature row")
	}
	withoutHumans := *table
	withoutHumans.Humans = nil
	if got := speakerFaces(&withoutHumans)[623]; got != want {
		t.Error("a creature needs no Humans collection")
	}
	if _, held := speakerFaces(readUnder(base.GameROM1, table))[623]; held {
		t.Error("first-game speaker table acquired a creature row")
	}
}
