package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"slices"
	"testing"

	"againrom/pkg/screenart"
	"againrom/pkg/town"
)

type townRecordSource struct {
	reads []string
	data  map[string][]byte
}

func (s *townRecordSource) ReadFile(name string) ([]byte, error) {
	s.reads = append(s.reads, name)
	if raw, ok := s.data[name]; ok {
		return raw, nil
	}
	return nil, os.ErrNotExist
}

func TestTownRoomArtReadsSceneBeforePage(t *testing.T) {
	for _, description := range []string{"rom1", "rom2-town1", "rom2-town2", "rom2-town3"} {
		f, _, screen := secondCampaignFixture(t, false)
		src := &townRecordSource{data: map[string][]byte{}}
		f.Archives.art = screenart.NewLibrary(screenArtSource{src})
		screen.composed = newTownSceneState(nil)
		d := *townDescriptions[description]
		d.Rooms = append([]town.RoomSpec(nil), d.Rooms...)
		entry := func(key string) []town.ArtSpec {
			src.data[key] = synthBMP(2, 2, color.RGBA{R: 1, A: 255})
			return []town.ArtSpec{{Name: "fixture", Key: key, Format: "picture", Required: true}}
		}
		d.Art = entry("fixture/scene/square.bmp")
		expected := []string{"fixture/scene/square.bmp"}
		for i := range d.Rooms {
			r := &d.Rooms[i]
			r.Scene = &town.SceneSpec{Art: entry("fixture/scene/" + r.Name + ".bmp")}
			r.Art = entry("fixture/page/" + r.Name + ".bmp")
			expected = append(expected, "fixture/scene/"+r.Name+".bmp", "fixture/page/"+r.Name+".bmp")
		}
		rooms := []string{d.Square.Name}
		for _, r := range d.Rooms {
			rooms = append(rooms, r.Name)
		}
		for _, room := range rooms {
			art := screen.roomArt(&d, room)
			if art.err != nil || art.scene == nil {
				t.Fatalf("%s/%s: %v", description, room, art.err)
			}
		}
		if !slices.Equal(src.reads, expected) {
			t.Fatalf("%s reads %v, want %v", description, src.reads, expected)
		}
	}
}

func TestTownRoomArtRecord(t *testing.T) {
	want := map[string]string{
		"rom1/gates":        "d12ce2c7c3e0feff9b38bf094d8421441bc07cc9ebf2abe274e7e333f0c68175",
		"rom1/school":       "76c32198f68ddb5f858d5c2de6e22524713ed4063cf38d200141ce767610f7e2",
		"rom1/shop":         "153c0b85ff7548d92d8ce2ef99ab61af670d9e18987f1317c49fb9a554c2b65f",
		"rom1/square":       "787d656b93fb7e156007c8e2ec2dbc92edb38d45cc6a310b7d4671fdd8730041",
		"rom1/tavern":       "0447725ca150c51359f570f10fa751688697f2a3d6d2118e39a506a25c6d43f7",
		"rom2-town1/inn":    "eebf0118937df6c803e20a38a3f1e99759710825af7556f9071860e1dcae7110",
		"rom2-town1/shop":   "a88b72e1138186e92c3040fa996c356e078ad990359f9ca3c297ea420113238e",
		"rom2-town1/square": "787d656b93fb7e156007c8e2ec2dbc92edb38d45cc6a310b7d4671fdd8730041",
		"rom2-town2/inn":    "b875c14211a0576c1b922561b0a7f67723a977fe436a91fbaca2e55428ec3626",
		"rom2-town2/shop":   "19550a1f1e729e47fd82a5f089fec3963b5293f5e377191b99e3d32a7bf9fd6e",
		"rom2-town2/square": "6b109198c71ba7a36df88f4df84fdef84f44639371c344489ad014e66ba92b24",
		"rom2-town3/inn":    "709e8afe42a193997552368625d6a85e121ac421112f65bfb20734a981d94eb9",
		"rom2-town3/shop":   "fb17358d05b092d0467f83c29e0cb93119a3ef8281040bda11b66f6ee1dbf82a",
		"rom2-town3/square": "0480c017c13c005d09dc142fd2fba4e311c64abfbde4fa613dd2c4d962956b22",
	}
	for _, description := range []string{"rom1", "rom2-town1", "rom2-town2", "rom2-town3"} {
		f, _, screen := secondCampaignFixture(t, false)
		src := &townRecordSource{}
		f.Archives.art = screenart.NewLibrary(screenArtSource{src})
		screen.composed = newTownSceneState(nil)
		d := townDescriptions[description]
		rooms := []string{d.Square.Name}
		for _, room := range d.Rooms {
			rooms = append(rooms, room.Name)
		}
		for _, room := range rooms {
			a := screen.roomArt(d, room)
			count := len(src.reads)
			if screen.roomArt(d, room) != a || len(src.reads) != count {
				t.Fatalf("%s/%s was read twice", description, room)
			}
			name := description + "/" + room
			got := townRecordHash(t, []any{src.reads, fmt.Sprint(a.err), a.scene == nil, a.page == nil})
			if got != want[name] {
				t.Errorf("%s: %s, want %s", name, got, want[name])
			}
		}
	}
}

