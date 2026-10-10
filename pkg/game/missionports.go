package game

import (
	"fmt"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// installMission is the production missionInstall: the install's archives,
// bundles and tables, and the developer marker setting the map viewer decodes
// with. It reads nothing else.
type installMission struct {
	in      *InstallResources
	markers Markers
}

func (s installMission) missionTable() *mapload.Table { return s.in.Table }

func (s installMission) constructCurrentBuildings(ms *Mission) error {
	return constructCurrentBuildings(ms, s.in.Archives.Containers)
}

func (s installMission) decodeMission(addr string) (*MapView, error) {
	mapBytes, err := s.in.Archives.Containers.ReadFile(addr)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", addr, err)
	}
	// The mission viewer receives every cycle-capable object. Its live fog
	// gate decides which of them moves on each rendered frame.
	mv, err := LoadMapViewerFor(s.in.Archives.Game(), s.in.Tiles, mapBytes, addr, s.markers,
		StaticLayer{Set: s.in.Statics, Art: true, AnimGate: terrain.AnimGateAll},
		StructureLayer{Set: s.in.Structures, Art: true})
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", addr, err)
	}
	return mv, nil
}

func (s installMission) nameJoinedRoster(ms *Mission) { s.in.canonicalizeJoinedRoster(ms) }

func (s installMission) loadPartyBodies(ms *Mission, units *terrain.UnitSet) {
	for _, member := range ms.Party {
		LoadHeroBody(s.in.Archives.Containers, units, member.BodyDir, data.HeroBody(member.Body))
	}
}

// openMissionDriver hands the driver the archive set the mission's words come
// from, rather than letting the driver look for an install itself: the tier
// that owns the world opens nothing, and a driver that went looking would give
// one mission two behaviours depending on when it ran.
func (s installMission) openMissionDriver(ms *Mission, units *terrain.UnitSet, v *ui.Viewer) *mapWorld {
	return openMission(ms, s.in.Table, units, v, s.in.Archives.Containers, s.in.Faces, s.in.NPCFaces)
}

// missionProjectiles is a fact about the install rather than about the
// mission: one load serves every mission this front end opens. A driver that is
// never given one draws no spell art.
func (s installMission) missionProjectiles() *terrain.EffectSet { return s.in.Projectiles }

func (s installMission) missionBriefing(n int) string {
	return MissionObjectivesFor(s.in.Archives.Containers, s.in.Archives.Game(), n)
}

// missionPorts is the production wiring of a mission entry over this front
// end's components. It is the one place the entry's services meet the
// FrontEnd; the sequence in enterMission sees only the ports.
func (f *FrontEnd) missionPorts() missionPorts {
	return missionPorts{
		audio:   f.runtimeAudio(),
		install: installMission{in: &f.InstallResources, markers: f.Markers},
		profile: &f.PersistenceContext,
		session: &f.CampaignSession,
		display: missionDisplay{
			pathfinding:        f.showPathfinding,
			graphics:           f.graphics,
			textSmoothing:      !f.smoothingOff.text,
			frameSmoothing:     !f.smoothingOff.frame,
			deterministicFrame: f.runtime.deterministicFrames,
			chicken:            f.runtime.chicken,
		},
		art:      viewerArtSource{in: &f.InstallResources, pr: &f.Presentation, pc: &f.PersistenceContext}.resolve,
		cityBase: cityBaseFrom(f, f.Table),
		advance:  advanceFrom(frontTransitions{f}),
		campaign: f.campaign(),
	}
}

// gameSnapshotSource is the whole game state as a SAVE would capture it. A
// city graph for a party is projected from all of it (the party, purse,
// campaign, town, shop and application state), so the port is named for that
// dependency and is not narrowed to a part.
type gameSnapshotSource interface {
	Snapshot(onMap bool) (Snapshot, string, error)
}

// openCityBase is the open town's city graph and document namespace for party,
// projected from the whole game state as a SAVE would write it.
func openCityBase(src gameSnapshotSource, table *mapload.Table, party []mapload.PartyMember) (*cityObjectTopology, sav.DocumentData, error) {
	snapshot, _, err := src.Snapshot(false)
	if err != nil {
		return nil, sav.DocumentData{}, err
	}
	snapshot.Party = mapload.CloneParty(party)
	namespace, _, err := cityBaseDocument(table, snapshot)
	if err != nil {
		return nil, sav.DocumentData{}, err
	}
	return snapshot.CityObjects, namespace, nil
}

// cityBaseFrom is the cityBaseFunc port over a game snapshot source.
func cityBaseFrom(src gameSnapshotSource, table *mapload.Table) cityBaseFunc {
	return func(party []mapload.PartyMember) (*cityObjectTopology, sav.DocumentData, error) {
		return openCityBase(src, table, party)
	}
}
