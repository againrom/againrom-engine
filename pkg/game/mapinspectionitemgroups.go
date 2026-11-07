package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) itemChoiceGroups(choices []ui.InspectionChoice) []ui.InspectionChoiceGroup {
	var groups []ui.InspectionChoiceGroup
	seen := map[[2]string]int{}
	for _, choice := range choices {
		if choice.Value <= 0 || choice.Value > 65535 {
			continue
		}
		code := data.ItemCode(choice.Value)
		kind, material := "", "None"
		if x.table != nil {
			collection := map[int]data.Collection{1: x.table.Weapons, 2: x.table.Shields}
			if slot, ok := data.EquipSlotFor(code); ok {
				rows := collection[slot]
				if rows == nil && slot >= 3 {
					rows = x.table.Armors
				}
				if rows != nil && code.D() < rows.Len() {
					kind = rows.EntryName(code.D())
				}
				material = fmt.Sprintf("#%d", code.A())
				if x.table.Materials != nil && code.A() < x.table.Materials.Len() && x.table.Materials.EntryName(code.A()) != "" {
					material = x.table.Materials.EntryName(code.A())
				}
			} else if code.B() == data.ItemClassCarried {
				kind = map[uint8]string{3: "Potions", 4: "Scrolls", 5: "Books"}[mapload.ItemInstanceFromCode(uint16(code), x.table).Kind]
				if kind == "" {
					kind = "Other items"
				}
			}
		}
		if kind == "" {
			kind = fmt.Sprintf(x.word("editor.item.unknownkind"), code.B(), code.D())
		}
		key := [2]string{kind, material}
		i, ok := seen[key]
		if !ok {
			i = len(groups)
			seen[key] = i
			groups = append(groups, ui.InspectionChoiceGroup{Block: x.itemGroupWord("kind", kind), Head: x.itemGroupWord("material", material)})
		}
		groups[i].Values = append(groups[i].Values, choice.Value)
	}
	return groups
}

