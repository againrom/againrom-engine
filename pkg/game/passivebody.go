package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// passiveRecordContents reads the worn and carried items a dead actor record
// references, in slot and container order. A record keeps its held Weapon
// elsewhere when heldWeapon is false. An item this reader cannot read stays
// out of the result.
func passiveRecordContents(doc *sav.DocumentData, object uint16, heldWeapon bool) sim.PassiveBody {
	var out sim.PassiveBody
	if doc == nil || object == 0 || int(object) > len(doc.Objects) {
		return out
	}
	r := &doc.Objects[object-1]
	read := func(ref uint16) (sim.ItemStack, bool) {
		if ref == 0 {
			return sim.ItemStack{}, false
		}
		v, err := savedItemRecord(doc, ref)
		if err != nil {
			return sim.ItemStack{}, false
		}
		return v.Value, true
	}
	for slot, field := range []string{"HeldWeapon", "HeldShield"} {
		if slot == 0 && !heldWeapon {
			continue
		}
		if refs, _ := savedObjectRefs(r, field); len(refs) == 1 {
			if v, ok := read(refs[0]); ok {
				out.Worn[slot] = v.Instance()
			}
		}
	}
	if r.Class != "Unit" {
		refs, _ := savedObjectRefs(r, "Worn")
		for slot := 2; slot < len(refs) && slot < sim.EquipSlots; slot++ {
			if v, ok := read(refs[slot]); ok {
				out.Worn[slot] = v.Instance()
			}
		}
	}
	refs, _ := savedObjectRefs(r, "Inventory")
	for _, ref := range refs {
		if v, ok := read(ref); ok {
			out.Carried = append(out.Carried, v)
		}
	}
	return out
}

func passiveBodyEmpty(b sim.PassiveBody) bool {
	for _, item := range b.Worn {
		if !item.Empty() {
			return false
		}
	}
	return len(b.Carried) == 0
}

// departedBodies are the bodies the World kept for departed actors without a
// map unit or a party entry, in ID order.
func departedBodies(w *sim.World) []sim.Entity {
	var out []sim.Entity
	for _, p := range w.PassiveBodies() {
		if p.Body != nil {
			out = append(out, *p.Body)
		}
	}
	return out
}

func validateDepartedBodies(bodies []sim.Entity) error {
	if len(bodies) > 65535 {
		return fmt.Errorf("departed bodies exceed their bound")
	}
	for i, e := range bodies {
		if e.MapUnitID != 0 || e.SourceBinding.Class != 0 || i > 0 && e.ID <= bodies[i-1].ID {
			return fmt.Errorf("departed body %d is placed, source-bound or out of order", e.ID)
		}
	}
	return nil
}

// restoreDepartedContents gives each departed actor the worn and carried
// items its record references, and no others. A departed actor without a map
// unit takes the body the native action leaf kept for it.
func restoreDepartedContents(ms *Mission, a *currentActionData, rows []sim.CurrentTerminalActor) error {
	bodies := map[sim.EntityID]sim.Entity{}
	for _, e := range a.DepartedBodies {
		bodies[e.ID] = e
	}
	var kept []sim.PassiveBody
	for _, row := range rows {
		if row.LeavesNoBody() {
			continue
		}
		object, _, err := currentTerminalActionObject(ms.savedDocument, a, row.ID)
		if err != nil {
			return err
		}
		// The record is the authority: a row it gives no items removes what
		// the retirement of a constructed actor may have kept.
		contents := passiveRecordContents(ms.savedDocument.Document, object, true)
		contents.ID = row.ID
		if body, ok := bodies[row.ID]; ok && row.MapUnitID == 0 && !passiveBodyEmpty(contents) {
			contents.Body = &body
		}
		kept = append(kept, contents)
	}
	if len(kept) == 0 {
		return nil
	}
	return ms.World.RestorePassiveBodies(kept)
}
