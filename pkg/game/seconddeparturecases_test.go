package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The case stores, adds and outputs below are written from R2-ENGINE-146,
// R2-ENGINE-148 and R2-SESSION-048, not from the controller.
var (
	departureStores = map[int]map[int]int32{
		20:  {532: 1, 552: 2},
		40:  {773: 23},
		50:  {771: 1, 534: 1, 554: 3},
		60:  {537: 1, 557: 2, 774: 1},
		70:  {535: 1, 555: 3, 777: 1, 770: 1},
		80:  {778: 1},
		100: {538: 1, 558: 2},
	}
	departureStage = map[int32]int32{30: 10000, 40: 22000, 50: 60000, 60: 150000, 70: 400000,
		80: 800000, 90: 1500000, 100: 5000000, 110: 10000000}
)

func departureAdds(n int, bank [1024]int32) []partyPlace {
	switch n {
	case 10:
		return []partyPlace{{1, 20}}
	case 20:
		if bank[772] != 0 {
			return []partyPlace{{2, 2}, {1, 21}}
		}
		return []partyPlace{{2, 2}}
	case 31:
		return []partyPlace{{1, 32}}
	case 40:
		return []partyPlace{{1, 50}, {1, 60}}
	case 50:
		return []partyPlace{{2, 3}}
	case 60:
		return []partyPlace{{1, 80}}
	}
	return nil
}

func departureOutput(n int, bank [1024]int32) int {
	switch {
	case n == 10:
		return 1
	case n == 30:
		return 2
	case n == 70 && bank[777] == 0 && bank[778] != 0, n == 80 && bank[778] == 0 && bank[777] != 0:
		return 3
	case n == 110 && bank[779] != 0:
		return 5
	case n == 110:
		return 4
	}
	return -1
}

// auxOf is the departure's auxiliary view of the shop: records 2 and 3.
func auxOf(s secondShop) secondAux {
	var aux secondAux
	for i := range aux {
		e := s.entry(i)
		aux[i] = [5]int32{e.Min, e.Max, e.Draws, e.Bound, int32(e.Mask)}
	}
	return aux
}

// shopWithAux is a new game's shop whose records 2 and 3 hold aux.
func shopWithAux(aux secondAux) secondShop {
	s := newSecondShop()
	for i, f := range aux {
		*s.entry(i) = secondshop.Category{Min: f[0], Max: f[1], Draws: f[2], Bound: f[3], Mask: uint32(f[4])}
	}
	return s
}

// departureMasks are the claim's mask writes by mission: auxiliary entry,
// field +0x10 (R2-ENGINE-330).
var departureMasks = map[int]map[int]uint32{
	30: {0: 0x1383c22c, 1: 0x2bc3c324, 3: 0x10438324},
	40: {0: 0x1387c26c, 1: 0x2bc7c324, 3: 0x10478264},
	50: {4: 0x13c7d318, 5: 0x2bc7d318, 6: 0x04000000, 7: 0x2bc042f0},
	60: {0: 0x1387c268, 1: 0x2bc7c268, 3: 0x10478260},
	80: {0: 0x138752e0, 1: 0x2bc752e0, 3: 0x104702e0},
	90: {0: 0x138772c0, 1: 0x2bc772c0, 3: 0x104722c0},
}

func departureIncoming(stage int32, gates int) ([1024]int32, secondAux) {
	var bank [1024]int32
	for i := range bank {
		bank[i] = int32(7*i - 3001)
	}
	for i := 0; i < 20; i++ {
		bank[512+i], bank[532+i] = int32(i%2), int32(i%3-1)
	}
	bank[768], bank[773], bank[775] = stage, 19, 0
	for bit, slot := range []int{772, 777, 778, 779, 780} {
		bank[slot] = 0
		if gates&(1<<bit) != 0 {
			bank[slot] = int32(-5 + bit)
		}
	}
	var aux secondAux
	for i := range aux {
		for j := range aux[i] {
			aux[i][j] = int32(-(10*i + j + 1))
		}
	}
	return bank, aux
}

