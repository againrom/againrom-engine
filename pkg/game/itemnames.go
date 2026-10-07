package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/itemname"
)

// The shipped item-name table's address and read (0151, ITEM-DISPNAME-036).
// The parse itself is pkg/formats/itemname's Parse, one tier down; this
// file is only the two addresses and the read, exactly as bodylist.go is
// only the address and the read of the shipped body list.

// ItemNameKeyAddress and ItemNameTextAddress are the two files the shipped
// item-name table is read from: mainPrefix, the same prefix HeroPictureAddress
// and EventTextPath compose from, and the payload's own two names under
// main.res's text tree.
const (
	ItemNameKeyAddress  = mainPrefix + "text/itemname.bin"
	ItemNameTextAddress = mainPrefix + "text/itemname.txt"
)

// ReadItemNames reads the shipped item-name table off src and reports
// whether both halves were there — ReadBodyList's own contract, one pair of
// files over.
//
// THERE IS NO ERROR RETURN, for ReadBodyList's own reason: a front end that
// cannot read the table still opens missions. What it loses is every stored
// name; itemName (world.go) already falls back to a code's own weapon
// recovery and, past that, its seven-digit digits, so a missing table
// narrows what is named rather than blocking anything.
func ReadItemNames(src entrySource) (data.ItemNames, bool) {
	if src == nil {
		return nil, false
	}
	bin, err := src.ReadFile(ItemNameKeyAddress)
	if err != nil {
		return nil, false
	}
	txt, err := src.ReadFile(ItemNameTextAddress)
	if err != nil {
		return nil, false
	}
	raw := itemname.Parse(bin, txt)
	names := make(data.ItemNames, len(raw))
	for k, v := range raw {
		names[data.ItemCode(k)] = v
	}
	return names, true
}
