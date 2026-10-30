package mapload_test

import (
	"againrom/pkg/mapload"
	"testing"
)

func TestCampaignActorUsesTheLastUnitsStageRowAndKeepsItsClass(t *testing.T) {
	row := func(key, typ, face int32) []int32 {
		p := wideUnitRow(key)
		p[0], p[4], p[29], p[30], p[31], p[32] = 40, 200, typ, face, 1, 1
		return p
	}
	tbl := &mapload.Table{Units: defCollection{{},
		{name: "base", params: row(10030, 100, 1)},
		{name: "early", params: row(10034, 100, 2)},
		{name: "late", params: row(10034, 100, 4)},
	}, UnitKeys: mapload.ServerUnitKeys}
	profile := mapload.CampaignActorProfile{ServerBase: 10000, ServerStride: 10, StageDivisor: 10, StageBias: -2, Temporary: true, RowDefinition: true}
	for _, tc := range []struct {
		stage int
		name  string
		face  int
	}{{30, "base", 1}, {60, "late", 4}, {70, "base", 1}, {-10, "base", 1}} {
		p, ok := mapload.CampaignActor(tbl, mapload.CampaignActorSpec{Profile: profile, Identity: 3, Stage: tc.stage, Hire: true}, nil)
		if !ok || p.Name != tc.name || p.FigureFace != tc.face || p.Class != 100 || p.ID != "hire:3" || p.MercenaryType != 0 || !p.Hired() || !p.UnitMember() {
			t.Fatalf("stage %d: %+v, %v", tc.stage, p, ok)
		}
		if tc.name == "late" {
			actor, err := mapload.SiegeHireActor(p, tbl, 1)
			if err != nil || actor.Basis.SourceBinding.TokenRow != 3 || actor.Basis.TypeID != 100 || actor.Basis.MaxHP != 200 || actor.Basis.SourceBinding.DisplayBacking != 3 {
				t.Fatalf("selected Units constructor: %+v, %v", actor.Basis, err)
			}
		}
	}
	if _, ok := mapload.CampaignActor(tbl, mapload.CampaignActorSpec{Profile: profile, Identity: 8, Hire: true}, nil); ok {
		t.Fatal("missing template accepted")
	}
}

func TestNamedCampaignActorDoesNotResolveThroughAnotherRowsServerID(t *testing.T) {
	params := composedHumanRow(15, 2, 0)
	params[24] = -1
	tbl := &mapload.Table{Humans: defCollection{{},
		{name: "named03_1", params: params},
		{name: "other", params: params},
	}}
	profile := mapload.CampaignActorProfile{HumanName: "named%02d_%d", Temporary: true, RowFigure: true, RowRotation: true, RowDefinition: true}
	p, ok := mapload.CampaignActor(tbl, mapload.CampaignActorSpec{Profile: profile, Identity: 3, Stage: 1, MercenaryType: 3}, nil)
	if !ok || p.Name != "named03_1" || p.DefinitionRow != 1 {
		t.Fatalf("named actor reached another row: %+v, %v", p, ok)
	}
}

func TestCampaignHireHumansRowAboveTheTokenByteTakesItsTypesFirstRow(t *testing.T) {
	keyed := func(key, typ int32) []int32 {
		p := composedHumanRow(typ, 1, 0)
		p[24] = key
		return p
	}
	humans := make(defCollection, 300)
	humans[5] = defEntry{name: "first crossbow", params: composedHumanRow(15, 1, 0)}
	humans[8] = defEntry{name: "low byte", params: composedHumanRow(3, 1, 0)}
	humans[263] = defEntry{name: "hire base", params: keyed(10010, 15)}
	humans[264] = defEntry{name: "hire thirty", params: keyed(10011, 15)}
	humans[200] = defEntry{name: "narrow hire", params: keyed(10020, 15)}
	humans[290] = defEntry{name: "no narrow row", params: keyed(10030, 77)}
	tbl := &mapload.Table{Humans: humans}
	profile := mapload.CampaignActorProfile{ServerBase: 10000, ServerStride: 10, StageDivisor: 10, StageBias: -2,
		Temporary: true, RowFigure: true, RowRotation: true, RowDefinition: true}
	for _, tc := range []struct {
		identity int
		name     string
		row      uint8
	}{{1, "hire thirty", 5}, {2, "narrow hire", 200}} {
		p, ok := mapload.CampaignActor(tbl, mapload.CampaignActorSpec{Profile: profile, Identity: tc.identity, Stage: 30, Hire: true}, nil)
		if !ok || p.Name != tc.name || p.DefinitionRow != tc.row || p.Class != 15 {
			t.Fatalf("hire %d: name %q row %d class %d ok %v, want %q row %d", tc.identity, p.Name, p.DefinitionRow, p.Class, ok, tc.name, tc.row)
		}
		if got := mapload.PartyHumanRow(p, mapload.PartyMember{}, tbl); got != int(tc.row) {
			t.Fatalf("hire %d: party row %d, want %d", tc.identity, got, tc.row)
		}
	}
	if p, ok := mapload.CampaignActor(tbl, mapload.CampaignActorSpec{Profile: profile, Identity: 3, Stage: 30, Hire: true}, nil); ok {
		t.Fatalf("a hire whose TypeID has no row below 256 resolved: %+v", p)
	}
}
