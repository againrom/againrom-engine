package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCreaturePortraitTierRoutes(t *testing.T) {
	for _, game := range []base.Game{base.GameROM1, base.GameROM2} {
		t.Run(string(game), func(t *testing.T) {
			second := game == base.GameROM2
			for _, tc := range []struct {
				name, route                     string
				face, saved, party, existing    int
				row, bindingType, class         int
				noRows, duplicate, wrongType    bool
				invalidRecord, zeroRow, noState bool
				want                            [2]int
				found                           bool
			}{
				{name: "placement row", route: "placement", want: [2]int{3, 3}, found: true},
				{name: "placement without row", route: "placement", noRows: true},
				{name: "constructor face", route: "constructor", want: [2]int{3, 3}, found: true},
				{name: "saved face beats row", route: "loaded", saved: 7, want: [2]int{7, 7}, found: true},
				{name: "legacy saved key takes row", route: "loaded", saved: 1, want: [2]int{1, 3}, found: true},
				{name: "zero saved face takes placement", route: "loaded", want: [2]int{3, 3}, found: true},
				{name: "invalid record takes placement", route: "loaded", saved: 7, invalidRecord: true, want: [2]int{3, 3}, found: true},
				{name: "duplicate placement keeps saved face", route: "loaded", saved: 7, duplicate: true, want: [2]int{7, 7}, found: true},
				{name: "duplicate placement has no fallback", route: "loaded", duplicate: true},
				{name: "wrong placement type has no fallback", route: "loaded", wrongType: true},
				{name: "no state keeps placement indices", route: "loaded", noState: true, want: [2]int{3, 3}, found: true},
				{name: "saved zero placement remains present", route: "loaded", zeroRow: true, want: [2]int{0, 0}, found: true},
				{name: "saved key with zero row stays key", route: "loaded", saved: 1, zeroRow: true, want: [2]int{1, 1}, found: true},
				{name: "source face beats row", route: "source", face: 6, want: [2]int{6, 6}, found: true},
				{name: "source zero face", route: "source", want: [2]int{0, 3}, found: true},
				{name: "source missing row", route: "source", noRows: true, want: [2]int{0, 0}, found: true},
				{name: "source row zero", route: "source", row: -1, want: [2]int{0, 0}, found: true},
				{name: "source row outside table", route: "source", row: 65535, want: [2]int{0, 0}, found: true},
				{name: "source binding type mismatch", route: "source", bindingType: 71, want: [2]int{0, 0}, found: true},
				{name: "source row type mismatch", route: "source", row: 2, want: [2]int{0, 0}, found: true},
				{name: "source face without row", route: "source", face: 6, noRows: true, want: [2]int{6, 6}, found: true},
				{name: "loaded source beats record", route: "loaded source", face: 6, saved: 7, want: [2]int{6, 6}, found: true},
				{name: "loaded source zero falls through", route: "loaded source", saved: 7, existing: 5, want: [2]int{5, 3}, found: true},
				{name: "loaded source without row uses placement", route: "loaded source", noRows: true, existing: 5, want: [2]int{5, 5}, found: true},
				{name: "loaded source zero without placement", route: "loaded source", noRows: true, duplicate: true},
				{name: "party stored face", route: "party", party: 4, want: [2]int{4, 4}, found: true},
				{name: "party existing tier wins", route: "party", party: 4, existing: 7, want: [2]int{7, 7}, found: true},
				{name: "party zero face", route: "party"},
				{name: "party negative face", route: "party", party: -1},
				{name: "party composed figure", route: "party", party: 4, class: 1},
				{name: "hire stored face", route: "hire", party: 4, want: [2]int{4, 4}, found: true},
				{name: "hire existing tier wins", route: "hire", party: 4, existing: 7, want: [2]int{7, 7}, found: true},
				{name: "manifest explicit face", route: "manifest", face: 6, existing: 5, want: [2]int{6, 6}, found: true},
				{name: "manifest zero overwrites existing", route: "manifest", existing: 5, want: [2]int{0, 3}, found: true},
				{name: "manifest missing row overwrites existing", route: "manifest", noRows: true, existing: 5, want: [2]int{0, 0}, found: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					row := func(typ, face, server int32) dbEntry {
						p := make([]int32, 56)
						for i := range p {
							p[i] = -1
						}
						p[29], p[30], p[55] = typ, face, server
						return dbEntry{name: "creature", params: p}
					}
					edition := game.Edition()
					table := &mapload.Table{Edition: &edition, Units: dbCollection{{}, row(70, 3, 5032), row(71, 8, 5033)}}
					key := uint16(3)
					if second {
						table.UnitKeys, key = mapload.ServerUnitKeys, 1
					}
					placement := alm.Unit{UnitID: 32, ClassID: 70, ClassSubID: key, ServerID: 5032}
					m := &alm.Map{Units: []alm.Unit{placement}}
					e := sim.Entity{ID: 9, MapUnitID: 32, TypeID: 70, Class: 70}
					placed := map[sim.EntityID]int{0: 3}
					if tc.zeroRow {
						placed[0] = 0
					}
					if tc.duplicate {
						m.Units = append(m.Units, placement)
					}
					if tc.wrongType {
						e.TypeID = 71
					}
					if tc.noRows {
						table.Units = nil
					}
					var got int
					var found bool
					switch tc.route {
					case "placement":
						got, found = placementTier(placement, table)
					case "constructor":
						bound, _, err := mapload.ConstructActorBasis(e, mapload.PartyMember{}, &placement, table, 1, 1, nil, 0)
						if err != nil {
							t.Fatal(err)
						}
						got, found = int(bound.SourceBinding.Face), true
					case "source", "loaded source", "manifest":
						row, typ := 1, 70
						if tc.row != 0 {
							row = max(0, tc.row)
						}
						if tc.bindingType != 0 {
							typ = tc.bindingType
						}
						e.SourceBinding = sim.SourceBinding{Class: sim.GeneratedUnitBinding, TokenRow: uint16(row), TypeID: uint16(typ), Face: uint8(tc.face)}
						if tc.route == "source" {
							got, found = sourceActorTier(e, table)
							break
						}
						if tc.route == "manifest" {
							w, err := sim.NewWorld(1, sim.Bounds{Width: 1, Height: 1}, sim.ModeCanonical, nil, []sim.Entity{e})
							if err != nil {
								t.Fatal(err)
							}
							mw := &mapWorld{world: w, tiers: map[sim.EntityID]int{9: tc.existing}}
							mw.installActorManifest(&SnapshotActorManifest{Actors: []SnapshotActor{{ID: 9}}}, table)
							got, found = mw.tiers[9]
							break
						}
						fallthrough
					case "loaded":
						if tc.existing != 0 {
							placed[0] = tc.existing
						}
						state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{
							savedTierRecord(32, 70, uint32(tc.saved)),
						}}, Actors: []SnapshotSAVActor{{EntityID: 9, ObjectIndex: 1}}}
						if tc.invalidRecord {
							state.Actors[0].ObjectIndex = 2
						}
						id := e.ID
						if tc.noState {
							state, id = nil, 0
						}
						got, found = savedActorTiers(m, table, placed, []sim.Entity{e}, state)[id]
					case "party", "hire":
						class := int32(70)
						if tc.class != 0 {
							class = int32(tc.class)
						}
						member := mapload.PartyMember{Class: class, FigureFace: tc.party}
						if tc.route == "hire" {
							member.MercenaryType = 1
						}
						var tiers map[sim.EntityID]int
						if tc.existing != 0 {
							tiers = map[sim.EntityID]int{9: tc.existing}
						}
						got, found = partyTiers(tiers, []mapload.PartyMember{member}, []sim.EntityID{9})[9]
					default:
						t.Fatalf("unknown route %q", tc.route)
					}
					index := 0
					if second {
						index = 1
					}
					if got != tc.want[index] || found != tc.found {
						t.Fatalf("tier/present = %d/%t, want %d/%t", got, found, tc.want[index], tc.found)
					}
				})
			}
		})
	}
}
