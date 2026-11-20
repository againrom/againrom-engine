package ui

import "reflect"

// InspectionSeams is implemented by a host that offers only some of its
// optional seams on the map it holds, as the map's format allows.
type InspectionSeams interface {
	Offers(seam reflect.Type) bool
}

// Seam is edits read as the optional seam T: the host must implement T, and
// when it names the seams it offers it must offer T on its map.
func Seam[T any](edits any) (T, bool) {
	t, ok := edits.(T)
	if ok {
		if g, gated := edits.(InspectionSeams); gated && !g.Offers(reflect.TypeFor[T]()) {
			var none T
			return none, false
		}
	}
	return t, ok
}