var itemGroupWords = map[string]struct{ word string }{
	"kind:BareHands":          {word: "editor.item.kind.barehands"},
	"kind:Dagger":             {word: "editor.item.kind.dagger"},
	"kind:Short Sword":        {word: "editor.item.kind.shortsword"},
	"kind:Long Sword":         {word: "editor.item.kind.longsword"},
	"kind:Bastard Sword":      {word: "editor.item.kind.bastardsword"},
	"kind:Two Handed Sword":   {word: "editor.item.kind.twohandedsword"},
	"kind:Club":               {word: "editor.item.kind.club"},
	"kind:Spiked Club":        {word: "editor.item.kind.spikedclub"},
	"kind:Mace":               {word: "editor.item.kind.mace"},
	"kind:Morning Star":       {word: "editor.item.kind.morningstar"},
	"kind:Pick Hammer":        {word: "editor.item.kind.pickhammer"},
	"kind:War Hammer":         {word: "editor.item.kind.warhammer"},
	"kind:Staff":              {word: "editor.item.kind.staff"},
	"kind:Shaman Staff":       {word: "editor.item.kind.shamanstaff"},
	"kind:Pike":               {word: "editor.item.kind.pike"},
	"kind:Halberd":            {word: "editor.item.kind.halberd"},
	"kind:Lance":              {word: "editor.item.kind.lance"},
	"kind:Axe":                {word: "editor.item.kind.axe"},
	"kind:Two Handed Axe":     {word: "editor.item.kind.twohandedaxe"},
	"kind:Short Bow":          {word: "editor.item.kind.shortbow"},
	"kind:Long Bow":           {word: "editor.item.kind.longbow"},
	"kind:Crossbow":           {word: "editor.item.kind.crossbow"},
	"kind:Sonic Beam":         {word: "editor.item.kind.sonicbeam"},
	"kind:Flame Thrower":      {word: "editor.item.kind.flamethrower"},
	"kind:Boulder Thrower":    {word: "editor.item.kind.boulderthrower"},
	"kind:Plasma Sword":       {word: "editor.item.kind.plasmasword"},
	"kind:Ring":               {word: "editor.item.kind.ring"},
	"kind:Amulet":             {word: "editor.item.kind.amulet"},
	"kind:Hat":                {word: "editor.item.kind.hat"},
	"kind:Cap":                {word: "editor.item.kind.cap"},
	"kind:Low Hat":            {word: "editor.item.kind.lowhat"},
	"kind:Helm":               {word: "editor.item.kind.helm"},
	"kind:Soft Helm":          {word: "editor.item.kind.softhelm"},
	"kind:Chain Helm":         {word: "editor.item.kind.chainhelm"},
	"kind:Full Helm":          {word: "editor.item.kind.fullhelm"},
	"kind:Plate Helm":         {word: "editor.item.kind.platehelm"},
	"kind:Cloak":              {word: "editor.item.kind.cloak"},
	"kind:Cape":               {word: "editor.item.kind.cape"},
	"kind:Robe":               {word: "editor.item.kind.robe"},
	"kind:Dress":              {word: "editor.item.kind.dress"},
	"kind:Soft Mail":          {word: "editor.item.kind.softmail"},
	"kind:Chain Mail":         {word: "editor.item.kind.chainmail"},
	"kind:Scale Mail":         {word: "editor.item.kind.scalemail"},
	"kind:Cuirass":            {word: "editor.item.kind.cuirass"},
	"kind:Plate Cuirass":      {word: "editor.item.kind.platecuirass"},
	"kind:Bracers":            {word: "editor.item.kind.bracers"},
	"kind:Soft Bracers":       {word: "editor.item.kind.softbracers"},
	"kind:Plate Bracers":      {word: "editor.item.kind.platebracers"},
	"kind:Gloves":             {word: "editor.item.kind.gloves"},
	"kind:Soft Gauntlets":     {word: "editor.item.kind.softgauntlets"},
	"kind:Chain Gauntlets":    {word: "editor.item.kind.chaingauntlets"},
	"kind:Scale Gauntlets":    {word: "editor.item.kind.scalegauntlets"},
	"kind:Shoes":              {word: "editor.item.kind.shoes"},
	"kind:Soft Boots":         {word: "editor.item.kind.softboots"},
	"kind:Chain Boots":        {word: "editor.item.kind.chainboots"},
	"kind:Plate Boots":        {word: "editor.item.kind.plateboots"},
	"kind:Buckler":            {word: "editor.item.kind.buckler"},
	"kind:Small":              {word: "editor.item.kind.small"},
	"kind:Soft Small":         {word: "editor.item.kind.softsmall"},
	"kind:Wooden Small":       {word: "editor.item.kind.woodensmall"},
	"kind:Large":              {word: "editor.item.kind.large"},
	"kind:Soft Large":         {word: "editor.item.kind.softlarge"},
	"kind:Wooden Large":       {word: "editor.item.kind.woodenlarge"},
	"kind:Tower":              {word: "editor.item.kind.tower"},
	"kind:Wooden Tower":       {word: "editor.item.kind.woodentower"},
	"kind:Potions":            {word: "editor.item.kind.potions"},
	"kind:Scrolls":            {word: "editor.item.kind.scrolls"},
	"kind:Books":              {word: "editor.item.kind.books"},
	"kind:Other items":        {word: "editor.item.kind.otheritems"},
	"material:Iron":           {word: "editor.item.material.iron"},
	"material:Bronze":         {word: "editor.item.material.bronze"},
	"material:Steel":          {word: "editor.item.material.steel"},
	"material:Silver":         {word: "editor.item.material.silver"},
	"material:Gold":           {word: "editor.item.material.gold"},
	"material:Mithrill":       {word: "editor.item.material.mithrill"},
	"material:Adamantium":     {word: "editor.item.material.adamantium"},
	"material:Meteoric":       {word: "editor.item.material.meteoric"},
	"material:Wood":           {word: "editor.item.material.wood"},
	"material:Magic Wood":     {word: "editor.item.material.magicwood"},
	"material:Leather":        {word: "editor.item.material.leather"},
	"material:Hard Leather":   {word: "editor.item.material.hardleather"},
	"material:Dragon Leather": {word: "editor.item.material.dragonleather"},
	"material:Bone":           {word: "editor.item.material.bone"},
	"material:Crystal":        {word: "editor.item.material.crystal"},
	"material:None":           {word: "editor.item.material.none"},
}

func (x *MapInspector) itemGroupWord(axis, name string) string {
	if caption, ok := itemGroupWords[axis+":"+name]; ok {
		return x.inspectionUTF8(x.word(caption.word))
	}
	return x.inspectionUTF8(name)
}

func (s *mapInspectionEdits) itemPictures(choices []ui.InspectionChoice) map[int64]*image.RGBA {
	for _, choice := range choices {
		_, _ = s.ItemLook(choice.Value)
	}
	return s.inspector.recordPictures["item"]
}
