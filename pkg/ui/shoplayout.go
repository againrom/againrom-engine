package ui

import "image"

// ShopLayout is the shop page's geometry and the panels it shows. The page is
// one kind for both games; each game's shop is a value of this type, and the
// view names the value it is drawn with.
//
// The grids keep their counts (a 2x3 shelf grid, two strips of five) and
// their 80x80 cells; a layout moves their first cells, the arrows, the
// buttons, the shelf hit rectangles and the merchant.
type ShopLayout struct {
	// The regions: the shelf grid, the table strip, the pack strip and the
	// button panel.
	Shelf, Table, Pack, Buttons image.Rectangle

	// The top-left corner of each grid's first cell.
	ShelfCell, TableCell, PackCell image.Point

	// The shelf grid's two scroll arrows and the pack strip's two corners.
	ArrowUp, ArrowDown, PackLeft, PackRight image.Rectangle

	// ShelfPick is where shelf i is clicked; Merchant is where the merchant
	// answers his hover.
	ShelfPick [4]image.Rectangle
	Merchant  image.Rectangle

	// Button is the four command buttons, top to bottom.
	Button [4]image.Rectangle

	// Room and Frame are where the room picture and its frame are drawn.
	Room, Frame image.Point

	// Message is the line the page's last answer is printed on.
	Message image.Rectangle

	// Character is the borrowed character panel: the member's figure and
	// doll, the picker and the mode corner. Book is the spell book the panel's
	// corner opens over the table strip; it needs Character.
	Character, Book bool

	// BookOffset moves the open book from the table strip's corner.
	BookOffset image.Point

	// NoWear keeps the panel's doll out of the drags: its items answer a
	// hover, and no drag takes an item off it or puts one on it.
	NoWear bool

	// TableScroll lets the wheel turn the table strip one place at a time,
	// for a deal whose table holds more places than the strip shows.
	TableScroll bool

	// LabelOffset moves each command button's label line from the shared
	// label band.
	LabelOffset [4]image.Point
}

// firstShopLayout is the first game's page (SHOP-SCREEN-030 … 035,
// SHOP-SHELF-047, SHOP-MERCHANT-046, SHOP-FIGURE-041).
var firstShopLayout = ShopLayout{
	Shelf: shopShelfRegion, Table: shopTableRegion, Pack: shopPackRegion, Buttons: shopButtonRegion,
	ShelfCell: image.Pt(1, 31), TableCell: image.Pt(32, 303), PackCell: image.Pt(32, 395),
	ArrowUp: shopArrowUpRect, ArrowDown: shopArrowDownRect, PackLeft: shopPackLeftRect, PackRight: shopPackRightRect,
	ShelfPick: shopShelfPickRects, Merchant: shopMerchantRect, Button: shopButtonRects,
	Room: shopRoomOrigin, Frame: shopFrameOrigin, Message: shopMessageRect,
	Character: true, Book: true,
}

// FirstShopLayout answers the first game's page geometry, every panel shown.
func FirstShopLayout() ShopLayout { return firstShopLayout }

// layout is the geometry the view is drawn with: its own, else the first
// game's.
func (v ShopScreenView) layout() *ShopLayout {
	if v.Layout != nil {
		return v.Layout
	}
	return &firstShopLayout
}

// ShelfCellRect is cell i of the shelf grid, two columns.
func (l *ShopLayout) ShelfCellRect(i int) image.Rectangle {
	col, row := i%shopShelfCols, i/shopShelfCols
	return shopCellAt(l.ShelfCell.Add(image.Pt(shopCellSize*col, shopCellSize*row)))
}

// TableCellRect is cell i of the table strip.
func (l *ShopLayout) TableCellRect(i int) image.Rectangle {
	return shopCellAt(l.TableCell.Add(image.Pt(shopCellSize*i, 0)))
}

// PackCellRect is cell i of the pack strip.
func (l *ShopLayout) PackCellRect(i int) image.Rectangle {
	return shopCellAt(l.PackCell.Add(image.Pt(shopCellSize*i, 0)))
}

func shopCellAt(p image.Point) image.Rectangle {
	return image.Rectangle{Min: p, Max: p.Add(image.Pt(shopCellSize, shopCellSize))}
}

// WheelAt answers which grid a wheel turn at p scrolls.
func (l *ShopLayout) WheelAt(p image.Point) ShopWheelRegion {
	switch {
	case p.In(l.Shelf):
		return ShopWheelShelf
	case p.In(l.Pack):
		return ShopWheelPack
	case l.TableScroll && p.In(l.Table):
		return ShopWheelTable
	}
	return ShopWheelNone
}
