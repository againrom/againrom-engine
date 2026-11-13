package game

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// modTownLoss is what a town LOAD under changed mods takes away once the town
// is restored: each item to take out of the party, and the rows that list all
// it took or lowered.
type modTownLoss struct {
	items []modLostItem
	rows  []ui.ModMismatchRow
}

// modLostItem is one item a member holds that the active mods no longer
// define: its city object, and the worn slot (1-based) or 0 for the pack.
type modLostItem struct {
	member string
	id     sim.SavedObjectID
	slot   int
}

// modSkillWords name the five trainable skill slots under load.mods.skill, a
// fighter's and a mage's.
var modSkillWords = [2][6]string{
	{"", "blade", "axe", "bludgeon", "pike", "shooting"},
	{"", "fire", "water", "air", "earth", "astral"},
}

// townLoss finds the holder of each lowered skill and gone item among the
// restored party, which chars bind to the document's actors in order, and the
// city object of each gone item in graph. A gone item outside the party's
// packs and worn slots refuses the load: nothing could take it away.
func (l modMarkLoad) townLoss(saved []byte, party []mapload.PartyMember, chars []sav.Character, graph *cityObjectTopology, off []modMarkLayer) (modTownLoss, error) {
	var loss modTownLoss
	if !l.changed {
		return loss, nil
	}
	holder := func(object int) int {
		if len(chars) != len(party) {
			return -1
		}
		return slices.IndexFunc(chars, func(c sav.Character) bool { return int(c.ArchiveIndex) == object+1 })
	}
	for _, s := range l.lowered {
		i := holder(s.Object)
		if i < 0 {
			continue
		}
		mage := 0
		if party[i].Mage {
			mage = 1
		}
		loss.rows = append(loss.rows, ui.ModMismatchRow{Kind: ui.ModMismatchSkill, Who: party[i].Name, What: modSkillWords[mage][s.Slot],
			Saved: strconv.Itoa(int(s.Saved)), Current: strconv.Itoa(int(s.Now))})
	}
	if len(l.gone) != 0 {
		doc, err := sav.DecodeDocumentData(saved)
		if err != nil {
			return loss, err
		}
		for _, g := range l.gone {
			lost, ok := goneModItem(&doc, g.Object, party, chars, graph)
			if !ok {
				return loss, fmt.Errorf("item object %d, which the active mods no longer define, is in no pack or worn slot of the party", g.Object)
			}
			loss.items = append(loss.items, lost)
			kind := ui.ModMismatchItemPack
			if lost.slot != 0 {
				kind = ui.ModMismatchItemWorn
			}
			i := slices.IndexFunc(party, func(p mapload.PartyMember) bool { return p.ID == lost.member })
			loss.rows = append(loss.rows, ui.ModMismatchRow{Kind: kind, Who: party[i].Name, Mod: g.Mod, What: g.Key})
		}
	}
	for _, layer := range off {
		if i := slices.IndexFunc(party, func(p mapload.PartyMember) bool { return p.ID == layer.Member }); i >= 0 {
			loss.rows = append(loss.rows, ui.ModMismatchRow{Kind: ui.ModMismatchLayer, Who: party[i].Name, Mod: layer.Mod, What: layer.Key})
		}
	}
	loss.rows = append(loss.rows, l.dropped...)
	return loss, nil
}

// pruneUndeclaredMissions drops from a town save's campaign projection each
// mission the active campaign no longer declares: a retained child, and an
// inn, school or shop candidate. A selection of a dropped child falls to the
// main mission. Only a mod mission can vanish, as the base is the same; a
// main mission that vanished still refuses the load. It answers the dropped
// numbers in order, each once.
func pruneUndeclaredMissions(c Campaign, in *sav.CampaignProjection) []int {
	declared := func(n int) bool {
		return containsMission(c.Main, n) || containsMission(c.Side, n) || containsMission(c.Offered, n)
	}
	var dropped []int
	drop := func(n int) {
		if !slices.Contains(dropped, n) {
			dropped = append(dropped, n)
		}
	}
	in.Children = slices.DeleteFunc(slices.Clone(in.Children), func(r sav.CampaignRecord) bool {
		n := int(r.Mission)
		if n <= 0 || declared(n) {
			return false
		}
		drop(n)
		return true
	})
	if n := int(in.SelectedMission); n > 0 && n != int(in.Main.Mission) && slices.Contains(dropped, n) {
		in.SelectedMission = in.Main.Mission
	}
	candidates := func(values []uint16) []uint16 {
		return slices.DeleteFunc(slices.Clone(values), func(v uint16) bool {
			if v == 0 || declared(int(v)) {
				return false
			}
			drop(int(v))
			return true
		})
	}
	if len(in.InnNPC) == len(in.InnMission) {
		var npcs, missions []uint16
		for i, v := range in.InnMission {
			if v != 0 && !declared(int(v)) {
				drop(int(v))
				continue
			}
			npcs, missions = append(npcs, in.InnNPC[i]), append(missions, v)
		}
		in.InnNPC, in.InnMission = npcs, missions
	}
	in.TCMission, in.ShopMission = candidates(in.TCMission), candidates(in.ShopMission)
	return dropped
}

