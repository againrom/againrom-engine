package game

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// unbuiltRosterMaps are the campaign maps whose triggers name a node the
// primary-alone roster leaves unbuilt (TestReleaseScriptUnbuiltNodeCensus).
var unbuiltRosterMaps = []int{30, 60, 81, 90, 100, 120, 130, 131, 151}

// engineWrittenSAV reports whether raw carries this build's native leaves;
// such a file is not a roster the original produced.
func engineWrittenSAV(raw []byte) bool {
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		return false
	}
	if _, present, err := sav.NativeActions(doc.State); err != nil || present {
		return true
	}
	_, found, err := sav.ReadNativeSession(doc.State)
	return err != nil || found
}

// rosterRoles names the hero ordinals 10002..10006 a party resolves on m.
func rosterRoles(m *alm.Map, f *FrontEnd, party []mapload.PartyMember) []uint32 {
	refs := campaignScriptRefs(m, f.Table, party)
	var out []uint32
	if refs.HasCompanion {
		out = append(out, 10002)
	}
	for v := uint32(10003); v <= 10006; v++ {
		if _, ok := refs.Roles[v]; ok {
			out = append(out, v)
		}
	}
	return out
}

// TestReleaseScriptUnbuiltNodeRealRosters binds each changed map against the
// party this build restores from every original-produced SAV whose campaign
// position enters that map: a mission SAV of that map, or a town SAV that
// offers it. AGAINROM_UNBUILT_SAVES lists the save directories, separated by
// ';', else AGAINROM_SAVE_CORPUS names one; a file carrying this build's native leaves, or under a directory named
// for generated kits, is not counted.
// AGAINROM_UNBUILT_CENSUS names a directory the table is written to. It fails
// when such a roster reaches an outcome or off-map arm through subscript 0.
func TestReleaseScriptUnbuiltNodeRealRosters(t *testing.T) {
	dirs := os.Getenv("AGAINROM_UNBUILT_SAVES")
	if dirs == "" {
		dirs = os.Getenv("AGAINROM_SAVE_CORPUS")
	}
	if dirs == "" || os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("no save directories (AGAINROM_UNBUILT_SAVES or AGAINROM_SAVE_CORPUS) or no AGAINROM_ASSETS")
	}
	base := releaseFront(t)
	maps := map[int]*alm.Map{}
	for _, n := range unbuiltRosterMaps {
		addr, _ := MissionMap(n)
		b, err := base.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatal(err)
		}
		m, err := alm.Open(b)
		if err != nil {
			t.Fatal(err)
		}
		mapload.WithdrawBorderPlacements(m)
		maps[n] = m
	}
	var text, reached []string
	changed := map[string]bool{}
	engine, generated, refused, irrelevant := 0, 0, 0, 0
	for _, dir := range strings.Split(dirs, ";") {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if skip := corpusDirSkip(d); skip != nil {
				return skip
			}
			if d.IsDir() && strings.Contains(strings.ToLower(d.Name()), "generated") {
				generated++
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".sav") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sf, err := sav.Open(raw)
			if err != nil {
				return nil
			}
			if engineWrittenSAV(raw) {
				engine++
				return nil
			}
			mission := int(sf.Head.Mission)
			if mission != 0 && !slices.Contains(unbuiltRosterMaps, mission) {
				irrelevant++
				return nil
			}
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			open, town, err := f.RestoreOriginal(raw)
			if err != nil {
				refused++
				text = append(text, fmt.Sprintf("refused %s: %v", path, err))
				return nil
			}
			var party []mapload.PartyMember
			var targets []int
			if town {
				party = f.Carried
				for _, n := range f.Town.Available() {
					if slices.Contains(unbuiltRosterMaps, n) {
						targets = append(targets, n)
					}
				}
			} else {
				app := f.App("unbuilt roster")
				if err := app.OpenMission(open); err != nil {
					refused++
					text = append(text, fmt.Sprintf("refused %s: open: %v", path, err))
					return nil
				}
				party = f.live.mission.party
				targets = []int{mission}
			}
			if len(targets) == 0 {
				irrelevant++
				return nil
			}
			kind := "mission " + strconv.Itoa(mission)
			if town {
				kind = fmt.Sprintf("town offering %v", targets)
			}
			for _, n := range targets {
				m := maps[n]
				c, err := censusUnbuiltNodes(m, campaignScriptRefs(m, f.Table, party))
				if err != nil {
					t.Fatal(err)
				}
				c.address, c.roster = fmt.Sprintf("%d.alm", n), fmt.Sprintf("%s (%s, party %d, roles %v)", path, kind, len(party), rosterRoles(m, f, party))
				text = append(text, c.lines()...)
				if c.visible {
					changed[c.address] = true
				}
				if s0 := c.subscript0; s0 != nil {
					for _, node := range c.nodes {
						if what, outcome := unbuiltCensusInstantName[s0.Opcode]; outcome && !node.check && len(node.triggers) != 0 {
							reached = append(reached, fmt.Sprintf("%d.alm %s via %s", n, what, path))
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(reached)
	text = append(text, fmt.Sprintf("engine-written skipped=%d generated directories skipped=%d refused=%d not entering a changed map=%d", engine, generated, refused, irrelevant),
		fmt.Sprintf("outcome or off-map arms reached through subscript 0: %v", reached))
	for _, line := range text {
		t.Log(line)
	}
	var named []string
	for name := range changed {
		named = append(named, name)
	}
	sort.Strings(named)
	text = append(text, fmt.Sprintf("maps whose triggers name an unbuilt node for a real roster: %v", named))
	if out := os.Getenv("AGAINROM_UNBUILT_CENSUS"); out != "" {
		path := filepath.Join(out, "rosters-"+filepath.Base(os.Getenv("AGAINROM_ASSETS"))+".txt")
		if err := os.WriteFile(path, []byte(strings.Join(text, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
