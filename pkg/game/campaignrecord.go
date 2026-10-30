package game

import (
	"slices"

	"againrom/pkg/formats/sav"
)

// The campaign record a SAV carries is built here: one projection per
// document kind. A kind's route lists the record's sources in the order they
// are tried and the overlays each source's record takes. The campaign's
// profile supplies what is the game's own: a record the campaign states
// itself, that record's home word, and the source of the offer labels a SAVE
// writes.

// campaignRecordKind is the document a campaign record is written into. A
// resumed mission is a mission snapshot over a loaded record whose World
// holds generated actors; its record is seated when the snapshot is taken.
type campaignRecordKind uint8

const (
	missionRecord campaignRecordKind = iota
	townRecord
	resumedMissionRecord
	campaignRecordKinds
)

// campaignRecordSource is where a campaign record comes from.
type campaignRecordSource uint8

const (
	// recordStated is the record the campaign states itself (the profile's
	// stated record).
	recordStated campaignRecordSource = iota
	// recordCarried is the record the snapshot carries: a town's loaded
	// campaign progress, or a resumed mission's record.
	recordCarried
	// recordChapter is the native projection of the campaign's chapter.
	recordChapter
)

// campaignHome decides a record's FirstMapPoint word.
type campaignHome uint8

const (
	// homeAsBuilt keeps the source's word.
	homeAsBuilt campaignHome = iota
	// homeTail keeps a set word and otherwise sets it when the selected
	// mission's map object shares MapPoint 0's point (campaignTail).
	homeTail
	// homeTown sets the word: the party stands in town.
	homeTown
)

// campaignRecordOverlays are what a source's record takes, in field order.
type campaignRecordOverlays struct {
	// playedChild adds the played side mission's child record when absent.
	playedChild bool
	// playedSelected selects the played mission unless a guest plays it.
	playedSelected bool
	// successor writes the played mission's AutoGetMission (REG-SCN-063).
	successor bool
	// onMissionObject clears the home word: the party stands on the
	// mission's map object (SAV-1331).
	onMissionObject bool
	// home decides the home word; a stated record takes the profile's.
	home campaignHome
	// markerPaths repairs bare marker picture names.
	markerPaths bool
	// mapObjects writes each record's registry map object.
	mapObjects bool
}

type campaignRecordStep struct {
	source   campaignRecordSource
	overlays campaignRecordOverlays
}

type campaignRecordRoute struct {
	// chapter is the native projection of the kind's chapter.
	chapter func(f *FrontEnd, s Snapshot) (sav.CampaignProjection, error)
	steps   []campaignRecordStep
}

// campaignRecordRouteOf is the one projection's data for kind. The chapter
// source is the chapter progress shape: its child record rule, its
// successor and its tail word.
func campaignRecordRouteOf(kind campaignRecordKind) campaignRecordRoute {
	return [campaignRecordKinds]campaignRecordRoute{
		missionRecord: {chapter: generatedChapterProjection, steps: []campaignRecordStep{
			{source: recordStated},
			{source: recordCarried, overlays: campaignRecordOverlays{home: homeTail}},
			{source: recordChapter, overlays: campaignRecordOverlays{playedChild: true, playedSelected: true,
				successor: true, onMissionObject: true, home: homeTail}},
		}},
		townRecord: {chapter: nativeCampaignProjection, steps: []campaignRecordStep{
			{source: recordStated, overlays: campaignRecordOverlays{markerPaths: true}},
			{source: recordCarried, overlays: campaignRecordOverlays{home: homeTail, markerPaths: true}},
			{source: recordChapter, overlays: campaignRecordOverlays{home: homeTail, markerPaths: true, mapObjects: true}},
		}},
		resumedMissionRecord: {chapter: generatedChapterProjection, steps: []campaignRecordStep{
			{source: recordChapter, overlays: campaignRecordOverlays{playedSelected: true, onMissionObject: true}},
		}},
	}[kind]
}

// campaignRecordProfile is a campaign's own part of the record projection.
type campaignRecordProfile struct {
	// stated is the record the campaign states itself for a snapshot, false
	// where the snapshot holds none; nil where the chapters state it.
	stated func(s Snapshot) (sav.CampaignProjection, bool)
	// statedHome is a stated record's home word in each document kind.
	statedHome [campaignRecordKinds]campaignHome
	// offerLabels is the source of the offer labels a SAVE writes.
	offerLabels offerLabelSource
}

// offerLabelSource is where the offer labels a SAVE writes come from.
type offerLabelSource uint8

const (
	// offerLabelsOfChapter: the town's current chapter offers. A campaign
	// with no main list holds chapter 0, so its town writes the labels LOAD
	// restored.
	offerLabelsOfChapter offerLabelSource = iota
	// offerLabelsAsLoaded: the labels LOAD restored, while the town holds no
	// campaign progress; a town LOAD restored none for writes its current
	// offers.
	offerLabelsAsLoaded
)