// droppedMissionRows lists the dropped campaign missions.
func droppedMissionRows(dropped []int) []ui.ModMismatchRow {
	var rows []ui.ModMismatchRow
	for _, n := range dropped {
		rows = append(rows, ui.ModMismatchRow{Kind: ui.ModMismatchMission, What: strconv.Itoa(n)})
	}
	return rows
}

// adoptChangedGuests holds a town save's guest records to the missions the
// changed mods declare: a record whose mission is still declared takes the
// number it has now, and one whose mission is gone is dropped and listed.
// The session no longer waits for a guest when none is left.
func adoptChangedGuests(s *currentSessionData, mods mapload.ModContext) []ui.ModMismatchRow {
	if s == nil {
		return nil
	}
	var rows []ui.ModMismatchRow
	for at, depth := &s.Guest, 0; *at != nil && depth < maxWaitingGuests; depth++ {
		r := *at
		m, ok := mods.Missions.Find(r.Mod, r.Key)
		if !ok {
			rows = append(rows, ui.ModMismatchRow{Kind: ui.ModMismatchGuest, Mod: r.Mod, What: strconv.Itoa(r.Number)})
			*at = r.Previous
			continue
		}
		r.Number = m.Number
		at = &r.Previous
	}
	if s.Guest == nil {
		s.GuestWaiting = false
	}
	return rows
}

// goneModItem is the party place of a document item object: the member whose
// actor holds it in Inventory or a worn slot, and its city object there.
func goneModItem(doc *sav.DocumentData, object int, party []mapload.PartyMember, chars []sav.Character, graph *cityObjectTopology) (modLostItem, bool) {
	if graph == nil || len(chars) != len(party) {
		return modLostItem{}, false
	}
	ref := uint16(object + 1)
	for i, c := range chars {
		at := int(c.ArchiveIndex) - 1
		if at < 0 || at >= len(doc.Objects) {
			continue
		}
		root, err := cityMutationParty(graph, party[i].ID)
		if err != nil {
			continue
		}
		r := &doc.Objects[at]
		pack, _, err := cityTopologyRefs(r, "Inventory")
		if k := slices.Index(pack, ref); err == nil && k >= 0 && k < len(root.Pack) {
			return modLostItem{member: party[i].ID, id: root.Pack[k]}, true
		}
		equipment, err := readSavEquipment(r, cityTopologyEquipment, func(field string) ([]uint16, bool, error) { return cityTopologyRefs(r, field) })
		if err != nil {
			continue
		}
		if s := slices.Index(equipment.Refs[:], ref); s >= 0 {
			return modLostItem{member: party[i].ID, id: root.Worn[s], slot: s + 1}, true
		}
	}
	return modLostItem{}, false
}

// takeModLoss takes each lost item out of the restored session s through the
// town's own item operations, as the shop room runs them for the member who
// holds it: a pack unit leaves as a use takes it, a worn item comes off
// without a pack and its slot stays empty.
func (f *FrontEnd) takeModLoss(s *CampaignSession, loss modTownLoss) error {
	if len(loss.items) == 0 {
		return nil
	}
	in := f.InstallResources
	t := detachedShop(s, &in)
	t.detachedHolder = true
	for _, lost := range loss.items {
		i := slices.IndexFunc(s.Carried, func(p mapload.PartyMember) bool { return p.ID == lost.member })
		t.shopMember = i
		if i < 0 || t.shopMemberIndex() != i {
			return fmt.Errorf("a removed item's holder %q is not a member the town serves", lost.member)
		}
		var err error
		action := t.withCityShopMutation(func(n *shopController) (ui.TownAction, error) {
			err = n.takeModLostItem(lost)
			return ui.TownAction{}, err
		})
		if err != nil {
			return err
		}
		if action.Msg != "" {
			return errors.New(action.Msg)
		}
	}
	return nil
}

// takeModLostItem takes one lost item off its holder, the shown member.
func (t *shopController) takeModLostItem(lost modLostItem) error {
	root, err := cityMutationParty(t.sess.Town.cityObjects, lost.member)
	if err != nil {
		return err
	}
	if lost.slot == 0 {
		k := slices.Index(root.Pack, lost.id)
		if k < 0 {
			return fmt.Errorf("a removed item is no longer in %q's pack", lost.member)
		}
		_, err := t.cityShopTakePack(k, true, true)
		return err
	}
	if root.Worn[lost.slot-1] != lost.id {
		return fmt.Errorf("a removed item is no longer worn by %q", lost.member)
	}
	_, worn, err := t.cityShopTakeWorn(lost.slot, false)
	if err == nil && !worn {
		err = fmt.Errorf("a removed item's slot of %q is empty", lost.member)
	}
	return err
}
