package menu_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/render/menu"
	"againrom/pkg/screenart"
)

// menuFrame is a state's overlay rectangle, overlay flag and frame hash.
type menuFrame struct {
	selected int
	pressed  bool
	at       image.Rectangle
	ok       bool
	hash     string
}

// Captured from the two loaders the layout rows replaced, over layoutSet.
var captureFirst = []menuFrame{
	{0, false, image.Rect(0, 0, 0, 0), false, "fbfb9eb5d415708fd62acb27cf08608e5bf0da4951d7fd88c68cecf0ceff16cf"},
	{1, false, image.Rect(112, 64, 324, 200), true, "5800a7a3bdb4c9dc501f9c48b5f7a968efd330cd9685fc2671f4466010993515"},
	{1, true, image.Rect(116, 64, 324, 202), true, "806408c3336e03f8f5354f59c0979d8c8fa5bab9365603d8d5efc45c5c2df44d"},
	{2, false, image.Rect(84, 88, 320, 236), true, "a6b08cffb4bbb7484794b7c132a087421be0f04e3c0a1671db990da5570c4a21"},
	{2, true, image.Rect(88, 88, 324, 240), true, "f2f67e9b0cb5b661e3eb9e88db98892e2445bcb954d5912fa53e9c6285088c6e"},
	{3, false, image.Rect(84, 236, 320, 388), true, "58332b3fd56c8041632de645c98442ceb3c6926b35edad3bf5744d46f864758b"},
	{3, true, image.Rect(88, 236, 324, 388), true, "54443eca6ba32512802607da8a70e50de4ffc446882d94d3821659e42428af73"},
	{4, false, image.Rect(112, 276, 324, 412), true, "c03e6ca91ecdbcd9f437dabeb1f20ca1025148ceb29e1de9694050d8f54f71cc"},
	{4, true, image.Rect(116, 272, 324, 412), true, "efa319ff05cbfe027feb94a53499933f3ca1c1ac4c0c846c7f93273c9051533a"},
	{5, false, image.Rect(320, 60, 532, 200), true, "65a613ef5d2a62c54e1384c6d1eca93955ba4bc23b3a4ed3fe93cbda9c82a46e"},
	{5, true, image.Rect(320, 64, 532, 204), true, "3cd5f7bbda69f883fc136f45db565574eeecd0c6e5923c746d10bb268f521268"},
	{6, false, image.Rect(324, 88, 560, 236), true, "dd369fb310d3ed39d028f2bc2c278c4cf7d127ac2f257a896abbb52e11acde51"},
	{6, true, image.Rect(324, 88, 556, 240), true, "d6726d03560908b988b39b8d699fa652c86a1d1410217f470b605157d5315124"},
	{7, false, image.Rect(324, 236, 560, 388), true, "efb7aa29fb668d68107aa60da3304bcfaac69eaa1e3cc9dfb9db47e24129b9b2"},
	{7, true, image.Rect(324, 236, 560, 388), true, "079995b0fa205c4692d0bf55c65bb5d5570a6a204032836077e87bcca7689ca7"},
	{8, false, image.Rect(324, 276, 532, 412), true, "1e953c2d42f12c2f7565f6f7a0d4023b0b8189befd09eb6b2d95a0b664cf01dd"},
	{8, true, image.Rect(320, 272, 532, 412), true, "fb6d918335f3b02df0d8a0cfe248656fd1d37e093c93b47e649e0f185e5e44be"},
}

var captureSecond = []menuFrame{
	{0, false, image.Rect(0, 0, 0, 0), false, "fbfb9eb5d415708fd62acb27cf08608e5bf0da4951d7fd88c68cecf0ceff16cf"},
	{1, false, image.Rect(204, 52, 308, 148), true, "ece318690a4641efe4b87e31ac64d2d1969d7d3abdfc166d7a4628f807a745d2"},
	{1, true, image.Rect(204, 52, 308, 148), true, "f039af743dc2246729210f4013064efd06863371f770012bf85f7e1b907090bf"},
	{2, false, image.Rect(124, 156, 232, 232), true, "1a8875cc5e607512b4ca268baef8b2e78a4151c8b39677c4309437ea61326c54"},
	{2, true, image.Rect(124, 156, 232, 232), true, "be4720f957381ba0248eca433fcb6d33bead82ee877d0fbd9cd1e6fc3b268619"},
	{3, false, image.Rect(124, 252, 220, 340), true, "8e0fecf75b2fc93d3b7b1dfe9757f31a88fc683007727f0d89d81375936b63e3"},
	{3, true, image.Rect(124, 252, 220, 340), true, "4201b61c3d427b464367c64ac21b11d6cf6cf14d3e672fe944bfc98cae9842ff"},
	{4, false, image.Rect(208, 340, 308, 440), true, "0a14cabc8a17f956bcc6d97b82a46ea5492c900034586596d1e5a7122fa73995"},
	{4, true, image.Rect(208, 340, 308, 440), true, "41b7c4d930d7456d4f4e5d2c0662d3fbe4af5aa6da845e0d4ac3338a4cdb0789"},
	{5, false, image.Rect(340, 52, 428, 152), true, "1ec6642aa71ceb0612f4436a4b59679bf8749ea40824a23f78e9f06de283da74"},
	{5, true, image.Rect(340, 52, 428, 152), true, "17b381eb8798bd6240cca75bac8d6696bd42ce874be2f4fe2a2a4909460cd851"},
	{6, false, image.Rect(424, 152, 508, 240), true, "36db1062d15ec6ad4da394dc98e28ce53576aabc31c80576026352865cade583"},
	{6, true, image.Rect(424, 152, 508, 240), true, "b735e511c86a08ef4985085b898cc6af132e87dfabf6f21d92f843ab43b328aa"},
	{7, false, image.Rect(412, 260, 508, 344), true, "284110b0c3e76a2bb7805015578c9bea3d831e76d6f8c103b83c61535c65cbd4"},
	{7, true, image.Rect(412, 260, 508, 344), true, "04accd8659c23ef16ec80f12e8cc8346cd5a1a7d18e1b9529a66c244d959c10d"},
	{8, false, image.Rect(344, 348, 416, 428), true, "6dc90eb48e4f8404441527b1336cdce37d4c346e1576ade7d03606057e27236d"},
	{8, true, image.Rect(344, 348, 416, 428), true, "2473a908402bc52d0e935483d70acdead89653ef19393a5ae6057573f9f1495b"},
}