// Every ordinary ID runs the common prefix, its case body, the join and the
// stage switch, for every combination of the gate slots and for a stage
// that the join moves into the switch and one that it moves past it.
func TestSecondDepartureCaseBodies(t *testing.T) {
	ids := []int{10, 20, 30, 31, 40, 50, 60, 70, 80, 90, 100, 110, 25, 41, 120, 127}
	for _, n := range ids {
		for _, stage := range []int32{20, 100, 110} {
			for gates := 0; gates < 32; gates++ {
				incoming, aux := departureIncoming(stage, gates)
				other := partyPlace{2, 2}
				c := &secondCampaign{bank: incoming, shop: shopWithAux(aux), current: partyPlace{1, n},
					available: []partyPlace{other, {1, n}}}
				output := c.completeBank(incoming)
				want := incoming
				want[773] = 0
				for i := 0; i < 20; i++ {
					if incoming[532+i] != 0 {
						want[532+i] = 1
						if incoming[512+i] != 0 {
							want[532+i] = 2
						}
					}
					want[512+i] = 0
				}
				want[896+n] = 1
				for slot, value := range departureStores[n] {
					want[slot] = value
				}
				if n%10 == 0 {
					want[768] += 10
				}
				wantAux := aux
				if n == 50 {
					for i, f0 := range map[int]int32{4: 499, 5: 499, 6: 0, 7: 499} {
						wantAux[i][0] = f0
						wantAux[i][2], wantAux[i][3] = 100, 2
						if i == 5 || i == 6 {
							wantAux[i][2], wantAux[i][3] = 20, 1
						}
					}
				}
				for i, mask := range departureMasks[n] {
					wantAux[i][4] = int32(mask)
				}
				if value, ok := departureStage[want[768]]; ok {
					entries := 8
					if want[768] < 50 {
						entries = 4
					}
					for i := 0; i < entries; i++ {
						wantAux[i][1] = value
					}
					if extra := map[int32]int32{90: 12000, 100: 40000}[want[768]]; extra != 0 {
						wantAux[0][0], wantAux[1][0], wantAux[3][0] = extra, extra, extra
					}
				}
				wantAvailable := append([]partyPlace{other}, departureAdds(n, incoming)...)
				if n == 20 {
					wantAvailable = append([]partyPlace{other}, departureAdds(n, incoming)[1:]...)
				}
				name := fmt.Sprintf("ID %d stage %d gates %05b", n, stage, gates)
				if c.bank != want {
					for slot := range want {
						if c.bank[slot] != want[slot] {
							t.Fatalf("%s: bank[%d]=%d want %d", name, slot, c.bank[slot], want[slot])
						}
					}
				}
				if got := auxOf(c.shop); got != wantAux {
					t.Fatalf("%s: aux %v want %v", name, got, wantAux)
				}
				if !reflect.DeepEqual(c.available, wantAvailable) {
					t.Fatalf("%s: available %v want %v", name, c.available, wantAvailable)
				}
				if output != departureOutput(n, incoming) {
					t.Fatalf("%s: output %d want %d", name, output, departureOutput(n, incoming))
				}
				if c.current != (partyPlace{}) {
					t.Fatalf("%s: current kept", name)
				}
			}
		}
	}
}

