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
	fresh, err := loadOriginalFront(front, saved)
	if err != nil {
		return nil, OriginalSaveResume{}, err
	}
	mission := fresh.CurrentMission()
	return mission, mission.OriginalSaveReport(), nil
}

// loadOriginalFront resumes saved on a copy of front with a fresh session
// and returns the copy with its mission open.
func loadOriginalFront(front *FrontEnd, saved []byte) (*FrontEnd, error) {
	fresh := *front
	fresh.CampaignSession = CampaignSession{}
	open, town, err := fresh.RestoreOriginal(saved)
	if err != nil {
		return nil, err
	}
	if town {
		return nil, fmt.Errorf("mission fixture loaded a town")
	}
	if err := fresh.App("original mission fixture").OpenMission(open); err != nil {
		return nil, err
	}
	if fresh.CurrentMission() == nil {
		return nil, fmt.Errorf("mission fixture opened no mission")
	}
	return &fresh, nil
}