const (
	captureFirstBindings  = `menu.Actions{Buttons:[8]menu.Button{menu.Button{Action:0x1, Name:"new game"}, menu.Button{Action:0x0, Name:""}, menu.Button{Action:0x4, Name:"cutscenes"}, menu.Button{Action:0x5, Name:"credits"}, menu.Button{Action:0x2, Name:"load game"}, menu.Button{Action:0x0, Name:""}, menu.Button{Action:0x3, Name:"hall of fame"}, menu.Button{Action:0x6, Name:"exit"}}, LoadNeedsSave:false, EscapeStays:false, HallKeepsEscape:false, CaptionAfterMove:false}`
	captureSecondBindings = `menu.Actions{Buttons:[8]menu.Button{menu.Button{Action:0x1, Name:"new game"}, menu.Button{Action:0x0, Name:"multi player"}, menu.Button{Action:0x4, Name:"cutscenes"}, menu.Button{Action:0x5, Name:"credits"}, menu.Button{Action:0x2, Name:"load game"}, menu.Button{Action:0x0, Name:"server"}, menu.Button{Action:0x3, Name:"hall of fame"}, menu.Button{Action:0x6, Name:"exit"}}, LoadNeedsSave:true, EscapeStays:true, HallKeepsEscape:true, CaptionAfterMove:true}`
)

func layoutSolid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
	return img
}

// layoutSet is solid-colour art, one colour per entry; captions adds caption/0..7.
func layoutSet(captions bool) *screenart.Set {
	set := &screenart.Set{
		Frames: map[string][]image.Image{"base": {layoutSolid(640, 480, color.RGBA{1, 1, 1, 255})}},
		Masks:  map[string]*image.Paletted{"mask": image.NewPaletted(image.Rect(0, 0, 640, 480), color.Palette{color.Black})},
	}
	for i := 0; i < menu.ButtonCount; i++ {
		set.Frames[fmt.Sprintf("hover/%d", i)] = []image.Image{layoutSolid(300, 200, color.RGBA{uint8(10 + i), 1, 2, 255})}
		set.Frames[fmt.Sprintf("pressed/%d", i)] = []image.Image{layoutSolid(300, 200, color.RGBA{uint8(10 + i), 3, 4, 255})}
		if captions {
			set.Frames[fmt.Sprintf("caption/%d", i)] = []image.Image{layoutSolid(180, 80, color.RGBA{uint8(10 + i), 5, 6, 255})}
		}
	}
	return set
}

func checkRow(t *testing.T, a *menu.Assets, want []menuFrame, bindings string) {
	t.Helper()
	for _, w := range want {
		s := menu.State{Selected: w.selected, Pressed: w.pressed}
		_, at, ok := a.Overlay(s)
		if at != w.at || ok != w.ok {
			t.Errorf("button %d pressed %v: overlay %v ok %v, want %v ok %v", w.selected, w.pressed, at, ok, w.at, w.ok)
		}
		sum := sha256.Sum256(a.Compose(s).Pix)
		if got := hex.EncodeToString(sum[:]); got != w.hash {
			t.Errorf("button %d pressed %v: frame hash %s, want %s", w.selected, w.pressed, got, w.hash)
		}
	}
	if got := fmt.Sprintf("%#v", a.Bindings()); got != bindings {
		t.Errorf("bindings %s, want %s", got, bindings)
	}
}

// The first row names no captions, so caption art in the set changes nothing.
func TestFirstLayoutReproducesTheFirstLoader(t *testing.T) {
	for _, captions := range []bool{false, true} {
		a, err := menu.Load(layoutSet(captions), menu.FirstLayout)
		if err != nil {
			t.Fatal(err)
		}
		checkRow(t, a, captureFirst, captureFirstBindings)
	}
}

func TestSecondLayoutReproducesTheSecondLoader(t *testing.T) {
	a, err := menu.Load(layoutSet(true), menu.SecondLayout)
	if err != nil {
		t.Fatal(err)
	}
	checkRow(t, a, captureSecond, captureSecondBindings)
}

func TestSecondLayoutNeedsItsCaptionArt(t *testing.T) {
	_, err := menu.Load(layoutSet(false), menu.SecondLayout)
	if err == nil || err.Error() != `menu: art entry "caption/0" is not an RGBA picture` {
		t.Fatalf("Load without caption art = %v", err)
	}
}
