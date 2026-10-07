package main

import (
	"fmt"

	"againrom/pkg/formats/alm"
)

func cmdLoot(path string) error {
	m, _, err := loadMap(path)
	if err != nil {
		return err
	}

	loot, err := m.Loot()
	if err != nil {
		return err
	}

	fmt.Printf("map:   %q  %dx%d  version=%d\n", m.Name, m.Width, m.Height, m.FormatVersion)
	fmt.Printf("type8: present=%v  #records (meta +0x2c)=%d  body=%d bytes\n",
		m.Present(8), m.Meta.Word2C, len(m.LootSection.Body))

	var ground, stock, elements int
	var totalGold uint64
	for i, r := range loot.Records {
		kind := "stock"
		if r.Ground() {
			kind = "ground"
			ground++
		} else {
			stock++
		}
		fmt.Printf("  [%d] %-6s cell=(%d,%d) gold=%d elements=%d\n",
			i, kind, r.CellX(), r.CellY(), r.Gold, len(r.Elements))
		for j, e := range r.Elements {
			code := e.ItemCode()
			fmt.Printf("      element[%d] code=%#04x class=%d index=%d\n",
				j, code, alm.ItemClass(code), alm.ItemIndex(code))
		}
		elements += len(r.Elements)
		totalGold += uint64(r.Gold)
	}

	fmt.Printf("census: %d record(s) (%d ground, %d stock), %d element(s), total gold=%d, payload closed exactly\n",
		len(loot.Records), ground, stock, elements, totalGold)
	return nil
}
