package game

// CurrentMission returns the mission installed in the running session.
func (s *CampaignSession) CurrentMission() *Mission {
	if s == nil || s.live == nil || s.live.mission == nil {
		return nil
	}
	return s.live.mission.state
}

// OriginalSaveReport returns the observations from the mission's SAV load.
func (m *Mission) OriginalSaveReport() OriginalSaveResume {
	if m == nil || m.originalSaveReport == nil {
		return OriginalSaveResume{}
	}
	return *m.originalSaveReport
}
