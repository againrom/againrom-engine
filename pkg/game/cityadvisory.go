package game

// citySaveAdvisory admits the settled-town form of the old map-list message.
// FinishMissionWithRoster reports the current chapter in Offered; no town or
// world-map action reads it (confirmed: no read site outside this file, the
// baseline capture in originalsave.go/originalcity_persistence.go, and the
// snapshot/restore plumbing in resume.go/savedialog1173.go — all either write
// it or compare a stored baseline, none act on it). SAV has no equivalent
// field and LOAD defaults it to zero. Selected mission, announcement latches
// and building candidates retain their independent values regardless
// (SAV-CAMPAIGN-079/080/081); pending travel still requires AGS, and both
// callers reject that before reaching this function.
//
// DIV-1322: s.Offered == chapter is the common case (the advisory still names
// the town's own settled chapter) and needs no approximation. A city saved
// right after finishing a chapter's main mission can carry an Offered naming
// the NEXT chapter's own main mission instead (observed across four of the
// owner's own saves) — a real value this session held, but one SAV has no
// field for and gameplay never reads back. Refusing the whole city save over
// a UI banner that already cannot survive the round trip traded a real
// refusal for an inert field; this now writes the SAV with the banner
// silently reset for exactly that one further shape, exactly as an ordinary
// Offered-0 LOAD already resets it.
func citySaveAdvisory(s Snapshot, camp Campaign, chapter int) error {
	if s.Offered == 0 || (s.Open && chapter > 0 && s.Offered == chapter) {
		return nil
	}
	if s.Open && chapter > 0 {
		for i, m := range camp.Main {
			if m == chapter && i+1 < len(camp.Main) && s.Offered == camp.Main[i+1] {
				return nil
			}
		}
	}
	return originalCityUnsupportedf("town save cannot normalize offered mission %d outside current settled chapter %d; that session field requires lossless .ags", s.Offered, chapter)
}
