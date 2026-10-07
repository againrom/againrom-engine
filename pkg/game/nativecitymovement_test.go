package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func checkNativeMovementWire(t *testing.T, f *FrontEnd, raw []byte) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := file.ActorGraph()
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Actors) == 0 {
		t.Fatal("empty Human wire population")
	}
	for _, a := range graph.Actors {
		if a.Class != "Human" {
			t.Fatalf("unexpected generated class %s", a.Class)
		}
		params := f.Table.Humans.EntryParams(int(a.DefRow))
		// Independent raw installed operands and the promoted constructor
		// constants, rather than the implementation's HumanDef/initializer.
		size, domain := params[21], params[22]
		if size == -1 {
			size = 1
		}
		if domain == -1 {
			domain = 1
		}
		mask := map[int32]byte{1: 0x41, 2: 0x44, 3: 0x82}[domain]
		if file.Body[a.ControlOff] != byte(size) || file.Body[a.ControlOff+1] != byte(domain) || a.Mover[5] != mask {
			t.Errorf("%s archive%d body%#x: size/domain/mask=%d/%d/%#x; installed row%d + constructor=%d/%d/%#x", a.Character.Name, a.ArchiveIndex, a.ControlOff, file.Body[a.ControlOff], file.Body[a.ControlOff+1], a.Mover[5], a.DefRow, size, domain, mask)
		}
	}
}

func TestReleaseNativeCityHumanMovement(t *testing.T) {
	releaseFront(t)
	for sex := 0; sex < 2; sex++ {
		for mage := 0; mage < 2; mage++ {
			t.Run(fmt.Sprintf("sex%d_class%d", sex, mage), func(t *testing.T) {
				f := releaseFront(t)
				f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Movement constructor", Choices: []int{sex, mage, 0}, Stats: []int{30, 30, 30, 30}})
				f.arriveInTown()
				f.addChapterCompanions(f.Town.Chapter())
				s, _, err := f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := f.ExportNativeCitySave(s, "Movement constructor")
				if err != nil {
					t.Fatal(err)
				}
				checkNativeMovementWire(t, f, raw)
			})
		}
	}
	t.Run("table_overrides_and_bounds", nativeCityHumanMovementOverridesAndBounds)
	if os.Getenv("AGAINROM_N3_AUDIT_ROOT") != "" {
		t.Run("owner_resaves", nativeCityMovementOwnerResaves)
	}
}

type movementOverrideCollection struct {
	data.Collection
	row    int
	params []int32
}

func (c movementOverrideCollection) EntryParams(row int) []int32 {
	if row == c.row {
		return c.params
	}
	return c.Collection.EntryParams(row)
}

func nativeCityHumanMovementOverridesAndBounds(t *testing.T) {
	f := releaseFront(t)
	for _, tc := range []struct {
		size, domain int32
		mask         byte
		valid        bool
	}{
		{2, 2, 0x44, true}, {3, 3, 0x82, true}, {0, 1, 0, false}, {256, 1, 0, false}, {1, 0, 0, false}, {1, 4, 0, false},
	} {
		t.Run(fmt.Sprintf("size%d_domain%d", tc.size, tc.domain), func(t *testing.T) {
			table := *f.Table
			params := append([]int32(nil), table.Humans.EntryParams(27)...)
			params[21], params[22] = tc.size, tc.domain
			table.Humans = movementOverrideCollection{Collection: table.Humans, row: 27, params: params}
			u := sav.CityUnitData{Token: make([]byte, 37), Scalar1: make([]byte, 19), Raw154: make([]byte, 180)}
			u.Token[16] = 27
			err := nativeCityInitializeHumanMovement(&u, &table)
			if !tc.valid {
				if err == nil {
					t.Fatal("unsupported movement admitted")
				}
				return
			}
			if err != nil || u.Scalar1[0] != byte(tc.size) || u.Scalar1[1] != byte(tc.domain) || u.Raw154[5] != tc.mask {
				t.Fatal("table override lost", u.Scalar1[:2], u.Raw154[5], err)
			}
		})
	}
}

// Private owner inputs remain outside the repository. This named discriminator
// is optional in the public suite and mandatory in the owner hotfix receipt.
func nativeCityMovementOwnerResaves(t *testing.T) {
	root := os.Getenv("AGAINROM_N3_AUDIT_ROOT")
	if root == "" {
		t.Skip("requires read-only owner N3/R1 inputs")
	}
	read := func(rel, want string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(b)); got != want {
			t.Fatalf("input %s SHA=%s", rel, got)
		}
		return b
	}
	for _, tc := range []struct {
		path, hash   string
		size, domain byte
	}{
		{"owner-N3-f1e76b6/candidates/game0069.sav", "60a5f1550df6a01414921b0be72e344192db0fc6624a76df96bb33e39af2084a", 0, 0},
		{"owner-N3-f1e76b6/candidates/game0068.sav", "878a84ac7dbd3fdc8e9cfea2712957ce441a0883e20e7b55e386b142eacd9a91", 0, 0},
		{"owner-current-sav-434f8b5-town/candidates/game0066.sav", "54e5200627c9e3319f7272168f093eaae9fe3721efb0d49f5169bb77053256f3", 1, 1},
	} {
		file, err := sav.Open(read(tc.path, tc.hash))
		if err != nil {
			t.Fatal(err)
		}
		graph, err := file.ActorGraph()
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, a := range graph.Actors {
			if a.OwnerSlot == 1 {
				count++
				if a.TokenSize != tc.size || a.Domain != tc.domain {
					t.Fatal("owner discriminator changed", tc.path, a)
				}
			}
		}
		if count != 2 {
			t.Fatal("owner party count", tc.path, count)
		}
	}
	baseline := read("story1168/f68f3f9/en/TestReleaseTownReturnCurrentCampaign1168-fresh-mission20-baseline.ags", "5b04851951e81595d2a8ec12d4229c98ea490deba32a20235b6b2616ba7054af")
	s, _, err := DecodeSave(baseline)
	if err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	_, town, err := f.Restore(s)
	if err != nil || !town {
		t.Fatal(town, err)
	}
	raw, err := f.ExportNativeCitySave(s, "N3 movement correction")
	if err != nil {
		t.Fatal(err)
	}
	checkNativeMovementWire(t, f, raw)
}