// campaignRecord is the campaign record a document of kind carries for s.
func (f *FrontEnd) campaignRecord(s Snapshot, kind campaignRecordKind) (sav.CampaignProjection, error) {
	route := campaignRecordRouteOf(kind)
	profile := f.campaign().recordProfile()
	for _, step := range route.steps {
		var record sav.CampaignProjection
		home := step.overlays.home
		switch step.source {
		case recordStated:
			if profile.stated == nil {
				continue
			}
			stated, ok := profile.stated(s)
			if !ok {
				continue
			}
			record, home = stated, profile.statedHome[kind]
		case recordCarried:
			if !s.CampaignState {
				continue
			}
			record = s.Campaign
		case recordChapter:
			var err error
			if record, err = route.chapter(f, s); err != nil {
				return sav.CampaignProjection{}, err
			}
		}
		return f.overlayCampaignRecord(record, s, step.overlays, home), nil
	}
	return s.Campaign, nil
}

func (f *FrontEnd) overlayCampaignRecord(record sav.CampaignProjection, s Snapshot, o campaignRecordOverlays, home campaignHome) sav.CampaignProjection {
	mission := s.Mission
	campaign := f.Campaign.Value()
	if o.playedChild && mission != 0 && !containsMission(campaign.Main, mission) &&
		(containsMission(campaign.Side, mission) || containsMission(campaign.Offered, mission)) &&
		!slices.ContainsFunc(record.Children, func(child sav.CampaignRecord) bool { return int(child.Mission) == mission }) {
		child := progressRecordFromCampaign(campaign, mission).savRecord(true)
		child.Age = 0
		record.Children = append(record.Children, child)
	}
	// A guest's mission is no mission of the waiting campaign.
	if o.playedSelected && !s.guest.playing(mission) {
		record.SelectedMission = uint32(mission)
	}
	// Mission 10 carries its successor 20 under AutoGetMission; every other
	// mission carries the no-successor sentinel -1 (SAV-1094).
	if o.successor {
		record.AutoGetMission = autoGetMissionValue(campaign, mission)
	}
	if o.onMissionObject {
		record.FirstMapPoint = false
	}
	switch home {
	case homeTail:
		record = f.campaignTail(record)
	case homeTown:
		record.FirstMapPoint = true
	}
	if o.markerPaths {
		record = f.cityCampaignMarkerPaths(record)
	}
	if o.mapObjects {
		record = f.campaignMapObjects(record)
	}
	return record
}

// generatedChapterProjection is the native projection of the chapter a
// mission over generated actors belongs to.
func generatedChapterProjection(f *FrontEnd, s Snapshot) (sav.CampaignProjection, error) {
	chapter, err := f.generatedCampaignChapter(s)
	if err != nil {
		return sav.CampaignProjection{}, err
	}
	return nativeCampaignProjectionForChapter(f, s, chapter)
}

// seatCarriedCampaignRecord gives the record a snapshot carries what every
// SAVE adds to it before the document is built: the save's fame, the
// world-map markers with their picture paths, and the registry's map objects
// unless the record is a town's loaded progress.
func (f *FrontEnd) seatCarriedCampaignRecord(s *Snapshot) {
	if s.CampaignState {
		fame := fameFromSnapshot(*s)
		s.Fame = &fame
		s.Campaign.MissionTime, s.Campaign.ScoreEvents, s.Campaign.ScoreEventsKnown = fame.Time, fame.Events, true
	}
	s.Campaign = f.cityCampaignMarkerPaths(f.currentCampaignMarkers(s.Campaign, s.WorldSelectedOnce))
	if !s.CampaignState || s.Mission != 0 {
		s.Campaign = f.campaignMapObjects(s.Campaign)
	}
}

// captureCampaignRecord seats a town's carried campaign record and the
// offer labels in a snapshot, by the campaign's profile.
func captureCampaignRecord(t *Town, s *Snapshot, profile campaignRecordProfile) {
	s.OfferLabels = capturedOfferLabels(t, profile.offerLabels)
	if t == nil || t.progress == nil {
		return
	}
	s.CampaignState = true
	s.Campaign = progressCampaignRecord(t)
	s.CampaignMarkerSelected = t.progress.markerSelected
}

// capturedOfferLabels is the offer labels a SAVE of t writes.
func capturedOfferLabels(t *Town, source offerLabelSource) *currentOfferLabels {
	loaded := t != nil && t.offerLabels != nil && t.progress == nil
	if loaded && (source == offerLabelsAsLoaded || len(t.camp.Main) == 0) {
		return cloneOfferLabels(t.offerLabels)
	}
	return snapshotOfferLabels(t)
}

// progressCampaignRecord is the record of a town over loaded campaign
// progress: the progress's record with the town's live tavern pool, its
// document payload and its documents.
func progressCampaignRecord(t *Town) sav.CampaignProjection {
	record := t.progress.projection()
	record.MercenaryWorking = record.MercenaryWorking[:0]
	record.MercenaryPristine = record.MercenaryPristine[:0]
	record.MercenaryHired = record.MercenaryHired[:0]
	for i := 1; i < 16; i++ {
		record.MercenaryWorking = append(record.MercenaryWorking, uint16(t.mercPool[i]))
		record.MercenaryPristine = append(record.MercenaryPristine, uint16(t.mercCapacity[i]))
		record.MercenaryHired = append(record.MercenaryHired, t.mercHired[i])
	}
	record.Payload = t.docPayload.Clone()
	record.Documents = record.Documents[:0]
	for _, d := range t.documents {
		record.Documents = append(record.Documents, sav.CampaignDocument{Value: uint32(d.Value), Kind: uint32(d.Kind)})
	}
	return record
}