// The join adds ten to slot 768 for every incoming ID divisible by ten,
// with or without a case body, and the stage switch runs on the result
// whether or not the join moved it (R2-SESSION-049).
func TestSecondDepartureJoinAndStageSwitch(t *testing.T) {
	for _, tc := range []struct {
		n             int
		stage, joined int32
	}{
		{10, 10, 20}, {20, 20, 30}, {21, 30, 30}, {90, 80, 90}, {120, 100, 110}, {25, 40, 40},
		{41, 35, 35}, {30, 110, 120}, {31, 0, 0}, {100, -10, 0},
	} {
		c := &secondCampaign{current: partyPlace{1, tc.n}, available: []partyPlace{{1, tc.n}}}
		var bank [1024]int32
		bank[768] = tc.stage
		c.completeBank(bank)
		if c.bank[768] != tc.joined {
			t.Fatalf("ID %d from stage %d joined at %d, want %d", tc.n, tc.stage, c.bank[768], tc.joined)
		}
		var want secondAux
		for i, mask := range departureMasks[tc.n] {
			want[i][4] = int32(mask)
		}
		if value, ok := departureStage[tc.joined]; ok {
			for i := 0; i < 8 && (i < 4 || tc.joined >= 50); i++ {
				want[i][1] = value
			}
			if tc.joined == 90 || tc.joined == 100 {
				extra := map[int32]int32{90: 12000, 100: 40000}[tc.joined]
				want[0][0], want[1][0], want[3][0] = extra, extra, extra
			}
		}
		if got := auxOf(c.shop); got != want {
			t.Fatalf("ID %d at stage %d stored aux %v, want %v", tc.n, tc.joined, got, want)
		}
	}
}

// Each movie exit is produced with its gates held and withheld when any
// gate is inverted. The gates read the bank before the case stores.
func TestSecondDepartureMovieExitsFollowTheirGates(t *testing.T) {
	for _, tc := range []struct {
		n      int
		slots  map[int]int32
		output int
	}{
		{10, nil, 1},
		{30, nil, 2},
		{70, map[int]int32{777: 0, 778: 4}, 3},
		{70, map[int]int32{777: 2, 778: 4}, -1},
		{70, map[int]int32{777: 0, 778: 0}, -1},
		{70, map[int]int32{777: 2, 778: 0}, -1},
		{80, map[int]int32{778: 0, 777: -1}, 3},
		{80, map[int]int32{778: 1, 777: -1}, -1},
		{80, map[int]int32{778: 0, 777: 0}, -1},
		{80, map[int]int32{778: 1, 777: 0}, -1},
		{110, map[int]int32{779: 6}, 5},
		{110, map[int]int32{779: 0}, 4},
		{20, nil, -1}, {90, nil, -1}, {100, nil, -1}, {41, nil, -1},
	} {
		c := &secondCampaign{current: partyPlace{1, tc.n}, available: []partyPlace{{1, tc.n}}}
		var bank [1024]int32
		bank[768] = 20
		for slot, value := range tc.slots {
			bank[slot] = value
		}
		if got := c.completeBank(bank); got != tc.output {
			t.Errorf("ID %d with %v stored output %d, want %d", tc.n, tc.slots, got, tc.output)
		}
	}
}

