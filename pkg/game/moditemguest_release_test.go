package game

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/sim"
)

// guestSword is the entity of the guest-chain sellsword in f's open mission.
func guestSword(t *testing.T, f *FrontEnd) sim.EntityID {
	t.Helper()
	for i, p := range f.live.mission.party {
		if p.ID == "char:sellsword" {
			return f.live.mission.ids[i]
		}
	}
	t.Fatal("no sellsword in the open mission")
	return 0
}

// TestReleaseGuestPartyMemberWieldsAnAddedWeaponRow: the guest-chain party's
// sellsword is armed with the war-glaive fixture's added weapon row instead of
// the sword, both mods in one load. The second link opens with her holding the
// glaive at its range; a SAVE inside it marks the glaive with its true row;
// a cold LOAD under both mods gives her the glaive back, and a LOAD under the
// chain alone refuses, naming the glaive's mod.
func TestReleaseGuestPartyMemberWieldsAnAddedWeaponRow(t *testing.T) {
	dir := assembledMods(t, chainModID, chainModID, warGlaiveMod)
	missions := filepath.Join(dir, chainModID, "data", "missions.toml")
	src, err := os.ReadFile(missions)
	if err != nil {
		t.Fatal(err)
	}
	armed := strings.Replace(string(src), `weapon = "Iron Short Sword"`, `weapon = "Iron war_glaive"`, 1)
	if armed == string(src) {
		t.Fatal("the guest-chain fixture no longer arms the sellsword with the Iron Short Sword")
	}
	if err := os.WriteFile(missions, []byte(armed), 0o644); err != nil {
		t.Fatal(err)
	}
	plain := releaseFront(t)
	f := chainFront(t, dir, chainModID, warGlaiveMod)
	g := glaiveOf(t, f, plain)
	if w, err := data.ResolveWeapon("Iron war_glaive", f.Table.Shapes, f.Table.Materials, f.Table.Weapons); err != nil || int(w.Row) != g.row || uint16(w.Code) != g.code {
		t.Fatalf("the cell names row %d code %#x (%v), the glaive is row %d code %#x", w.Row, w.Code, err, g.row, g.code)
	}

	app, _ := guestRoute(t, f, []int{0, 0, 0})
	chainSecond(t, f, app)
	sword := guestSword(t, f)
	if worn, _ := f.live.world.EquippedItems(sword); worn[0].Code != g.code {
		t.Fatalf("the sellsword holds %#x, the glaive is %#x", worn[0].Code, g.code)
	}
	se, _ := f.live.entity(sword)
	if se.Reach != 3 {
		t.Fatalf("the sellsword's reach with the glaive is %d", se.Reach)
	}

	saved := guestMissionSave(t, f)
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil {
		t.Fatal(err)
	}
	mark, present, err := readModMark(doc)
	if err != nil || !present || !slices.ContainsFunc(mark.Mods, func(e mod.EntryRecord) bool { return e.ID == warGlaiveMod }) {
		t.Fatalf("the save's mark: %v %v", present, err)
	}
	if !slices.ContainsFunc(mark.Items, func(it modMarkItem) bool { return it.Code == g.code && int(it.Row) == g.row }) {
		t.Fatalf("the mark names no glaive: %+v", mark.Items)
	}

	h := chainFront(t, dir, chainModID, warGlaiveMod)
	open, town, err := h.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("cold LOAD of the second link: town %v %v", town, err)
	}
	if err := h.App("guest glaive load").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	lsword := guestSword(t, h)
	if worn, _ := h.live.world.EquippedItems(lsword); worn[0].Code != g.code {
		t.Fatalf("after the LOAD the sellsword holds %#x", worn[0].Code)
	}
	if le, _ := h.live.entity(lsword); le.Reach != se.Reach || le.DamageBase != se.DamageBase || le.DamageSpread != se.DamageSpread {
		t.Fatalf("after the LOAD: reach %d base %d spread %d, before %d %d %d", le.Reach, le.DamageBase, le.DamageSpread, se.Reach, se.DamageBase, se.DamageSpread)
	}

	if err := os.WriteFile(missions, src, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := chainFront(t, dir, chainModID).RestoreOriginal(saved); err == nil || !strings.Contains(err.Error(), warGlaiveMod) {
		t.Fatalf("a LOAD without the glaive's mod: %v", err)
	}
}

// itemCharacterFront is a front end on the install under the mods ids of the
// folder dir with the report of the data step, or the first refusal.
func itemCharacterFront(t *testing.T, dir string, ids ...string) (*FrontEnd, ModDataReport, error) {
	t.Helper()
	f := releaseFront(t)
	entries, err := mod.Resolve(dir, ids)
	if err != nil {
		return nil, ModDataReport{}, err
	}
	res, err := modrt.Load(entries, BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))), nil, modrt.Options{})
	if err != nil {
		return nil, ModDataReport{}, err
	}
	if err := f.SetMods(res.Rules, res.Set, false); err != nil {
		return nil, ModDataReport{}, err
	}
	r, err := f.SetModData(res.Data)
	return f, r, err
}

