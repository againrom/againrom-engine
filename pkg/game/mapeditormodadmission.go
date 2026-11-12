package game

import (
	"errors"
	"fmt"
	"slices"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

func checkMissionProjection(word func(string) string, expected []mod.Mission, actual mod.MissionData) error {
	want, _, err := mod.MergeMissions(mod.MissionData{Missions: expected})
	if err != nil {
		return err
	}
	got, _, err := mod.MergeMissions(actual)
	if err != nil {
		return err
	}
	if len(want.Missions) != len(got.Missions) {
		return errors.New(word("editor.mod.err.entryall"))
	}
	for _, m := range want.Missions {
		if !slices.ContainsFunc(got.Missions, func(v mod.Mission) bool { return v.Key == m.Key && v.Number == m.Number && v.Kind == m.Kind }) {
			return fmt.Errorf(word("editor.mod.err.entryone"), m.Key, m.Number)
		}
	}
	return nil
}

// admitMissionPackage refuses a staged package the game would refuse: through
// the admitted fork when an install table is open, else by the mission set
// and map checks alone.
func (x *MapInspector) admitMissionPackage(result modrt.Result) error {
	if x.contextBase().table == nil {
		if err := x.checkMissionSet(result.Missions.Missions); err != nil {
			return err
		}
		for _, m := range result.Missions.Missions {
			address, _ := MissionMap(m.Number)
			if rows := x.CheckModMap(m, address); hasError(rows) {
				return fmt.Errorf(x.word("editor.mod.err.mission"), m.Key, rows[0].Line())
			}
		}
		return nil
	}
	_, _, err := x.admittedFork(result)
	return err
}
