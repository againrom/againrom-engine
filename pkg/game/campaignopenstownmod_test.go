package game

import (
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

func referenceOpensTown(c Campaign, n int) (Offer, int, bool) {
	if s, auto := c.AutoAdvance(n); auto {
		return Offer{}, s, false
	}
	offer, ok := c.NextMission(n)
	if !ok || !offer.Town {
		return Offer{}, -1, false
	}
	return offer, -1, true
}

func campaignFromRegistry(t *testing.T, raw []byte) Campaign {
	t.Helper()
	r, err := reg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return ReadCampaign(r)
}

// pins stock answers.
func TestOpensTownKeepsTheStockAnswers(t *testing.T) {
	stocks := map[string]Campaign{
		"scenario fixture": readFixture(t),
		"graph registry":   graphCampaign(t),
		"route sections":   routeCampaign(t, routeSections()),
		"test play":        campaignFromRegistry(t, testPlayRegistry()),
		"town boundary":    campaignFromRegistry(t, townAfterRegistry(20)),
	}
	for name, c := range stocks {
		if len(c.Main) == 0 {
			t.Fatalf("%s: no main missions", name)
		}
		for _, n := range c.Main {
			wo, wn, wok := referenceOpensTown(c, n)
			gt, gn, gok := c.opensTown(n)
			if gt != wo || gn != wn || gok != wok {
				t.Errorf("%s: opensTown(%d) = %v, %d, %t; before %v, %d, %t", name, n, gt, gn, gok, wo, wn, wok)
			}
		}
	}
}

func townAfterRegistry(first int32) []byte {
	return synth.Reg(kindRoot, []synth.RegNode{
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{{Name: "AutoGetMission", Kind: kindInt, Int: first}}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{
			{Name: "InnMission", Kind: kindInt, Int: 20},
			{Name: "AddHero", Kind: kindIntArray, Ints: []int32{22}},
		}},
		{Name: "Mission30", Kind: kindDir, Children: []synth.RegNode{{Name: "InnMission", Kind: kindInt, Int: 30}}},
	})
}

func TestOpensTownReadsTheWinDecisionOfAReplacedMission(t *testing.T) {
	stock := campaignFromRegistry(t, townAfterRegistry(20))
	if _, next, ok := stock.opensTown(10); ok || next != 20 {
		t.Fatalf("stock opensTown(10) = %d, %t; want the declared successor 20", next, ok)
	}
	c, _ := routeMod(t, stock, routeLeadOff)
	offer, next, ok := c.opensTown(10)
	after := c.after(10, 10, false)
	if !ok || next != -1 || !after.town || offer != after.offer {
		t.Fatalf("opensTown(10) = %v, %d, %t; want the town offer %v", offer, next, ok, after.offer)
	}
	if !offer.Town || !slices.Contains(c.Chapters[offer.Mission].AddHero, 22) {
		t.Fatalf("town offer %v does not grant companion 22", offer)
	}
	src, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := mod.Mission{Mod: "m", File: mod.MissionsFile, Kind: mod.MissionGuest, Key: "g", Number: 15,
		After: 10, Guest: 22, Chapter: offer.Mission, Carry: mod.CarryJoin}
	if err := checkGuest(c, src, mod.MissionData{Missions: []mod.Mission{g}}, g); err != nil {
		t.Fatalf("checkGuest after 10: %v", err)
	}
	if err := checkGuest(stock, src, mod.MissionData{Missions: []mod.Mission{g}}, g); err == nil {
		t.Fatal("checkGuest after 10 on the stock campaign admits a guest")
	}
	picked, err := testPlayGuest(c, src)
	if err != nil || picked.After != 10 || picked.Chapter != offer.Mission {
		t.Fatalf("testPlayGuest = %+v, %v; want a guest after 10 in chapter %d", picked, err, offer.Mission)
	}
}
