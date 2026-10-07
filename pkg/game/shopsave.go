package game

import (
	"fmt"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type shopOwnerQuantity struct {
	PartyID   string
	Member    int
	PackIndex int
	Count     int32
}

func (t *townScreen) ownShopItem(item ShopItem, pack int) ShopItem {
	index := t.shopMemberIndex()
	member := t.shopPartyMember(index)
	if member != nil {
		item.owners = []shopOwnerQuantity{{PartyID: member.ID, Member: index, PackIndex: pack, Count: item.Count}}
	}
	return item
}

func takeShopOwners(owners []shopOwnerQuantity, count int32) (taken, remaining []shopOwnerQuantity) {
	for _, owner := range owners {
		n := min(owner.Count, count)
		if n > 0 {
			part := owner
			part.Count = n
			taken = append(taken, part)
			owner.Count -= n
			count -= n
		}
		if owner.Count > 0 {
			remaining = append(remaining, owner)
		}
	}
	return taken, remaining
}

func resizedShopOwners(owners []shopOwnerQuantity, count int32) []shopOwnerQuantity {
	if len(owners) == 0 {
		return nil
	}
	var total int32
	for _, owner := range owners {
		total += owner.Count
	}
	if count < total {
		_, owners = takeShopOwners(owners, total-count)
		return owners
	}
	owners = slices.Clone(owners)
	owners[0].Count += count - total
	return owners
}

func (s *Shop) growOwnedTablePlace(index int, incoming ShopItem) bool {
	if !s.GrowTablePlace(index, incoming.Count) {
		return false
	}
	s.table[index].owners = append(slices.Clone(s.table[index].owners), incoming.owners...)
	return true
}

func (f *FrontEnd) captureShop() *Shop {
	if f.Shop == nil || len(f.Shop.table) == 0 {
		return nil
	}
	shop := cloneShopMutation(f.Shop)
	index := 0
	if f.townUI != nil {
		index = townPickerIndex(f.Carried, f.townUI.shopMember)
	}
	for i, place := range shop.table {
		if place.Mine && len(place.owners) == 0 && index >= 0 && index < len(f.Carried) {
			shop.table[i].owners = []shopOwnerQuantity{{PartyID: f.Carried[index].ID, Member: index, PackIndex: -1, Count: place.Count}}
		}
	}
	return shop
}

func shopOwnerMember(party []mapload.PartyMember, owner shopOwnerQuantity) (int, error) {
	if owner.PartyID != "" {
		for i, member := range party {
			if member.ID == owner.PartyID {
				return i, nil
			}
		}
	} else if owner.Member >= 0 && owner.Member < len(party) {
		return owner.Member, nil
	}
	return 0, fmt.Errorf("staged shop item has no current owner")
}

func groupedShopOwners(owners []shopOwnerQuantity) []shopOwnerQuantity {
	var out []shopOwnerQuantity
	for _, owner := range owners {
		found := false
		for i := range out {
			if out[i].PartyID == owner.PartyID && (owner.PartyID != "" || out[i].Member == owner.Member) {
				out[i].Count += owner.Count
				found = true
				break
			}
		}
		if !found {
			out = append(out, owner)
		}
	}
	return out
}

func (t *townScreen) restoreShopOwner(item ShopItem, owner shopOwnerQuantity) error {
	index, err := shopOwnerMember(t.sess.Carried, owner)
	if err != nil {
		return err
	}
	member := t.sess.Carried[index]
	stacks := slices.Clone(cityMemberStacks(member, t.in.Table))
	at := len(stacks)
	if owner.PackIndex >= 0 {
		at = min(owner.PackIndex, len(stacks))
	}
	graph, err := moveCityItemRoot(t.sess.Town.cityObjects, item.cityID, cityItemLocation{}, cityItemLocation{PartyID: member.ID, Kind: cityItemPack, Index: at})
	if err != nil {
		return err
	}
	stacks = slices.Insert(stacks, at, sim.StackItem(item.Instance(), uint32(item.Count)))
	if err := t.cityShopSetPack(index, stacks, false); err != nil {
		return err
	}
	t.sess.Town.cityObjects = graph
	return nil
}

// DIV-1443: only the detached SAV capture settles the table. Live trade,
// payment and selection remain unchanged after SAVE, cancellation or failure.
func (f *FrontEnd) resolveShopSnapshot(s Snapshot) (Snapshot, error) {
	if s.Mission != 0 || s.shop == nil || len(s.shop.table) == 0 {
		return s, nil
	}
	in := f.InstallResources
	n := &CampaignSession{
		Town: restoreTown(f.Campaign.Value(), s), Carried: mapload.CloneParty(s.Party), Shop: cloneShopMutation(s.shop)}
	t := &townScreen{sess: n, in: &in, pc: &PersistenceContext{}, room: roomShop}
	t.bindServices(&RuntimeServices{}, &in, &Presentation{})
	p, err := newCityObjectProjection(n.Town.cityObjects, n.Carried, in.Table)
	if err != nil {
		return Snapshot{}, err
	}
	n.Town.cityObjects = p.graph
	if err := t.validateCityShopValues(p); err != nil {
		return Snapshot{}, err
	}
	for len(n.Shop.table) != 0 {
		index := len(n.Shop.table) - 1
		place := n.Shop.table[index]
		if !place.Mine {
			n.Shop.table = slices.Delete(n.Shop.table, index, index+1)
			shelf := place.From
			if shelf < 0 || shelf >= numShopShelves {
				shelf = ShelfMagic
			}
			n.Shop.shelves[shelf] = append(n.Shop.shelves[shelf], place.ShopItem.Clone())
			continue
		}
		item, err := t.cityShopBind(place.ShopItem)
		if err != nil {
			return Snapshot{}, err
		}
		owners := groupedShopOwners(item.owners)
		var total int64
		for _, owner := range owners {
			if owner.Count <= 0 {
				return Snapshot{}, fmt.Errorf("staged shop owner has invalid quantity")
			}
			total += int64(owner.Count)
		}
		if len(owners) == 0 || total != int64(item.Count) {
			return Snapshot{}, fmt.Errorf("staged shop ownership differs from current quantity")
		}
		n.Shop.table[index].ShopItem = item
		for i := len(owners) - 1; i > 0; i-- {
			part, err := t.cityShopSplitQuantity(item, cityItemLocation{}, owners[i].Count)
			if err != nil {
				return Snapshot{}, err
			}
			item = n.Shop.table[index].ShopItem
			item.owners = slices.Clone(owners[:i])
			n.Shop.table[index].ShopItem = item
			if err := t.restoreShopOwner(part, owners[i]); err != nil {
				return Snapshot{}, err
			}
		}
		n.Shop.table = slices.Delete(n.Shop.table, index, index+1)
		if err := t.restoreShopOwner(item, owners[0]); err != nil {
			return Snapshot{}, err
		}
	}
	if err := t.validateCityShopRoots(); err != nil {
		return Snapshot{}, err
	}
	s.Party, s.CityObjects, s.shop = n.Carried, n.Town.cityObjects, nil
	return s, nil
}
