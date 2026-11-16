package game

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// generatorSecondFront is a second-game front end over a fixture table with
// three of the four template rows and the town text the commit checks.
func generatorSecondFront(t *testing.T) *FrontEnd {
	t.Helper()
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	row := func(name string, body, reaction, mind, spirit, face int32) dbEntry {
		e := companionHumansRow(name, 10210, face)
		e.params[0], e.params[1], e.params[2], e.params[3] = body, reaction, mind, spirit
		return e
	}
	humans := dbCollection{{},
		row("Start_MF", 40, 30, 20, 20, 32),
		row("Start_FF", 30, 40, 20, 20, 33),
		row("Start_FM", 20, 20, 40, 30, 34),
	}
	src := townTextFS(t, []synth.File{{Path: "text/town.txt", Data: []byte("[npc1talk1]\r\nHello\r\n")}})
	f := &FrontEnd{InstallResources: InstallResources{
		Archives: &Archives{Base: base.Match{Profile: profile}, Containers: src},
		Campaign: resolved(Campaign{}, nil),
		Table: readUnder(base.GameROM2, &mapload.Table{Humans: humans, Armors: dressArmors(),
			Shapes: emptyScale{}, Materials: emptyScale{}, NPCCatalog: &data.NPCCatalog{}}),
		Bodies: data.NewBodyList("unarmed", "mage"),
	}, CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	f.Humans = f.Table.Humans
	f.Maps = []MapEntry{{Source: "10.alm", Name: "B", Mission: 10}}
	return f
}
