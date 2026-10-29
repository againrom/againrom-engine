package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

func checkMissionProjection(expected []mod.Mission, actual mod.MissionData) error {
	want, _, err := mod.MergeMissions(mod.MissionData{Missions: expected})
	if err != nil {
		return err
	}
	got, _, err := mod.MergeMissions(actual)
	if err != nil {
		return err
	}
	if len(want.Missions) != len(got.Missions) {
		return fmt.Errorf("the mod entry script does not load every edited mission; reopen it in a supported script/settings context")
	}
	for _, m := range want.Missions {
		if !slices.ContainsFunc(got.Missions, func(v mod.Mission) bool { return v.Key == m.Key && v.Number == m.Number && v.Kind == m.Kind }) {
			return fmt.Errorf("the mod entry script does not load edited mission %s (%d)", m.Key, m.Number)
		}
	}
	return nil
}

func (x *MapInspector) admitMissionPackage(result modrt.Result) error {
	if x.table == nil {
		if err := x.checkMissionSet(result.Missions.Missions); err != nil {
			return err
		}
		for _, m := range result.Missions.Missions {
			address, _ := MissionMap(m.Number)
			if rows := x.CheckModMap(m, address); hasError(rows) {
				return fmt.Errorf("mission %s: %s", m.Key, rows[0].Line())
			}
		}
		return nil
	}
	c, err := LoadCampaign(x.archives.Containers)
	if err != nil {
		return err
	}
	table, archives := *x.table, *x.archives
	table.Mods = mapload.ModContext{}
	archives.Containers, archives.mods = archives.Containers.Fork(), nil
	f := &FrontEnd{InstallResources: InstallResources{Table: &table, Archives: &archives, Campaign: resolved(c, nil), Humans: table.Humans}}
	if !result.Files.Empty() {
		layer := f.modFileLayer()
		layer.order = result.Files.Order
		if err := layer.serveInstallFiles(f.Archives, result.Files.Files); err != nil {
			return err
		}
		t, err := LoadTable(f.Archives.Containers)
		if err != nil {
			return err
		}
		f.Table, f.Humans = t, t.Humans
	}
	if err := f.SetMods(result.Rules, result.Set, false); err != nil {
		return err
	}
	_, err = f.SetModData(result.Data)
	return err
}
