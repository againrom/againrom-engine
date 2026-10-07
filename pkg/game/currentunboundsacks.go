package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// Unregistered ground items still have exact ordinary Sack/Contents addresses.
// Their absent native IDs do not suppress the ordinary value reader.
func currentUnboundSackValues(doc *sav.DocumentData, rows []currentOwnedObject, items map[uint16]sim.ItemStack) ([]sim.Sack, error) {
	objects := map[uint16]bool{}
	for _, row := range rows {
		if row.Kind == 4 && row.ID == 0 && row.Object != 0 {
			objects[row.Object] = true
		}
	}
	var ground []sim.Sack
	var count uint64
	for _, object := range doc.World.Sacks {
		if !objects[object] {
			continue
		}
		delete(objects, object)
		record := &doc.Objects[object-1]
		token, gold, _, err := savedSackRecord(record)
		if err != nil {
			return nil, err
		}
		sack := sim.Sack{X: int32(token.Position[2]), Y: int32(token.Position[3]), Gold: gold}
		refs, _ := savedObjectRefs(record, "Contents")
		for _, ref := range refs {
			value, ok := items[ref]
			if !ok {
				return nil, fmt.Errorf("current unregistered Sack lacks an ordinary Item binding")
			}
			count += uint64(value.Count) * uint64(1+len(value.Effects))
			if count > 1<<20 {
				return nil, fmt.Errorf("current unregistered Sack exceeds expanded ground population")
			}
			for range value.Count {
				sack.ItemInstances = append(sack.ItemInstances, value.Instance())
			}
		}
		ground = append(ground, sack)
	}
	return ground, nil
}