func townRecordHash(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

func TestTownStateAndArtRecord(t *testing.T) {
	record := map[string]string{}
	f, screen := travelFront(t, travelCatalog)
	fixture := shellFrontEnd()
	f.Table, f.Humans, f.Carried = fixture.Table, fixture.Humans, fixture.Carried
	f.Carried[0].ID, f.Carried[0].StartingHero = "hero", true
	observe := func(name string) {
		tw := f.Town
		record[name] = townRecordHash(t, []any{tw.townID(), tw.Offers(TownTavern), tw.Offers(TownShop), tw.mercPool, tw.mercCapacity, tw.mercHired, tw.townsRecord()})
		record[name+"/sav"] = fmt.Sprintf("%x", sha256.Sum256(exportTown(t, f)))
	}
	observe("chapters/entry")
	screen.Choose(3)
	travelCard(t, screen, placeTown, 2)
	f.Town.gold = 1 << 20
	f.Town.Take(TownShop, 0)
	observe("chapters/east")
	screen.Choose(3)
	travelCard(t, screen, placeTown, 1)
	observe("chapters/return")

	second, _, secondScreen := secondCampaignFixture(t, false)
	second.Table = identityDollTable()
	second.Carried = fixture.Carried
	second.Town.second = newSecondCampaign()
	c := second.Town.second
	if os.Getenv("AGAINROM_TOWN_RECORD_MUTATION") == "1" {
		c.bank[770]++
	}
	observeSecond := func(name string) {
		secondScreen.campaignTown().townID()
		record[name] = townRecordHash(t, captureSecondCampaign(c))
		record[name+"/sav"] = fmt.Sprintf("%x", sha256.Sum256(exportTown(t, second)))
	}
	observeSecond("bank/entry")
	c.bank[769] = 1
	c.available = []partyPlace{{placeTown, 2}, {placeTown, 3}, {placeMission, 30}}
	c.current = partyPlace{placeTown, 2}
	c.bank[768] = 30
	c.shop.records[2][0].Mask = 7
	observeSecond("bank/travel")
	c.current = partyPlace{placeTown, 3}
	observeSecond("bank/return")
	c.current = partyPlace{placeMission, 30}
	c.completeBank(c.bank)
	c.current = partyPlace{placeTown, 2}
	observeSecond("bank/mission-return")

	for _, name := range []string{"rom1", "rom2-town1", "rom2-town2", "rom2-town3"} {
		d := townDescriptions[name]
		if d == nil {
			t.Fatalf("missing description %s", name)
		}
		src := &townRecordSource{}
		loads := newTownArtLoads(d, src)
		if len(src.reads) != 0 {
			t.Fatal("art was read before its first use")
		}
		for _, room := range []string{"square", "tavern", "school", "shop"} {
			switch room {
			case "square":
				loads.square.resolve()
			case "tavern":
				loads.tavern.resolve()
			case "school":
				loads.school.resolve()
			case "shop":
				loads.shop.resolve()
			}
			record[name+"/"+room] = townRecordHash(t, src.reads)
		}
		before := len(src.reads)
		loads.square.resolve()
		loads.tavern.resolve()
		loads.school.resolve()
		loads.shop.resolve()
		if len(src.reads) != before {
			t.Fatal("a failed art read was retried")
		}
	}
	want := map[string]string{
		"bank/entry":              "6d7c529090e2fc7dcaf75a942954de73fc842cae2f6a440bc313d23e60e47f6d",
		"bank/entry/sav":          "d2dd26c59ec9bd48c3d804fb81f9d27040191a4867f8fed75ec4badb21853e56",
		"bank/mission-return":     "547b7b1245345639bfa00f4a07c45c737ba19dc2a76d5230f7d80bac210235e1",
		"bank/mission-return/sav": "52b0c76958685171872ceb3105258fb4fff72e1ffbb877da06e67522b012969e",
		"bank/return":             "365859704d53f4f2da2dc4d149ef8b7a1e3b5753fbea9f737862766d3b6bbd95",
		"bank/return/sav":         "d0cf7fd6d0a9cb498c8eb419e0514b806e857b17e789a28107197b3b6ba46715",
		"bank/travel":             "b62cf863696d74813be085bc7d1632fb53bb466f60125bf9c531ec1d107b3b08",
		"bank/travel/sav":         "78cf6fba7fd5bb76db8d394737385325773e9fe54b7b29fbf43b18b8bb24c0e2",
		"chapters/east":           "1c12c9ad5825c0a5f36628b2a4b93e9281f937b1bb7fdb98e8f3edb60f15f1df",
		"chapters/east/sav":       "2a119b68b397dc79f5850d8ef67272c82bf0e051b5807dc097103f8e5c01cf20",
		"chapters/entry":          "a31c10aa2138e0b7d4537f982fc090d2bd891a77c1c152264a37ed704276b3f9",
		"chapters/entry/sav":      "2c03494dec7d90430a9a334e97e55387642c740c5f508c0b4a45dcba436ed5e3",
		"chapters/return":         "327a582ec435d0e91cdb95ce17911943f5c89b2ad064fb0d7b1b33ee5c697d2e",
		"chapters/return/sav":     "5e4728259950e58f3acb764fd11254de790bc4edebe99d71014f42ec5ff25bcf",
		"rom1/school":             "9cd8dd1e9a9a8857aa098315d1feff346035b665f87b5504333117bf4e8e1857",
		"rom1/shop":               "7af2ee37b03decfc94738582cbb1b2b73a01c698147e409803e6e1b2f01a5dee",
		"rom1/square":             "2749ec00a87468c412f3aaa9c117b2317d13f3b6b1a4b2c4b7016a86f4de5655",
		"rom1/tavern":             "9d0966b8a4d867b1aa178473fe4908c803aeb7d05ebf9d400231208d0ae1ec80",
		"rom2-town1/school":       "2749ec00a87468c412f3aaa9c117b2317d13f3b6b1a4b2c4b7016a86f4de5655",
		"rom2-town1/shop":         "0e305543f6819e97d666d02e20431087387572a792cd3a4117bc525582765b81",
		"rom2-town1/square":       "2749ec00a87468c412f3aaa9c117b2317d13f3b6b1a4b2c4b7016a86f4de5655",
		"rom2-town1/tavern":       "2749ec00a87468c412f3aaa9c117b2317d13f3b6b1a4b2c4b7016a86f4de5655",
		"rom2-town2/school":       "79cf8c5a14e4939066e7f21a4f10c2ee5a4db6eae17887a086793fadcde8460b",
		"rom2-town2/shop":         "686766ede14443124b961d5f87ba0375a2ad37f6d2c9e254486fd86c46421eac",
		"rom2-town2/square":       "79cf8c5a14e4939066e7f21a4f10c2ee5a4db6eae17887a086793fadcde8460b",
		"rom2-town2/tavern":       "79cf8c5a14e4939066e7f21a4f10c2ee5a4db6eae17887a086793fadcde8460b",
		"rom2-town3/school":       "085fa28ffac9c8c839ca4bb6f34db2a20605a11210a276c4b62ce0c1eede8b24",
		"rom2-town3/shop":         "8c37eeef33ef3aed82e9f590a83ca3a02d6b1bb64442243922eb192b0a96f22a",
		"rom2-town3/square":       "085fa28ffac9c8c839ca4bb6f34db2a20605a11210a276c4b62ce0c1eede8b24",
		"rom2-town3/tavern":       "085fa28ffac9c8c839ca4bb6f34db2a20605a11210a276c4b62ce0c1eede8b24",
	}
	if len(record) != len(want) {
		t.Fatalf("record has %d rows, want %d", len(record), len(want))
	}
	for name, got := range record {
		if got != want[name] {
			t.Errorf("%s: %s, want %s", name, got, want[name])
		}
	}
}

// TestSecondTownIDStaysTheChapterTown pins Town.townID() for the second game:
// binding the rows, a cold load and travel to towns 2 and 3 leave it at the
// chapter town, as before the rows were bound.
func TestSecondTownIDStaysTheChapterTown(t *testing.T) {
	second, _, screen := secondCampaignFixture(t, false)
	second.Town.second = newSecondCampaign()
	c := second.Town.second
	check := func(name string, want int) {
		t.Helper()
		if got := second.Town.townID(); got != 1 {
			t.Fatalf("%s: Town.townID() = %d, want 1", name, got)
		}
		if got := screen.campaignTown().townID(); got != want {
			t.Fatalf("%s: campaign townID() = %d, want %d", name, got, want)
		}
	}
	screen.campaignTown().bank()
	check("first bank read", 1)
	c.available = []partyPlace{{placeTown, 2}, {placeTown, 3}}
	c.current = partyPlace{placeTown, 2}
	check("town 2", 2)
	c.current = partyPlace{placeTown, 3}
	check("town 3", 3)
	second.Town.second = captureSecondCampaign(c).restore()
	c = second.Town.second
	check("cold load", 3)
	c.current = partyPlace{placeTown, 2}
	check("cold load town 2", 2)
}
