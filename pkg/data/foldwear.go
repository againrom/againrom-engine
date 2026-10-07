package data

func FoldWear(e Equipment, shapes, materials ScaleTable, shields, armors Collection) EquipMod {
	var mod EquipMod
	for n := 1; n <= EquipSlots; n++ {
		code, _ := e.Code(n) // n is always 1..12; the discarded bool is a
		// fact about the slot number n, not about this fold (HeroBodyFor's
		// own precedent, equip.go).
		foldPiece(&mod, code, shapes, materials, shields, armors)
	}
	return mod
}

// FoldLayers is the contribution of clothing layers worn beside the twelve
// slots: each code adds its defence and absorption exactly as a worn item of its
// table does, so a layer enters the same EquipMod sum.
func FoldLayers(mod *EquipMod, codes []ItemCode, shapes, materials ScaleTable, shields, armors Collection) {
	for _, code := range codes {
		foldPiece(mod, code, shapes, materials, shields, armors)
	}
}

func foldPiece(mod *EquipMod, code ItemCode, shapes, materials ScaleTable, shields, armors Collection) {
	switch code.B() {
	case shieldItemClass:
		if shields == nil {
			return
		}
		piece, err := ShieldFromCode(code, shapes, materials, shields)
		if err == nil {
			mod.Defence += piece.Defence
			mod.Absorption += piece.Absorption
		}
	default:
		if armors == nil {
			return
		}
		piece, err := ArmorFromCode(code, shapes, materials, armors)
		if err == nil {
			mod.Defence += piece.Defence
			mod.Absorption += piece.Absorption
		}
	}
}
