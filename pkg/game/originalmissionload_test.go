package game

import (
	"fmt"

	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

func originalMissionFixture(source missionSource) *FrontEnd {
	files := &vfs.FS{}
	for address, bytes := range source {
		files.Overlay(address, func() ([]byte, error) { return bytes, nil })
	}
	return &FrontEnd{InstallResources: InstallResources{
		Archives: &Archives{Containers: files},
		Tiles:    &terrain.Tileset{},
		Campaign: resolved(Campaign{}, nil),
	}}
}

func loadOriginalMission(front *FrontEnd, saved []byte) (*Mission, OriginalSaveResume, error) {
	fresh := *front
	fresh.CampaignSession = CampaignSession{}
	open, town, err := fresh.RestoreOriginal(saved)
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	if town {
		return nil, OriginalSaveResume{}, fmt.Errorf("mission fixture loaded a town")
	}
	if err := fresh.App("original mission fixture").OpenMission(open); err != nil {
		return nil, OriginalSaveResume{}, err
	}
	mission := fresh.CurrentMission()
	if mission == nil {
		return nil, OriginalSaveResume{}, fmt.Errorf("mission fixture opened no mission")
	}
	return mission, mission.OriginalSaveReport(), nil
}
