package game

import "againrom/pkg/formats/sav"

// lastMissionValue is the campaign +0x11c word the record loader stores when
// it loads main record n: the section's own LastMission, default 0
// (REG-SCN-063). The shipped scenario declares it once, as 1.
func lastMissionValue(c Campaign, n int) uint32 {
	if c.Last[n] {
		return 1
	}
	return 0
}

// missionAtFirstMapPoint reports whether mission's map object stands on the
// point of MapPoint 0. A mission the registry does not map takes object 0,
// as the original's lookup defaults to 1 (SAV-1331).
func (d *globalMapData) missionAtFirstMapPoint(mission int) bool {
	if d == nil || len(d.Objects) == 0 || !d.Objects[0].Valid {
		return false
	}
	object, ok := d.Missions[mission]
	if !ok {
		object = 0
	}
	if object < 0 || object >= len(d.Objects) || !d.Objects[object].Valid {
		return false
	}
	return d.Objects[object].Point == d.Objects[0].Point
}

// campaignTail sets the position-dependent tail word of a campaign record a
// SAVE writes. The record's FirstMapPoint enters as the campaign state's
// answer to "the party stands on MapPoint 0"; when it does not, the party
// stands on the selected mission's map object, and the word is set when that
// object shares MapPoint 0's point (SAV-1331).
func (f *FrontEnd) campaignTail(campaign sav.CampaignProjection) sav.CampaignProjection {
	if campaign.FirstMapPoint {
		return campaign
	}
	if a := f.worldMapAssets(); a != nil {
		campaign.FirstMapPoint = a.data.missionAtFirstMapPoint(int(campaign.SelectedMission))
	}
	return campaign
}
