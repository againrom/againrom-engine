package game

import "againrom/pkg/formats/sav"

func (f *FrontEnd) campaignMapObjects(campaign sav.CampaignProjection) sav.CampaignProjection {
	if f == nil {
		return campaign
	}
	assets := f.worldMapAssets()
	if assets == nil || assets.data == nil {
		return campaign
	}
	project := func(record *sav.CampaignRecord) {
		object, ok := assets.data.Missions[int(record.Mission)]
		if ok && object >= 0 && object < len(assets.data.Objects) {
			record.MapObject = uint32(object)
		}
	}
	project(&campaign.Main)
	if campaign.Children != nil {
		campaign.Children = append([]sav.CampaignRecord{}, campaign.Children...)
		for i := range campaign.Children {
			project(&campaign.Children[i])
		}
	}
	return campaign
}
