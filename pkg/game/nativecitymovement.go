package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// Missing Human movement starts from Unit defaults. Retained movement fields
// are projected separately, including explicit zero values.
// TERR-MOVE-057: -1 keeps the base defaults, otherwise the Human table supplies
// Unit+49/+4a. MOVE-DOM-025 supplies the new mover's corresponding mask.
func nativeCityInitializeHumanMovement(unit *sav.CityUnitData, table *mapload.Table) error {
	if unit == nil || len(unit.Token) != 37 || len(unit.Scalar1) != 19 || len(unit.Raw154) != 180 {
		return originalCityUnsupportedf("native city Human movement has no complete constructor")
	}
	base := data.UnitDefaults()
	size, domain := base.TokenSize, base.MovementType
	row := int(unit.Token[16])
	if table != nil && table.Humans != nil && row > 0 && row < table.Humans.Len() && table.Humans.EntryName(row) != "" {
		params := make([]int32, data.MinHumanRow)
		for i := range params {
			params[i] = -1
		}
		copy(params, table.Humans.EntryParams(row))
		d, err := data.NewHumanDef(table.Humans.EntryName(row), params)
		if err != nil {
			return originalCityUnsupportedf("native city Human movement definition: %v", err)
		}
		size, domain = d.TokenSize, d.MovementType
	}
	if size < 1 || size > 255 || domain < 1 || domain > 3 {
		return originalCityUnsupportedf("native city Human movement cannot represent size %d domain %d", size, domain)
	}
	unit.Scalar1[0], unit.Scalar1[1] = byte(size), byte(domain)
	unit.Raw154[5] = [...]byte{0, 0x41, 0x44, 0x82}[domain]
	return nil
}
