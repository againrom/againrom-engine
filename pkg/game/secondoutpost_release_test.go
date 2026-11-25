//go:build release

package game

import (
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
)

// The install's census of the type-10 pass: every campaign map keeps the
// gate shut and holds no entry, every root map opens it and holds entries
// (R2-ASSET-130). A root map started through the mission builder holds an
// Outpost for each of its computer Groups; an entry's group takes its reload,
// and the radius is 0 in the English build.
func TestReleaseSecondOutpostCensusAndRootMapPass(t *testing.T) {
	f := releaseFront(t)
	if f.Archives.Base.Profile.GameOf() != base.GameROM2 {
		t.Skip("requires a second-game install")
	}
	x := installedInspector(t)
	keys, maps := installedMapBytes(t, x)
	var rootWith, rootWithout, campaignWith, campaignWithout, started, outposts, spread int
	for i, data := range maps {
		m, err := MapOpener(base.GameROM2)(data)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		in := mapload.Outposts(m, f.Table)
		holds := len(in.Entries) != 0
		if !strings.HasPrefix(keys[i], "loose/") {
			if holds {
				campaignWith++
			} else {
				campaignWithout++
			}
			if in.Runs {
				t.Errorf("%s: a campaign map opens the gate", keys[i])
			}
			continue
		}
		if !holds || !in.Runs {
			rootWithout++
			continue
		}
		rootWith++
		s, _, err := mapload.CompileROM2Script(m, mapload.ScriptRefs{})
		if err != nil {
			t.Fatal(keys[i], err)
		}
		w, _, err := mapload.StartMissionScripted(m, f.Table, mapload.DifficultyNormal, nil, s)
		if err != nil {
			t.Fatal(keys[i], err)
		}
		started++
		got := w.Outposts()
		if len(got) == 0 {
			t.Errorf("%s: no Outpost", keys[i])
		}
		for _, o := range got {
			outposts++
			if o.Radius != 0 {
				spread++
			}
			want := int32(120)
			for _, e := range in.Entries {
				if e.Group == o.Group {
					want = e.Percent
				}
			}
			if o.Reload != want || o.Countdown != want || len(o.Members) == 0 {
				t.Errorf("%s: Outpost of owner %d group %d reload %d, want %d", keys[i], o.Owner, o.Group, o.Reload, want)
			}
		}
	}
	if f.Archives.Base.Profile.ID == base.ROM2EN && spread != 0 {
		t.Errorf("the English build gave %d Outposts a radius", spread)
	}
	if campaignWith != 0 || rootWith == 0 {
		t.Errorf("campaign maps with entries %d, root maps with entries %d", campaignWith, rootWith)
	}
	t.Logf("PASS %s: campaign maps %d with type-10 entries, %d without; root maps %d with, %d without; %d root maps started, %d Outposts, %d with a radius",
		installLanguage(x), campaignWith, campaignWithout, rootWith, rootWithout, started, outposts, spread)
}