// The output selects the movie at acknowledgement on the production route;
// an inverted gate leaves no movie request.
func TestSecondDepartureMovieAtAcknowledgement(t *testing.T) {
	cutpaths := synth.File{Path: "text/cutpaths.txt", Data: []byte("unused\r\none\r\ntwo\r\nthree\r\nfour\r\nfive\r\n")}
	for _, tc := range []struct {
		n     int
		slots map[int]int32
		movie string
	}{
		{70, map[int]int32{778: 1}, "three"},
		{70, map[int]int32{777: 1, 778: 1}, ""},
		{80, map[int]int32{777: 1}, "three"},
		{80, nil, ""},
		{110, map[int]int32{779: 1}, "five"},
		{110, nil, "four"},
		{30, nil, "two"},
		{40, nil, ""},
	} {
		t.Run(fmt.Sprint(tc.n, tc.slots), func(t *testing.T) {
			f, app, _ := secondCampaignFixtureFiles(t, true, cutpaths)
			movies := &completionMovieRequests{}
			app.SetCutscenes(movies)
			c := newSecondCampaign()
			c.bank[768] = int32(tc.n - tc.n%10)
			c.current, c.available = partyPlace{1, tc.n}, []partyPlace{{1, tc.n}, {2, 2}}
			f.Town.second = c
			w := secondGameScriptWorld(t, []sim.ScriptInstant{{Op: sim.ScriptInstantWin}}, []sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
			bank := c.bank
			for slot, value := range tc.slots {
				bank[slot] = value
			}
			w.SetROM2ScenarioState(bank)
			mw, view := missionDriverFor(t, w, nil, nil)
			mw.mission.table = readUnder(base.GameROM2, &mapload.Table{})
			ms := &Mission{Number: tc.n, World: w}
			advance := continueMission(frontTransitions{f}, tc.n, ms, mw.advanceNotice)
			f.live, f.liveMission = mw, tc.n
			if err := app.OpenMission(openPrepared(preparedMap{viewer: view, tick: mw.deterministicFrame, cadence: mw.setCadenceMode, advance: advance}, func() {})); err != nil {
				t.Fatal(err)
			}
			for tick := 0; tick < 32 && !mw.mission.open; tick++ {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if !mw.mission.open || mw.mission.kind != ui.NoticeSuccess || len(movies.names) != 0 {
				t.Fatal("script victory did not reach acknowledgement without a movie", movies.names)
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenTown || c.current != (partyPlace{}) || c.bank[896+tc.n] != 1 {
				t.Fatalf("acknowledgement did not depart: screen=%v current=%v message=%q", app.Screen(), c.current, app.HeadlessMessage())
			}
			if tc.movie == "" {
				if len(movies.names) != 0 {
					t.Fatal("withheld output requested a movie", movies.names[0])
				}
				return
			}
			if len(movies.names) != 99 || movies.names[0] != tc.movie+"/01.smk" {
				t.Fatalf("movie requests %d starting %v, want %s", len(movies.names), movies.names[:min(1, len(movies.names))], tc.movie)
			}
		})
	}
}

// A current SAV this engine wrote before the shop held the records carries
// the auxiliary array: it decodes, its records are rebuilt from the bank's
// departures, and capture writes the records in the shop, not the array.
// Malformed arrays refuse without replacing the target.
func TestCurrentSecondLegacyAuxiliaryArrayRebuildsRecords(t *testing.T) {
	c := &secondCampaign{current: partyPlace{1, 31}, available: []partyPlace{{2, 2}, {1, 31}}, shop: newSecondShop()}
	c.bank[768] = 30
	plain, err := json.Marshal(captureSecondCampaign(c))
	if err != nil || strings.Contains(string(plain), "Aux") {
		t.Fatal("capture wrote the auxiliary array", err)
	}
	// The legacy document: departures 10 and 20, stage 30, mission 30 not
	// yet left; and one that also left 30 and 50 at stage 60.
	legacy := strings.Replace(string(plain), `"Available"`, `"Aux":[[0,10000,0,0,0],[0,10000,0,0,0],[0,10000,0,0,0],[0,10000,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0]],"Available"`, 1)
	var decoded currentSecondCampaign
	if err := json.Unmarshal([]byte(legacy), &decoded); err != nil || decoded.Aux == nil {
		t.Fatal("legacy auxiliary array refused", err)
	}
	got := decoded.restore().shop.records
	want := secondshop.NewGameRecords
	for k := range want[2] {
		want[2][k].Max = 10000
	}
	if got != want {
		t.Fatalf("stage-30 records %v, want %v", got, want)
	}
	bank := c.bank
	bank[768], bank[896+30], bank[896+50], bank[770] = 60, 1, 1, 2
	records := replayRecords(bank)
	if records[2][0].Mask != 0x1383c22c || records[3][0] != (secondshop.Category{Min: 499, Max: 150000, Draws: 100, Bound: 2, Mask: 0x13c7d318}) ||
		records[3][3].Mask != 0x2bc042f0|1<<19 || records[2][2].Max != 150000 {
		t.Fatalf("stage-60 records %v", records)
	}
	full := strings.Replace(string(plain), `"Available"`, `"Aux":[[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0],[0,0,0,0,0]],"Available"`, 1)
	var explicit currentSecondCampaign
	if err := json.Unmarshal([]byte(full), &explicit); err != nil || explicit.Aux != nil {
		t.Fatal("explicit zero auxiliary array did not decode as absent", err)
	}
	for _, bad := range []string{`[]`, `[[1,2,3,4,5]]`, `null`, `[[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4]]`,
		`[[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5,6]]`,
		`[[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5]]`,
		`[[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],[1,2,3,4,5],["a",2,3,4,5]]`} {
		changed := strings.Replace(string(plain), `"Available"`, `"Aux":`+bad+`,"Available"`, 1)
		target := decoded
		before := *target.clone()
		if json.Unmarshal([]byte(changed), &target) == nil || !reflect.DeepEqual(before, target) {
			t.Fatalf("malformed auxiliary array %s admitted or replaced the target", bad)
		}
	}
}

// Town 3 is the composed druid town, a save point, and its type-2
// departure keeps its node (R2-ENGINE-144).
func TestSecondCampaignThirdTown(t *testing.T) {
	f, app, screen := secondCampaignFixture(t, true)
	c := &secondCampaign{current: partyPlace{1, 50}, available: []partyPlace{{2, 2}, {1, 50}, {1, 60}}}
	c.bank[768] = 40
	f.Town.second = c
	app.SetTown(screen)
	c.completeBank(c.bank)
	want := []partyPlace{{2, 2}, {1, 60}, {2, 3}}
	if !reflect.DeepEqual(c.available, want) || c.bank[768] != 50 {
		t.Fatal("mission 50 departure", c.available, c.bank[768])
	}
	choose := func(target string) {
		t.Helper()
		for i, row := range screen.Rows() {
			if row.Text == target && row.Choosable {
				screen.Choose(i)
				return
			}
		}
		t.Fatalf("unavailable row %s in %v", target, screen.Rows())
	}
	if err := secondMapTravel(app, screen, partyPlace{2, 3}); err != nil {
		t.Fatal(err)
	}
	if c.current != (partyPlace{2, 3}) || screen.Header() != "ROM2 campaign: town 3" || !reflect.DeepEqual(screen.Rows(), []ui.TownRow{{Text: "TAVERN", Choosable: true}, {Text: "GATES", Choosable: true}, {Text: "INNKEEPER", Choosable: true}, {Text: "SHOPKEEPER", Choosable: true}, {Text: "SHOP", Choosable: true}}) {
		t.Fatal("town 3 screen", c.current, screen.Header(), screen.Rows())
	}
	if got := screen.Footer(); !reflect.DeepEqual(got, []string{"Town 2 services are unavailable.", "Town 3 services are unavailable."}) {
		t.Fatal("town 3 footer", got)
	}
	if !screen.CanSave() {
		t.Fatal("quiet town 3 is not a save point")
	}
	raw, err := json.Marshal(captureSecondCampaign(c))
	if err != nil {
		t.Fatal(err)
	}
	var cold currentSecondCampaign
	if err := json.Unmarshal(raw, &cold); err != nil || !reflect.DeepEqual(c, cold.restore()) {
		t.Fatal("town 3 restoration", err)
	}
	choose("GATES")
	if c.current != (partyPlace{}) || !reflect.DeepEqual(c.available, want) {
		t.Fatal("town 3 departure changed availability", c.available)
	}
	for _, bad := range []func(*currentSecondCampaign){
		func(c *currentSecondCampaign) { room := secondTownShop + 1; c.Room = &room },
		func(c *currentSecondCampaign) { c.Room = nil },
		func(c *currentSecondCampaign) { c.Available = c.Available[:2] },
		func(c *currentSecondCampaign) { c.Bank[775] = 1 },
		func(c *currentSecondCampaign) { c.Available = append(c.Available, c.Available[0]) },
	} {
		changed := cold.clone()
		bad(changed)
		if changed.validateTown() == nil {
			t.Fatal("malformed town 3 admitted", changed)
		}
	}
}
