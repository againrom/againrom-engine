package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"reflect"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func main() {
	assets := flag.String("assets", "", "read-only second-game install")
	flag.Parse()
	if err := run(*assets); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	f, err := game.NewFrontEnd(root)
	if err != nil {
		return err
	}
	f.SetRandomLaunch(71, true, false)
	f.SetDeterministicFrames(true)
	app := f.App("inn stock witness")
	app.Layout(640, 480)
	if err := app.HeadlessActivate("new game"); err != nil {
		return err
	}
	previous := ""
	for step := 0; step < 3; step++ {
		if step > 0 {
			if err := app.HeadlessActivate("TAVERN"); err != nil {
				return err
			}
			before, _ := f.Town.SecondState()
			gold := f.Town.Gold()
			x, y, ok := app.HeadlessNativePoint(image.Pt(550, 205))
			if !ok {
				return fmt.Errorf("the inn has no native point")
			}
			for _, action := range []string{"press", "release"} {
				if err := app.HeadlessPointer(action, x, y); err != nil {
					return err
				}
			}
			after, _ := f.Town.SecondState()
			if !reflect.DeepEqual(before, after) || f.Town.Gold() != gold || app.HeadlessMessage() != "" {
				return fmt.Errorf("Sleep changed the campaign, room, gold or message")
			}
			if err := app.HeadlessActivate("EXIT"); err != nil {
				return err
			}
		}
		if err := app.HeadlessActivate("SHOP"); err != nil {
			return err
		}
		shop := f.TownScreen().(ui.TownShopScreen)
		var signature strings.Builder
		for shelf := 0; shelf < 4; shelf++ {
			shop.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfPick, Index: shelf})
			seen := map[int]bool{}
			var first []string
			for page := 0; page < 512; page++ {
				view := shop.ShopScreen()
				for i, cell := range view.Shelf {
					row := view.ShelfOffset + i
					if !cell.Occupied() || seen[row] {
						continue
					}
					seen[row] = true
					fmt.Fprintf(&signature, "%d/%d/%d/%v;", row, cell.Count, cell.Price, cell.Info)
					if len(first) < 3 {
						name := strings.Join(cell.Text, " ")
						if len(cell.Info) > 0 {
							name = cell.Info[0]
						}
						first = append(first, vfs.DecodeText([]byte(name)))
					}
				}
				if !view.ShelfForward {
					break
				}
				shop.ShopClick(ui.ShopControl{Kind: ui.ShopControlArrowDown})
			}
			fmt.Printf("Sleep %d shelf %d: %d rows; %s\n", step, shelf+1, len(seen), strings.Join(first, " | "))
		}
		if step > 0 && signature.String() == previous {
			return fmt.Errorf("Sleep %d kept the old stock", step)
		}
		previous = signature.String()
		ex, ey, err := app.HeadlessShopPoint("button", 3)
		if err != nil {
			return err
		}
		for _, action := range []string{"press", "release"} {
			if err := app.HeadlessPointer(action, ex, ey); err != nil {
				return err
			}
		}
	}
	return nil
}
