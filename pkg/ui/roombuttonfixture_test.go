package ui

import (
	"againrom/pkg/render/text"
	"image"
)

// DrawRoomButtons draws a composed room page's buttons through the button
// panel builder: each off picture at its rectangle and its caption centred
// and cut to fit, in the style's font and ink, else in f and the town
// caption ink. A button with no on picture keeps its off picture held.
func DrawRoomButtons(dst *image.RGBA, f *text.Font, style *RoomButtonStyle, buttons []RoomButton) {
	DrawRoomPage(dst, roomButtonsPage(f, style, buttons))
}

func roomButtonsPage(f *text.Font, style *RoomButtonStyle, buttons []RoomButton) RoomPage {
	comp := panelComposition{Plaques: make([]image.Rectangle, len(buttons)), FitLine: true,
		Ink: roomButtonInk}
	art := panelArt{Plaques: make([][2]image.Image, len(buttons))}
	in := make([]panelButton, len(buttons))
	for i, b := range buttons {
		comp.Plaques[i] = b.Rect
		art.Plaques[i] = [2]image.Image{b.Off, b.On}
		if b.On == nil {
			art.Plaques[i][1] = b.Off
		}
		if b.Caption != "" {
			in[i].Captions = []string{b.Caption}
		}
		in[i].Hover, in[i].Inside = b.Hover, b.Hover
	}
	return RoomPage{Font: f, Layers: []RoomLayer{{Kind: RoomLayerButtons}}, Panel: RoomPanel{Composition: comp, Art: art, Buttons: in, Style: style}}
}
