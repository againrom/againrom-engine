package game

import (
	"bytes"
	"encoding/binary"
	"image"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func scrollDescriptionInstalledOracle(t *testing.T, f *FrontEnd) func(uint16, int32, sim.SpellCharacteristics) []string {
	t.Helper()
	main := LoadTextTable(f.Archives.Containers, MainTextPath, f.textCode())
	spells := LoadTextTable(f.Archives.Containers, SpellNamesTextPath, f.textCode())
	items := LoadTextTable(f.Archives.Containers, ItemNameTextAddress, f.textCode())
	keys, err := f.Archives.Containers.ReadFile(ItemNameKeyAddress)
	quest81OK(t, err)
	if main == nil || spells == nil || items == nil {
		t.Fatal("installed scroll text tables are absent")
	}
	label := func(index int) string {
		value, ok := main.At(index)
		if index >= main.Lines() || !ok && index != 91 {
			t.Fatal("installed main.txt label is absent", index)
		}
		return value
	}
	prefixes := make(map[uint16]string)
	for id := uint16(1); id <= 28; id++ {
		params := f.Table.Spells.EntryParams(int(id))
		if len(params) < 19 {
			t.Fatal("installed spell row is absent", id)
		}
		code := uint16(shopSchoolTemplateCode(f.Table, "Scroll", params[2]))
		name := ""
		for row := 0; row*2+1 < len(keys); row++ {
			if binary.LittleEndian.Uint16(keys[row*2:row*2+2]) == code {
				name, _ = items.At(row)
			}
		}
		spell, ok := spells.At(int(id) - 1)
		if !ok || name == "" {
			t.Fatal("installed scroll or singular spell name is absent", id, code)
		}
		if id == 11 && LanguageSelector(f.Archives.Containers) == 1 {
			spell = EncodeInstallText("Вампиризм", 1)
		}
		prefixes[id] = name + " " + label(90) + " " + spell + label(91)
	}
	return func(id uint16, power int32, c sim.SpellCharacteristics) []string {
		return append([]string{prefixes[id]}, scrollDescriptionExpectedDetails(id, power, c, label)...)
	}
}

func scrollDescriptionInstalledItem(t *testing.T, f *FrontEnd, id uint16, power int32) sim.ItemInstance {
	t.Helper()
	params := f.Table.Spells.EntryParams(int(id))
	if len(params) < 19 {
		t.Fatal("installed scroll spell row is absent", id)
	}
	code := uint16(shopSchoolTemplateCode(f.Table, "Scroll", params[2]))
	if code == 0 {
		t.Fatal("installed scroll template is absent", id, params[2])
	}
	return sim.ItemInstance{Code: code, Kind: 4, Price: 1000 + int32(id),
		Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(id) | uint32(uint16(power))<<16}}}
}

func TestReleaseScrollDescriptionCompleteFieldMatrix(t *testing.T) {
	f := releaseFront(t)
	oracle := scrollDescriptionInstalledOracle(t, f)
	rules := mapload.SpellRules(f.Table)
	if len(rules) != 28 {
		t.Fatal("installed descriptor rows", len(rules))
	}
	count := 0
	for id := uint16(1); id <= 27; id++ {
		if rules[id-1].ID != id {
			t.Fatal("installed rule identity differs", id, rules[id-1])
		}
		for power := int32(1); power <= 100; power++ {
			item := scrollDescriptionInstalledItem(t, f, id, power)
			c := sim.SpellCharacteristicsAtPower(mapload.TableRules(f.Table), rules[id-1], power)
			want := oracle(id, power, c)
			if got := itemInstanceInfoLines(item, f.Table, f.Words); !slices.Equal(got, want) {
				t.Fatalf("ordinary spell%d power%d = %q, want %q", id, power, got, want)
			}
			count++
		}
	}
	if count != 2700 {
		t.Fatal("ordinary description population", count)
	}
	for _, stored := range []int32{0, 1, 80, 100, 101, 255, 256, -1} {
		power := int32(uint8(stored))
		item := scrollDescriptionInstalledItem(t, f, 28, stored)
		c := sim.SpellCharacteristicsAtPower(mapload.TableRules(f.Table), rules[27], power)
		want := oracle(28, power, c)
		if got := itemInstanceInfoLines(item, f.Table, f.Words); !slices.Equal(got, want) {
			t.Fatalf("Slow descriptor stored%d byte%d = %q, want %q", stored, power, got, want)
		}
	}
	t.Logf("%d ordinary field/label compositions plus 8 Slow descriptor controls; numeric input is the shared engine projection, not a native numeric oracle", count)
}

