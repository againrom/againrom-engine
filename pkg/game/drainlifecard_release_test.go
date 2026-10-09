package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"

	"golang.org/x/text/encoding/charmap"
)

func drainLifeCardWitnessDir(t *testing.T) string {
	t.Helper()
	base := os.Getenv("AGAINROM_DRAIN_LIFE_CARD_WITNESS_DIR")
	if base == "" {
		t.Skip("AGAINROM_DRAIN_LIFE_CARD_WITNESS_DIR is required for the installed Drain Life card witness")
	}
	if !filepath.IsAbs(base) {
		t.Fatal("Drain Life witness output must be absolute")
	}
	assets := filepath.Clean(os.Getenv("AGAINROM_ASSETS"))
	if relative, err := filepath.Rel(assets, base); err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("Drain Life witness output must remain outside the install")
	}
	namespace := fmt.Sprintf("%x", sha256.Sum256([]byte(assets)))
	dir := filepath.Join(base, namespace)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReleaseDrainLifeItemCardUsesLiteralInstalledNamesDamageAndRange(t *testing.T) {
	f := releaseFront(t)
	dir := drainLifeCardWitnessDir(t)
	app, screen := releaseShopApp(t)
	defer app.StopAudio()
	f = frontOf(screen)
	screen.CloseTip()
	app.SetTooltipDelayPreference(0, nil)
	const spellID uint16 = 11
	const power int32 = 30
	item := mapload.ItemInstanceFromCode(0x812d, f.Table)
	item.Effects = []sim.ItemEffect{{Kind: 41, Operand: uint32(spellID) | uint32(uint16(power))<<16}}
	beforeItem, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	row := f.Table.Spells.EntryParams(int(spellID))
	if len(row) <= 17 || row[6] <= 0 || row[6] > 255 || row[16] <= 0 || row[17] < row[16] {
		t.Fatalf("installed Drain Life row has no bounded range/damage pair: %v", row)
	}
	nameUTF8, rawNameUTF8, wrongName := "Drain Life", "Drain Life", "Heal"
	nameBytes, rawNameBytes, encoding := nameUTF8, rawNameUTF8, "ASCII"
	if f.textSelector() == 1 {
		nameUTF8, rawNameUTF8, wrongName, encoding = "Вампиризм", "Высасывание жизни", "Drain Life", "CP866"
		nameBytes, err = charmap.CodePage866.NewEncoder().String(nameUTF8)
		if err != nil {
			t.Fatal(err)
		}
		rawNameBytes, err = charmap.CodePage866.NewEncoder().String(rawNameUTF8)
		if err != nil {
			t.Fatal(err)
		}
	}
	nameInput := rawTextLine(t, f, SpellNamesTextPath, 10)
	castCaption := rawTextLine(t, f, MainTextPath, 92)
	damageCaption := rawTextLine(t, f, StatsTextPath, 43)
	rangeCaption := rawTextLine(t, f, StatsTextPath, 38)
	minimum := int64(row[16]) * (int64(power) + 30) / 30
	maximum := minimum + max(int64(0), int64(row[17])*(int64(power)+30)/30-minimum)
	bookWant := itemName(data.ItemCode(0x0e17), f.Table) + " " + rawTextLine(t, f, MainTextPath, 90) + " " + nameBytes + rawTextLine(t, f, MainTextPath, 91)
	book := itemInstanceInfoLines(gameSpellBook(11), f.Table, f.Words)
	bookControl := slices.Equal(book, []string{bookWant})
	teach := item.Clone()
	teach.Effects[0] = sim.ItemEffect{Kind: 42, Operand: 11}
	teachLines := itemInstanceInfoLines(teach, f.Table, f.Words)
	teachWant := rawTextLine(t, f, MainTextPath, 189) + " " + rawTextLine(t, f, MainTextPath, 90) + " " + nameBytes + rawTextLine(t, f, MainTextPath, 91)
	teachControl := teachLines[len(teachLines)-1] == teachWant
	for _, line := range teachLines {
		if line == fmt.Sprintf("%s %d-%d", damageCaption, minimum, maximum) || line == fmt.Sprintf("%s %d", rangeCaption, row[6]) {
			teachControl = false
		}
	}
	flagControl := false
	for _, rule := range mapload.SpellRules(f.Table) {
		if rule.ID == 11 {
			flagControl = !rule.Damaging && !rule.Restorative
		}
	}
	want := []string{itemName(data.ItemCode(item.Code), f.Table), rawTextLine(t, f, MainTextPath, 189),
		castCaption + " " + nameBytes, fmt.Sprintf("%s %d-%d", damageCaption, minimum, maximum),
		fmt.Sprintf("%s %d", rangeCaption, row[6])}
	f.Shop.shelves[ShelfMagic] = []ShopItem{shopItemFromInstance(item, 1)}
	x, y, err := app.HeadlessShopPoint("shelf_pick", 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	view := screen.ShopScreen()
	x, y, err = app.HeadlessShopPoint("shelf", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	state, actual := app.HeadlessTooltip()
	compose := func(lines []string) *image.RGBA {
		pic, _, ok := ui.ComposeTooltipHint(lines, f.tipFont(), image.Pt(x, y), image.Rect(0, 0, 640, 480), f.HoverBall())
		if !ok || pic == nil {
			t.Fatal("independent expected card did not compose")
		}
		return pic
	}
	expected := compose(want)
	writePicture := func(name string, pic *image.RGBA) {
		if pic == nil {
			return
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, pic); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".png"), encoded.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writePicture("actual", actual)
	writePicture("expected", expected)
	wrong := slices.Clone(want)
	wrong[2] = castCaption + " " + wrongName
	losses := map[string]bool{}
	for _, loss := range []struct {
		name  string
		lines []string
	}{
		{"wrong-name", wrong},
		{"without-damage", append(slices.Clone(want[:3]), want[4:]...)},
		{"without-range", slices.Clone(want[:4])},
	} {
		pic := compose(loss.lines)
		losses[loss.name] = !samePicture(expected, pic)
		writePicture(loss.name, pic)
	}
	afterItem, err := json.Marshal(f.Shop.Shelf(ShelfMagic)[0].Instance())
	if err != nil {
		t.Fatal(err)
	}
	linesHex := func(lines []string) []string {
		out := make([]string, len(lines))
		for i, line := range lines {
			out[i] = hex.EncodeToString([]byte(line))
		}
		return out
	}
	manifest := struct {
		ActualAssetRoot          string         `json:"actualAssetRoot"`
		SyntheticEffect          sim.ItemEffect `json:"explicitSyntheticEffect"`
		SyntheticSpellID         uint16         `json:"syntheticSpellID"`
		SyntheticPower           int32          `json:"syntheticPower"`
		ItemCode                 uint16         `json:"itemCode"`
		RawRange, RawMin, RawMax int32
		RawSpellParameters       []int32 `json:"rawInstalledSpellParameters"`
		RawSpellNameUTF8         string  `json:"rawSpellNameUTF8"`
		RawSpellNameEncoding     string  `json:"rawSpellNameEncoding"`
		RawSpellNameSHA256       string  `json:"rawSpellNameSHA256"`
		CastCaptionSource        string  `json:"castCaptionSource"`
		WantedSpellNameUTF8      string  `json:"wantedSpellNameUTF8"`
		InstalledSpellNameHex    string  `json:"installedSpellNameHex"`
		WantedMin, WantedMax     int64
		ExpectedLinesHex         []string `json:"expectedLinesHex"`
		ActualLinesHex           []string `json:"actualLinesHex"`
		RawNameMatches           bool     `json:"rawNameMatchesIndependentSourceLiteral"`
		BookCaptionOnly          bool     `json:"bookCaptionOnly"`
		TeachCaptionOnly         bool     `json:"kind42CaptionOnly"`
		DrainFlagsUnchanged      bool     `json:"drainGameplayFlagsUnchanged"`
		Visible, PixelsMatch     bool
		CompleteItemUnchanged    bool            `json:"completeItemUnchanged"`
		LossControls             map[string]bool `json:"lossControlsChangePixels"`
	}{ActualAssetRoot: os.Getenv("AGAINROM_ASSETS"), SyntheticEffect: item.Effects[0], SyntheticSpellID: spellID,
		SyntheticPower: power, ItemCode: item.Code, RawRange: row[6], RawMin: row[16], RawMax: row[17],
		RawSpellParameters: slices.Clone(row), RawSpellNameUTF8: rawNameUTF8, RawSpellNameEncoding: encoding,
		RawSpellNameSHA256:  fmt.Sprintf("%x", sha256.Sum256([]byte(nameInput))),
		CastCaptionSource:   "main/text/main.txt[92]",
		WantedSpellNameUTF8: nameUTF8, InstalledSpellNameHex: hex.EncodeToString([]byte(nameInput)),
		WantedMin: minimum, WantedMax: maximum, ExpectedLinesHex: linesHex(want), ActualLinesHex: linesHex(view.Shelf[0].Info),
		RawNameMatches: nameInput == rawNameBytes, BookCaptionOnly: bookControl, TeachCaptionOnly: teachControl,
		DrainFlagsUnchanged: flagControl, Visible: state.Visible,
		PixelsMatch: samePicture(actual, expected), CompleteItemUnchanged: bytes.Equal(beforeItem, afterItem), LossControls: losses}
	report, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), report, 0o644); err != nil {
		t.Fatal(err)
	}
	if !manifest.RawNameMatches || !manifest.Visible || !manifest.PixelsMatch || !manifest.CompleteItemUnchanged || !slices.Equal(view.Shelf[0].Info, want) || !bookControl || !teachControl || !flagControl {
		t.Errorf("Drain Life shelf card: rawName=%v visible=%v pixels=%v itemUnchanged=%v book=%v kind42=%v flags=%v; got %q want %q",
			manifest.RawNameMatches, manifest.Visible, manifest.PixelsMatch, manifest.CompleteItemUnchanged, bookControl, teachControl, flagControl, view.Shelf[0].Info, want)
	}
	for name, changed := range losses {
		if !changed {
			t.Errorf("loss control %s did not change the expected pixels", name)
		}
	}
	t.Logf("explicit synthetic kind41 spell11 power30 code=%#04x raw=%d-%d/range%d wanted=%d-%d name=%q output=%s",
		item.Code, row[16], row[17], row[6], minimum, maximum, nameUTF8, dir)
}
