package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// manyRowsTown is a row-list room with more rows than the room shows at once,
// some choosable and some not.
type manyRowsTown struct{ fakeTown }

func (manyRowsTown) Rows() []TownRow {
	rows := make([]TownRow, 30)
	for i := range rows {
		rows[i] = TownRow{Text: "row " + strconv.Itoa(i), Choosable: i%3 != 1}
	}
	return rows
}

// townListFrameCase is one row-list room: its model, a message and how far
// the selection moves before the frame is taken.
type townListFrameCase struct {
	name  string
	town  TownScreen
	msg   string
	moves int
	// golden is the SHA-256 of the room's pixels.
	golden string
}

// The golden hashes are the frames the row-list room produced before its
// buttons were painted through the shared push button painter.
var townListFrameCases = []townListFrameCase{
	{name: "square-first", town: &fakeSquareTown{}, msg: "", moves: 0,
		golden: "92772c84c9b916ee27fc41ff6b53b837dde21310629e31c79a7a2c561375640a"},
	{name: "square-second", town: &fakeSquareTown{}, msg: "a message", moves: 1,
		golden: "f50c0850d1eaaf1f396b8a64951f0b43a5630faf859e9f96b8dfe0a6ddb3256f"},
	{name: "square-third", town: &fakeSquareTown{}, msg: "", moves: 2,
		golden: "8fa3d9e5fda2b966ae13f30fa5fe161bbba3e59cb8f2b6e599e9200ba01f1e12"},
	{name: "inside-first", town: &fakeSquareTown{fakeTown{inside: true}}, msg: "", moves: 0,
		golden: "1ab095db6da02b1353d6845e079b6280edbd7833a57863671b9914d7b948d943"},
	{name: "inside-moved", town: &fakeSquareTown{fakeTown{inside: true}}, msg: "inside message", moves: 1,
		golden: "a902005efdcab7db03a9f05270867a49d27a71e910844ebccb9e7057c41a974f"},
	{name: "plain-rows", town: &fakeTown{}, msg: "", moves: 1,
		golden: "1a8daeec146535366df1e33eb57fdad7cdc8a285ad2a75b6b45f0a55c5c59117"},
	{name: "many-rows-top", town: &manyRowsTown{}, msg: "", moves: 0,
		golden: "c13400a5c8c7838f2be58f3ee9e8a4fc7fd4145e6032f642adab302762ec2ff7"},
	{name: "many-rows-scrolled", town: &manyRowsTown{}, msg: "scrolled", moves: 12,
		golden: "00b7d33d5b48908504075338d05aa2dc33138c6a5272110c7d9388f8fd39813a"},
}

func townListFrameApp(t *testing.T, c townListFrameCase) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(c.town)
	if !a.flow.showTown(c.msg) {
		t.Fatal("showTown refused the town")
	}
	for i := 0; i < c.moves; i++ {
		a.flow.townList.MoveChoice(1)
	}
	return a
}

// canvasFrame reads the whole canvas back from the pixel log, failing when a
// cell is not exactly known.
func canvasFrame(t *testing.T, a *App) *image.RGBA {
	t.Helper()
	b := a.canvas.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c, ok := a.canvasLog.value(len(a.canvasLog.ops), x, y)
			if !ok {
				t.Fatalf("canvas pixel (%d,%d) is not exactly known", x, y)
			}
			out.SetRGBA(x, y, c)
		}
	}
	return out
}

// The row-list room's frame is the same on the CPU compose and on the canvas
// the app draws, and both equal the frame captured before the buttons moved
// to the shared painter.
func TestTownRowListRoomFramesAreUnchanged(t *testing.T) {
	for _, c := range townListFrameCases {
		t.Run(c.name, func(t *testing.T) {
			a := townListFrameApp(t, c)
			cpu, err := a.composeTownRoom()
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(cpu.Pix)
			got := hex.EncodeToString(sum[:])
			if got != c.golden {
				t.Errorf("CPU frame hash = %s, want %s", got, c.golden)
			}

			a.canvas = ebiten.NewImage(frame.W, frame.H)
			a.canvasLog.reset(a.canvas.Bounds())
			a.drawTown()
			a.canvasLog.end()
			gpu := canvasFrame(t, a)
			if gpu.Rect != cpu.Rect || string(gpu.Pix) != string(cpu.Pix) {
				t.Error("the canvas frame differs from the CPU frame")
			}
		})
	}
}

// Every button of the row-list room is one shared push button call carrying
// the row-list style, selected only on the selected row.
func TestTownRowListButtonsAreSharedPushButtons(t *testing.T) {
	a := townListFrameApp(t, townListFrameCase{town: &fakeSquareTown{}, moves: 1})
	var calls []pushButton
	widgetRecord = func(c widgetCall) {
		if b, ok := c.state.(pushButton); ok && c.kind == widgetPushButton {
			calls = append(calls, b)
		}
	}
	defer func() { widgetRecord = nil }()
	if _, err := a.composeTownRoom(); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 {
		t.Fatalf("%d push button calls, want the 3 square rows", len(calls))
	}
	for i, b := range calls {
		if b.Style != &pushButtonRowList {
			t.Errorf("button %d does not carry the row-list style", i)
		}
		if want := i == 1; b.Selected != want {
			t.Errorf("button %d selected = %v, want %v", i, b.Selected, want)
		}
	}
}

// The row-list style paints its fill, the selected fill and a two-pixel
// border on all four edges, inside the button's rectangle only.
func TestPushButtonStylePaintsFillAndBorder(t *testing.T) {
	st := pushButtonRowList
	for _, selected := range []bool{false, true} {
		pix := image.NewRGBA(image.Rect(0, 0, 40, 30))
		r := image.Rect(5, 6, 35, 24)
		drawPushButton(pix, nil, pushButton{Rect: r, Style: &st, Selected: selected})
		fill := st.Fill
		if selected {
			fill = st.Selected
		}
		for y := 0; y < 30; y++ {
			for x := 0; x < 40; x++ {
				want := color.RGBA{}
				if p := image.Pt(x, y); p.In(r) {
					want = fill
					if !p.In(r.Inset(st.Width)) {
						want = st.Border
					}
				}
				if got := pix.RGBAAt(x, y); got != want {
					t.Fatalf("selected=%v pixel (%d,%d) = %v, want %v", selected, x, y, got, want)
				}
			}
		}
	}
}