func TestReleaseScrollDescriptionPopulationSAVAndNextSale(t *testing.T) {
	app, screen := releaseShopApp(t)
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f := frontOf(screen)
	f.SetDeterministicFrames(true)
	oracle := scrollDescriptionInstalledOracle(t, f)
	rules := mapload.SpellRules(f.Table)
	var source []sim.ItemStack
	expected := make(map[uint16][]string)
	for id := uint16(1); id <= 28; id++ {
		power := int32(1 + id*7%100)
		quantity := uint32(1)
		if id%2 == 0 {
			quantity = 3
		}
		item := scrollDescriptionInstalledItem(t, f, id, power)
		source = append(source, sim.StackItem(item, quantity))
		c := sim.SpellCharacteristicsAtPower(mapload.TableRules(f.Table), rules[id-1], power)
		expected[id] = oracle(id, power, c)
	}
	cityShopSetTestPack(t, f, 0, source...)
	cityShopGraph(t, f)
	screen.CloseTip()
	f.Carried[0].Hero.Mind = 1
	scrollDescriptionCheckPack(t, f, app, screen, source, expected)
	f.Carried[0].Hero.Mind = 200
	scrollDescriptionCheckPack(t, f, app, screen, source, expected)
	store, _, _ := campaignSave(t, f, false)
	cold, coldApp := campaignCold(t, store)
	t.Cleanup(coldApp.StopAudio)
	coldApp.Layout(1024, 768)
	quest81OK(t, coldApp.HeadlessActivate("SHOP"))
	coldScreen := registerTownFront(cold.TownScreen().(*townScreen), cold)
	coldScreen.CloseTip()
	scrollDescriptionCheckPack(t, cold, coldApp, coldScreen, source, expected)
	scrollDescriptionSellRestoredPack(t, cold, coldApp, coldScreen, source)
	t.Logf("27 ordinary families and Slow descriptor control retain source fields, fixed power, quantities1/3, App pixels, cold SAV and next stored-price sale; %d source tuples", len(source))
}

func scrollDescriptionCheckPack(t *testing.T, f *FrontEnd, app *ui.App, screen *townScreen,
	source []sim.ItemStack, expected map[uint16][]string) {
	t.Helper()
	app.SetTooltipDelayPreference(0, nil)
	for _, stack := range source {
		item := stack.Instance()
		spell, _, ok := sim.ScrollSpell(item)
		if !ok {
			t.Fatal("source is not a scroll", item)
		}
		want, ok := expected[spell]
		if !ok {
			t.Fatal("source spell has no independent description expectation", spell)
		}
		stacks := screen.shopPackStacks()
		index := slices.IndexFunc(stacks, func(value sim.ItemStack) bool {
			return value.Count == stack.Count && staffSaleValueEqual(value.Instance(), item)
		})
		if index < 0 {
			t.Fatal("pack lost source item fields or quantity", stack, stacks)
		}
		cell := shopScrollPackCell(t, app, screen, index)
		view := screen.ShopScreen()
		wantSell := (item.Price + 1) / 2
		if got := view.Pack[cell]; !slices.Equal(got.Info, want) || got.Count != stack.Count || got.Price != wantSell {
			t.Fatalf("spell%d pack info/count/price = %+v, want %q x%d price%d", spell, got, want, stack.Count, wantSell)
		}
		x, y, err := app.HeadlessShopPoint("pack", cell)
		quest81OK(t, err)
		quest81OK(t, app.HeadlessPointer("hover", x, y))
		state, pic := app.HeadlessTooltip()
		wantPic, _, drawable := ui.ComposeTooltipHint(want, f.tipFont(), image.Pt(x, y), image.Rect(0, 0, 1024, 768), f.HoverBall())
		if !state.Visible || pic == nil || !drawable || pic.Bounds() != wantPic.Bounds() || !bytes.Equal(pic.Pix, wantPic.Pix) {
			t.Fatal("pack hover did not draw the independent expected description", spell, state)
		}
	}
}

func scrollDescriptionSellRestoredPack(t *testing.T, f *FrontEnd, app *ui.App, screen *townScreen, source []sim.ItemStack) {
	t.Helper()
	for _, stack := range source {
		item := stack.Instance()
		index := slices.IndexFunc(screen.shopPackStacks(), func(value sim.ItemStack) bool {
			return value.Count == stack.Count && staffSaleValueEqual(value.Instance(), item)
		})
		if index < 0 {
			t.Fatal("restored pack lost the source sale tuple", stack)
		}
		cell := shopScrollPackCell(t, app, screen, index)
		cityShopPointer(t, app, "pack", cell, "shift-press", "shift-release")
		placed := f.Shop.Table()
		if len(placed) != 1 || placed[0].ShopItem.Count != int32(stack.Count) || !placed[0].Mine || !staffSaleValueEqual(placed[0].ShopItem.Instance(), item) {
			t.Fatal("next App action lost the restored sale tuple", stack, placed)
		}
		gold := f.Town.Gold()
		cityShopPointer(t, app, "button", 2, "press", "release")
		wantPaid := (item.Price*int32(stack.Count) + 1) / 2
		if f.Town.Gold()-gold != int(wantPaid) || len(f.Shop.Table()) != 0 {
			t.Fatal("next sale lost stored price, quantity, or table completion", stack, f.Town.Gold()-gold, wantPaid, f.Shop.Table())
		}
	}
}
