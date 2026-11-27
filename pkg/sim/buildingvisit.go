package sim

import "slices"

type VisitClass uint8

const (
	VisitNone VisitClass = iota
	VisitShop
	VisitInn
)

type buildingKind struct {
	kind  uint16
	class VisitClass
	town  int
}

type buildingKinds []buildingKind

var secondGameBuildings = buildingKinds{
	{34, VisitShop, 1}, {35, VisitShop, 1}, {93, VisitShop, 2}, {94, VisitShop, 2}, {95, VisitShop, 2},
	{105, VisitShop, 3}, {106, VisitShop, 3}, {107, VisitShop, 3},
	{67, VisitInn, 1}, {68, VisitInn, 1}, {69, VisitInn, 1}, {99, VisitInn, 2}, {100, VisitInn, 2}, {101, VisitInn, 2},
	{111, VisitInn, 3}, {112, VisitInn, 3}, {113, VisitInn, 3},
}

var firstGameBuildings = buildingKinds{{34, VisitShop, 0}, {35, VisitShop, 0}}

type BuildingVisit struct {
	Entity    EntityID
	Structure StructureID
	Class     VisitClass
	Town      int
}

func (w *World) SetBuildingVisits(on bool, visitors []EntityID) {
	w.visits, w.visitors = on, slices.Clone(visitors)
}

func (w *World) VisitsBuildings() bool { return w.visits }

func (k buildingKinds) of(kind uint16) buildingKind {
	for _, b := range k {
		if b.kind == kind {
			return b
		}
	}
	return buildingKind{}
}

func (k buildingKinds) class(kind uint16) VisitClass { return k.of(kind).class }

func (w *World) visitClass(kind uint16) VisitClass {
	if !w.visits {
		return VisitNone
	}
	return w.game.buildings.class(kind)
}

func (w *World) StructureUsable(s Structure) bool {
	return s.Usable() || w.visitClass(s.Kind) != VisitNone
}

func (w *World) enterBuilding(i int, s Structure, class VisitClass) {
	if !w.takeOffMap(i) {
		return
	}
	if o := w.castObservation; o != nil {
		o.visits = append(o.visits, BuildingVisit{Entity: w.entities[i].ID, Structure: s.ID, Class: class,
			Town: w.game.buildings.of(s.Kind).town})
	}
}

func (w *World) leaveBuilding(i int) {
	e := w.entities[i]
	if !e.OffMap || w.returnToMap(i) {
		return
	}
	if x, y, ok := nearestFreeCell(e.X, e.Y)(w, i); ok {
		w.placeAt(i, x, y)
	}
}