// TestReleaseModsChangeAnItemAndACharacterInOneLoadOrder: one mod changes the
// Pike's price, a definition row's face and adds a catalog character; a mod
// loaded after it changes the same three. Both kinds take the one load order:
// listed in either order, the later mod's price, face and character stand, and
// the report names the item and the character override. Without load-after the
// same item field refuses the launch, while a second face edit alone applies.
func TestReleaseModsChangeAnItemAndACharacterInOneLoadOrder(t *testing.T) {
	dir := t.TempDir()
	both := "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n    game.data.add(\"data/characters.toml\")\n"
	edit := func(id, after string, price, face int, row string) {
		files := map[string]string{"main.star": both,
			"data/items.toml": "[[change]]\nitem = \"Pike\"\nprice = " + strconv.Itoa(price) + "\n",
			"data/characters.toml": "[[character]]\ntarget = \"NPC06_1\"\nface = " + strconv.Itoa(face) + "\n\n" +
				"[[character]]\nkey = \"pikeman\"\nrow = \"" + row + "\"\n"}
		if after != "" {
			files["mod.toml"] = "load-after = [\"" + after + "\"]\n"
		}
		itemRowsMod(t, dir, id, files)
	}
	edit("edit-a", "", 500, 3, "NPC06")
	edit("edit-b", "edit-a", 700, 5, "NPC06_2")
	itemRowsMod(t, dir, "price-c", map[string]string{"data/items.toml": "[[change]]\nitem = \"Pike\"\nprice = 700\n"})
	itemRowsMod(t, dir, "face-d", map[string]string{
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/characters.toml\")\n",
		"data/characters.toml": "[[character]]\ntarget = \"NPC06_1\"\nface = 5\n"})

	plain := releaseFront(t)
	pike := weaponRowNamed(t, plain.Table, "Pike")
	npc := humansRow(t, plain, "NPC06_1")
	look := func(f *FrontEnd) (params []int32, face int32, row string) {
		c, ok := catalogCharacter(f.Table, "pikeman")
		if !ok {
			t.Fatal("no pikeman in the character catalog")
		}
		return f.Table.Weapons.EntryParams(pike), f.Table.Humans.EntryParams(npc)[data.HumanSlotFace], c.Row
	}
	a, _, err := itemCharacterFront(t, dir, "edit-a")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := itemCharacterFront(t, dir, "edit-b", "edit-a")
	if err != nil {
		t.Fatal(err)
	}
	aParams, aFace, aRow := look(a)
	if slices.Equal(aParams, plain.Table.Weapons.EntryParams(pike)) || aFace != 3 || aRow != "NPC06" {
		t.Fatalf("one mod's item and character edits: Pike %v face %d pikeman %q", aParams, aFace, aRow)
	}
	bParams, bFace, bRow := look(b)
	for _, ids := range [][]string{{"edit-a", "edit-b"}, {"edit-b", "edit-a"}} {
		f, r, err := itemCharacterFront(t, dir, ids...)
		if err != nil {
			t.Fatalf("%v: %v", ids, err)
		}
		params, face, row := look(f)
		if !slices.Equal(params, bParams) || face != bFace || face != 5 || row != bRow {
			t.Fatalf("%v: Pike %v face %d pikeman %q; edit-b alone gives %v %d %q", ids, params, face, row, bParams, bFace, bRow)
		}
		item := slices.ContainsFunc(r.Overrides, func(o string) bool {
			return strings.Contains(o, "Pike") && strings.Contains(o, "price") && strings.Contains(o, `mod "edit-b"`)
		})
		character := slices.ContainsFunc(r.Overrides, func(o string) bool {
			return strings.Contains(o, "pikeman") && strings.Contains(o, `mod "edit-b"`)
		})
		if !item || !character {
			t.Fatalf("%v: the report %q names the item override %v, the character override %v", ids, r.Overrides, item, character)
		}
	}
	if _, _, err := itemCharacterFront(t, dir, "edit-a", "price-c"); err == nil || !strings.Contains(err.Error(), "Pike") || !strings.Contains(err.Error(), "price") {
		t.Fatalf("two mods on the Pike's price without load-after: %v", err)
	}
	d, _, err := itemCharacterFront(t, dir, "edit-a", "face-d")
	if err != nil {
		t.Fatalf("a second face edit without load-after: %v", err)
	}
	if params, face, _ := look(d); !slices.Equal(params, aParams) || face != 3 && face != 5 {
		t.Fatalf("edit-a with face-d: Pike %v face %d", params, face)
	}
}
